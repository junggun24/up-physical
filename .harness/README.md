# .harness — up-physical 백엔드 하네스

회사가 소유하는 **에이전트 개발 시스템**. 모델 능력을 *검증 가능한 결과*로 바꾼다.
([Harness 프레임워크 원문](https://app.notion.com/p/3a7b61e8ae4281cc9151eed3a00797d7))

## 4개 레이어 ↔ 이 디렉토리

| 레이어 | 역할 | 여기서 |
| --- | --- | --- |
| ① 지식·Context | 무엇을·어디·왜 | `wiki/` + 루트 `AGENTS.md` |
| ② 실행·도구 | 빌드/부분실행/디버깅 | `runners/` |
| ③ 검증·완료판정 | 무엇이 맞고 끝인가 | `fixtures/` + `golden/` + `wiki/verification.md` |
| ④ 피드백·환류 | 실패·배움 축적 | 작업 후 `wiki/`·`skills/` 갱신 (self-review) |

## 구조

```
.harness/
├─ wiki/                      # AI Wiki (장기기억, 검색용으로 분할)
│  ├─ system-map.md           #   아키텍처·모듈·진입점
│  ├─ domain/skeleton-stream.md  # 골격 스트림 계약·불변식·DTW 도메인
│  ├─ runtime.md              #   빌드·실행·인프라·환경변수
│  └─ verification.md         #   불변식·Golden·완료조건
├─ skills/                    # 재사용 절차기억 (How)
├─ fixtures/                  # 최소 입력(골격 스트림 샘플)
├─ golden/                    # 승인된 기준 출력
├─ runners/                   # build/test/smoke/validate CLI 래퍼
└─ e2e/                       # 배포후 E2E + Notion 리포터 (예정)
```

## 검증 루프 (Harness의 심장)

```
Fixture → 실행 → Invariant/Golden 검증 → 수정 → 재검증 → 완료
```

`컴파일 성공 ≠ 기능 완료`. 완료 조건이 있어야 Agent가 끝까지 루프를 돈다.
현 상태의 최소 루프: `runners/check.sh` (build+vet+test+fixture 검증).
