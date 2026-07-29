#!/usr/bin/env python3
"""fixture 생성기 — 골격 스트림 샘플(유효/무효)을 결정적으로 생성한다.

blazepose_33 · 2d · 1 subject · 10프레임(30fps). 좌표는 결정적 함수로 생성해
재실행해도 같은 파일이 나온다(diff 안정성).
"""

import copy
import json
import math
import pathlib

HERE = pathlib.Path(__file__).parent
N_KP = 33
N_FRAMES = 10
FPS = 30.0


def keypoint(kp_id: int, t: float) -> dict:
    # 결정적·매끄러운 가짜 궤적 (스윙 흉내일 뿐 물리적 의미 없음)
    phase = t * 2.0 * math.pi + kp_id * 0.1
    return {
        "id": kp_id,
        "x": round(0.5 + 0.2 * math.sin(phase), 6),
        "y": round(0.5 + 0.2 * math.cos(phase), 6),
        "z": None,
        "visibility": 0.95,
    }


def build_valid() -> dict:
    frames = []
    for i in range(N_FRAMES):
        t = round(i / FPS, 6)
        frames.append({"t": t, "keypoints": [keypoint(k, t) for k in range(N_KP)]})
    return {
        "schema_version": "1.0",
        "session_id": "11111111-1111-4111-8111-111111111111",
        "created_at": "2026-01-01T00:00:00Z",
        "capture": {
            "source": "on_device",
            "model": "blazepose",
            "model_variant": "full",
            "model_version": "0.10.0",
            "fps": FPS,
            "coordinate_space": "image_normalized",
            "dimensions": "2d",
            "keypoint_topology": "blazepose_33",
            "frame_count": N_FRAMES,
        },
        "subjects": [
            {"subject_id": "player-1", "label": "user", "frames": frames}
        ],
    }


def main() -> None:
    valid = build_valid()

    # INV-2 위반: 3번째 프레임 t 를 뒤로 되돌림
    inv2 = copy.deepcopy(valid)
    inv2["subjects"][0]["frames"][2]["t"] = inv2["subjects"][0]["frames"][0]["t"]

    # INV-4 위반: 첫 프레임에서 키포인트 하나 제거(토폴로지 불완전)
    inv4 = copy.deepcopy(valid)
    inv4["subjects"][0]["frames"][0]["keypoints"].pop()

    out = {
        "valid-forehand-2d.json": valid,
        "invalid-inv2-time.json": inv2,
        "invalid-inv4-topology.json": inv4,
    }
    for name, data in out.items():
        path = HERE / name
        path.write_text(json.dumps(data, ensure_ascii=False, indent=1) + "\n")
        print(f"wrote {path.name}")


if __name__ == "__main__":
    main()
