# 생성 AI 실측 — Runpod GPU (2026-11-21)

작업: Claude (PO 요청). 이전 상태 `generator/server.py`의 `infer()`·워터마크는 스텁, G2 `NOT TESTED`.
이번 결과: **ACE-Step 1.5로 실제 생성 성공, `infer()` 구현, 생성 서버 v1 왕복 확인.** 워터마크·유사도 검사·운영 연결은 남음.

## 1. 환경

| 항목 | 값 |
|---|---|
| 공급사 | Runpod Pod (Secure Cloud, EU-RO-1) — 커뮤니티 재고는 당시 전부 매진 |
| GPU | NVIDIA RTX PRO 4000 Blackwell, VRAM 24GB, 드라이버 595.112.07 |
| 요금 | 시간당 $0.57 (Pod 읽기 기준). 실행 약 40분 |
| 이미지 | `runpod/pytorch:1.0.2-cu1281-torch280-ubuntu2404` (템플릿 `runpod-torch-v280`) |
| Pod | 실측용 `efohwl29h24oda`는 삭제. **지금은 Pod 없음** — 필요할 때 `scripts/runpod-gen.sh up` (§6) |
| 볼륨 | **네트워크 볼륨 `y7tzkyd0g5` (`avibe-gen`, EU-RO-1, 40GB)**: ACE-Step 1.5 + venv + 가중치 9.5GB + 생성 서버가 설치돼 있음. Pod를 지워도 유지 |
| 모델 | ACE-Step 1.5 (`github.com/ACE-Step/ACE-Step-1.5` @ `ca1e85f`), DiT `acestep-v15-turbo` |
| 라이선스 | 코드 MIT, 가중치 MIT. 모델 카드: 생성물 상업 이용 가능, 라이선스·저작권 없는 데이터로 학습했다고 명시 |

## 2. 설치 (1회, 네트워크 볼륨에 완료됨)

아래는 최초 실측 때의 절차다. 지금은 같은 내용이 네트워크 볼륨에 설치돼 있어 반복할 필요가 없다(`/workspace/gen/install.sh`가 볼륨용 설치 스크립트).

```sh
cd /root && git clone --depth 1 https://github.com/ACE-Step/ACE-Step-1.5.git
pip install -q uv && cd ACE-Step-1.5 && UV_LINK_MODE=copy uv sync     # .venv 약 7.6GB
# 가중치(turbo DiT, LM 1.7B, VAE)는 첫 실행 때 checkpoints/에 자동 다운로드
```

- 최초 실측은 컨테이너 디스크(`/root`)에 설치했는데, **컨테이너 디스크는 Pod 정지·재시작 시 지워지고**(Runpod 문서), 서버에 묶인 볼륨은 그 서버 GPU가 차면 Pod를 다시 켤 수 없었음(`not enough free GPUs on the host`).
- 그래서 **네트워크 볼륨**(데이터센터 단위, 같은 DC의 어느 Pod에나 연결)에 다시 설치: `UV_PYTHON_INSTALL_DIR`·`HF_HOME`도 볼륨 안. venv 12GB(네트워크 디스크라 하드링크 없음) + 가중치 9.5GB.
- 볼륨 `/workspace/gen/`: `server.py`, `start.sh`, `install.sh`, `.token`(서버 토큰), `demo_songs.py`, `manifest.json`.

## 3. 실측 (turbo, 8 steps, LM 미사용)

ACE-Step `profile_inference.py`:

| 곡 길이 | 전체 시간 | 파이프라인 | 웜업(최초 1회) |
|---|---|---|---|
| 60초 | 1.96초 | 1.26초 | 3.44초 |
| 180초 | 4.77초 | 3.52초 | 5.81초 |

생성 서버(`generator/server.py`) 경유, HTTP 왕복:

| 요청 | 시간 |
|---|---|
| 30초 곡, 첫 요청(모델 로딩 포함) | 13.1초 |
| 30초 곡, 이후 | 약 1.55초 |
| 180초 곡 | 약 5.1초 |
| 210초 곡 6개 연속 | 첫 곡 17.8초, 이후 각 약 6.3초 |

네트워크 볼륨 + RTX 2000 Ada(16GB, 시간당 $0.24):

| 요청 | 시간 |
|---|---|
| 최초(가중치 9.5GB 다운로드 포함) | 146.5초 |
| 180초 곡, 웜 | 10.7초 |
| **새 Pod에 볼륨만 연결**: `up`(Pod 생성~서버 시작) | 78초 |
| 그 직후 첫 생성(네트워크 디스크에서 모델 로딩) | 71.3초 |
| 이후 30초 곡 | 2.3초 |

