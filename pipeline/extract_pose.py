#!/usr/bin/env python3
"""extract_pose — 영상 파일 → 랜드마크 시계열(JSON) + 품질 리포트.

역할 경계: 이 스크립트는 **랜드마크 추출만** 한다. 골격 스트림 조립은
:core-stream 의 SkeletonStreamBuilder 가 담당하고(계약 구현체를 늘리지 않는다),
검증은 Go 의 cmd/validate 가 한다. 파이프라인:

    영상 → extract_pose.py → landmarks.json → :core-stream buildStream → cmd/validate → cmd/seed

편집된 영상(컷·슬로모션)에서 레퍼런스를 뽑을 때를 대비해 **불연속 구간을 탐지해 보고**한다.
컷을 넘어간 구간을 한 스윙으로 오인하면 계약은 통과해도 쓸모없는 레퍼런스가 된다.

사용:
    python extract_pose.py IN.mp4 OUT.json [--start 12.0] [--end 14.5] [--model PATH]
"""

from __future__ import annotations

import argparse
import json
import pathlib
import sys

import cv2
import mediapipe as mp
from mediapipe.tasks import python as mp_python
from mediapipe.tasks.python import vision

KEYPOINT_COUNT = 33  # blazepose_33 — 계약이 요구하는 토폴로지
DEFAULT_MODEL = pathlib.Path(__file__).parent / "models" / "pose_landmarker_full.task"

# 프레임 간 평균 이동량이 이 값을 넘으면 편집 컷/장면 전환으로 의심한다
# (정규화 좌표 기준. 실제 스윙의 프레임 간 이동은 통상 이보다 훨씬 작다.)
DISCONTINUITY_THRESHOLD = 0.15


def build_landmarker(model_path: pathlib.Path) -> vision.PoseLandmarker:
    if not model_path.exists():
        sys.exit(f"모델 파일 없음: {model_path}\n  README 의 모델 다운로드 절차를 확인하세요.")
    options = vision.PoseLandmarkerOptions(
        base_options=mp_python.BaseOptions(model_asset_path=str(model_path)),
        running_mode=vision.RunningMode.VIDEO,
        num_poses=1,  # MVP: 단일 인물. 다인 클립은 레퍼런스로 쓰지 않는다.
        output_segmentation_masks=False,
    )
    return vision.PoseLandmarker.create_from_options(options)


def mean_displacement(prev: list[dict], cur: list[dict]) -> float:
    total = sum(abs(a["x"] - b["x"]) + abs(a["y"] - b["y"]) for a, b in zip(prev, cur))
    return total / len(cur)


def extract(video: pathlib.Path, model: pathlib.Path, start: float, end: float | None) -> dict:
    cap = cv2.VideoCapture(str(video))
    if not cap.isOpened():
        sys.exit(f"영상을 열 수 없음: {video}")

    fps = cap.get(cv2.CAP_PROP_FPS)
    total = int(cap.get(cv2.CAP_PROP_FRAME_COUNT))
    if not fps or fps <= 0:
        sys.exit("영상 fps 를 읽을 수 없음 — 파일이 손상됐거나 코덱 미지원")

    landmarker = build_landmarker(model)
    frames: list[dict] = []
    seen = detected = incomplete = 0
    discontinuities: list[float] = []

    while True:
        ok, frame_bgr = cap.read()
        if not ok:
            break
        t = seen / fps
        seen += 1
        if t < start:
            continue
        if end is not None and t > end:
            break

        rgb = cv2.cvtColor(frame_bgr, cv2.COLOR_BGR2RGB)
        mp_image = mp.Image(image_format=mp.ImageFormat.SRGB, data=rgb)
        result = landmarker.detect_for_video(mp_image, int(t * 1000))

        if not result.pose_landmarks:
            continue  # 인물 미검출 프레임은 버린다
        detected += 1

        lms = result.pose_landmarks[0]
        if len(lms) != KEYPOINT_COUNT:
            incomplete += 1  # 계약 INV-4 를 만족할 수 없는 프레임
            continue

        pts = [
            {
                "x": round(float(lm.x), 6),
                "y": round(float(lm.y), 6),
                "z": round(float(lm.z), 6),
                "visibility": round(float(lm.visibility), 6),
            }
            for lm in lms
        ]
        if frames and mean_displacement(frames[-1]["landmarks"], pts) > DISCONTINUITY_THRESHOLD:
            discontinuities.append(round(t, 3))
        frames.append({"t": round(t, 6), "landmarks": pts})

    cap.release()
    landmarker.close()

    avg_vis = 0.0
    if frames:
        vis = [lm["visibility"] for f in frames for lm in f["landmarks"]]
        avg_vis = round(sum(vis) / len(vis), 4)

    return {
        "meta": {
            "source_file": video.name,
            "fps": round(float(fps), 6),
            "video_frames_total": total,
            "frames_scanned": seen,
            "frames_with_pose": detected,
            "frames_incomplete_topology": incomplete,
            "frames_kept": len(frames),
            "avg_visibility": avg_vis,
            "discontinuities_at": discontinuities,
            "model": "blazepose",
            "model_variant": "full",
            "keypoint_topology": "blazepose_33",
        },
        "frames": frames,
    }


def main() -> None:
    ap = argparse.ArgumentParser(description="영상에서 33 랜드마크 시계열을 추출한다")
    ap.add_argument("video", type=pathlib.Path)
    ap.add_argument("out", type=pathlib.Path)
    ap.add_argument("--start", type=float, default=0.0, help="추출 시작 초")
    ap.add_argument("--end", type=float, default=None, help="추출 종료 초")
    ap.add_argument("--model", type=pathlib.Path, default=DEFAULT_MODEL)
    args = ap.parse_args()

    data = extract(args.video, args.model, args.start, args.end)
    args.out.write_text(json.dumps(data, ensure_ascii=False) + "\n")

    m = data["meta"]
    print(f"■ {m['source_file']} → {args.out.name}")
    print(f"  fps {m['fps']} · 스캔 {m['frames_scanned']} · 인물검출 {m['frames_with_pose']}"
          f" · 토폴로지불완전 {m['frames_incomplete_topology']} · 채택 {m['frames_kept']}")
    print(f"  평균 visibility {m['avg_visibility']}")
    if m["frames_scanned"]:
        rate = m["frames_kept"] / m["frames_scanned"] * 100
        print(f"  채택률 {rate:.1f}%")
    if m["discontinuities_at"]:
        print(f"  ⚠ 불연속(편집 컷 의심) {len(m['discontinuities_at'])}건 @ {m['discontinuities_at'][:8]}")
        print("    → 한 스윙이 컷을 넘어가면 레퍼런스로 쓸 수 없다. --start/--end 로 구간을 좁히세요.")
    if not data["frames"]:
        sys.exit("채택 프레임 0 — 레퍼런스로 쓸 수 없음 (인물이 전신으로 보이는 구간인지 확인)")


if __name__ == "__main__":
    main()
