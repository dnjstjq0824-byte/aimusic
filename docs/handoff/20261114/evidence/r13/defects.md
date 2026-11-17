# R13 최종 하네스 결함표 (수용: 체크리스트 기준 결함 0)

2차 실행 2026-11-17: 1530화면 완주, 200×1500 + 404×30(게이트 정상) + 500×0.
1차 46건 지적 → 실결함 1건 수정 → 2차에서 게이트 패턴 외 0건으로 해소.

| # | viewport | role | lang | route | url | 현상 | 기준(수용체크리스트) | 판정 |
|---|----------|------|------|-------|-----|------|----------------------|------|
| D-1 | 3종 전뷰포트 | guest | 5언어 | landing-mood·landing-use | /explore/mood/cafe·/explore/use/youtube | 404 "Landing not found" 30화면 | S4 랜딩 게이트(12곡 미만 404) | 기대 동작 — E2E T20 게이트 정방향 검증済. 200 경로(landing-genre)는 오류 0 |
| D-2 | 3종 전뷰포트 | listener | 5언어 | playlist-public | /playlists/{slug-short} | Hydration mismatch 15화면(1차만) | R-4 콘솔 0 | 실결함 → 수정済: isOwner 직접 v-if 2곳을 ownerView(mount 게이트)로 교체. 2차 0건 + 프로브 4종 0건 |
| D-3 | d1440 | guest | zh-cn | embed-track | /zh-cn/embed/track/{short} | goto 30초 타임아웃 1화면(1차만) | R-4 콘솔 0 | 1회성 — 재측정 200·0건, 2차 전수에서 미재현 |

수집 지표(collect.mjs): 전체 캡처, 콘솔 오류, 4xx/5xx, 가로 스크롤, 44px 미만 1차 행동,
24px 미만 대상, 12px 미만 텍스트, axe serious/critical(미설치 시 수동 프록시), LCP.

2차 지표 요약: hScroll 0·tiny 0·unlabeled 0·noAlt 0 전 화면. s44(44px 미만 대상)는
R0 기준선부터 존재하는 정보성 지표로 유지(모바일 탭바 자체는 5탭·규격 충족).

스크린샷 명명: `{viewport}-{role}-{lang}-{route}.png` (총 1530, 로컬 보관·커밋 제외).
매트릭스: evidence/r13/harness-matrix.txt. 실행 로그: baseline.json + screens.txt.