- 같은 시드(42)로 두 번 생성 → SHA-256 동일. **재현성 확인.**
- GPU 비용 추정(시간당 $0.57 × 생성 시간, 웜 상태): 3분 곡 약 $0.0008. 콜드 스타트·대기 시간 제외.
- **NOT TESTED**: 5Hz LM(thinking/CoT 메타데이터) 경로. `--thinking`으로 돌렸지만 프로파일러가 LM을 로드하지 않아 DiT 단독과 같은 결과. XL 모델, 배치 생성, 동시 요청.

## 4. 코드 변경 (`generator/server.py`)

1. `infer()` 구현: `AceStepHandler.initialize_service()` 1회 지연 로딩 → `acestep.inference.generate_music()` → WAV 바이트. `params`의 `durationSec`, `seed`, `inferenceSteps` 반영. 요청 직렬화(GPU당 1건).
2. 환경변수: `ACESTEP_ROOT`(체크아웃 경로, 가중치는 첫 생성 때 `checkpoints/`로 다운로드), `ACESTEP_CONFIG`(기본 `acestep-v15-turbo`), `ACESTEP_INFERENCE_STEPS`(기본 8), `GENERATOR_API_TOKEN`.
3. **토큰 인증 추가**: `GENERATOR_API_TOKEN`이 설정되면 `/healthz` 외 모든 요청에 `Authorization: Bearer <token>` 필수(불일치 401). Runpod 프록시 주소(`https://<podId>-8000.proxy.runpod.net`)는 공개라서 필요.
4. **기존 버그 수정**: `GenerateResponse`의 `mimeType`·`durationMs`가 `serialization_alias`만 있어 응답 생성 시 검증 실패(500). 모델이 없어 이 경로가 실행된 적이 없어 숨어 있었음. `alias`로 수정.
5. 응답에서 null 필드 제외(`response_model_exclude_none`). v1 응답은 원래 3개 키만 나감.
6. v2(워터마크 필수)는 여전히 **503으로 막힘(의도된 fail closed)**.

검증: Pod에서 `/healthz` 200(`modelLoaded: true`), 토큰 없는 `/generate` 401, v1 생성 200(30초 → `durationMs: 30000`, 180초 → `180000`), v2 503.

## 5. 데모 곡 6개

- 스타일: 피아노 중심 J-pop 밴드 사운드, 감성 미드템포, 남성 팔세토 보컬. **프롬프트·제목에 아티스트명과 곡명 없음, 가사는 새로 작성.**
- 각 210초, 시드 11/23/37/41/58/64. 프롬프트와 가사는 `/workspace/gen/demo_songs.py`.
- 파일: PO PC `C:\Users\dnjst\Desktop\avibe-demo-20261121\` (MP3 192kbps). 저장소에는 넣지 않음.
- **운영 업로드 보류**: PO가 데모 파트너 계정(`demo-partner-20261110-180055@avibemusic.com`)에 등록을 요청함. 제공자 `AVIBE` 인증 게이트는 서버가 확인한 워터마크·유사도 증거가 있어야 통과하는데, 둘 다 미구현이라 정직한 경로가 없음. 다른 제공자로 신고하거나 DB에 직접 넣는 것은 거짓 인증이라 하지 않음. 워터마크·유사도 검사가 들어간 뒤 생성 작업 경로로 등록할 것(작업지시 WI-N12, N13).

## 6. 다시 쓸 때 (설치 없이)

```sh
bash scripts/runpod-gen.sh up       # EU-RO-1에서 재고 있는 가장 싼 GPU로 Pod 생성 + 볼륨 연결 + 서버 시작
bash scripts/runpod-gen.sh test     # 30초 곡 1개 생성(첫 요청은 모델 로딩 약 70초)
bash scripts/runpod-gen.sh status   # 상태·요금·SSH 명령
bash scripts/runpod-gen.sh sync     # 저장소의 generator/server.py를 볼륨에 반영(이후 up 또는 start.sh 재실행)
bash scripts/runpod-gen.sh down     # Pod 삭제(GPU 과금 중단, 볼륨은 유지)
```

- 기본은 Pod 안 127.0.0.1에서만 열림(SSH로 접근). 외부(Go 워커 등)에서 부르려면 `PUBLIC=1 bash scripts/runpod-gen.sh up` → `https://<podId>-8000.proxy.runpod.net`, 토큰은 볼륨의 `/workspace/gen/.token`.
- 필요 조건: Runpod CLI 로그인(`~/.local/bin/runpod.exe`, PO 계정), SSH 키 `~/.ssh/runpod_ed25519`.
- 요금: Pod는 켜져 있는 동안만 시간당 과금(RTX 2000 Ada $0.24). 네트워크 볼륨 40GB는 보관료가 계속 나감. 완전히 그만 쓸 때 `runpod volume delete y7tzkyd0g5`.
- GPU는 EU-RO-1 재고에 따라 `GPUS` 순서대로 시도. 볼륨이 EU-RO-1에 있으므로 다른 데이터센터 GPU는 붙일 수 없음.
