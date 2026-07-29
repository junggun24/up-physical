# golden — 승인된 기준 출력

**아직 비어 있다 (의도됨).** 분석 엔진의 결정성이 확보되면 대표 fixture
(`fixtures/valid-forehand-real.json` 예정)의 분석 결과를 사람이 검토·승인해 여기 고정한다.

## 규칙

- Golden 은 **사람 승인**을 거쳐야만 갱신된다 (diff 검토 → 승인 커밋).
- 형식: `forehand-<fixture명>.result.json` — 워커가 저장하는 결과와 동일 구조.
- 비교는 러너가 수행한다(수치 허용오차 정책은 승인 시 함께 정의).
