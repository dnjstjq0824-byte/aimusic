# 진행 로그 — AVIBE 리디자인 연속 개발 (Muse Spark, 2026-11-14~)

브랜치: `feat/redesign-20261114` (from `docs/handoff-20261114` @ 6a5cd65).
모드: 연속 개발 (Phase 승인 대기 없음). 운영 쓰기 금지, main 직접 push/병합 금지.

## WI 상태판

| WI | 상태 | 커밋 | 변경파일 | 검증 | 캡처 | 남은 결함 |
|----|------|------|---------|------|------|----------|
| R0 | PASS | f76dd5a | 16 + evidence | 하네스 165/165 | baseline-samples 6종 | axe-core 미실행(의존성 상한) |
| P1 | PASS | 0091050 | 23 + 이동 92(면제) | CSR 동등 162/165·SSR 162/165·vitest 47·tsc·이중빌드·grant 14 | nuxt-csr/ssr json | legal alias 경고(개발 로그만, R8 해소) |
| R1 | PASS | 3753448+3 | 4분할(상한 준수) | tsc·vitest 47·하네스 9동일·tiny 0·빌드 | r1 json+showcase 2종 | 장식 raw 팔레트 일부 잔존 |
| R2 | IN PROGRESS (90%) | eae45f0(WIP) | 2분할 예정 | tsc·vitest 52·checker PASS·스팟 EN/JA/TH/ZH | r2-*-1440/390+switcher | 전체 하네스·prod 빌드 미완 |

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

### WI-R2 (P1b) — i18n 기반 — IN PROGRESS
- 모듈 @nuxtjs/i18n 10.6.0 exact(+@nuxt/kit 호이스팅). 전략 prefix_except_default(ko), langDir ../i18n/locales, files 배열 지연 로딩, vueI18n은 파일 방식(i18n.config.ts, 인라인 불가).
- 로케일 5+1: ko/en/ja/zh-cn/th + en-XA(dev 의사로케일, 생성 스크립트). 네임스페이스 17종 골격, common/errors/player 완성(ko 원문·en 자필·ja/zh/th machine-draft).
- 공개 페이지만 접두(27페이지 defineI18nRoute(false)), 미들웨어 ___xx 접미 처리 + i18n 타이틀(반응형) + 가드 토스트 + 후행슬래시 301(로케일 루트 제외).
- 이름 기반 내비 14파일 localePath 전환(v10은 ko 포함 전 접미사), route.name 비교는 base명 computed.
- 감지 플러그인(공개면 스킵)+제안 배너(세션 1회)+스위처 실동작(Beta 뱃지)+useFormat(+테스트)+useApiError(+테스트)+check-i18n(허용목록)+CI 잡.
- 검증 중간: tsc 0·vitest 52/52·check-i18n PASS·스팟 4언어 내비/플레이어/타이틀/배너/전환 전부 PASS. 전체 하네스·prod 빌드는 미완.
- WIP 커밋 (아래): 구현 완료, 검증 2건 남음. 재개 시 순서: ①dev 기동(npm run dev, 15173) ②하네스 setup+run → evidence/r2 (15분, historic: 캐시 문제 시 .nuxt 삭제 후 재시작) ③npm run build ④결과 기입 후 R2a/R2b 분할 커밋 정리(WIP 유지 or 분할) ⑤R3 착수.
- 주의: 하네스 실행 중 .nuxt 캐시가 오래되면 오탐(151 bad 사례) — 하네스 전 dev 재시작 권장. B-09 토큰 회전 때문에 하네스는 setup 직후 바로 실행.
