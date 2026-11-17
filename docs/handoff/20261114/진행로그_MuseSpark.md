# 진행 로그 — AVIBE 리디자인 연속 개발 (Muse Spark, 2026-11-14~)

브랜치: `feat/redesign-20261114` (from `docs/handoff-20261114` @ 6a5cd65).
모드: 연속 개발 (Phase 승인 대기 없음). 운영 쓰기 금지, main 직접 push/병합 금지.

## WI 상태판

| WI | 상태 | 커밋 | 변경파일 | 검증 | 캡처 | 남은 결함 |
|----|------|------|---------|------|------|----------|
| R0 | PASS | f76dd5a | 16 + evidence | 하네스 165/165 | baseline-samples 6종 | axe-core 미실행(의존성 상한) |
| P1 | PASS | 0091050 | 23 + 이동 92(면제) | CSR 동등 162/165·SSR 162/165·vitest 47·tsc·이중빌드·grant 14 | nuxt-csr/ssr json | legal alias 경고(개발 로그만, R8 해소) |
| R1 | PASS | 3753448+3 | 4분할(상한 준수) | tsc·vitest 47·하네스 9동일·tiny 0·빌드 | r1 json+showcase 2종 | 장식 raw 팔레트 일부 잔존 |
| R2 | PASS | eae45f0+eab2387+fix | WIP+수정 1·증거 json/txt | tsc0·vitest 52·checker·하네스 165/165·빌드 | r2 baseline.json+screens.txt(+PNG 로컬 165) | volar 경고(무해)·legal alias 경고(R8) |
| R3 | PASS | (아래 2건) | 코드 20·로케일 6언어 | tsc0·vitest 60·checker·하네스 189/189·빌드 | r3 json/txt(+PNG 로컬 189)+r3-i18n 20종 | 소셜 무제공시 단위테스트만·legal alias(R8) |
| R4 | PASS | (아래 2건) | 코드 11·로케일 6언어 | tsc0·vitest 60·checker·하네스 189/189·프로브 19·빌드 | r4 json/txt(+PNG 로컬 189)+r4-i18n 10종 | 차트·비즈니스 404(R5·R7)·legal alias(R8) |
| R5 | PASS | (아래 3건) | 코드 36·로케일 6언어 | go全·E2E 435·tsc·vitest 64·checker·하네스 192·빌드 | r5 json/txt(+PNG 192)+r5-i18n 20종 | AiBadge 미키화·용도facet·BPM슬라이더·legal alias |
| R6 | PASS | (아래 3건) | 코드 33·로케일 6언어 | go全·E2E 466·tsc·vitest 64·checker·하네스 198·빌드 | r6 json/txt(+PNG 198)+r6-i18n 20종 | AiBadge·크리에이터팔로우UI(S4)·legal alias |
| R7 | PASS | (아래 3건) | 코드 26·로케일 6언어 | go全·E2E 483·tsc·vitest 64·checker·하네스 210·빌드 | r7 json/txt(+PNG 210)+r7-i18n 20종 | 플랜명 한글 고정·문의UI R11·legal alias |
| R8 | PASS | 1aca943+검증수정 | 코드 15·로케일 6언어 | E2E 701/0(T18 포함)·하네스 1530 무오류 | r13 baseline+screens(+PNG 1530) | 없음(전용 증거는 최종 매트릭스로 계승) |
| L1 | PASS | 미커밋(금지) | 초안 1 | cspell 0·게시금지 확인 | – | G-3 법률 검토·게시(범위 밖) |
| A1 | PASS | 1aca943+setup수정 | 코드 33·로케일 6언어 | E2E 701/0(가드)·스텁 18종·하네스 소셜 5언어 무오류 | a1 gates+stub·r13 | 실키 5종 실측 deferred(PO) |
| A2 | PASS | 1aca943 | 엔진 35·화면 30 | E2E 701/0(T19 전 항목)·하네스 인증 5언어 무오류 | a2 gates·r13 | SMTP 실발송 deferred |
| S1 | PASS | 1aca943+000025수정 | 코드 32 | E2E 701/0(T20 slug/short/handle)·하네스 무오류 | s1 gates·r13 | 없음(substr 캐스트 수정済) |
| S2 | PASS | 1aca943 | 코드 13·로케일 6언어 | E2E 701/0(T20 oEmbed)·하네스 5언어 무오류 | s2 gates+OG·r13 | 없음 |
| S3 | PASS | 1aca943 | 코드 40 | E2E 701/0(T20 사이트맵·robots·indexnow·purge)·하네스 무오류 | s3 gates·r13 | 콘솔등록·IndexNow키(PO) |
| S4 | PASS | 1aca943 | (P3c 포함) | E2E 701/0(T20 게이트)·하네스 임베드·가이드·랜딩(404 기대) | s4 gates·r13 | 가이드문구 PO검토 대기 |
| S5 | PASS | 1aca943 | (P3c 포함) | E2E 701/0(T20 이벤트)·하네스 무오류 | s5 gates·r13 | RUM 실측·서명(PO) |
| R9 | PASS | 1aca943 | 코드 19·로케일 6언어·삭제 6 | E2E 701/0·하네스 스튜디오 8면×5언어 무오류 | r9 gates·r13 | 없음 |
| R10 | PASS | 1aca943+스케줄수정 | 엔진 24·화면 22 | E2E 701/0(T21 역할·콘솔)·하네스 콘솔 5언어 무오류 | r10 gates·r13 | 없음(TIME 스캔 수정済) |
| R11 | PASS | 1aca943 | 코드 16·로케일 ko/en | E2E 701/0(T21 admin)·하네스 admin 14면×5언어 무오류 | r11 gates·r13 | 없음 |
| G0 | PASS | 1aca943 | 코드 11 | E2E 701/0(T22 스키마·트리거·핀) | g0 gates | 없음 |
| G1 | PASS | 1aca943 | 코드 6 | E2E 701/0(T22 핀·게이트)+계약단위 | g1 gates | 없음 |
| G2 | BLOCKED | 미커밋(워킹트리) | 코드 1(PoC 드라이버) | NOT TESTED (GPU·가중치 부재) | g2 리포트 json+gates | 자택 GPU PC 실측 deferred |
| G3 | PASS | 미커밋(워킹트리) | 문서 1(비교표) | 비교틀 완성·가격기입 0(가드레일 4) | g3 gates.txt | 견적일 가격기입·PO 선택 deferred |
| G4 | PASS | 미커밋(워킹트리) | 코드 3(eval) | self-test 2종 그린 | g4 gates.txt | GPU 백엔드·임계값 교정 deferred |
| G5 | PASS | 1aca943 | 코드 13·로케일 6언어·문서 2·사전 2 | E2E 701/0(T22 가드·위반행)+단위·하네스 create 자리 5언어 | g5 gates·r13 | 없음 |
| R12 | PASS | 1aca943 | P5a 41 | analyze 0·test 22·ARB동치·OAuth단위(런타임 불필요) | r12 gates | 없음 |
| R13 | PASS | 1aca943+검증수정 | P5b 44 | E2E 701/0·하네스 1530(200×1500+게이트404×30)·트리아지 0 | r13 baseline+screens+defects·CSV 3종 | DEFECT-R13-01 PO 백로그 유지 |

## 항목 기록

