#!/usr/bin/env python3
"""dev_engine_server — 개발용 엔진 스텁 (상주 프로세스, 라인 JSON 프로토콜).

⚠️ 검증된 분석 엔진이 아니다. 파이프라인(워커↔엔진 경계·큐·저장) 검증 전용.
   이 출력으로 Golden 을 승인하거나 코칭 품질을 판단하지 않는다.
   실엔진(engine/ 원본, 포핸드 오프라인 검증 완료본) 확보 시 ENGINE_SCRIPT 만 교체한다.

프로토콜 (internal/analysis/engine.go 와 계약):
  요청  {"user": <stream>, "ref": <stream|null>, "ref_key": "<id>"}  # ref 는 키당 최초 1회만
  응답  {"ok": true, "results": [...], "comparison": {...}} | {"ok": false, "error": "..."}

결정적: 같은 입력 → 같은 출력 (난수·시간 미사용).
"""

import json
import sys

import numpy as np

RIGHT_WRIST = 16  # blazepose_33

_ref_cache: dict[str, dict] = {}


def wrist_track(stream: dict, subject_idx: int = 0) -> np.ndarray:
    """subject의 오른 손목 (x,y) 시계열. 없으면 첫 키포인트로 폴백."""
    frames = stream["subjects"][subject_idx]["frames"]
    pts = []
    for f in frames:
        kp = next((k for k in f["keypoints"] if k["id"] == RIGHT_WRIST), f["keypoints"][0])
        pts.append((kp["x"], kp["y"]))
    return np.asarray(pts, dtype=np.float64)


def dtw(a: np.ndarray, b: np.ndarray) -> float:
    """단순 DTW 누적거리 (경로 길이 정규화)."""
    n, m = len(a), len(b)
    cost = np.full((n + 1, m + 1), np.inf)
    cost[0, 0] = 0.0
    for i in range(1, n + 1):
        for j in range(1, m + 1):
            d = float(np.linalg.norm(a[i - 1] - b[j - 1]))
            cost[i, j] = d + min(cost[i - 1, j], cost[i, j - 1], cost[i - 1, j - 1])
    return float(cost[n, m] / (n + m))


def analyze(user: dict, ref: dict) -> dict:
    ref_track = wrist_track(ref)
    results = []
    for idx, subj in enumerate(user["subjects"]):
        d = dtw(wrist_track(user, idx), ref_track)
        score = round(100.0 * float(np.exp(-8.0 * d)), 1)  # 거리→점수 (임의 스케일)
        results.append({
            "subject_key": subj["subject_id"],
            "overall_score": score,
            "dtw_distance": round(d, 6),
            "score_breakdown": {"stub": True, "wrist_dtw": round(d, 6)},
            "feedback": [{
                "priority": 1,
                "segment": "full_swing",
                "message": "[개발 스텁] 실엔진 결과 아님 — 파이프라인 검증용 출력",
            }],
        })
    return {"ok": True, "results": results,
            "comparison": {"stub": True, "ref_frames": len(ref_track)}}


def main() -> None:
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            req = json.loads(line)
            ref_key = req["ref_key"]
            if req.get("ref") is not None:
                _ref_cache[ref_key] = req["ref"]
            ref = _ref_cache.get(ref_key)
            if ref is None:
                resp = {"ok": False, "error": f"ref 미캐시: {ref_key}"}
            else:
                resp = analyze(req["user"], ref)
        except Exception as e:  # 프로토콜상 오류도 한 줄 응답으로
            resp = {"ok": False, "error": f"{type(e).__name__}: {e}"}
        sys.stdout.write(json.dumps(resp) + "\n")
        sys.stdout.flush()


if __name__ == "__main__":
    main()
