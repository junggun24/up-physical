# fixtures — 골격 스트림 최소 샘플

검증 루프의 입력. **실제 경계 코드**(`internal/contract`)로 검증된다 (`cmd/validate`).

| 파일 | 기대 | 설명 |
| --- | --- | --- |
| `valid-forehand-2d.json` | 통과 | blazepose_33 · 2d · 1 subject · 10 프레임 (최소 유효 스트림) |
| `invalid-inv2-time.json` | 무효 | INV-2 위반: 프레임 `t` 비증가 |
| `invalid-inv4-topology.json` | 무효 | INV-4 위반: 토폴로지 불완전(키포인트 1개 누락) |

## 규칙

- fixture 를 추가하면 이 표와 `runners/check.sh` 의 목록에 등록한다.
- 유효 fixture 는 `validate-fixture.sh <file>`, 무효 fixture 는
  `validate-fixture.sh -expect-invalid <file>` 로 검증한다.
- 실전 데이터(실측 포핸드 클립 추출 스트림)를 확보하면 `valid-forehand-real.json` 으로
  추가하고 Golden 승인 파이프라인의 입력으로 쓴다.

재생성: `python3 gen_fixtures.py` (이 디렉토리에서).
