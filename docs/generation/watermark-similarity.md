# 워터마크·유사도 검사 선정 기록 (WI-N13, 2026-11-21)

AVIBE 제공자 인증 게이트(`licensegate.ValidateCommercialEligibility`)가
요구하는 "서버가 확인한 워터마크·유사도 증거"의 구현 선정 기록이다.
실측은 로컬 CPU(torch 2.15+cpu, audioseal 0.2.1)에서 수행.

## 1. 워터마크 라이브러리 선정

| 후보 | 라이선스 (코드/가중치) | 상업 이용 | MP3 내성 | 잘라내기 내성 | 판정 |
|---|---|---|---|---|---|
| AudioSeal (Meta, ICML 2024) | MIT / MIT (2024-04-02 전체 MIT 전환) | 가능 | 실측 통과 (192k, 검출 1.0) | 실측 통과 (8초 중 4초, 0.9933) | **선정** |
| WavMark (학계, arXiv 2308.12770) | MIT (PyPI `wavmark` 동봉 가중치 포함) | 가능 | 논문 주장 10종 공격 내성 | NOT TESTED | 예비 (AudioSeal 대체용) |
| SynthID (Google) | 공개 라이브러리 아님 (이용 제한) | 조건부 | NOT TESTED | NOT TESTED | 탈락 (라이선스·가용성) |
| audiowmark (S. Westerfeld) | GPLv3 | 조건부(카피레프트) | 문서상 강함 | NOT TESTED | 탈락 (GPLv3) |

선정 근거: MIT(상업 제약 없음) + 로컬 실측 내성 + 메시지 페이로드(16비트,
추적 ID용은 아니고 존재 판정용).

### 1.1 로컬 실측 (2026-11-21, CPU)

- 삽입→검출 왕복: clean 0.0000 / marked 1.0000 (임계 0.5).
- MP3 192k 왕복 후 검출: 1.0000.
- 8초 중 중간 4초만 남기고 잘라내기: 0.9933.
- -20dB 감쇠: 1.0000.
- 메시지 비트: 16비트 중 15비트 일치(1비트 오류 관측). 따라서
  `watermarkId`(`wm-<job>-<rand>`)는 서버 생성 핸들로 별도 추적하고,
  오디오에서 디코드하지 않는다.

### 1.2 구현 (`generator/server.py`)

- `insert_watermark()` (C-FIX-1, 2026-11-24): 원본 디코드 후 16kHz
  mono 캐리어로 AudioSeal 잔차 신호만 계산 → 원본 샘플레이트로
  선형 업샘플 → 원본 전 채널에 가산. 반환 WAV는 입력의
  샘플레이트·채널 수·비트뎁스를 그대로 유지 (구 동작인 16kHz mono
  다운샘플 반환은 폐기). `watermarkId` 부여, `audioseal` 미설치 시
  `NotImplementedError` → v2는 503 (fail closed, 변경 없음).
- 주의: §1.1 내성 수치는 구 경로(16kHz mono 반환) 실측치이므로,
  C-FIX-1 이후 GPU 실측으로 검출률 재확인이 필요하다.
- `detect_watermark()`: (존재 여부, 확률). 검증·감사 경로용.
- 임계: `WATERMARK_PROB_MIN = 0.5` (실측 1.0/0.0 분리).

## 2. 유사도 검사

### 2.1 방식

`eval/memorization.py`의 PCM 프리필터(256블록 부호 해시, stdlib)와
같은 알고리즘을 `server.py:check_similarity()`에 내장. 참조 목록은
`GENERATOR_REFERENCE_FPS` JSON
(`{"fingerprints": [{"id", "hash"}]}`).

- 통과 기준: `best_similarity < GENERATOR_SIMILARITY_MAX` (기본 0.90).
- 참조 목록이 없으면 검사를 수행하지 않고 `compared: 0`으로 기록하며,
  safety verdict에 `"similarity"`를 포함하지 않는다 → AVIBE 게이트는
  참조 데이터가 갖춰지기 전까지 fail-closed 유지.
- 한계: exact·근접 재사용만 잡는다. 스펙트럼 핑거프린트(numpy +
  peak-hash)는 `fingerprint_spectral()` 업그레이드 지점으로 문서화돼
  있음 (GPU PC 작업).

### 2.2 로컬 실측

- 참조 없음: `passed=True, compared=0`.
- 자기 참조: `passed=False, best=1.0`.
- 다른 클립(440Hz vs 880Hz 톤): `passed=True, best=0.5`.

### 2.3 safety verdict 구성 (v2)

`checks = ["prompt", "watermark"] + (["similarity"] if compared > 0)`.
`passed=False`면 Go 워커가 작업을 실패시킨다(재시도는 워커 정책,
401만 재시도 제외).

## 3. 프롬프트 사전 차단 (엔진)

`backend/internal/service/generation.go`:

- `BannedVoicePhrases`·`BannedStylePatterns`: 기존 유지 (5개 언어).
- `BannedArtistNames`: 17개 → 58개로 확장 (글로벌·한국·일본·중국
  명확한 이름). 매칭 규칙 변경: CJK 이름은 substring (단어 경계가
  없어 `米津玄師みたいな`·`지민처럼`도 차단), 라틴 이름은 word
  boundary 유지.
- 의도적 제외 (과차단 방지): 일반명사 충돌 (queen/정국/태양/대성/
  윈터/슈가), 1음절 이름 (뷔), 곡 제목 (범용 단어 — 건별
  `GENERATION_BANNED_KEYWORDS`로 차단).
- 곡 제목 일반 차단은 불가: "밤", "사랑" 같은 단어를 막으면 정상
  프롬프트가 죽는다. 특정 곡 모방 의심은 유사도 검사(사후)가 담당.

## 4. 로컬 E2E (stub 생성 서버 → AVIBE READY)

절차와 결과는 진행로그 WI-N13 참조. 요약:

1. stub 생성 서버 (v2 + safety 전 항목 + 실제 WAV) 기동.
2. 워커 `GENERATION_BACKEND_URL`·`GENERATION_BACKEND_TOKEN` 설정.
3. `POST /partner/generations` → 워커 처리 → `SUCCEEDED`.
4. 스테이징 오디오 SHA로 `POST /partner/pre-check` (provider `AVIBE`)
   → `eligible: true`.
5. `POST /partner/finalize` → 트랙 생성 → 트랜스코딩 → `READY`,
   `origin: AVIBE_MODEL`.

## 5. 데모 6곡

같은 프롬프트·시드(`demo_songs.py`, PoC 문서 §5)로 생성 작업 경로를
통해 다시 생성해야 하나, ACE-Step 실행에 GPU Pod가 필요하다.
Pod 켜기·끄기는 PO 또는 Claude 담당(P6)이라 생성 실측은 BLOCKED.
§4의 등록 리허설 경로로 대체 검증했다. 운영 계정 등록은
G-MODEL·G-2 승인 후.
