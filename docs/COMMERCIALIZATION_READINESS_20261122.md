# AVIBE AI 음원 생성 & 스트리밍 플랫폼 상용화 준비 전수 검증 보고서

- **검증 일시**: 2026-11-22 (KST)
- **검증 대상 커밋**: `main` (`a7c8780` — 2026-11-21 롤아웃 반영본)
- **검증 범위**: 백엔드(`backend/`), AI 생성 서버(`generator/`), 웹 프론트엔드(`web/`), 모바일 앱(`mobile/`), 엣지 워커(`worker.js`), 배포·백업·운영 스크립트(`deploy/`, `scripts/`), DB 마이그레이션(`000001`~`000027`), 라이브 프로덕션(`https://avibemusic.com`) 실측

---

## 1. 최종 종합 판정 (Executive Summary)

| 구분 | 판정 | 핵심 요약 |
|---|---|---|
| **소프트웨어 코드 & 기술 엔진 준비도** | **✅ READY (구축·검증 완료)** | AI 음원 생성(ACE-Step 1.5 + AudioSeal 워터마크 + 유사도 프리필터 + 5개 국어 프롬프트 가드), 4단계 상업 라이선스 인증 게이트, HLS 서명 스트리밍(Cloudflare Worker + Private R2), 70:30 순매출 정산 엔진, B2B 매장 플레이어, 유료 다운로드 태깅, OSP 권리보호센터, 5개 국어 Nuxt 4 SSR 웹 및 Flutter 모바일 앱 코드 전수 구현 및 테스트 통과 |
| **운영 인프라 & 라이브 서비스 가동** | **✅ LIVE (정상 가동 중)** | Vultr 서울 VPS + Cloudflare Worker + Private R2 정식 연결 완료(과거 530 터널 단절 해소). 현재 `avibemusic.com`에서 웹 SSR 및 API 200 정상 응답, 자체 생성(`AVIBE_MODEL`) 데모 5곡 + 외부 AI(`SUNO`) 3곡 등 총 8곡 `READY` 스트리밍 가동 중 |
| **상용 유료 과금 & 공개 생성 오픈** | **⚠️ CONDITIONAL READY (조건부 완료 — PO/외부 게이트 6건 대기)** | 기술 코드는 fail-closed 안전장치까지 완비되었으나, 실제 고객 유료 결제와 상시 AI 생성을 열기 위해서는 **① Toss 실결제 키 주입 ② GPU 서버(Runpod Serverless/상시 Pod) 상시 연결 ③ 요금제 DB 시드 및 크레딧 차감 연결 ④ OAuth 5종·SMTP 실키 주입 ⑤ 사업자 정보·수령인 기입 및 약관 법률 검토(G-3) ⑥ 생성 워터마크 16kHz 다운샘플링 음질 개선**이 선행되어야 함 |

---

## 2. 자동화 빌드·테스트 게이트 & 라이브 실측 결과 (2026-11-22 직접 검증)

### 2.1 코드 정적 분석 및 테스트 스위트
| 영역 | 검증 명령 | 결과 | 세부 내역 |
|---|---|---|---|
| **Go 백엔드 빌드** | `cd backend && go build ./... && go vet ./...` | **✅ PASS (exit 0)** | `cmd/api`, `cmd/worker`, `cmd/migrate`, `cmd/adminctl` 및 내부 패키지 전수 컴파일·정적 검사 오류 0건 |
| **Go 백엔드 단위 테스트** | `cd backend && go test ./...` | **✅ PASS (14/14 패키지)** | `worker`, `config`, `domain`, `handler`, `audio`, `crypto`, `generator`, `licensegate`, `mail`, `storage`, `toss`, `repository`, `search`, `service` 전 패키지 통과 |
| **웹 TypeScript 검사** | `cd web && npm run typecheck` | **✅ PASS (exit 0)** | Nuxt 4 (`vue-tsc`) 타입 오류 0건 |
| **웹 단위 테스트** | `cd web && npx vitest run` | **✅ PASS (127/127)** | 19개 테스트 파일, 127개 케이스 전수 통과 (스토어, API 클라이언트, 플레이어 그랜트 경합, 다국어 로케일, SEO, 환불 정책 등) |
| **다국어(i18n) 무결성** | `cd web && node scripts/check-i18n.mjs` | **✅ PASS (exit 0)** | `ko/en/ja/zh-CN/th/en-XA` 6개 로케일 키 동치, 템플릿 내 한글 하드코딩 0건, ICU 구문 오류 0건 |
| **모바일 앱 (Flutter)** | `cd mobile && flutter analyze && flutter test` | **✅ PASS (28/28)** | Analyzer 이슈 0건, 위젯·모델·소셜 이메일 보완·인증서·스토어 모드 테스트 28건 통과, ARB 5개 언어×210키 동치 |