### WI-R0 (P0) — 기준선 하네스 + 보고 — PASS
- 기준선: 브랜치 feat/redesign-20261114 (from docs/handoff-20261114 @ 6a5cd65), 착수 시 미커밋 0. 앱 코드 변경 0 (하네스·설정·문서만).
- 로컬 스택: start-local.sh 재빌드, 전 컨테이너 healthy. API 18081, web dev 15173 (VITE_API_PROXY=18081). 운영 계정·데이터 생성 0.
- 하네스: web/e2e/screens/{config,collect,setup,run}.mjs + @playwright/test 1.57.0 (devDep 1개) + npm scripts harness/harness:setup.
- 실행: 전 라우트 × 역할 5 × 뷰포트 3 = 165화면 전수. 출력 evidence/baseline/{baseline.json,screens.txt} + PNG 165종(31MB, 로컬 보관·커밋 제외) + baseline-samples 6종 커밋.
- 결과: HTTP 200 165/165, 가로 스크롤 0, LCP 165/165 수집(예: 모바일 게스트 홈 824ms). 오류 9건 = H-11 인증서 404 ×6 + 무효 인증서 404 ×3. 44px 미만 1차 행동이 대다수(홈 121/152), 12px 미만 텍스트 다수(홈 155), 대비 미달 샘플 매 화면 존재.
- axe-core: devDependency 1개 상한으로 미설치 → NOT RUN. 대신 수동 프록시(무라벨 버튼·alt 누락·대비 샘플 계산) 수집. R13에서 재검토.
- H-xx/C-xx 재현: H-01(온보딩 모달 코드 확인)·H-03(기술 용어 9파일)·H-07(단일 URL 3탭)·H-08(인증 화면 플레이어/푸터)·H-10(비즈니스 요금 없음)·H-11(404 재현)·H-12(UUID 원문)·H-14(소셜 버튼 0)·C-01(3역할)·C-02(reset 겸용)·C-04(이메일 인증 없음)·C-05(i18n 없음)·C-11/C-12/C-16(oauth 코드)·C-14(SPA)·C-15(MyLibrary 혼재)·C-17(ILIKE) 전부 재현. C-10 확인(D5 미커밋·대체, D7 유효).
- 신규 결함: N-01 게스트 공개 플레이리스트 상세(/playlists/:id, isPublic=true)가 로그인으로 리다이렉트 → 공개 SSR 요건(R6)과 충돌, R6에서 해소.
- 역할 분리 (b) 영향 범위: 마이그레이션(user_roles + 백필) 2, entities/user_partner/auth-서비스/middleware/admin/adminctl/overview 8, JWT roles 클레임, web 가드·스토어·메뉴 6, E2E. 예상 16~18파일(R10). 이관 스크립트는 로컬·스테이징만, 운영 실행 금지(G-ROLE).
- Nuxt 사전 조사: web/src 95파일(이동 대상, rename 면제). 브라우저 전역 24파일 — hls.js는 playerStore·storePlayerStore 2곳(정적 import → 동적 전환 필요), Toss는 Checkout·CheckoutConfirm·Pricing + types(전부 ssr:false 경로라 격리 용이), localStorage 키 avibe.accessToken/refreshToken/deviceToken/onboarded(SSR 페이지는 비로그인 동일 HTML + 하이드레이션 승격). 라우터 가드 → 미들웨어(auth·role·guest) + safeRedirect 이식. 예상: 이동 ~95(면제) + 내용 P1 15·R1 20·R2 10·R3 12·R4 10.
- 라이브러리 API: 좋아요·저장·팔로우 테이블·API 전무(playlists.is_public은 존재). 엔진 추가 예상 7~9파일(R6).
- H-11: 엔진 certificate.go VerifyTrackCertificate 404 → 200 {data:null}로 수정(R5). TrackDetail은 이미 .catch null 처리 → null 분기로 정리.
- 운영 OAUTH_REDIRECT_BASE_URL=https://avibemusic.com/api/v1/auth/oauth 확인(읽기 전용, 비밀 아님).
- Phase 파일 추정(내용 변경): P1 ~67 → P1a(P1+R1) + P1b(R2+R3+R4)로 분할. P2 ~75 → P2a(R5+R6) + P2b(R7+R8+L1) + P2c(A1+A2)로 분할. P3 ~50 → 초과 시 분할. P4 ~40. Track G ~25. P5 ~36(ARB 언어당 1 계산).
- 검증: 하네스 165/165 완주, baseline.json 커밋, vue-tsc·vitest·go test는 앱 변경 없어 생략(다음 WI부터 매 항목 실행).

