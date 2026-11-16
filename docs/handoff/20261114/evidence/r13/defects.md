# R13 최종 하네스 결함표 (수용: 체크리스트 기준 결함 0)

상태: NOT RUN — Docker 엔진 API 500으로 하네스 미실행. 복구 후 `setup.mjs → run.mjs` 실행하고 아래 표를 채울 것.

| # | viewport | role | lang | route | url | 현상 | 기준(수용체크리스트) | 판정 |
|---|----------|------|------|-------|-----|------|----------------------|------|
| - | - | - | - | - | - | (실행 후 기입) | - | - |

수집 지표(collect.mjs): 전체 캡처, 콘솔 오류, 4xx/5xx, 가로 스크롤, 44px 미만 1차 행동,
24px 미만 대상, 12px 미만 텍스트, axe serious/critical(미설치 시 수동 프록시), LCP.

스크린샷 명명: `{viewport}-{role}-{lang}-{route}.png` (총 1530).
매트릭스: evidence/r13/harness-matrix.txt.