### 2.2 라이브 프로덕션 (`https://avibemusic.com`) 읽기 전용 프로브
| 엔드포인트 / 화면 | 응답 코드 | 실측 상태 |
|---|---|---|
| `GET https://avibemusic.com/` (홈 SSR) | **200 OK** | Nuxt 4 SSR 정상 렌더링, CSP nonce·HSTS·보안 헤더 정상 |
| `GET https://avibemusic.com/explore` · `/pricing` · `/create` | **200 OK** | 탐색·요금제·생성 웨이트리스트 화면 정상 렌더링 |
| `GET https://avibemusic.com/api/v1/tracks?limit=10` | **200 OK** | 총 **8곡 `READY`** 반환 (`AVIBE_MODEL` 자체 생성 5곡: *Unsent Letter*, *Night Bus*, *Paper Moon*, *Summer Afterimage*, *Glass Station* / `EXTERNAL_AI` Suno 3곡) |
| `GET https://avibemusic.com/api/v1/billing/plans` | **200 OK** | `LISTENER_PLUS_MONTHLY` (5,900원), `STUDIO_PRO_MONTHLY` (12,900원) 2종 활성 반환 |
| `GET https://avibemusic.com/api/v1/legal/terms` | **200 OK** | `TOS`, `PRIVACY`, `PARTNER_CONTRACT`, `COPYRIGHT_POLICY` 4종(v1.0/v1.1) 정상 반환 |
| `GET https://avibemusic.com/api/v1/auth/oauth/providers` | **200 OK** | `GOOGLE`, `APPLE`, `KAKAO`, `NAVER`, `LINE` 5종 모두 `enabled: false` (키 미주입으로 UI 자동 숨김 동작) |

---

## 3. 10대 핵심 도메인별 코드 구현 및 상용화 준비 상태 전수 점검

### 3.1 AI 음원 생성 엔진 & GPU 서빙 (`generator/`, `backend/internal/pkg/generator/`, `backend/internal/service/generation.go`)
- **구현 완료 사항**:
  - **모델 및 라이선스**: 상업적 이용이 가능한 **ACE-Step 1.5** (`acestep-v15-turbo`, 코드·가중치 모두 MIT 라이선스) 채택 및 [`generator/server.py`](../generator/server.py) `infer()` 실제 추론 연동 완료.
  - **성능 실측**: Runpod GPU(RTX PRO 4000 24GB / RTX 2000 Ada 16GB)에서 웜 상태 60초 곡 1.96초, 180초 곡 4.77초 생성 및 시드(`seed`) 기반 SHA-256 재현성 검증 완료 ([`docs/generation/poc-runpod-20261121.md`](generation/poc-runpod-20261121.md)).
  - **계약 v2 및 보안**: `contractVersion: 2` 요청 시 워터마크 삽입(`watermark: true`) 강제, `modelVersion`·`seed`·`weightsHash`·`watermarkId`·`safety` 반환. 대용량 WAV 처리를 위한 사전서명 URL 업로드(`delivery.uploadUrl`) 지원.
  - **워커-생성서버 인증**: [`backend/internal/pkg/generator/generator.go`](../backend/internal/pkg/generator/generator.go)에서 `GENERATION_BACKEND_TOKEN`을 `Authorization: Bearer` 헤더로 전송하고, `401 Unauthorized` 시 토큰 노출 없이 재시도하지 않고 즉시 실패 처리. Runpod Cloudflare 프록시(1010 차단) 우회를 위한 전용 `User-Agent` 적용 완료.
  - **컨테이너화 & 온디맨드 스크립트**: 가중치 베이킹을 지원하는 멀티 타겟 [`generator/Dockerfile`](../generator/Dockerfile)(`server`/`gpu` 타겟) 및 재설치 없이 네트워크 볼륨(`y7tzkyd0g5`)으로 Pod를 기동·종료하는 [`scripts/runpod-gen.sh`](../scripts/runpod-gen.sh) 구축 완료.