### WI-P1 (P1a) — Nuxt 4 이전 + 공개 SSR — PASS
- 버전 고정(exact): nuxt 4.6.4 · vue 3.5.45 · pinia 4.1.0 · @pinia/nuxt 1.0.3 (pinia 2→4 상향, peer 충돌 해소). Nitro 프리셋 실측: cloudflare-module + node-server.
- 이동: web/src 95파일 → web/app (pages/layouts/components/stores/api/composables/assets). 라우트명은 pages:extend로 구명 유지(코드 참조 60곳 무수정). 가드·타이틀·safeRedirect → middleware app.global + utils.
- SSR 격리: tokenStore/deviceToken localStorage 가드, hls.js 동적 import 2곳, UiModal immediate watcher 가드, WaveformSeekbar 소수점 3자리(하이드레이션), UiNavbar·playlists 인증 영역 ClientOnly.
- routeRules: 공개 8경로 SSR+SWR, 앱·콘솔 명시 CSR, /api/** 프록시(NUXT_API_PROXY). 주의: /** 포괄 규칙은 개별 규칙을 덮어써서 사용 금지, / 는 ssr:true 명시 필요, nuxt.config 변경은 dev 수동 재시작(.nuxt 캐시 삭제 병행).
- 동등성: CSR 하네스 162/165 일치(3건은 신규 매장행 데이터 드리프트, 오류 9건 baseline과 동일). SSR 전환 후 동일 162/165 + 하이드레이션 경고 0.
- 이중 빌드: node-server + cloudflare-module 둘 다 성공. node preview 실측: 홈 SSR 45KB+h1, login 200, /api 프록시 200.
- 검증: vitest 47/47, nuxt typecheck 0, worker-grant 14/14(worker.js 무변경). E2E는 API 전용이라 미실행(CI에서 수행).
- 파일: 내용 23(이동 92 면제, 84% 1건 포함). 증거: evidence/nuxt-csr + nuxt-ssr (json/txt 커밋, PNG 로컬).
- 잔여: /legal alias 경고(개발 로그만, R8에서 legal 구조 변경 시 해소). 공개 페이지 데이터는 아직 onMounted 로딩 → P3에서 useAsyncData 전환.

### WI-R1 (P1a) — 디자인 토큰 v2 + Ui + 신규 컴포넌트 — PASS (4분할 커밋)
- R1a 3753448: 토큰(text-muted #8A909C·verified·success #22C55E·focus·on-brand·서체 7단계·xs 390) + 셀프호스팅 글꼴 219파일(10.6MB: Pretendard/JP dynamic-subset, Noto SC/TH 400·500·700 + OFL + README) + :lang() 스택 + CDN 글꼴 제거 + latin 서브셋 preload.
- R1b 5a23b4e: UiButton(v2 4종+높이 36/44/52, primary=시안) + UiInput(min-h-11) + Toast success 분리 + 신규 6종(VerifiedSeal·CoverMosaic·LocaleSwitcher·BottomSheet·Stepper·UiMenu) + /dev/ui(dev 전용, 비 dev 404).
- R1c 5c140c9 + R1d d9fe6c3: neutral-* 428건 + text-[8~11px] 144건 + [#04121a] 19건 + verified/success 의미 분리 20건 전수 교체(컴포넌트 27 + 페이지 34).
- 검증: neutral 0·tiny 0(repo grep), 12px 미만 노드 8411→0(하네스), 대비 muted 6.23/5.61·ink 10.72(실측), 외부 글꼴 요청 0·font loaded true·에러 0, ja/zh/th computed 스택 전환 확인, vitest 47/47·tsc 0·하네스 9동일(기존 404)·prod 빌드(219 폰트 포함).
- 교훈: PS Get/Set-Content 재기록은 한글 손상 → 대량 교체는 Node utf8 스크립트로만. (1회 손상 후 git 복구·Node 재실행으로 수습, 무결성 확인済)
- 잔여: 장식용 raw 팔레트(gradient용 blue/teal/amber 등) 일부 잔존 — 수용 기준(neutral·임의크기)은 만족. LocaleSwitcher 실동작은 R2에서 연결.
- 후속 발견(WI-R2에서 수습): R1 스윕이 `text-[#04121a]` 19건을 `-on-brand`로 깨뜨림(PS 인라인 정규식 인용 간섭) + `hover:`/`selection:` 접두 4건 손실. R2a에 포함해 복구, prefix 복원 확인済. 교훈 강화: 인라인 node -e/PS 정규식 금지, 파일 스크립트만.

### WI-R2 (P1b) — i18n 기반 — PASS
- 모듈 @nuxtjs/i18n 10.6.0 exact(+@nuxt/kit 호이스팅). 전략 prefix_except_default(ko), langDir ../i18n/locales, files 배열 지연 로딩, vueI18n은 파일 방식(i18n.config.ts, 인라인 불가).
- 로케일 5+1: ko/en/ja/zh-cn/th + en-XA(dev 의사로케일, 생성 스크립트). 네임스페이스 17종 골격, common/errors/player 완성(ko 원문·en 자필·ja/zh/th machine-draft).
- 공개 페이지만 접두(27페이지 defineI18nRoute(false)), 미들웨어 ___xx 접미 처리 + i18n 타이틀(반응형) + 가드 토스트 + 후행슬래시 301(로케일 루트 제외).
- 이름 기반 내비 14파일 localePath 전환(v10은 ko 포함 전 접미사), route.name 비교는 base명 computed.
- 감지 플러그인(공개면 스킵)+제안 배너(세션 1회)+스위처 실동작(Beta 뱃지)+useFormat(+테스트)+useApiError(+테스트)+check-i18n(허용목록)+CI 잡.
- 검증 중간: tsc 0·vitest 52/52·check-i18n PASS·스팟 4언어 내비/플레이어/타이틀/배너/전환 전부 PASS. 전체 하네스·prod 빌드는 미완.
- WIP 커밋 (아래): 구현 완료, 검증 2건 남음. 재개 시 순서: ①dev 기동(npm run dev, 15173) ②하네스 setup+run → evidence/r2 (15분, historic: 캐시 문제 시 .nuxt 삭제 후 재시작) ③npm run build ④결과 기입 후 R2a/R2b 분할 커밋 정리(WIP 유지 or 분할) ⑤R3 착수.
- 주의: 하네스 실행 중 .nuxt 캐시가 오래되면 오탐(151 bad 사례) — 하네스 전 dev 재시작 권장. B-09 토큰 회전 때문에 하네스는 setup 직후 바로 실행.
- 검증 완결(11-15): 하네스 165/165 HTTP 200, 오류 9건 = H-11 기존 패턴(track certificate 404×6 + 무효인증서 404×3, R5 해소 예정)과 동일. hScroll 0·tiny 0·unlabeled 0·noAlt 0·failed 0. prod 빌드 2m02s 성공. 증거 evidence/r2/{baseline.json,screens.txt} 커밋(PNG 165종 로컬 보관).
- 수습 2건: ①locale-init.client.ts가 useI18n()을 setup 밖(플러그인)에서 호출 → 비공개 페이지 전역 크래시. nuxtApp.$i18n.setLocale로 교체(미들웨어 app.global.ts와 동일 패턴), /login·/disputes 브라우저 프로브로 해소 확인. tsc·vitest·스팟(공개면만)은 이 버그를 못 잡음 — 비공개면 프로브를 R3+ 게이트에 추가할 것. ②dev 기동 시 구 Vite 관행(VITE_API_PROXY=18081 포트만) 사용 시 Nitro 프록시가 `18081/api/**`로 깨져 /api 전구간 502. Nuxt는 NUXT_API_PROXY 전체 URL(기본 http://localhost:18081). 해당 런은 폐기 후 재실행.
- 무효 런 2건 폐기: 11-14 18:30~19:30분(114 타임아웃, stale dev 추정) + 11-15 502 런(프록시 오설정). 본 결과는 fresh .nuxt + 정상 프록시 재실행분.
- tsc 참고: `[Vue] Load plugin failed: vue-router/volar/sfc-route-blocks` 경고 2종은 exit 0·오류 0이라 무해 확정. 원인=vue-router 4.6.4/v5.5.1이 volar 플러그인 미동봉 + Nuxt 생성 tsconfig가 참조. 프로젝트에 `<route>` 블록이 없어 영향 없음. 수정 금지(toolchain 영역).
- P1b 분할 선언: R2 내용 변경이 74건(코드·설정, 번역·증거·락·로그 제외)으로 Phase 상한 50 초과 — R1 스윕 파손 전수 복구가 포함된 탓. P1b-1=R2, P1b-2=R3+R4로 분할 진행. R3=코드 20+로케일 6언어.

### WI-R3 (P1b-2) — 인증 화면 분리 — PASS
- AuthLayout(신규): 로고+언어 상단, 카드 슬롯, 최소 푸터(©+약관 2종), 플레이어·메뉴 없음. 미들웨어에서 인증 6경로 매핑.
- 페이지: /login(탭 제거·소셜 상단·TOTP 2단계 유지·dev필 보존) /signup(3카드) /signup/listener /signup/creator(3단계·Stepper) /signup/business(3단계·도입문의 고지) /signup/complete-email(A1 자리) /forgot-password(분리) /reset-password(토큰 전용) /verify-email(A2 자리) /oauth/callback(키화). 문자열 전부 auth 네임스페이스 5언어(ko원문·en자필·ja/zh/th machine-draft).
- SocialButtons(신규): 엔진 활성 공급자만·언어별 순서(UI §3.11). 순서 로직은 utils/social.ts+social.test.ts 8건으로 분리 검증.
- 리다이렉트(미들웨어): ?tab=register→/signup/listener, ?tab=partner→/signup/creator, 무토큰 reset→/forgot-password. 로그인 상태 진입 차단도 signup/forgot까지 확장.
- 비고: business 제출은 R10 전이라 registerPartner로 등록(파트너 계정 즉시 사용 가능)+화면에 도입문의 접수 명시. creator 국가·AI플랫폼/business 사업자번호·업종은 엔진 필드 없어 선택 수집만 — R9/R10에서 배선. 매장 이용 조건은 법문서 없어 라벨만(창작 금지). verify/complete-email은 엔진 없이 껍데기(A1/A2 TODO).
- 검증: tsc 0·vitest 60/60(신규 8)·checker PASS(allowlist에서 login/reset/oauth 제거, UNUSED 32=기존 18+동적키 14 실사용)·하네스 189/189 HTTP 200(오류 9=H-11 기존·리다이렉트 9/9·hScroll/tiny/unlabeled/noAlt 0)·5언어×2화면×2뷰포트 20종 무오류 캡처·빌드 20.5s.
- 수습: ①vue-i18n `@` 링크문법 — emailPh를 `{'@'}`로, 체커 ICU 규칙에 리터럴 허용 추가. ②첫 R3 하네스의 하이드레이션 10건은 빌드 동시 실행이 .nuxt를 오염시킨 환경 사고(코드 회귀 아님) — fresh .nuxt 재실행분으로 증거 교체. 교훈: 빌드와 dev/하네스 절대 병행 금지. ③terminate 뒤 nuxt 좀비가 포트 점유 — 포트 주인 확인 후 PID kill.
- 잔여: 로컬 엔진에 소셜 공급자 미설정이라 SocialButtons 실렌더 미확인(단위테스트로 커버, A1에서 실증). legal alias 경고는 R8.

### WI-R4 (P1b-2) — 앱 셸·내비게이션 — PASS
- AppLayout(user.vue→app.vue 개명·개편): 상단 내비 + 푸터 + 플레이어 + 모바일 탭바 + 큐/전체화면 상태. 본문 하단 여백=플레이어+탭바+safe-area(H-09).
- 상단바 개편: 데스크톱 5항목(홈·탐색·플레이리스트·라이브러리·차트)+검색창(/explore?q)+[비즈니스]+언어+계정 메뉴(계정설정·스튜디오*·관리자*·로그아웃). 햄버거·드로어 삭제(H-15), 모바일 상단은 스크롤 시 숨김.
- MobileTabBar(신규): 5탭 아이콘+라벨(신규 tabs 키 6언어, 전 언어 한 줄 확인). 계정 탭은 /account/security 고정(가드 경유, SSR 안전).
- 푸터 개편: H-03 기술 스트립 삭제(allowlist 해제)+언어 선택 추가+요금제 링크(상단에서 이동).
- 플레이어: 모바일 미니 64(2px 진행선, 파형 제거·H-15)+탭바 위 배치. FullscreenPlayer(신규, §2.4): 미니 탭/커버 클릭·Esc·큐/가사/라이선스 탭·D 우측 420 패널. 좋아요는 R6 엔진 없어 생략, ⋯ 대신 큐 버튼.
- 검증: tsc 0·vitest 60·checker PASS(UNUSED 55=기존 32+미사용 nav 20+동적 3, R5~R11에서 재사용)·하네스 189/189(오류 9=H-11·리다이렉트 9/9·hScroll/tiny/unlabeled/noAlt 0)·프로브 19/19(데스크톱 8·모바일 5·재생중 6)·5언어 홈 10종 무오류·빌드 17.9s.
- P1 수용 판정: 인증 플레이어 0·푸터 가림 0·모바일 5탭·5언어 캡처(인증 20+R3·공통 10) 전부 충족. P1 완결.
- 잔여: /charts·/business 링크는 R5·R7까지 404 껍데기(정본 경로 선반영). 프로필 메뉴·비즈니스 콘솔 메뉴는 R8·R10. 좋아요 R6. legal alias R8.

### WI-R5 (P2a) — 홈·탐색·트랙 상세 — PASS
- P2a 분할 선언: R5 내용 변경이 코드 36+로케일 6언어로 상한 50 초과 가능 — P2a-1=R5a 엔진(17)·P2a-2=R5b 화면(19+6)·P2a-3=R6으로 분할. A2 예정 000020은 R5가 사용 → A2 마이그레이션은 000021로 이월.
- R5a 엔진: 000020 pg_trgm+GIN 3종. 검색v2(제목·크리에이터 ILIKE+trgm·태그표시명 확장·relevant 정렬·certified/partnerId/bpm/dur 필터). 태그 레지스트리 19키+5언어(search/tags.go). 크리에이터명 DTO(GetByID·SearchReady·TopByPlays). /charts(주·월·전체)·/playback/recent·/playlists/public·/tracks/:id/playlists·공개 트랙목록(JWTOptional). H-11 양 인증서 조회 200 null.
- R5b 화면: 홈 전면 개편(게스트 히어로+스트립+바로듣기+무드칩+차트10+추천플리+고객카드·로그인 이어듣기+최근+내플리·G5 만들기 자리·온보딩 삭제). 탐색 §3.2(필터패널+칩+리스트/그리드+BottomSheet). 트랙 §3.3(라이선스 패널·인증상태·AI표시·비슷한곡·같은크리에이터·담긴플리·ID복사·하단신고). /charts(주·월·전체). TrackRow 신규·TrackCard v2(재생+⋯). home/explore/track/tags 6언어(ko원문·en자필·ja/zh/th machine-draft).
- 검증: go build/vet/test 전체 그린·E2E 435/0(T14 검색5언어·T15 발견API·T11b 404→null 개정)·tsc 0·vitest 64(신규 provider/tags 4)·checker PASS·하네스 192/192 무오류(404 9건도 해소)·프로브 18·5언어 홈/트랙 20종·빌드 42.9s.
- 수습: ①ListPublicContaining 컬럼 모호성 500 → 한정자로 수정. ②T15 고정 인증서번호 재실행 충돌 → 트랙ID 파생. ③게스트 홈 추천 모자이크 401 → 트랙목록 라우트 JWTOptional+서비스 가시성(상세 N-01은 R6). ④인증 홈 하이드레이션 12건 → 마운트 후 분기+ClientOnly.
- 판단 기록: 용도 facet 저장 컬럼 없어 mood_tags 공유 풀로 매칭(데이터모델은 S4). BPM/길이는 프리셋 칩(슬라이더 대신). 비즈니스 CTA·고객카드는 R7까지 기존 경로(/signup/business·/for-creators·/business 껍데기). 크리에이터명 텍스트(S4 링크). 좋아요 R6. AiBadge 키화 미실시(후속 WI). th track.json 누락 등록.

### WI-R6 (P2a-3) — 플레이리스트/라이브러리 분리 — PASS
- R6a 엔진: 000021 좋아요·저장·팔로우 3테이블. 좋아요/저장/팔로우 CRUD+상태+목록 11종(멱등). N-01 공개 상세 게스트 개방(서비스 가시성 유지). Playlist ownerName. E2E T16 31어설션. A2 마이그레이션 000022로 재이월.
- R6b 화면: /library 4탭+내플리 카드·/me/playlists 관리(그리드/리스트·정렬·⋯)·/playlists 발견(추천/최신·히어로)·상세 §3.5(헤더·모자이크·드래그+⋯정렬·저장·팔로우·공유·N-01)·/account/billing(이용권+해지+탈퇴 임시·R8 분리 예정)·상세 ♡·/my-library→/library 301. library/playlist/account 6언어(ko원문·en자필·ja/zh/th machine-draft).
- 검증: go全·E2E 466/0·tsc 0·vitest 64·checker·하네스 198/198 무오류·프로브 12·5언어 20종·빌드 43.6s.
- 수습: ①T15 차트 동점 탈락(누적 데이터) → 시드 10건+limit 50. ②me 목록 태그 오타 500 → 수정. ③B-09 회전 프로브 토큰 만료 → setup 후 실행. ④인증 분기 하이드레이션 → ClientOnly/마운트 패턴(상세 오너 포함). ⑤R6 신규 앱 페이지 접두 누락 → defineI18nRoute(false)+charts 공개명단 등록.
- 판단 기록: 발견 탭은 추천/최신만(무드/용도/인기/믹스는 데이터 모델 없어 보류). 탈퇴 카드는 billing 임시 배치(R8 privacy로). 크리에이터 팔로우 UI는 S4(상세엔 플리 팔로우만). 저장+팔로우 병존(스펙). 토글 줄바꿈 공통 수정.

### WI-R7 (P2b) — 요금제·랜딩·도입 문의 — PASS
- R7a 엔진: 000022 문의+대기자 테이블. POST /business/inquiries(허니팟·레이트리밋·운영메일·수신처 OPS_NOTIFY_EMAIL)+관리자 목록·상태. POST /waitlist(멱등). E2E T17 17어설션. A2 마이그레이션 000023으로 재이월.
- R7b 화면: 요금제 3탭(리스너/크리에이터/비즈니스 무가격 3카드+문의 모달)+비교표+FAQ+환불 정본. 랜딩 /business(히어로·가치·샘플·4단계·FAQ·인라인 문의)·/for-creators·/trust·/create(대기자). pricing/business 6언어(ko원문·en자필·ja/zh/th machine-draft).
- 검증: go全·E2E 483/0·tsc 0·vitest 64·checker·하네스 210/210 무오류·프로브 8(실제 문의·대기자 제출)·5언어 20종·빌드 61s.
- 판단 기록: 플랜명은 엔진 nameKo 원문 표시(C-05 후속). 환불문구 경로만 갱신(내 라이브러리→내 이용권, 정책 불변). 문의 모달은 전 사이즈 공통(BottomSheet 분기 생략). 문의 관리 UI는 R11. 비즈니스 탭 가격은 CEO 확정대로 전면 배제.

### WI-R8 (P2b) — 계정 설정 분리 — BLOCKED (환경, 트리 그린)
- 구현 확인(전 세션 워킹트리, 읽기 검증): R8a 엔진 PUT /auth/profile(router.go:95·UpdateProfile·UpdateDisplayName) + E2E T18 4어설션(test-e2e.sh). R8b 화면 account 7종(index 셸+profile/security/billing/language/privacy/connections)·/disputes→/rights-center 301(미들웨어)·rights-center 개명·legal transPendingT/D 배너+6언어 키·legalBody 유틸+테스트 5·alias 301(nuxt.config rights-center 규칙, legal alias 경고 해소).
- Docker리스 게이트 전부 그린: go build/vet 무출력·go test -count=1 3패키지 PASS·tsc exit 0(volar 경고만)·vitest 69/69(신규 legalBody 5)·i18n:check PASS(UNUSED 212)·nuxt build 32.2s. gofmt -l 전수 표시는 HEAD부터의 CRLF 체크아웃 artifact(autocrlf=true) — R8 4파일 LF 변환본으로 포맷 정상 확인.
- 환경 블로커: Docker Desktop Start-Process 기동 후 엔진 API가 15분+ 계속 500("check if the server supports the requested API version", client 29.9.2/API 1.56, desktop-linux). 승인 대기 없이 폴링 종료. 하네스(78라우트×3뷰포트=234, config에 account 6+rights-center 반영済)·E2E(T18 포함 487/0 재확인) 미실행.
- 증거: evidence/r8/gates.txt(게이트 기록) + evidence/r8-i18n 20종(profile+rights × 5언어 × 2뷰포트, 전 세션분 유지).
- 다음 세션 재개 순서: ①Docker Desktop 복구 확인(docker ps) ②로컬 스택 기동(API 18081·DB 55432·Redis 16380·S3 14566·web dev 15173, NUXT_API_PROXY 전체 URL) ③E2E는 e2e-runner 컨테이너+--env-file, psql은 컨테이너 내 직통(전 세션 FAIL=컨테이너에서 host psql 접속·docker port 출력 없음) ④.nuxt 삭제 후 dev 재시작 → harness setup 직후 run(B-09) → evidence/r8 ⑤5언어 캡처 보완 후 R8 PASS 판정·커밋.

### WI-L1 (P2b) — 개인정보·약관 개정안 초안 — PASS
- 산출물 1건: `docs/legal/drafts/privacy-terms-amendment-l1.md`. 재정의서 §8 항목 전부 대비표(현행 v1.1 ↔ 개정안): 소셜 5사 수집 항목·이메일 인증·언어 설정·도입 문의·대기자 등록·국외 이전 구체화(Cloudflare 전 세계 엣지·LINE 일본·공급자 인증 서버)·쿠키 없는 분석 + 약관(용어 2·신규 조항안 3). 가격·생성물 권리 창작 없음.
- D7은 별도 문서 유지(합치지 않음 — D7=환불 산식, L1=수집·이전, 게시 시 묶음 검토).
- 게시 금지 준수: ToS·privacy md·seed·마이그레이션·DB 무변경(git status 확인).
- 검증: cspell 0건(초회 18건→사전 5단어 추가로 해소, 초안 §5에 기록). 실행형 게이트 해당 없음(D7 §7과 동일 논거).
- 증거: evidence/l1/gates.txt.

### WI-A1 (P2c) — 소셜 로그인 5사 + C-11/C-12/C-16 — BLOCKED (환경, 트리 그린)
- 엔진: 어댑터 인터페이스+5사 구현(service/oauth_adapters.go 신규) — Google·Kakao 이식(PKCE 추가·이메일 검증 플래그 파싱), Apple(ES256 client_secret·form_post·JWKS id_token·최초 이름·릴레이 허용)·Naver(plain flow·state 에코·미검증)·LINE(OIDC JWKS·nonce) 신규. C-11(미검증 이메일 기존계정 자동연결 거부→ERR_OAUTH_LINK_LOGIN_REQUIRED)·C-12(서명+10분+1회용+HttpOnly 쿠키 nonce, Apple만 SameSite=None)·C-16(교환 코드 60초 1회용+POST /auth/oauth/exchange, 콜백 URL 토큰 0). 이메일 미제공→임시 토큰→/signup/complete-email(A2에서 인증 후 생성). 연결/해제 API(link-token 5분·마지막 수단 해제 불가)+마이그레이션 000023(oauth_accounts CHECK 5사, A2는 000024로 재이월)·부팅 로그(빠진 변수명만). WeChat 미구현(인터페이스+문서 고지).
- 화면: SocialButtons 브랜드 5색(socialBrand+테스트 2)·callback 교환 코드 플로우·complete-email 배선(pending/무토큰 상태)·connections 실구현(목록·연결·2단계 해제·linked 토스트)·login locale 전달. auth/account/errors 6로케일(신규 13키·삭제 2키, ko원문·en자필·ja/zh/th Beta·en-XA).
- 문서: docs/OAUTH_SETUP.md(5사 콘솔 절차·콜백 URI·권한·빈칸 .env·키반영=재시작)·.env.example·API.md·ERD.md·E2E T12 확장(5사 목록·locale·wechat 400·콜백 무토큰·exchange/complete-email/link-token/connections/unlink 가드·CHECK 실측).
- 검증 그린: go build/vet/test ./...全(14 패키지)·gofmt(LF 환산)·tsc 0·vitest 71/71·checker PASS·worker-grant 14/14·scripts-check·cspell 0·nuxt build 54.4s. 스텁 테스트 18그룹全(PASS): 공급자별 흐름·Apple JWT+form_post·C-11 6케이스·C-12 8케이스·C-16 4케이스·complete/link/unlink/warnings·env 토글(config+service). 라우터 스모크 1건(handler): 등록 충돌 없음·콜백 리다이렉트 무토큰 HTTP 실측.
- 환경 블로커: Docker Desktop 실행 중이나 엔진 API가 v1.56·v1.44 모두 500(R8과 동일). Restart-Service 실패(권한). E2E 해당분·하네스·000023 up/down 실측·5언어 캡처 NOT TESTED. E2E 추가분은 기존 패턴 미러+bash -n 통과라 CI(ubuntu)에서는 수행 가능.
- 실키 발급·연동 실측은 deferred(PO). 증거: evidence/a1/{gates.txt,oauth-stub-tests.txt}.
- 다음 세션: Docker 복구 후 ①E2E 해당분(신규 A1 어설션) ②하네스(브랜드 버튼·connections 캡처) ③000023 up/down 왕복 ④5언어 캡처 → A1 PASS 판정.

### WI-A2 (P2c) — 이메일 인증 + 사용자 언어 — BLOCKED (환경, 트리 그린)
- P2c 분할 선언: A1(코드 33+로케일 6=39)에 A2(65)가 더해져 상한 50 초과 → P2c-1=A1(기전) · P2c-2=A2a 엔진(35: 신규 8+수정 27) · P2c-3=A2b 화면·E2E·문서(30: 신규 2+수정 16+로케일 6+E2E 1+문서 5)로 분할. 마이그레이션 번호는 런타임 확인(기존 최댓값 000023 → A2는 000024).
- 엔진(A2a): 000024(users.locale CHECK 5종·기본 ko, users.email_verified_at, email_verifications VERIFY|OAUTH·해시·24h·1회용, 기존행 인증 백필). POST /auth/email/verify/request(JWT·1rps/5burst·202)+POST /auth/email/verify/confirm(익명·성공 시 토큰 발급=매직링크 로그인)+PATCH /auth/me/locale+/auth/me에 locale·emailVerified. 미들웨어 RequireEmailVerified→403 EMAIL_NOT_VERIFIED(정본 literal)를 결제 2·pre-check/finalize 4·매장등록·생성에만 적용, 스트리밍 개방. A1 연결: complete-email 제출 시 검증메일 발송→confirm에서 FinishVerifiedSignup(토큰 consume·계정 생성/연결)+로그인. 탈퇴 시 검증행 삭제(withdraw tx).
- 메일 5종 템플릿: pkg/mail 템플릿(재설정·인증 × ko원문·en자필·ja/zh/th machine-draft Beta), 수신자 locale 렌더·미지정 시 ko 폴백. 재설정 기존 한글 인라인→템플릿 전환.
- 한글→code: 사용자 노출 Go 오류 30여 곳을 코드 정본+영문 메시지로 전환(코드·상태 유지 — checkAge 400 등 E2E 핀 준수). docs/API.md 코드표 완성(기존 누락 15종 추가·LOGIN_LOCKED→ERR_TOO_MANY_ATTEMPTS 정정). 유지 분류(콘텐츠·데이터·E2E 핀): 탈퇴 라벨·worker rejection_reason_ko·거절사유 기본문·Toss 사유·탐지 키워드·주석·ko 템플릿.
- 화면(A2b): verify-email 실배선(verifying/success/failed+재발송)+EmailVerifyBanner(미인증 상시)+complete-email 로케일 전달+language/LocaleSwitcher PATCH(설정면 토스트·스위처 fire-and-forget)+감지순서 ① users.locale 적용(미들웨어 fresh-hydrate·로그인)+가입 3종→/login?registered=1 안내. auth/errors 6로케일(신규 12키·삭제 2키, ko원문·en자필·ja/zh/th machine-draft·en-XA).
- E2E: 픽스처 5계정 인증 백필(T02)+T01 스키마 3종+T19 30어설션(가입→미인증 로그인→로케일→4종 게이트 403→스트리밍 200→request 202→confirm 로그인→replay 401→결제 개방→CHECK 실측)+탈퇴 검증행 0. E2E 해당분은 bash -n만 통과(CI ubuntu에서 수행).
- 검증 그린: go build/vet/test ./...全(18 패키지)·gofmt(LF 환산, entities 정렬 1건 수정)·tsc 0·vitest 75/75(신규 locale 4)·checker PASS(UNUSED 212 동일)·worker-grant 14/14·scripts-check·cspell 신규 0·nuxt build 35.6s.
- 환경 블로커: Docker 엔진 API 500 지속(docker version 실측)+docker ps 행. 하네스·E2E 실측·000024 up/down·5언어 캡처 NOT TESTED. SMTP 실발송 deferred(발송계정 없음 — 스텁/단위/E2E로 대체 검증).
- 증거: evidence/a2/gates.txt. 다음 세션: Docker 복구 후 ①E2E T19 포함 전수 ②하네스(verify/banner/connections 캡처) ③000024 up/down 왕복 ④SMTP 실발송(PO) → A2 PASS 판정.

### WI-S1 (P3a) — URL·slug·handle — BLOCKED (환경, 트리 그린)
- P3 분할 선언: P3 총량 ~91로 상한 50 초과 → P3a=S1(코드 32: 엔진 25+웹 7) · P3b=S2(코드 13+로케일 6) · P3c=S3+S4+S5(코드 40)로 분할. 마이그레이션 번호 런타임 확인(최댓값 000024 → S1은 000025).
- 엔진: 000025(tracks/playlists short_id UNIQUE+slug NOT NULL·충돌방지 md5 백필, users.handle nullable UNIQUE+CHECK, analytics_events, indexnow_log)+domain/seo.go(Slugify 유니코드유지·NewShortID base62-8·handle+예약어40·SplitSlugID·canonical·allowlist/PII정제)+repository/seo.go(3형태 param 해결·canonical+moved·CreatorByHandle·ClaimHandle·사이트맵커서·LandingCount·이벤트/개요/IndexNow 로그)+service/seo.go(Resolve 로케일분리·IndexNow 5언어팬아웃+일일상한+재시도1+감사·PurgeCache 토큰없음 스킵)+handler/seo.go+router(resolve/sitemap/landing/creators/events/oembed+별칭/users/handle/admin seo 3종). GetTrack/TrackPlaylists/Playlist.Get/ListTracks slug 수용. Review/Takedown/Playlist-Update 후 notifySearchAsync(IndexNow+퍼지, 백그라운드 30s). RegisterPartner creator_* 자동발급(재시도3·nil가드). ErrHandleTaken 409+API.md+errors 6언어.
- 화면: trackLink/playlistLink/detailParam + 7곳 정본 링크(TrackCard/Row·플레이어2·홈·플리홈·트랙연관). 비공개·콘솔 링크는 uuid 유지(301로 해결).
- 검증 그린: go build/vet/test ./...全(18디렉터리·14ok·4notest)·신규 단위 13그룹(domain 6+service 7)·bash -n(T20 55어설션·INSERT 6곳 short/slug 대응). seed 파트너 handle 부여.
- 환경 블로커: Docker 엔진 API 500 지속(docker ps 실패, 본 세션 재확인). E2E T20 런타임·하네스(90라우트×3=270)·000025 실측 NOT TESTED.
- 증거: evidence/s1/gates.txt.

### WI-S2 (P3b) — 메타·hreflang·JSON-LD·OG — BLOCKED (환경, 트리 그린)
- 모듈(exact): sitemap 9.1.0·robots 7.0.2·schema-org 7.0.1·og-image 7.0.7·site-config 5.0.3·web-vitals 6.2.3·satori·resvg-js·resvg-wasm(CF용)·unhead 3.4.3 상단호이스트. 설치는 --legacy-peer-deps(루트 vite5 vs peerOptional 단일 충돌만).
- utils/seo.ts(정본 URL·hreflang5+x-default=/en·seo.json 템플릿 기반 title/description·언어별 트랙설명 §3.1·share utm·embed/oembed·JSON-LD 7종)+usePublicSeo 반응형 헤드(og-image 위임용 image:null)+트랙/플리 SSR 전환(useAsyncData 게스트뷰·정본301·비공개CSR·Music*+Breadcrumb·AvibeOg·공유)+탐색 ?q= noindex+app 레이아웃 폴백+routeRules robots 19경로+OgImageAvibeOg.satori(1200×630·라틴텍스트/VERIFIED·CJK태국어 무문자폴백).
- 로케일: seo.json 6종(ko원문·en자필·ja/zh/th machine-draft Beta·en-XA 생성)+titles 8종+ERR_HANDLE_TAKEN+files 등록. 중대수정: 타이틀의 bare "|"가 vue-i18n 복수형으로 파싱되어 접미사 소실 → {'|'} 이스케이프+체커 예외(R3 선례).
- 검증 그린: tsc 0(volar 무해)·vitest 94/94(신규 seo 19)·checker PASS(UNUSED 213)·node 34.0s·CF 45.4s 이중빌드. dev 실측: 가이드 3언어 SSR(본문·메타·JSON-LD·공유utm 전수 존재)·OG 2종 렌더확인(라틴문자·무문자폴백, evidence/s2 PNG)·hreflang 상호·임베드 ?lang=th 전환·가이드 초안배너.
- 환경 블로커: 동일. 트랙/플리/크리에이터 풀SSR·5언어 PNG·태국어 OG픽셀(동일 plain 분기, 단위+ko픽셀로 대체검증) 미실측.
- 증거: evidence/s2/{gates.txt,og-en.png,og-ko-plain.png}.

### WI-S3 (P3c) — sitemap·robots·IndexNow·llms.txt — BLOCKED (환경, 트리 그린)
- 엔진 seo/sitemap 6종(READY/공개·커서·1h캐시·랜딩12게이트·가이드/정적 고정)+웹 __sitemap__/engine-urls(25페이지 순회·가이드/정적 로컬·5언어 확장+6개 alternates·x-default=/en·1h캐시). 인덱스+6분할 검증(guides 15·static 50 URL, lastmod 실측).
- robots §2+§6.2 합집합(Allow /+Disallow 17종, 언어 5배 자동확장, Sitemap 절대경로). /embed 크롤허용 유지, AI봇 허용. dev 차단·prod-mock 전수 확인.
- IndexNow: 발송큐+로그+상한+재시도+키파일 미들웨어(/{key}.txt). 키 없이 skipped_no_key x5 기록 확인 경로(E2E). 캐시퍼지 자리+수동 API.
- llms.txt(사실형+5언어링크+엔드포인트+조사일자)+콘솔검증 슬롯 4종(빈값 무렌더)+Daum 파일자리 문서화.
- 교훈 3종(실측): ①sitemapName 명시+`sitemaps` 병용 시 인덱스 자기-302 루프 → 생략(멀티 기본값). ②/api/** 프록시가 서버라우트 삼킴 → /__sitemap__/ 이동. ③모듈 auto-i18n은 커스텀소스에 ko+x-default(ko)만 부여 → 엔드포인트에서 명시 확장.
- 검증 그린: dev 전수(robots 양모드·인덱스200·자식200·키파일·400케이스). E2E T20 해당분 bash -n.
- Deferred: 콘솔 등록·IndexNow 키 발급(PO), 엔진데이터 동반 재검증.
- 증거: evidence/s3/gates.txt.

### WI-S4 (P3c) — 임베드·크리에이터·랜딩·가이드·공유 — BLOCKED (환경, 트리 그린)
- 임베드 2종(embed 레이아웃·SSR경량·noindex+크롤·?lang=·AVIBE역링크·비공개결측·embed_loaded). oEmbed 2경로+discovery.
- 크리에이터 페이지(UI §3.8: 헤더+팔로우+공유·4탭·ProfilePage 스키마). 크리에이터 팔로우 UI(R6 잔여) 해소.
- 랜딩 3종+공용컴포넌트(게이트 12미만·미지키·엔진단절→404, SSR, 타이틀/설명/브레드크럼/OG). explore.vue→explore/index.vue 이동(파일+디렉터리 공존 불가, 라우트명 동일·rename 면제).
- 가이드 허브+3편(매장·Suno·유튜브, web/content ko/en 정본·ja/zh/th 초안배너, 2026-11-11 조사근거+확인일자, 사실표기만, Article+FAQPage). 문구 PO검토 후 게시(브랜치 미배포이므로 코드만).
- 공유 버튼utm 4채널+복사, 전 상세페이지 배치+share_clicked.
- 검증 그린: dev(임베드 noindex+th전환·가이드 3언어 SSR·랜딩 404·explore 200 회귀). tsc/vitest/checker 빌드 공통게이트 그린.
- Deferred: 임베드 권한E2E·팔로우E2E·랜딩12실측·가이드 PO검토·5언어 PNG.
- 증거: evidence/s4/gates.txt.

### WI-S5 (P3c) — 성능예산·RUM·분석·앱링크·퍼지 — BLOCKED (환경, 트리 그린)
- LHCI: lighthouserc.cjs(모바일1회·성능0.90err·SEO1.0err·a11y0.9warn·스크립트170KBerr·4URL)+CI e2e잡 확장(웹빌드+프리뷰+READY발견+autorun, FIXTURES=1).
- RUM 플러그인(web-vitals LCP/INP/CLS→web_vital)+분석배처(무쿠키·20건/5s·hide플러시·탭난수세션·실패폐기). §1.4 15종+web_vital 전수배선(가입3·로그인·재생2·플리생성·좋아요·문의2·결제2·언어·공유·임베드·대기자).
- 엔진 POST /events(allowlist·상한·PII제거·1-50건·202)+개요 signupsByLocale+inquiries(전환입력).
- 앱링크 2종(서명값 플레이스홀더·apple JSON content-type)+CF분석 슬롯+퍼지(자동훅+수동API·토큰없음 false).
- 검증 그린: dev(well-known 2종 content-type·llms·RUM무오류)·단위(SubmitEvents·Purge httptest)·cspell 0(사전 49단어·쌍둥이동기·로케일/스크립트 범위제외 HEAD대조)·worker-grant 14/14·공통게이트.
- Deferred: LHCI 첫 그린(CI)·RUM수집·퍼지 실호출·서명값/토큰(PO)·관리자 퍼널UI(R11)·Baidu 보류(정본).
- 증거: evidence/s5/gates.txt. 다음 세션(Docker 복구 후): ①000025 up/down ②E2E T20 포함 전수 ③하네스 setup→run 270 + 5언어 캡처 ④LHCI 디스커버리 확인 → S1~S5 PASS 판정.

### WI-R9 (P4a) — 크리에이터 스튜디오 — BLOCKED (환경, 트리 그린)
- P4 분할 선언: P4 총량 ~90으로 상한 50 초과 → P4a=R9(코드 19+로케일 6+삭제 6) · P4b=R10(엔진 24+화면 22+로케일 6) · P4c=R11(코드 16+로케일 ko/en)로 분할.
- 화면: layouts/studio.vue(사이드바 7메뉴·키화)+pages/studio 7종 — index(대시보드: KPI 4·할일 2종·최근 타임라인·Top5·70:30 카드)·create(Generations 이식+키화, G5 재구성 예정)·upload(4단계 재구성: 파일→출처·플랜·증빙+사전검증→메타데이터→확인+서약4+제출, 엔진 호출순서 불변)·catalog(표+상태필터+타임라인 드로어+인증서 링크)·proofs(구독증빙 이식)·earnings(신규: 기존 API 범위 내 재생·비중·70:30 안내)·settings(활동명·핸들·정산계좌). 컴포넌트 studio/KpiCard·ReviewTimeline 신규. 구 /partner 5면+레이아웃 삭제, 미들웨어 301 매핑(/partner/stores→/business/console/stores 포함·쿼리 유지).
- 스토어: partnerUploadStore에 canLeaveStep1·step3Blocked 추가(기존 finalize/toggle/reset 무변경, 테스트 유지). 네비: UiNavbar 역할별 진입(스튜디오/콘솔/관리자)+UiFooter 스튜디오 링크. LEGACY_NAMES 'studio'→'studio-dashboard'.
- 로케일: studio.json 6종(ko원문·en자필·ja/zh/th machine-draft Beta·en-XA 스크립트 생성)+common titles 12종 교체(partner-* 삭제·studio/business-console/admin 3종 추가)+nav 2종.
- 정본 대비 조정(기록): ①업로드 ①단계 커버 제외(커버 제출 엔진 미지원 — 앨범 커버만 존재) ②③단계 소개문 제외(finalize에 description 필드 없음) ③대시보드 "이번 달 재생" 대신 누적 재생(월별 집계 API 없음) ④/studio/create는 G5에서 PromptBox 구조로 재구성 예정.
- 검증 그린: tsc 0(volar 무해)·vitest 98/98(신규 4: roles 3+업로드게이트 1)·checker PASS(allowlist에서 partner 6종 제거)·이중빌드(node 28.6s·CF 43.2s)·worker-grant 14/14·cspell 신규 0.
- 환경 블로커: Docker 엔진 API 500 지속(Start-Process 후 5분 폴링 DOCKER_DOWN_AFTER_5MIN, 본 세션 실측). 하네스 309(103×3)·E2E T21·5언어 캡처 NOT TESTED.
- 증거: evidence/r9/gates.txt.

### WI-R10 (P4b) — 비즈니스 콘솔 + BUSINESS 역할 — BLOCKED (환경, 트리 그린)
- 엔진: 000026(users.role CHECK+BUSINESS, user_roles PK+CHECK+backfill, business_profiles, store_businesses.owner_user_id NOT NULL+인덱스+백필, partner_id nullable 전환, 매장보유 PARTNER에 BUSINESS 'migration' 부여). Claims.roles+ValidateAccess 재확인+RequireRole 다중허용(CurrentRoles, primary 폴백). RegisterBusiness(KR 체크섬 필수→APPROVED, 해외→PENDING 수동)+user_roles 'register'. /business 콘솔 그룹(프로필+매장/기기/편성/로그 9종, owner 키·RequireEmailVerified 매장등록)+기존 /partner/stores 유지(호환). StoreService owner 변형 9종+UpsertSchedule 비공개플리 검사를 owner 기준으로 통합(파트너 경로 동등). Admin SetUserRole BUSINESS 허용+primary grant 동기화(구 grant 유지로 양쪽 콘솔 도달)+role-grants/oauth-stats/businesses/review 4종. adminctl set-role/create-admin 동기화. OAuth 신규가입 grant 2곳. 단위 8종(claims roles·폴백·재확인·RegisterBusiness 검증 5케이스·ValidRole·RequireRole 2·owner 격리).
- 화면: layouts/business.vue(5메뉴)+console 5종(개요 KPI 4+오프라인 알림·매장=Stores 이식+consoleApi·플레이어=기기표+현재곡+발급·라이선스=매장 라이선스+인증곡·청구=계약 요약)+business.vue→business/index.vue 이동(라우트명 유지)+signup/business 실가입 연결(국가 5종·KR bizNo 필수)+store/player 5언어 키화+기기언어(?lang> navigator>저장값).
- 로케일: business.json console+player 6종(ko·en·ja/zh/th draft·en-XA 스크립트)+auth.business 8종+needBusiness.
- 데이터 이관: 마이그레이션 내 백필(steps 4·5)+scripts/role-migration-dryrun.sql(G-ROLE 드라이런 5종, 로컬에서만 실행·운영 금지). start-local.sh에 batch23~26 가드 추가(A1/A2/S1 누락분 포함).
- E2E T21 54어설션(스키마 3+제약 2·백필 4·KR가입 7·해외+승인 8·콘솔 19·격리/가드 5·admin 6·bash -n 통과).
- 문서: API.md(가입·me.roles·콘솔 9종·admin 4종)·ERD.md(관계 3·CHECK·신규표 2).
- 검증 그린: go build/vet/test ./...全(18디렉터리)·gofmt(LF 환산, entities Track/Playlist 선행 어긋남 2건 정리)·tsc·vitest 98·checker·이중빌드·bash-n 2종·worker-grant·cspell 신규 0(schs→weekRows·FEMAIL 계열 개명).
- 환경 블로커: 동일(Docker 다운). 000026 up/down 왕복·드라이런 실측·T21 런타임·하네스·5언어 캡처 NOT TESTED.
- 증거: evidence/r10/gates.txt.

### WI-R11 (P4c) — 관리자 — BLOCKED (환경, 트리 그린)
- 키화: layouts/admin.vue+10면 전수 ko/en 키화(admin.* 약 230키)+신규 inquiries(문의 목록·상태 변경)·role-grants(이관 로그 표)·social(공급자별 현황) 3면+네비 13메뉴+users면에 비즈니스 승인/정지 버튼(pending/approved 목록 대조)+check-i18n admin.* parity 면제(ko/en 필수·나머지 생략)+admin 레이아웃에서 ko/en 강제(en 폴백)+토큰 v2(text-amber-300 1건→accent-amber)+날짜 로케일화(ko-KR/en-US)+토스트 apiError 전환.
- 검증 그린: tsc·checker PASS(allowlist에서 admin 11종+player 제거, UNUSED 300 경고만)·이중빌드.
- 환경 블로커: 동일. admin 13면 하네스·ko/en 캡처 NOT TESTED.
- 증거: evidence/r11/gates.txt. 다음 세션(Docker 복구 후): ①000026 up/down+드라이런 실측 ②E2E T21 포함 전수 ③하네스 setup→run 309+5언어 캡처(관리자 ko/en) ④R9~R11 PASS 판정.

### WI-G0 (Track G) — 데이터 모델 + AVIBE 공급자 게이트 — BLOCKED (환경, 트리 그린)
- 마이그레이션 000027: generation_models(레지스트리)·generation_jobs에 model_version·seed·params(기존 params_json)·safety_result·watermark_id·user_id·tracks.origin(EXTERNAL_AI|AVIBE_MODEL)·training_consents(꺼짐)·data_provenance_ledger·dataset_manifests(추가전용 트리거)·credit_ledger(추가전용, 미사용). 공급자 CHECK 3종에 AVIBE 추가.
- 게이트: domain.ProviderAvibe + ValidProvider 편입, licensegate AVIBE 분기(정책조회 후 AvibeChecksPassed 필수, 티어·차단일·구독 스킵). Finalize에서 SUCCEEDED 작업의 워터마크+안전판정(similarity·watermark 포함)으로 서버 유도, 자기선언 불가. AVIBE 트랙 origin=AVIBE_MODEL. 유사도 축은 G2 졸업 전 충족 불가 → 구조적 fail-closed (G-MODEL).
- 시드: AVIBE 정책행(ON CONFLICT-safe). E2E T22 30어설션(스키마 14·정책·가드 4+스트라이크·핀 5·AVIBE 403 2·origin 1)+start-local batch27.
- 검증 그린: 네이티브 PG17 스크래치 왕복(up→객체 5+5+1+트리거 3→append-only 거부 2종→down 복원→fail-closed 중단→재up, 후 DROP). go build/vet/test ./...全·validator 4·AvibeProvenanceOK 8·bash-n. 문서 API/ERD 반영.
- Finding(범위 밖): 000025 substr(bigint) 전버전 실패 — SEO 후속 전달, G0 미수정. down 비트랜잭션 부분적용은 기존 convention 유지.
- 환경 블로커: Docker 엔진 API 500 (Start-Process 후 장시간, 본 세션 2회 실측). T22 런타임·하네스 NOT TESTED.
- 증거: evidence/g0/gates.txt.

### WI-G1 (Track G) — 생성 계약 v2 + 스키마 — BLOCKED (환경, 트리 그린)
- Go pkg/generator: contractVersion 2 + modelVersion/seed 송신·watermark:true 강제, 응답 weightsHash/safety/watermarkId 파싱, safety 실패 시 작업 실패, audioUrl 대용량 fetch. v1 응답은 공란 provenance로 디코드(호환).
- generator/server.py: v1/v2 이중 수신 (v2 검증: watermark:false→400·duration 1..600·variations 1..4·seed≥0), v1은 기존 응답, v2는 provenance+audioBase64|audioUrl(presigned PUT). 가중치·워터마크 없이 503 fail-closed. healthz에 contractVersions+weightsHash.
- 스키마: docs/generation/model-card.schema.json(G-MODEL 8항목)·dataset-manifest.schema.json(허용출처 6종). 워커 핀 전달+UpdateResult 증거 저장(v1은 요청 핀 유지).
- 검증 그린: generator 단위 4종(v2·v1호환·safety·audioUrl)+기존 5 유지, server 오프라인 8/8, 스키마 JSON 유효, go build.
- 환경 블로커: 동일 (E2E 해당분 미실행).
- 증거: evidence/g1/gates.txt.

### WI-G2 (Track G) — GPU PC PoC — BLOCKED (환경: GPU 부재, NOT TESTED)
- 드라이버 generator/poc/run_poc.py: ①ACE-Step 추론 ②워터마크 삽입·검출 ③지문 유사도 ④리포트. 결측 시 NOT TESTED 리포트 후 exit 0, 생성물 운영 업로드 금지.
- 이 PC 실행: NOT TESTED (GENERATOR_MODEL_DIR·acestep·audioseal·CUDA 전부 부재) → evidence/g2/poc-report.json. 무리한 대체 없음.
- 라이선스는 GPU PC 설치 산출물 해시로 재확인 (단정 없음).
- deferred: 자택 GPU PC 실측 (검출률·유사도 포함).

### WI-G3 (Track G) — GPU 운영안 비교표 — PASS
- docs/generation/gpu-options.md: 웜/콜드 분리 1분당 원가식+입력 출처표, 후보 4곳×6열 비교표(TBD), 계약 전 체크 5항, G-MODEL 결정 게이트. 가격 숫자 0·구매가입 금지 (가드레일 4).
- deferred: G2 실측+견적일 기입 후 PO 선택.
- 증거: evidence/g3/gates.txt.

### WI-G4 (Track G) — 평가 하네스 뼈대 — PASS
- generator/eval/run_metrics.py: synthetic 결정적 백엔드(self-test PASS)+fad/clap GPU 훅(미설치 시 안내 종료, syntheticWarning 표기).
- ab_schema.json: 블라인드 A/B 투표 기록. memorization.py: PCM 프리필터 지문(self-test PASS: 안정·정확일치 1.0)+스펙트럼 확장점.
- deferred: GPU 백엔드 구현·임계값 교정 후 워커 similarity 축 편입.
- 증거: evidence/g4/gates.txt.

### WI-G5 (Track G) — UI 자리 + 다국어 가드 — BLOCKED (환경, 트리 그린)
- 가드: 5언어 음성모방 구·스타일 구문(~스타일로/in the style of 등)·아티스트 시드 18종(단어경계)+전각/공백 정규화. 단위 12차단·4허용 통과.
- 화면: PromptBox·StyleTags(tags.* 재사용)·LyricsEditor·VariationCard 4종 + studio/create 플래그 패널(NUXT_PUBLIC_GENERATE_UI, 기본 꺼짐·길이/시드/변형2/이어·스템 자리·인증요청). studio.create.v2 20키×6로케일(ko/en 실번역·ja/zh/th 기계초안·en-XA 의사).
- 자리 재사용: 홈 만들기 칩(R5)·요금제 creditSoon(R7)·/create(R7+엔진) 그대로. 수용 조건(꺼진 상태 가시변화 /create뿐) 충족.
- 발견·수정: ①B3011 충돌(styleTags.ts→styleTagOptions.ts 개명) ②Nitro env 숫자 1 전달로 === true 실패→3종 허용 ③슬롯1 빈 문구 키 추가.
- 검증 그린: tsc 0·vitest 104/104·checker PASS·빌드 31.9s·grant 14/14·cspell 신규 0. Playwright+API스텁 실렌더: off(패널부재+목록+오류0)·on(패널+목록+오류0) PASS + 스크린샷 픽셀 확인 2종. /create SSR 200+generateUi:false.
- 환경 블로커: 동일. 하네스(이미 라우트 포함)·T22·5언어 캡처 NOT TESTED. 다음 세션(Docker 복구 후): ①E2E T22 포함 전수 ②하네스 setup→run(기본+플래그on 1회) ③5언어 캡처 ④G0·G1·G5 PASS 판정.
- 증거: evidence/g5/{gates.txt,studio-create-v2-on-1440.png,studio-create-v2-off-1440.png}.

### WI-R12 (P5) — 모바일 — BLOCKED (환경, 단독 게이트 전부 그린)
- ARB 5종(`app_{ko,en,ja,zh,th}`, 191키) + `l10n.yaml` + `gen-l10n` 커밋. Dart 한글 169행 전수 이관 → 잔여 0.
  실측 169행(정본 572는 과대 — 전수 확인). ja·zh·th machine-draft, 설정 화면 Beta 뱃지(G-L).
- 로그인/가입 분리(`/login`·`/signup`), 소셜 버튼은 `GET /auth/oauth/providers` 구동(키 없으면 숨김).
- 소셜: 시스템 브라우저(url_launcher externalApplication) + `avibe://oauth/callback` 복귀(app_links).
  엔진 `?platform=app` → 서명 state `.app` 마커 → 고정 avibe:// 리다이렉트(개방형 아님). 위조 마커 HMAC 거부 테스트 포함.
  iOS Apple 필수(스토어 정책)는 코드 주석+완료보고로 PO 전달(릴리스에 Apple 키 필수).
- 언어 선택+서버 동기화: `PATCH /auth/me/locale`(zh→zh-cn 변환), 로그인/가입 후 서버값 채택, 기기 zh-Hant→en.
- 하단 탭 5 유지(라벨 키화). 개수 동치이나 구성은 웹과 상이(재생·매장) — 정본 해석 메모로 보고.
- 테마: 웹 v2 토큰 전수 미러(brand/surface/text/status/verified/focus/radii) + CJK·태국어 시스템폰트 폴백.
- `check-mobile-release.sh`에 ARB 키 동치 검사(`--arb-only`, node 구현 — python3 스토어 스텁 회피).
- 검증 그린: analyze 0·test 22/22(신규 7)·ARB 5×191 OK·go build/vet/test 전종·OAuth 신규 3종·gofmt 자분.
- deferred: 모바일 complete-email 화면(`email_required` 우회 중), link-intent 복귀(취소 처리).
- 증거: evidence/r12/gates.txt.

### WI-R13 (P5) — 최종 하네스 + 번역 검수 패키지 — BLOCKED (환경, 트리 그린)
- strict 19파일 전수 키잉(api 2·스토어 4·게이트 2·legal 2·ui 4·페이지 5) + 소비자 연쇄 2(pricing·upload).
  `tKey`/`setKeyTranslator` 브리지 + `plugins/i18n-keys.ts`로 스토어 런타임 번역, 폴백 영어.
- 신규 키 119종×6로케일 병합(`r13-merge-locales.cjs`, en-XA pseudo 생성). `check-i18n` allowlist 삭제·전역 강제(CI 개명).
- 하네스 언어 차원: `LANGS`+URL 접두+stored-locale 고정+`HARNESS_LANGS` 부분 실행. 정적 매트릭스 1530면(102×3×5).
- `i18n-review/{ja,zh-CN,th}.csv` 1663행×3(키·원문·번역·캡처경로). 환불 ko 7/7 바이트 동일 검증.
- 발견·수정: ①인증서/결제확인 날짜 불기 회귀 → `dateISO`(서력 고정)+테스트 ②실 클릭 미배선 → DEFECT-R13-01 등록(보고 §5).
  ③`locale.test`에 감지순서 4종 추가(zh-TW→en 포함).
- 검증 그린: checker PASS·tsc 0·vitest 113/113·node 38s·CF 76s·grant 14·cspell 신규 0.
- 환경 블로커: Docker 엔진 API 500(Start-Process+5분 폴링 `DOCKER_DOWN_AFTER_5MIN`, 본 세션 3회 실측).
  E2E 전수·하네스 1530·결함 0 트리아지 NOT TESTED → 복구 후 1회 실행 시 R13 PASS 판정.
- 파일 상한: P5 합산 85 → **P5a(R12)=41·P5b(R13+보고)=44 분할 선언**. 부모는 반드시 2건+ 분리 커밋.
- 증거: evidence/r13/{gates.txt,harness-matrix.txt,defects.md} + i18n-review/ + 완료보고_리디자인_20261114.md.

### 검증 세션 (2026-11-17) — Docker 복구 후 E2E·하네스 완주, R8~R13 전항목 PASS
- 환경: Docker healthy, 로컬 스택(API 18081·DB 55432·Redis 16380·S3 14566·dev 15173).
  api/worker를 1aca943 코드로 리빌드, 배치 23~27+시드 적용(가드 전수 확인).
- E2E: e2e-runner 컨테이너+env-file로 3회 실행 → 최종 PASS 701 / FAIL 0 / SKIP 0
  (T01~T22, T18~T22 포함). 발견·수정 7건: ①00025 substr bigint 캐스트
  ②store TIME 스캔 500(제품) ③T11 owner_user_id 픽스처 ④T20 코드캡처 오타 2곳
  ⑤T21 백필 기대→런타임 경로+신규 리스너(명세 근거: 백필은 1회성) ⑥T22 동의 누락
  ⑦T14 short_id 실행별 고유화(UNIQUE 재실행 충돌).
- 하네스: setup→run 1530화면 2회 실행 → 최종 200×1500+404×30(게이트 정상)+500×0,
  hScroll·tiny·unlabeled·noAlt 0. 1차 지적 46건 중 실결함 1건
  (playlist-public 하이드레이션, ownerView 게이트로 수정·프로브 4종 확인),
  나머지는 게이트 404 기대 패턴 + 임베드 1회성 타임아웃(재측정 정상).
- 런타임 조치: JWT TTL 240분 임시 상향 후 15분 원복·api/worker 재생성, dev 서버
  종료+좀비 제거, .env 원복, 임시 비밀·스크립트 삭제. 운영 쓰기 0.
- 판정: R8·A1·A2·S1~S5·R9·R10·R11·G0·G1·G5·R12·R13 전부 PASS(위 상태판 반영).
  G2 NOT TESTED·외부 deferred(OAuth 실키·SMTP·GPU·법률·배포)는 PO 잔여로 유지.
- 미커밋(승인 대기): .gitignore·store.go·000025·test-e2e.sh·setup.mjs·
  playlists/[id].vue·진행로그·r13 증거 4종(baseline.json·screens.txt·gates·defects).
