# fixtures — 골격 스트림 최소 샘플

검증 루프의 입력. **실제 경계 코드**(`internal/contract`)로 검증된다 (`cmd/validate`).

| 파일 | 기대 | 설명 |
| --- | --- | --- |
| `valid-forehand-2d.json` | 통과 | blazepose_33 · 2d · 1 subject · 10 프레임 (최소 유효 스트림) |
| `valid-forehand-2d-lefty.json` | 통과 | 위 fixture 의 **좌우 반전**본 — 손잡이 정규화 회귀용 |
| `reference-forehand-2d.json` | 통과 | 레퍼런스(코치) 시드용 — 위상만 다른 유효 스트림 |
| `invalid-inv2-time.json` | 무효 | INV-2 위반: 프레임 `t` 비증가 |
| `invalid-inv4-topology.json` | 무효 | INV-4 위반: 토폴로지 불완전(키포인트 1개 누락) |

## 규칙

- fixture 를 추가하면 이 표와 `runners/check.sh` 의 목록에 등록한다.
- 유효 fixture 는 `validate-fixture.sh <file>`, 무효 fixture 는
  `validate-fixture.sh -expect-invalid <file>` 로 검증한다.
- 실전 데이터(실측 포핸드 클립 추출 스트림)를 확보하면 `valid-forehand-real.json` 으로
  추가하고 Golden 승인 파이프라인의 입력으로 쓴다.

## 재생성

- 대부분: `python3 gen_fixtures.py` (이 디렉토리에서, 결정적).
- `valid-forehand-2d-lefty.json` 은 **Go 변환기로** 만든다 — 좌우 반전 구현을 한 벌로
  유지하기 위해서다(Python 에 좌/우 쌍 테이블을 복제하면 드리프트가 생긴다):

```bash
go run ./cmd/mirror -in .harness/fixtures/valid-forehand-2d.json \
  -out .harness/fixtures/valid-forehand-2d-lefty.json \
  -session 66666666-6666-4666-8666-666666666666
```