- **상용화 전 보완·결정 필요 사항**:
  1. **🔴 [음질 핵심 이슈] 워터마크 삽입 시 16kHz 모노 다운샘플링 문제 ([`generator/server.py#L321-L344`](../generator/server.py#L321-L344))**:
     - 현재 `insert_watermark()`가 AudioSeal(16kHz 기반 모델)을 통과시키기 위해 원본 WAV를 `AUDIOSEAL_SR = 16000` 모노로 리샘플링한 뒤 **그 16kHz 모노 배열을 그대로 최종 WAV로 인코딩(`_encode_wav_mono16(out, AUDIOSEAL_SR)`)** 하여 반환하고 있습니다.
     - 16kHz 샘플레이트는 가청 주파수 상한(Nyquist 주파수)이 **8kHz**로 잘려 고음역이 먹먹해지고 스테레오 공간감이 사라집니다.
     - **해결 방법**: AudioSeal이 생성한 워터마크 잔차 신호(`wm_16k = gen.get_watermark(...)`)만 원본 샘플레이트(44.1kHz/48kHz)로 업샘플링하여 **원본 고음질 스테레오 파형에 더해주는 방식**으로 수정해야 상용 스트리밍 음질 기준을 충족합니다.
  2. **상시/서버리스 GPU 엔드포인트 고정 (`P6` / `G-MODEL`)**:
     - 현재는 비용 절약을 위해 데모 5곡 생성 후 Runpod Pod를 종료(`down`)해 둔 상태입니다. 상시 생성 서비스 오픈 시 Runpod Serverless(또는 상시 Pod)를 배포하고 운영 `.env`에 `GENERATION_BACKEND_URL`과 `GENERATION_BACKEND_TOKEN`을 고정해야 합니다.

### 3.2 생성물 안전성: 워터마크·유사도 검사·다국어 프롬프트 가드 (`backend/internal/service/generation.go`, `generator/server.py`)
- **구현 완료 사항**:
  - **비가청 워터마크**: Meta **AudioSeal**(코드·가중치 MIT) 삽입(`insert_watermark`) 및 검출(`detect_watermark`) 구현. MP3 192k 재인코딩(검출 확률 1.0), 50% 구간 절단(0.9933), -20dB 감쇠(1.0) 내성 실측 완료 ([`docs/generation/watermark-similarity.md`](generation/watermark-similarity.md)).
  - **다국어 프롬프트 사전 차단**: [`backend/internal/service/generation.go`](../backend/internal/service/generation.go)에서 전각 문자·공백 정규화(`normalizeGuardText`) 후 5개 국어(한/영/일/중/태) 보이스 클로닝 문구(`BannedVoicePhrases`), 스타일 모방 패턴(`BannedStylePatterns`, "~풍으로", "in the style of" 등), 실존 아티스트명 58개(`BannedArtistNames`, CJK 부분일치 + 라틴 단어경계 매칭) 및 운영자 추가 키워드(`GENERATION_BANNED_KEYWORDS`) 차단.
  - **3-Strike 생성 제한**: 24시간 내 금지 프롬프트 3회 적발 시 해당 파트너의 생성을 24시간 동안 자동 정지(`generation_violations` 테이블).
  - **Born Verified 게이트 연동**: `AvibeProvenanceOK()`가 `SUCCEEDED` 상태, `watermarkId` 존재, `safety.passed == true`, 그리고 `checks` 배열에 `"watermark"`와 `"similarity"`가 모두 포함된 경우에만 `provider=AVIBE` 트랙 등록을 허용(하나라도 누락 시 fail-closed).
- **상용화 전 보완·결정 필요 사항**:
  - **참조 핑거프린트 DB 적재 및 스펙트럼 유사도 고도화**: 현재 `server.py:check_similarity()`는 `GENERATOR_REFERENCE_FPS` JSON 파일과 256-블록 PCM 부호 해시를 대조합니다. 운영 환경에서 실제 등록 카탈로그 및 학습셋의 지문 JSON을 자동 동기화하고, 피치·템포 변조까지 탐지하는 스펙트럼/크로마 지문(`backend/internal/pkg/forensics/` 모듈 또는 `fingerprint_spectral`)으로 고도화하는 것이 권장됩니다.

### 3.3 4단계 상업 라이선스 인증 게이트 & 인증서 (`backend/internal/pkg/licensegate/validator.go`, `backend/cmd/worker/main.go`)
- **구현 완료 사항**:
  - **Gate 1 (프로바이더·플랜·모델 정책)**: 허용 프로바이더(`SUNO`, `UDIO`, `AIVA`, `STABLE_AUDIO`, `SOUNDRAW`, `MUBERT`, `ELEVENLABS`, `CUSTOM_OPEN_MODEL`, `AVIBE`) 검증, 비상업 플랜(`free`, `basic`, `trial`, `cc-by-nc`) 및 Sync-Only 플랜(`AIVA Standard`, `Soundraw/Mubert Creator`) 즉시 거부(`403`). 비상업 오픈 모델(`musicgen`, `audiogen`, `musiclm`, `stable-audio-open`) 하드코딩 차단.
  - **Gate 2 (구독 유효기간 윈도우 & 컷오프)**: 파트너의 검증된 유료 구독(`VERIFIED`) 기간(`start <= generated_at <= expires_at`) 내 생성 여부 대조(소급 생성 차단) 및 Udio 다운로드 중단 컷오프(`max_generated_at: 2025-10-30`) 이후 생성분 차단.
  - **Gate 3 (SHA-256 무결성 & FFprobe 포렌식)**: 스테이징 업로드 시 파일의 실제 SHA-256과 사전 신고된 SHA-256 일치 검증, 중복 음원(`409`) 차단, FFprobe 메타데이터 내 비상업 키워드(`free`, `non-commercial`, `trial`, `demo`, `watermarked` 등) 및 최소 비트레이트 검사 후 `ffprobe_report_json` 기록.
  - **Gate 4 (관리자 최종 검수 & 정책 스냅샷 & 인증서)**: 결정 시점의 정책 버전(`PolicyVersion: 2026-11-11`)과 게이트 통과 근거를 불변 JSON(`cert_snapshot_json`)으로 박제. 신탁단체(`KOMCA` 등 `TRUSTED`) 음원 등록 거부. 트랙별 고유 인증서 발급(`POST /admin/tracks/:id/certificate`) 및 공개 진위 확인 페이지(`/certificate/:number`), 목록·상세·플레이어 `Track.certNumber` 배지 링크(WI-N2) 연동 완료.
- **상용화 전 보완·결정 필요 사항**:
  - 현재 라이브 운영 DB에 등록된 `READY` 트랙 8곡의 `certNumber`가 `null` 상태입니다. 관리자 콘솔 또는 `POST /api/v1/admin/tracks/:id/certificate`를 호출해 8곡의 인증서를 발급하면 홈·탐색·상세 화면에서 인증 배지 클릭 시 즉시 인증서가 열립니다.

### 3.4 오디오 트랜스코딩·HLS 서명 스트리밍·다운로드 태깅 (`backend/internal/pkg/audio/`, `backend/internal/handler/track.go`, `worker.js`)
- **구현 완료 사항**:
  - **HLS 트랜스코딩 파이프라인**: FFmpeg `loudnorm`(EBU R128 음량 정규화) + 스트림 헤더 AI 고지 워터마크(`AI-Generated Audio (KR AI Basic Act Art.31 notice | commercial terms checked)`) + `128kbps (44.1kHz)` SQ 및 `256kbps (48kHz)` HQ 듀얼 비트레이트 6초 VOD 세그먼테이션 + 100-포인트 정규화 파형(`peaks.json`) 및 오디오 지문(`fp_*`) 생성.
  - **엣지 HMAC 서명 검증 (`worker.js`)**: Private R2 버킷(`HLS_BUCKET`)을 외부 네트워크 노출 없이 Cloudflare Worker 바인딩으로만 읽으며, `trackId|jti|exp|q` HMAC-SHA256 서명(5분 TTL)과 품질 티어(`SQ`는 `stream_0.m3u8`/`seg_0_*.ts`만 허용, `HQ`는 `master.m3u8` 전체 허용)를 검증한 뒤 `.m3u8` 내부 세그먼트 URI에 서명 쿼리를 동적 재작성(`rewritePlaylistUris`). Range 요청(`206 Partial Content`) 및 ETag(`304 Not Modified`) 캐싱 지원.
  - **유료 마스터 다운로드 & 인파일 AI 태깅 (`GET /api/v1/tracks/:id/download`)**: 유료 패스 보유자(트랙 소유자·관리자 예외) 전용, 24시간 50곡 쿼터 제한, 64KB 헤드 버퍼링 후 스트리밍 전송(메모리 과점유 방지). WAV 파일은 `LIST-INFO` 청크에, MP3 파일은 `ID3v2` 태그에 AI 생성 출처 및 검증 URL을 실시간 삽입하고 `X-AVIBE-AI-Disclosure` 헤더를 부착.

### 3.5 스트리밍 하트비트·사기 방어·70:30 정산 엔진 (`backend/internal/service/playlist_playback_settlement.go`)
- **구현 완료 사항**:
  - **하트비트 위변조 방어 (`PlaybackService.Heartbeat`)**: 서버가 실제 발급한 스트림 그랜트(`JTI`)가 없으면 거부(`403`). 이전 하트비트 시각과의 실제 경과시간(`elapsed + 15s tolerance`)으로 `playMs`를 클램핑(`capPlayMs`)하여 빠른 요청 연타 조작을 차단. 누적 30초(`ValidStreamMs = 30000`) 도달 시 정확히 1회 유효 재생으로 집계하며, 시간당 유저 전체 120회 / 동일 트랙 40회 초과 시 `FraudSuspected: true`로 집계를 차단.
  - **순매출 70:30 정산 엔진 (`SettlementService.Calculate`)**:
    - 재생당 10원 고정 단가 정산을 전면 폐기하고 **유통 순매출 70:30(파트너 70%)** 정본 적용 완료.
    - **배분 로직**: 일반 청취자 무료 스트리밍은 정산금을 만들지 않음(봇 재생 사기 원천 차단). 매장 구독료 풀(`StorePoolKRW`)은 승인된 매장에서 30초 이상 재생된 적격 매장 재생 시간(`StorePlaySecs / TotalStorePlaySecs`) 비례로 배분하고, 직접 판매·라이선스 매출(`DirectKRW`)은 해당 파트너에 100% 귀속한 뒤 합산액의 70%를 파트너 Gross로 산정.
    - **원천징수·보류·몰수·암호화**: 사업소득 3.3%(`BUSINESS_33`) 및 기타소득 8.8%(`OTHER_88`) 원천징수 계산, 정산 계좌번호 `AES-256-GCM` 암호화 저장 및 끝 4자리 마스킹 조회, 부정 의심 정산 보류(`HELD` — 사유·행위자 기록 및 해제 `Release`), 권리 침해 확정 시 몰수(`FORFEITED_VIOLATION`), 승인(`APPROVED`) 후 지급 완료(`PAID`, 이중 지급 409 차단) 워크플로우 완비.

### 3.6 B2B 매장 BGM 플레이어 & 글로벌 DSP 유통 (`backend/internal/service/store.go`, `backend/internal/service/distribution.go`)
- **구현 완료 사항**:
  - **매장 및 디바이스 관리**: 국세청 사업자등록번호 10자리 가중치 체크섬 검증(`NormalizeBusinessNumber`), 관리자 매장 승인(`APPROVED`), 매장 라이선스 수량(1~100대) 한도 내 디바이스 생성, 8자리 페어링 코드(15분 유효) 발급 및 클레임 시 32바이트 디바이스 토큰 발급(DB에는 SHA-256 해시만 저장).
  - **요일별 스케줄러 & 자동 편성 큐 (`DeviceQueue`)**: 요일(0~6)별 영업 시작·종료 시각(야간 심야 영업 지원), 휴무일(`isClosed`), 지정 플레이리스트(없을 시 공식 큐레이션 플레이리스트 자동 폴백) 기반 실시간 재생 큐 제공 및 원격 볼륨(0~100) 제어.
  - **매장 하트비트 위변조 방어 (`StoreService.Heartbeat`)**: 트랙 길이 및 직전 하트비트 경과시간(`elapsed + 5s tolerance`)으로 재생 시간을 클램핑(`clampStorePlaySecs`)하고 30초 이상 재생 시 정산 풀 집계에 반영.
  - **클라이언트 플레이어**: 웹 전용 매장 플레이어([`web/app/pages/store/player.vue`](../web/app/pages/store/player.vue), 5개 국어·오프라인 큐 스냅샷 폴백·일반 플레이어 동시 재생 방지) 및 Flutter 모바일 스토어 모드(`/store`) 완비.
  - **ISRC 및 DSP 배포 키트**: 트랙별 12자리 표준 ISRC 부여(`PUT /admin/tracks/:id/isrc`, `UNIQUE` 제약 및 감사 로그 기록) 및 관리자 선별곡(최대 100곡 상한, 대량 스팸 업로드 방지) 전용 DSP 배포 패키지(`GET /admin/distribution/dsp-kit`, CSV/JSON 메타데이터 + 1시간 유효 마스터 다운로드 서명 URL) 구현 완료.

### 3.7 결제·구독·청약철회/일할환불 엔진 (`backend/internal/service/billing.go`, `backend/internal/pkg/toss/`)
- **구현 완료 사항**:
  - **Toss Payments 연동**: 주문 생성(`POST /billing/orders`, 24시간 이내 환불규정 동의 시각 및 약관 버전 `2026-11-12-v2` 기록 필수) → 결제 승인(`POST /billing/payments/confirm`, 금액·주문번호 대조 및 1개월 이용권 활성화) → 웹훅(`POST /billing/webhooks/toss`, 이벤트 ID 멱등성 보장 및 Toss API 서버 간 재조회 검증).
  - **한국 전자상거래법·음원 스트리밍 표준 환불 정책 (`ComputeRefund`, `Cancel`)**:
    - **일반 해지 (`END_OF_PERIOD`)**: 만료일까지 이용권 유지, 환불 없음.
    - **즉시 해지 (`IMMEDIATE`)**: 결제 후 7일 이내이고 이용 이력(스트리밍·다운로드·생성 등 `FirstUsageSince`)이 전혀 없으면 **100% 전액 환불(청약철회 `WITHDRAWAL`)**, 이용 이력이 있거나 7일 경과 시 위약금 없이 **사용 일수만큼만 일할 공제한 잔액 환불(중도해지 `PRORATED`)** 및 Toss 부분취소/전체취소 API 자동 호출(이중 환불 방지 락 `ClaimRefund` 포함).
- **상용화 전 보완·결정 필요 사항**:
  1. **운영 `TOSS_SECRET_KEY` 주입 (`P2`)**: 현재 키 미설정으로 주문 API가 `503 ERR_PAYMENT_UNAVAILABLE`(의도된 fail-closed)로 닫혀 있습니다.
  2. **요금제 테이블 시드 갱신 및 플랜별 권한·크레딧 연동 (`P2-3`, `P2-8`)**:
     - 현재 운영 DB의 `billing_plans`에는 구 시드인 `LISTENER_PLUS_MONTHLY`(5,900원)와 `STUDIO_PRO_MONTHLY`(12,900원) 2종만 들어 있고, 웹 요금제 화면([`web/app/pages/pricing.vue`](../web/app/pages/pricing.vue))의 비즈니스 탭은 가격 미표시·도입 문의 모달(`InquiryForm`)로 운영 중입니다.
     - 기획 방향결정서의 최종 가격표(생성 구독 9,900원 / 매장 BGM 9,900·14,900원 / 영상 라이선스 19,900·29,900원)로 과금을 시작하려면 `billing_plans` DB 행 갱신 및 플랜별 생성 쿼터(무료 일 3~5곡 vs 유료 월 300곡 / `credit_ledger` 차감) 연결이 필요합니다.

### 3.8 권리보호센터(OSP)·계정 보안·컴플라이언스 (`backend/internal/service/review_dispute.go`, `auth.go`, `export.go`)
- **구현 완료 사항**:
  - **저작권법 제102·103조 OSP 절차**: 권리 침해 신고 즉시 임시 블라인드(`TAKEN_DOWN` — 스트리밍·하트비트·플레이리스트 추가 전면 차단), 파트너 반론(재개 요구 `COUNTER`), 관리자 인용/기각 판정, 72시간 미처리 분쟁 관리자 자동 알림 스위퍼(`sla.go`), 침해 확정 3회 누적 시 파트너 계정 자동 정지(3-Strike).
  - **계정·인증 보안**: 비밀번호 bcrypt 해싱, Access(15분)/Refresh(14일) JWT 분리 및 `token_version` 기반 전체 기기 로그아웃(`POST /auth/logout-all`), 로그인 실패 잠금(15분 내 10회 실패 시 `LOGIN_LOCKED`), 관리자 콘솔 TOTP 2단계 인증 강제(`ADMIN_TOTP_REQUIRED`), 이메일 인증(`RequireEmailVerified`) 게이트, 만 14세 이상 확인(`RequireAgeConfirmed`) 및 필수 약관 동의(`RequireConsents`) 미들웨어 강제.
  - **개인정보보호법 준수**: 회원 탈퇴(`POST /auth/withdraw`) 시 이메일·이름·소셜 연결 등 PII 즉시 파기 및 정산·세무 기록 익명 보존, 본인 데이터 전체 JSON 번들 내보내기(`GET /auth/export`, 비밀 컬럼 제외).

### 3.9 웹 프론트엔드(Nuxt 4 SSR, v3 디자인, 5개 국어, SEO) & 모바일 앱(Flutter)
- **구현 완료 사항**:
  - **Nuxt 4 하이브리드 렌더링**: 공개 페이지(`/`, `/explore/**`, `/charts`, `/playlists/**`, `/tracks/**`, `/creators/**`, `/certificate/**`, `/business`, `/for-creators`, `/trust`, `/pricing`, `/guides/**`, `/create`)는 SSR+SWR 캐싱으로 검색엔진 노출을 극대화하고, 콘솔·계정·결제·플레이어 화면은 CSR(`ssr: false, noindex`)로 분리.
  - **다국어(5개 언어) & 콘솔 번들 분리**: `ko`, `en`, `ja`, `zh-cn`, `th` 5개 언어 지원. 콘솔 전용 번역(`admin.json`, `studio-console.json`, `business-console.json`)은 콘솔 진입 시에만 지연 로드하여 공개 페이지 초기 번들 크기를 절감.
  - **글로벌 SEO**: 슬러그 URL(`/tracks/:slug-:shortId`), 다국어 동적 사이트맵(`/sitemap_index.xml`), IndexNow 즉시 색인 제출, Schema.org 구조화 데이터, Satori 기반 동적 OG 이미지, 외부 사이트 삽입용 oEmbed 및 임베드 플레이어(`/embed/track/:shortId`, `/embed/playlist/:shortId`), Android `assetlinks.json` 및 iOS `apple-app-site-association`.
  - **Flutter 모바일 앱 (`mobile/`)**: 5개 언어 ARB(210키), 소셜 로그인 및 이메일 보완 화면(`/complete-email`), 백그라운드 오디오 재생(`just_audio`), AI 라이선스 인증 배지 및 인증서 진위 확인 화면(`/certificate/:number`), 매장 전용 플레이어 모드(`/store`), 릴리스 빌드 시 디버그 키 서명 및 루프백 URL 하드코딩 차단 가드(`scripts/check-mobile-release.sh`).
- **상용화 전 보완·결정 필요 사항**:
  1. **`web/app/pages/charts.vue` 다국어 라우트 404 버그 (1줄 수정 필요)**:
     - [`web/app/pages/charts.vue#L66`](../web/app/pages/charts.vue#L66)에 `defineI18nRoute(false);`가 남아 있어 기본 한국어 `/charts`는 정상(200)이지만 `/en/charts`, `/ja/charts`, `/th/charts`, `/zh-cn/charts` 접속 시 404가 발생합니다. `defineI18nRoute(false);` 1줄만 제거하면 해결됩니다.
  2. **공개 생성 UI 활성화 스위치**:
     - 현재 `/create`([`web/app/pages/create.vue`](../web/app/pages/create.vue))는 출시 알림 웨이트리스트 화면이며, 스튜디오 생성 v2 패널([`web/app/pages/studio/create.vue#L22`](../web/app/pages/studio/create.vue#L22))은 `NUXT_PUBLIC_GENERATE_UI=1` 플래그로 가려져 있습니다. 생성 서비스 정식 공개 시 플래그를 켜고 `/create`를 생성 진입점으로 전환해야 합니다.

### 3.10 운영 인프라·CI/CD·무중단 배포·백업 (`deploy/`, `.github/workflows/ci.yml`)
- **구현 완료 사항**:
  - **운영 토폴로지**: Vultr 서울 VPS(`158.247.219.159`, Docker Compose: `api`, `worker`, `web`, `postgres`, `redis`, `cloudflared`) + Cloudflare Worker(`avibemusic.com`) + Cloudflare R2(`STORAGE_DRIVER=r2`). 개발자 로컬 PC가 꺼져도 24시간 독립 가동됨을 검증 완료.
  - **무중단 배포 & 롤백**: 풀스택 배포([`deploy/rollout-redesign.sh`](../deploy/rollout-redesign.sh)) 및 웹 단독 무중단 배포·롤백([`deploy/rollout-web.sh`](../deploy/rollout-web.sh) — 구·신 `/_nuxt/*` 정적 에셋 합본을 Worker에 먼저 배포한 뒤 웹 컨테이너 교체) 완비.
  - **DB 백업 & 오프사이트 R2 복제**: 매일 03:30 KST `pg_dump` 로컬 백업(14일 보존) + 전용 비공개 R2 버킷(`R2_DB_BACKUP_BUCKET`) 자동 업로드 및 크기 검증([`deploy/backup-db.sh`](../deploy/backup-db.sh)), 복원 리허설(50/50 테이블 완벽 일치) 완료.
  - **GitHub Actions CI**: Node 24 최신 액션(`checkout@v7`, `setup-node@v7`, `setup-go@v7`, `setup-java@v6`), `ubuntu-24.04` 고정, Gradle 캐시·재시도, Lighthouse CI 3회 중앙값 측정 파이프라인 구축 완료.

---

## 4. 상용화 정식 오픈(Launch)을 위한 최종 체크리스트 요약

### A. 코드 레벨 즉시 조치 권장 항목 (개발 세션에서 10분 내 패치 가능)
- [ ] **C-FIX-1 (고음질 워터마킹)**: [`generator/server.py`](../generator/server.py)의 `insert_watermark()`에서 원본 오디오 전체를 16kHz 모노로 다운샘플링해 반환하는 대신, AudioSeal 워터마크 잔차 신호만 원본 샘플레이트(44.1kHz/48kHz 스테레오)로 업샘플링하여 원본 파형에 합성하도록 개선
- [ ] **C-FIX-2 (차트 다국어 404 수정)**: [`web/app/pages/charts.vue#L66`](../web/app/pages/charts.vue#L66)의 `defineI18nRoute(false);` 1줄 제거 (`/en/charts` 등 다국어 차트 페이지 200 복구)
- [ ] **C-FIX-3 (라이브 8곡 인증서 발급)**: 현재 운영 서버에 `READY`로 등록된 8곡에 대해 `POST /api/v1/admin/tracks/:id/certificate`를 실행하여 `certNumber` 부여

### B. PO / CEO / 운영 설정 블로커 (외부 키 발급·법무·의사결정 항목)
- [ ] **P-1 (결제 개시)**: Toss Payments 가맹 심사 및 운영 `.env`에 `TOSS_SECRET_KEY`·`VITE_TOSS_CLIENT_KEY` 설정 후 100원 실결제→승인→환불 테스트 1회 수행
- [ ] **P-2 (생성 GPU 상시 연결)**: Runpod Serverless(또는 상시 Pod) 엔드포인트 확정 후 운영 `.env`에 `GENERATION_BACKEND_URL`·`GENERATION_BACKEND_TOKEN` 주입, 참조 지문 파일(`GENERATOR_REFERENCE_FPS`) 마운트, 웹 `NUXT_PUBLIC_GENERATE_UI=1` 활성화
- [ ] **P-3 (요금제 정본 시드 & 크레딧 차감)**: 확정 가격표(생성 9,900원 / 매장 9,900·14,900원 / 라이선스 19,900·29,900원)를 운영 `billing_plans` 테이블에 반영하고 무료/유료 생성 한도 차등(일 3~5곡 vs 월 300곡) 활성화
- [ ] **P-4 (소셜 로그인 & 메일 발송)**: Google·Apple·Kakao·Naver·LINE OAuth 클라이언트 키 5종([`docs/OAUTH_SETUP.md`](OAUTH_SETUP.md)) 및 `SMTP_HOST/USER/PASS` 실발송 계정 주입
- [ ] **P-5 (사업자 표시 & 법무 게이트 G-3)**: 사업자등록·통신판매업·음악산업진흥법 신고 완료 후 푸터 `VITE_BUSINESS_*` 환경변수 주입, 저작권법 제103조 복제·전송 중단 수령인 및 개인정보 보호책임자 실명 기재, 약관 4종 변호사 최종 검토 완료본 게시
- [ ] **P-6 (백업 R2 & 모니터링)**: ~~운영 `.env`에 `R2_DB_BACKUP_BUCKET=avibe-db-backups` 설정 및 Cloudflare R2 30일 수명주기 규칙 적용~~ → **완료(2026-11-27 KST)**: 비공개 버킷 `avibe-db-backups` 생성, 수명주기 `expire-30d` 적용, root crontab에 R2 변수 등록, cron 환경에서 업로드 및 크기 검증 통과([`PROD_RUNBOOK.md` §8](PROD_RUNBOOK.md#8-backups)). **남은 일**: UptimeRobot(`/readyz`, `/`) 5분 주기 알림 연결
  - 남은 개선 항목: 백업 버킷 전용 R2 키 발급. 대시보드에서만 발급할 수 있으며, 현재는 미디어 버킷과 같은 키를 씁니다.
