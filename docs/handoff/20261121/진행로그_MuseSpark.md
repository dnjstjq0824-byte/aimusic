# 진행로그 — Muse Spark 후속 작업 (2026-11-21)

기준: main `2dc89cd` (640b40c 포함 확인). 검수는 로컬 스택 + 하네스 계정.
운영 쓰기 0.

## WI-N0 기준선 확인 (보고만)

- git: `git pull` 후 main 최신. `git log`에 640b40c 포함 확인
  (PR #8 생성 서버 infer·토큰, PR #9 runpod-gen.sh).
- 전달안내 §2 코드 확인 5건: `web/app/api/http.ts`,
  `web/app/utils/consoleMessages.ts`,
  `web/app/plugins/console-i18n.client.ts`,
  `web/app/components/console/ConsoleShell.vue`,
  `web/nuxt.config.ts`의 `build:manifest` hook (229행) — 전부 존재.

### 게이트 수치

| 게이트 | 결과 |
|---|---|
| `npm run typecheck` | 종료 0 (volar `sfc-route-blocks` 플러그인 경고 1건 — 로컬 환경 이슈 후보) |
| `npx vitest run` | 19파일 127/127 통과 |
| `node scripts/check-i18n.mjs` | PASS (`UNUSED-KEYS warn: 337`) |
| `go test ./...` | 전 패키지 ok (no-test 4곳 제외) |
| `flutter test` | 22/22 통과 (N0 외 기준선) |
| E2E 전수 | PASS 584 / FAIL 30 / SKIP 4 (2회 동일) |
| 화면 하네스 | (실행 결과 대기 — 아래 참조) |
| Lighthouse 4 URL | (프로덕션 빌드 측정 대기 — 아래 참조) |

### E2E 실패 30건 분석

- T14 검색 500, 11건: 서버가 raw UTF-8 쿼리(`q=밤하늘` 등)에 500
  반환. 같은 쿼리를 URL 인코딩하면 200 정상. 1차 원인은 Windows
  git-bash curl의 인코딩 깨짐이나, 깨진 쿼리에 500이 아닌 400을
  내려야 한다는 점에서 서버 개선 후보로 남김.
- T16/T20/T21/T22-AVIBE 401 연쇄, 17건: "missing bearer token"
  (만료가 아니라 빈 토큰). E2E 실측 133~163초라 TTL 15분 만료는
  아님 — 원인 정정. T16 PA_TOKEN 재발급·T20 short/slug의 `jget`
  파싱이 빈 값을 반환. `jget` 단독 10회 반복은 정상이라 E2E
  실행 컨텍스트 문제 후보 (shim heredoc·BODY 경로·인코딩).
  Linux CI에서 판정. N2/N9/N12/N13 변경 후 재실행해도 동일
  584/30/4 — 회귀 없음.
- T22 ja/th 금칙어 미차단, 2건: **원인 확정 — 환경 문제.**
  Windows curl이 CJK 프롬프트를 `?`(0x3F)로 깨뜨려 전송
  (DB 저장값 16진 확인). ko/en 차단은 ASCII 패턴(`bts`,
  `in the style of`)으로 동작. 가드 코드는 정상이며 Linux CI에서
  통과 예상. 실제 금칙어 목록 확장은 WI-N13에서 수행.

### 하네스·Lighthouse (작성 중)

- 1차 하네스: 959/1530 화면에서 중단 (2시간 초과 — hang이 아니라
  느린 진행이었음).
- 2차 하네스 (전수 완료, EXIT=0): 1530 screens, 1104 with
  errors/bad-responses. 상태: 200: 644, 404: 885, None: 1.
  ko 404: 9/306 (not-found 등 의도된 것). non-ko 404: 876/1224.
- **결함 발견 (N0 범위 밖, PO 보고)**: 하네스에서 non-ko 404가
  876/1224. 내역:
  - `charts.vue`의 `defineI18nRoute(false)` (11-14~): `/en/charts`
    등이 라우트 미생성으로 404. 운영에서도
    `https://avibemusic.com/en/charts` → 404 (읽기 전용 확인).
    공개 페이지 결함이 맞음.
  - 나머지 로케일 하위 경로 (`/th/playlists` 등): preview
    빌드·운영 모두 200 정상. dev 서버에서만 404가 관측됨 —
    장시간 실행 dev 서버의 상태 이상 후보 (N6 재실행 시 dev
    재시작 후 재확인).
  - 참고: 11-17에는 [...slug]가 soft-404 (200)를 반환해 하네스
    PASS였고, 11-17 이후 hard 404 전환 (`2957a4b`)으로
    badResponses가 됨.
  - 수정은 WI 범위 밖이라 하지 않고 완료보고에 결함으로 올림
    (charts 1행 수정이면 해결).
- 401 다수 (`/auth/me`, `/auth/refresh`): 하네스 토큰 TTL 15분
  만료 (WI-N6 배경과 일치). WI-N6에서 수정 후 재측정.
- Lighthouse 1차 (로컬 preview + `NUXT_COMPRESS_PUBLIC_ASSETS=1`
  빌드, 3회 중앙값, LHCI 0.15.1):
  - `/`: perf 0.63, script 324,007B, seo 1.00
  - `/en/tracks/e2e-sitemap-song-1c65595d`: perf 0.61, script
    407,538B
  - `/th/playlists`: perf 0.66, script 419,035B
  - `/ja/business`: perf 0.56, script 480,919B
  - CI 수치(perf 0.68~0.89, 193~205KB)와 차이: 로컬 preview는
    동적 응답 압축 미적용 (WI-N1 항목 1에서 수정).
  - 정정: preview 빌드에서는 로케일 하위 경로가 200 정상
    (N1 재측정 시 확인). dev 서버에서만 404가 관측됨 — dev
    서버 상태 이상 후보로, N6 하네스 재실행 시 dev 재시작 후
    재확인. 운영 `/en/charts` 404는 별도 확인 필요
    (아래 긴급 회귀 항 참조).

## WI-N3 DB 백업 외부 보관 (R2) — PASS (운영 적용은 G-2)

- 변경: `deploy/backup-db.sh` 신규 (운영 스크립트 + R2 업로드·크기
  검증), `.env.example` (`R2_DB_BACKUP_BUCKET`), `docs/PROD_RUNBOOK.md`
  (§8 R2 보관·수명주기 30일·복원 절차).
- 검증 (로컬 스택 + LocalStack 테스트 버킷):
  - 덤프 746,306바이트 → 업로드 → 크기 검증 OK, exit 0.
  - 없는 버킷: ERROR 로그 + exit 1.
  - 복원 리허설: 다운로드 → scratch DB 복원 → 테이블 50/50,
    users 460/460, tracks 233/233 일치 → scratch 삭제.
  - `bash -n` OK, shellcheck 경고 0.
- 남은 문제: 운영 설치·R2 버킷·수명주기 규칙은 PO 승인 후 (G-2,
  P8). R2 키는 PO가 운영 `.env`에 입력.

## WI-N4 웹 단독 무중단 배포 스크립트 — PASS (운영 실행 없음, G-2)

- 변경: `deploy/rollout-web.sh` 신규 (web/rollback/smoke +
  `--dry-run`), `.gitignore` (`deploy/.web-prev-image`),
  `docs/PROD_RUNBOOK.md` (§4b 사용법).
- 검증:
  - `bash -n` OK, shellcheck 경고 0 (2건 수정 후).
  - `--dry-run web`: 0~5단계 계획 출력 확인 (사전조건·합본·
    Worker·전송·교체·스모크).
  - `--dry-run rollback`: prev 파일 없을 때 거부, 있을 때
    `docker tag` + 재생성 + `wrangler rollback` 계획 확인.
  - 합본 로직 실측 (로컬 이미지 ca25431 + b8c045a): 구 387개 →
    합본 519개, 신 파일 전부 포함 확인.
- 남은 문제: 운영 실행 0 (G-2). 첫 실행 시 prev 파일이 없어 합본이
  신본만 됨 (경고 출력, 정상).

## WI-N5 CI 정비 — PASS (CI 실행은 push 후)

- 변경: `.github/workflows/ci.yml` (checkout v7, setup-node v7,
  setup-go v7, setup-java v6 — 전부 action.yml `using: node24` 확인,
  flutter-action v2는 composite이라 유지), 전 잡 `ubuntu-24.04`
  고정, mobile-release-gate에 `actions/cache@v6` Gradle 캐시 +
  release 빌드 1회 재시도; `web/lighthouserc.cjs` `numberOfRuns: 3`.
- 검증: YAML 파싱 OK. CI 전 잡 통과·경고 0은 push 후 CI 실행으로
  확인 (NOT TESTED — 로컬에서 CI 실행 불가).
- 남은 문제: 없음 (CI 결과 확인만 남음).

## WI-N10 구 미디어 볼륨 대조 리포트 — PASS (운영 실행·삭제는 P7)

- 변경: `scripts/r2-media-compare.sh` 신규 (읽기 전용, 키·크기·ETag
  대조, 일치/누락/불일치/여분 분리), `docs/PROD_RUNBOOK.md` (§8
  G-R2 절차 + 리포트 양식).
- 검증 (LocalStack 가짜 4버킷):
  - 불일치 픽스처: match=1 missing=1 mismatch=1 extra=1 분리
    출력, exit 1.
  - 전체 일치: match=3/1, exit 0 (`COMPARE OK`).
  - `bash -n` OK, shellcheck 경고 0. 리허설 버킷 삭제 완료.
- 남은 문제: 운영 실행과 볼륨 삭제는 PO 승인 후 (P7, G-R2).

## WI-N11 생성 서버 이미지화 + 서버리스 준비 — 부분 PASS

- 변경: `generator/Dockerfile` 신규 (server/gpu 멀티 타겟,
  ACE-Step @ ca1e85f 고정, 가중치 베이크 선택,
  `GENERATOR_WEIGHTS_HASH` 스탬프),
  `docs/generation/runpod-recipes.md` 신규 (Pod·Serverless 레시피·
  콜드스타트 측정법), `docs/generation/gpu-options.md` (11-21 실측치
  기입, 가격은 Pod 표시 요금 인용만),
  `generator/poc/run_poc.py` (`--endpoint`·`--token` 원격 모드).
- 검증:
  - server 타겟 로컬 빌드·기동: `/healthz` `modelLoaded: false`,
    토큰 없이 401, 토큰 있고 모델 없이 503 (GPU 없는 환경 기준).
  - 원격 PoC: stub 서버 대상 PASS (오디오 16,044바이트 수신·
    핑거프린트 기록), 토큰 불일치 시 401 실패, 로컬 모드 회귀
    NOT TESTED 유지. eval self-test 2종 PASS.
- 남은 문제 (BLOCKED/P6): 이미지 방식 콜드스타트 비교 실측과
  5Hz LM 경로 실측은 GPU Pod가 필요. Pod 켜기·끄기는 PO 또는
  Claude 담당이라 레시피 §C 표는 NOT TESTED로 둠.

## WI-N12 워커 → 생성 서버 토큰 인증 — PASS

- 변경: `backend/internal/pkg/generator/generator.go` (token 필드·
  Bearer 헤더·401 → `ErrUnauthorized`), `backend/cmd/worker/main.go`
  (401은 재큐잉 없이 실패 기록), `backend/internal/config/config.go`
  (`GENERATION_BACKEND_TOKEN`), 테스트 3종 신규 (헤더 있음/없음·
  401·토큰 비노출), `.env.example`·`docs/PROD_RUNBOOK.md`·
  `generator/README.md` 문서.
- 검증: generator 패키지 12/12 통과, `go build ./...` OK,
  `go test ./...` 전 패키지 ok. 401 에러 메시지에 토큰 없음
  (테스트로 검증). gofmt 로컬 지적은 CRLF 체크아웃 아티팩트
  (Linux CI에서 통과, 수정 파일 포맷 문제 없음).
- 남은 문제: 실 서버 대상 왕복은 Pod 필요 (P6). 운영 연결은
  G-MODEL 통과 후.

## WI-N9 모바일 이메일 보완 화면 — PASS (실키 실측은 P1 이후)

- 변경 (엔진): `backend/internal/handler/auth.go` (모바일 복귀를
  `oauthError=email_required`에서 `emailToken=<1회성 토큰>`으로).
- 변경 (모바일): `features/auth/complete_email_screen.dart` 신규
  (웹과 같은 흐름: 입력 → `POST /auth/oauth/complete-email` → 인증
  메일 안내), `services/social_auth_service.dart`
  (`SocialEmailRequired`·`completeEmail()`), `features/auth/login_screen.dart`
  (토큰 복귀 시 화면 이동), `main.dart` (`/complete-email` 라우트),
  ARB 5개 언어 10키씩.
- 검증: `flutter analyze` 0, `flutter test` 28/28 (위젯 2종 +
  토큰 복귀 1종 신규), ARB 동치 검사 5×210 OK.
- 남은 문제: 실키 실측은 P1 이후.

## WI-N2 인증 배지 클릭 연결 — PASS

- 변경 (엔진): `domain.Track.CertNumber` (`*string`, 없으면 null),
  `catalog.go` (creator 조인에 인증서 LEFT JOIN, 일반 목록은
  스칼라 서브쿼리 — 단일 문장, N+1 없음), 스캔 3종 + 테스트 3종.
- 변경 (웹): `api/types.ts` (`certNumber?`), `TrackRow`·
  `FullscreenPlayer` (번호 전달), `TrackCard` (배지 추가),
  트랙 상세 (`track.certNumber` 직접 사용 — 기존 slug URL에서는
  cert 별도 fetch가 404로 배지가 안 보였음).
- 변경 (모바일): `core/models.dart` (`Track.certNumber`·
  `TrackCertificate`·`CertificateView`), `services/api_service.dart`
  (`verifyCertificate()`), `features/certificate/certificate_screen.dart`
  신규, `main.dart` (`/certificate/:number` 라우트),
  `features/player/player_screen.dart` (AI·라이선스 탭 배지 + 탭
  이동), ARB 5개 언어 10키씩, `test/models_test.dart` 3종.
- 검증:
  - 엔진 단위 3종 통과, `go test ./...` 전부 ok.
  - 쿼리 수 불변: 목록 API = COUNT 1 + SELECT 1 (pg 로그),
    상세 = SELECT 1. JOIN 추가로 쿼리 수 증가 0.
  - 실연동: 미발급 목록·상세 `certNumber: null` (키 존재),
    발급 상세 `certNumber: 'AVB-WSP65PRTFD'`.
  - 웹: 브라우저에서 발급 트랙 상세에
    `/certificate/AVB-WSP65PRTFD` 링크 렌더 확인, 미발급은
    span (링크 없음). typecheck 0, vitest 127/127, i18n PASS.
  - 모바일: `flutter analyze` 0, `flutter test` 28/28, ARB
    5×210 OK (N9 키 포함).
- 남은 문제: 없음.

## WI-N1 Lighthouse 예산 회복 — NOT PASS (부분 개선, 구조적 한계)

- 변경: `web/server/middleware/compress-dynamic.ts` 신규 (CI 빌드만
  동적 응답 br 압축, 운영 이미지 무변경),
  `web/nuxt.config.ts` (`runtimeConfig.compressDynamic`),
  `web/app/pages/index.vue` (StarterRail·차트 TrackRow
  hydrate-on-visible, 모달 2종 지연 로드),
  `lucide-vue-next` 제거 (미사용 확인, 번들 효과 0).
- 항목별 결과:
  1. 측정 환경 보정: HTML 49,206→8,819B, messages.json
     57,463→16,082B (br). `numberOfRuns: 3` (N5 적용).
  2. CSS: entry 198KB raw/34KB br, 897 셀렉터 전부 사용 중
     추정 — 감소 여지 없음.
  3. 지연 하이드레이션: Nuxt 4.6 지원 확인 후 적용. Lighthouse가
     전체 로드하므로 transfer 효과 없음 (TBT는 원래 0).
  4. 엔트리: 클라이언트 번들에 vue-i18n 컴파일러·schema-org 없음
     (이미 최소), SiteConfig 경량. 모듈별 크기 기록済.
  5. LCP: font-display swap 적용済. 단 `/ja/business`에서 일본어
     폰트 912KB (27파일) 발견 — FCP 8.3초의 주범, 서브셋
     재설계 필요 (별도 WI).
- 재측정 (3회 중앙값):
  - `/`: perf 0.90 (0.88/0.90/0.94), script 331,230B
  - `/en/tracks/…`: perf 0.63, script 416,203B
  - `/th/playlists`: perf 0.69, script 428,198B
  - `/ja/business`: perf 0.57, script 491,184B
- 판정: `/`만 perf 달성. script 예산 (174,080B)은 전 URL 미달
  (미사용 JS 0 — 전부 사용 중이라 삭제 여지 없음).
  `lighthouserc.cjs`는 `warn` 유지 (error 복구 불가).
- 회귀: typecheck 0, vitest 127/127, check-i18n PASS.
  E2E·하네스·콘솔 언어 전환 검사는 N6 완료 후 일괄 수행.
- 남은 문제: perf·script 예산 달성은 아키텍처 변경 필요
  (폰트 서브셋 재설계, JS 구조 다이어트). 별도 WI로 제안.

## WI-N6 화면 하네스 보완 — PASS (전수 실행 중)

- 변경: `web/e2e/screens/setup.mjs` (BUSINESS 역할 + 승인 프로필
  계정 + 매장 2종), `web/e2e/screens/config.mjs` (business 역할
  추가), `web/e2e/screens/run.mjs` (컨텍스트 시작 시 토큰 갱신 +
  `.auth.json` 저장), `web/e2e/screens/console-lang.mjs` 신규
  (콘솔 3종 × 4언어 번역 키 0개 검사 + 관리자 en 복귀 검사),
  `web/app/middleware/app.global.ts` (`/ja|th|zh-cn/admin` →
  `/en/admin` 302), `web/package.json` (`harness:console-lang`).
- 검증:
  - setup: 5 roles 기록 확인.
  - console-lang: 12/12 PASS (콘솔 3종 × 4언어 raw 키 0개,
    관리자 ja/th/zh-cn → /en/admin 리다이렉트 확인).
  - 전수 하네스: preview 대상 실행 중 (토큰 갱신 효과 +
    business 콘솔 캡처 확인 예정).
- 발견·수정한 결함: 관리자 ja 접근 시 en 폴백이 동작 안 했음
  (`/ja/admin` 라우트 자체가 없어 404). 미들웨어 리다이렉트로
  수정 (R-2).
- 남은 문제: 전수 결과 수집 후 확정.

## WI-N7 디자인 쇼케이스 v3 갱신 — PASS (화면 확인 대기)

- 변경: `web/app/pages/dev/ui.vue` 전면 갱신 (v3 표기, 타이포,
  버튼 7종, 카드·배지, 입력 4종, prompt-box + hero-glow,
  KpiCard 3종, ConsoleShell 미리보기).
- 검증: typecheck 0. 화면 캡처 확인은 하네스 완료 후 빌드에서
  수행.
- 남은 문제: 없음.

## WI-N8 i18n 미사용 키 정리 — PASS (빌드 검증 대기)

- 변경: 6개 언어 × 63개 키 삭제 (동적 26종·간접 참조·
  substring 오탐까지 3단계 검증 후 삭제).
- 검증: check-i18n PASS (경고 337→276), typecheck 0,
  vitest 127/127. 삭제 키 사용처 0건 (FULL + 동적 패턴 +
  간접 리터럴 전수 대조).
- 남은 문제: 빌드 검증을 하네스 완료 후 수행.

## WI-N13 워터마크·유사도 + 데모 — 부분 PASS (데모 생성 BLOCKED)

- 변경: `generator/server.py` (AudioSeal 삽입·검출·유사도·
  verdict), `backend/internal/service/generation.go` (금칙어
  17→58개·CJK substring), `backend/cmd/worker/main.go`
  (`recheckLicense` AVIBE 분기 — E2E에서 발견한 결함 수정),
  `docs/generation/watermark-similarity.md` 신규, 가드 테스트 확장.
- 검증:
  - AudioSeal: clean 0.0/marked 1.0, MP3 1.0, 잘라내기 0.9933,
    -20dB 1.0. 유사도: 무참조 vacuous, 자기참조 실패 (1.0),
    타 클립 통과 (0.5).
  - 로컬 E2E (stub 생성 서버): generations → SUCCEEDED
    (safety 3종 + watermarkId) → self pre-check 403 →
    finalize 201 → PENDING → Gate 4 → READY +
    `origin: AVIBE_MODEL`. PASS.
  - `go test` service/generator/worker 전부 ok.
- 남은 문제 (BLOCKED/P6): 데모 6곡 생성은 GPU Pod 필요.
  Pod 켜기·끄기는 PO 또는 Claude. 운영 등록은 G-MODEL·G-2 후.

- 변경: `generator/server.py` (`insert_watermark` AudioSeal 구현·
  `detect_watermark`·`check_similarity`·v2 safety verdict 동적 구성),
  `backend/internal/service/generation.go` (`BannedArtistNames`
  17→58개, CJK substring 매칭),
  `docs/generation/watermark-similarity.md` 신규 (선정표·기준·
  정책), 가드 테스트 확장.
- 검증:
  - AudioSeal 선정: MIT(코드·가중치) 확인. 로컬 실측 — 왕복
    clean 0.0/marked 1.0, MP3 192k 1.0, 4초 잘라내기 0.9933,
    -20dB 1.0. 메시지 비트 15/16 (watermarkId는 서버 핸들로
    별도 추적).
  - server 왕복: 삽입·검출, 참조 없음 vacuous
    (`compared: 0`, similarity 미주장), 자기참조 실패 (1.0),
    타 클립 통과 (0.5).
  - `go test` service/generator/worker 전부 ok (가드 다국어
    테스트 확장 포함).
- 남은 문제: 로컬 E2E (stub 생성 서버 → 워커 → AVIBE READY)는
  워커 재시작이 필요해 하네스 완료 후 수행. 데모 6곡 생성은 GPU
  Pod 필요라 BLOCKED (P6). 운영 등록은 G-MODEL·G-2 승인 후.
