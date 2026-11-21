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
- T16/T20/T21/T22-AVIBE 401 연쇄, 17건: E2E 초반 발급 토큰
  (PA_TOKEN/USER_TOKEN, TTL 15분) 만료 후보. E2E 전체 소요 시간
  미측정이라 후보로 유지 (하네스 완료 후 time 측정으로 확정).
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
- **긴급 회귀 발견 (N0 범위 밖, PO 보고)**: 로케일 하위 경로
  (`/en/charts`, `/ja/charts` 등)가 전부 404. `/en` 홈은 200,
  `/charts`는 200. 원인: 공개 페이지에 `defineI18nRoute(false)`가
  남발돼 로케일 라우트가 생성 안 됨 (11-14~) + 11-17 이후
  [...slug]가 hard 404를 반환 (`2957a4b`; 11-17에는 soft-404 200이라
  하네스 PASS였음). 운영(`https://avibemusic.com/en/charts` → 404,
  읽기 전용 확인)도 동일. 수정은 WI 범위 밖이라 하지 않고
  완료보고에 긴급 결함으로 올림.
- 401 다수 (`/auth/me`, `/auth/refresh`): 하네스 토큰 TTL 15분
  만료 (WI-N6 배경과 일치). WI-N6에서 수정 후 재측정.
- Lighthouse 1차 (로컬 preview + `NUXT_COMPRESS_PUBLIC_ASSETS=1`
  빌드, 3회 중앙값, LHCI 0.15.1):
  - `/`: perf 0.63, script 324,007B, seo 1.00
  - `/en/tracks/e2e-sitemap-song-1c65595d`: perf 0.61, script
    407,538B (로케일 회귀로 404 페이지 측정 — 수정 후 재측정)
  - `/th/playlists`: perf 0.66, script 419,035B (404 페이지 측정)
  - `/ja/business`: perf 0.56, script 480,919B (404 페이지 측정)
  - CI 수치(perf 0.68~0.89, 193~205KB)와 차이: 로컬 preview는
    동적 응답 압축 미적용 (WI-N1 항목 1) + 404 페이지 측정분.

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

## WI-N2 모바일 부분 (엔진·웹은 하네스 완료 후)

- 변경 (모바일): `core/models.dart` (`Track.certNumber`·
  `TrackCertificate`·`CertificateView`), `services/api_service.dart`
  (`verifyCertificate()`), `features/certificate/certificate_screen.dart`
  신규, `main.dart` (`/certificate/:number` 라우트),
  `features/player/player_screen.dart` (AI·라이선스 탭 배지 + 탭
  이동), ARB 5개 언어 10키씩, `test/models_test.dart` 3종.
- 검증: `flutter analyze` 0, `flutter test`에 포함 (28/28).
  엔진 `certNumber` 노출 후 실연동 확인 예정.

## WI-N13 워터마크·유사도 (E2E·데모 생성 대기)

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
