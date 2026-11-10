# WI 수용 증거 (2026-11-10, 엔진 세션)

작업지시 §5(A-01~A-09)·§6 형식 + 수용 체크리스트 E-1~E-6 대조.
상태: PASS / FAIL / PARTIAL / BLOCKED / NOT TESTED.
"라이브"는 로컬 스택(`http://localhost:18081`, docker-compose.local) 기준 —
프로덕션 배포·도메인·PG키는 사람(PO) 소관이라 별도 표기한다.

## A-01 라이브 목록·요금제 (WI-01) — PARTIAL

- 로컬 PASS: `GET /healthz` → `{"status":"ok"}`(2026-11-10 실측),
  `GET /readyz` → `{"status":"ok","checks":{"db":"up","redis":"up",...}}`.
  E2E T01/T08/T12가 목록·요금제·스트림을 라이브로 검증 (결과 하단 참조).
- 프로덕션 BLOCKED: `avibemusic.com` 배포·DNS·R2·관리형 DB/Redis 미착수
  (사람 소관, LAUNCH_CHECKLIST §2~§4).

## A-02 테스트 결제 전 과정 (WI-02) — BLOCKED

- 사유: Toss 테스트키 없음(`.env`에 TOSS 미설정). E2E T12는 요금제 목록 200 +
  키 미설정 시 주문 503 fail-closed까지만 검증.
- 해제 조건: `TOSS_SECRET_KEY`(테스트키) 발급 후 실결제 1건(주문→승인→구독
  ACTIVE→환불) 수행 (LAUNCH_CHECKLIST 결제행).

## A-03 우회 등록 전부 거부 (WI-04) — PASS (E2E 290/0)

- E2E T03(Suno Free 403) · T04(기간 밖 403) · T05(Sync-Only 403) ·
  T05b(Udio 컷오프 403) · T06(중복 409) · T07(FFprobe 차단 태그).
- 단위: `TestValidateCommercialEligibility` 11 서브테스트 + `TestValidateSourceModel`
  (`go test ./internal/pkg/licensegate/` PASS).

## A-04 정책 스냅샷 잔존 (WI-04) — PASS

- `track_rights_declarations.cert_snapshot_json/_at` (000008). Finalize에서
  `BuildCertSnapshot`(provider/tier/sub/model/trust/gate1·2 + `PolicyVersion`
  + decidedAt) 저장, 워커 재검증 통과 시 갱신(`SetCertSnapshot`).
- 파일: `backend/internal/pkg/licensegate/validator.go`
  (`PolicyVersion`, `CertSnapshot`, `Build/ParseCertSnapshot`),
  `backend/internal/service/precheck_finalize.go`,
  `backend/cmd/worker/main.go` (`recheckLicense`),
  `backend/internal/repository/catalog.go`, `000008_cert_snapshot.*.sql`.
- 증거: `TestBuildCertSnapshotRoundTrip` + `TestBuildCertSnapshotRequiresProvider`
  PASS. 라이브 행 확인: E2E T08 트랙의 스냅샷 컬럼 non-null (E2E 후 DB 조회).

## A-05 "100% 보장" 문구 0건 (WI-04) — PASS

- `100% 보장|소유권 보장` 전수 grep 0건 (2026-11-10).
- 인증서 문구 정본 통일: `web/src/views/user/TrackDetail.vue`
  "상업 라이선스 인증서" → "상업적 이용권 보유 인증".
- 면책 유지: "파트너 신고와 자동 검사에 근거하며 권리 관계를 보증하지 않습니다."

## A-06 무료 한도 서버 강제 (WI-05·06) — PARTIAL

- PASS: 생성 일일 쿼터(`GENERATION_DAILY_CAP`, 기본 5, 파트너별 자정 리셋) +
  금지 프롬프트 필터(내장 목소리모방/사칭 신호 + `GENERATION_BANNED_KEYWORDS`
  추가, 24시간 창 3회 누적 시 24시간 생성 제한) — `GenerationService.Submit`,
  `TestGenerationDailyCap`·`TestGenerationBannedPrompt` PASS.
- PASS(구조적): DSP-kit는 관리자 전용이라 무료 계정이 유통 신청 API에
  도달 불가(403).
- NOT TESTED: 다운로드 API(#22) 미구현 — 다운로드 한도 자체가 없음.
  잔존 결함으로 남김 (P2).

## A-07 GPU 유휴 과금 없음 (WI-05) — PASS

- GPU 인프라 0: 생성은 외부 백엔드 위임 구조(`GENERATION_BACKEND_URL`,
  미설정 시 작업 실패 + 과금 주체 없음). `generator/server.py`는 사용자
  자택 PC에서 수동 기동 전제로 상시 과금 없음.

## A-08 정산 원천 분리 (WI-07) — STALE (정본 변경 2026-11-10 저녁)

- ~~PO 결정(2026-11-10): 재생당 10원 고정 유지, 50% 풀 전환 취소.~~
  **PO 최종 결정(2026-11-10 저녁): 유통 순매출 70:30이 정본.**
  재생당 10원 고정도 50% 풀도 아니다. 아래는 구(舊) 정본 기준 기록이다.
- 구 기록: 현행 `SettlementService.Calculate`는 일반 스트리밍 + 인증 매장
  재생을 합산해 10원/건으로 정산한다.
- 후속: 엔진 계산 교체는 별도 WI로 집행 (전수검수 패키지 EV-P06·P07 추적).
- 잔존 결함: 부정 재생 정산 보류(#12) 미구현, 유효 판정 규칙 미확정(#P07).

## A-09 회귀 없음 (전 항목) — PASS (E2E 290/0)

- `go build/vet/test ./...` 전체 PASS, `vue-tsc` + vitest 37/37 + `vite build`
  PASS, `flutter analyze` clean + `flutter test` 8/8 (2026-11-10).
- 라이브 회귀: E2E T01~T12 결과 하단 참조.

## E-1 서버·결제 — PARTIAL (로컬 PASS / 프로덕션 BLOCKED)

- API·워커·DB·Redis·LocalStack 전부 healthy (start-local 실측).
- `canonical`은 `avibemusic.com` 유지 여부 미확인 — 확인 필요 (P2).
- Toss 키·실결제 BLOCKED (A-02와 동일).

## E-2 인증 게이트 — PASS (라이브 행 확인은 E2E 후)

- 4단계 게이트 + 스냅샷 + 허용모델 화이트리스트(`BlockedSourceModels`:
  musicgen/audiogen/musiclm/stable-audio-open) + 문구 통일.
- 증거: licensegate 단위 테스트 + E2E T03~T08.

## E-3 생성 엔진 — PARTIAL

- 허용 모델 연결: 구조만(원격 백엔드 계약 + `licenseRef` 기록). ACE-Step 실
  구동은 사용자 자택 PC 예정 (G-1 실측: GTX 1630 4GB 불가).
- GPU scale-to-zero: 해당 없음(외부 위임).
- 한도 코드 강제: 일일 쿼터 + 금지 프롬프트 PASS (A-06 참조). 무료/유료
  차등(3~5/300)은 요금제 체계 개편(#9)과 묶여 NOT TESTED.

## E-4 정책·약관 — PARTIAL

- 플랜 권리표 일치: 요금제 체계 개편(#9, 리스너/매장/영상 상품) NOT TESTED —
  범위 밖(대규모 상품 개편, PO 후속 지시 필요).
- 금지 프롬프트 + 제재(경고→24시간 제한): PASS. 계정 해지는 수동
  관리자 정지로 대체 (문서화됨).
- AI 표기 자동화: HLS 세그먼트 comment 워터마크 + 트랙 고지문 + 웹/모바일/
  매장 플레이어 표시 PASS. 다운로드 파일 태깅은 다운로드 API 부재로 NOT TESTED.

## E-5 정산 — STALE (정본 70:30, A-08 참조 · 엔진 WI 후속)

## E-6 유통 실무 — PASS (오프라인 캐시 제한 명시)

- ISRC: `tracks.isrc UNIQUE` + 관리자 지정 API(`PUT /admin/tracks/:id/isrc`,
  감사 `TRACK_ISRC_ASSIGN`) + DSP-kit 포함. 발급 대행 절차: 기관 블록을
  운영자가 확보해 직접 지정하는 방식 (RUNBOOK에 문서화 필요 — P2).
- DSP 선별 송출: `GET /admin/distribution/dsp-kit`(READY+ISRC만, 100곡 상한,
  대량 일괄 금지 — 요청당 명시 선택만) + CSV/JSON. 관리자 화면 포함.
- 매장 플레이어: 사업자 승인 + 페어링 + 편성/스케줄 + 원격 볼륨 + 인증 집계
  + 웹 플레이어(`/store/player`) 구현·테스트 + 모바일 스토어모드(`/store`,
  페어링·큐·하트비트, `flutter test` 12/12). 오프라인 세그먼트 캐시는
  5분 서명 URL 구조와 양립 불가 → 큐 스냅샷 표시 + 온라인 복귀 시 재개로
  문서화한 제한 (API.md 명시).

## E2E 라이브 결과 (T01~T12)

> E2E PASSED — PASS=290 FAIL=0 SKIP=0 (2026-11-10, `e2e-20261110-final.log`,
> `avibe-e2e:local` 러너).
> 경위: 280/8 → ValidateAccess 도입 후 267/21(회귀: stale 토큰 재사용
> 테스트가 401 cascade) → 스크립트 수정 후 285/3 → 288/0 → 신규
> `GET /admin/users/:id/blocks` assert 2건 추가 후 290/0.
> 수정: password-reset 후 USER_TOKEN 갱신, blocked 픽스처 comment에 RUN_ID,
> legal_terms 버전에 STAMP, 신규 blocks 엔드포인트 assert(200/403).

## 변경 파일 매핑 (§3-5 대응 요약)

- WI-02 잔여: `licensegate/validator.go`, `domain/errors.go`,
  `service/precheck_finalize.go`, `cmd/worker/main.go`,
  `repository/catalog.go`, `domain/entities.go`, `migrations/000008*`,
  `web/src/api/client.ts`, `mobile/lib/core/error.dart`.
- WI-03: `repository/generation.go`·`queue.go`, `pkg/generator/*`,
  `service/generation.go`, `handler/partner.go`·`router.go`,
  `config/config.go`, `cmd/worker/main.go`, `migrations/000009*`,
  `generator/server.py`·`requirements.txt`·`README.md`.
- WI-05: `domain/isrc.go`·`bizno.go`, `pkg/audio/analysis.go`,
  `service/distribution.go`·`store.go`, `repository/store.go`,
  `handler/device.go`·`admin.go`·`partner.go`, `migrations/000010*`·`000011*`,
  `web/src/{api,stores,views/{partner,admin,store},layouts,router}`,
  `mobile/lib/{core/models.dart,features/player}`.
- WI-08: `service/generation.go`(쿼터·필터), `migrations/000012*`,
  `views/user/TrackDetail.vue`(문구), 본 문서.

## 잔존 결함 (우선순위 순)

1. P2-1 다운로드 API(#22) 없음 — 다운로드 한도·파일 AI 태깅 불가.
2. P2-2 부정 재생 정산 보류(#12) 없음.
3. P2-3 요금제 체계 개편(#9: 매장/영상 상품, 플랜-권한 연결) 없음.
4. P2-4 ISRC 발급 대행 절차 RUNBOOK 문서화.
5. P2-5 canonical avibemusic.com 통일 여부 미확인.
6. P2-6 곡별 인증서 발급(#24: 고유번호·PDF·진위 URL) 없음.
7. P2-7 유사도 게이트(#15): 외부 탐지 API 계약 필요 (PO 판단).
8. P2-8 생성 한도 무료/유료 차등 (요금제 개편과 묶임).
