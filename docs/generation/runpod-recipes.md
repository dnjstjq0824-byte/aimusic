# 생성 서버 배포 레시피 — Runpod Pod + Serverless (WI-N11)

기준: `generator/Dockerfile`의 `gpu` 타겟(ACE-Step 1.5 @ `ca1e85f` +
turbo 가중치 포함). 실측치는 `poc-runpod-20261121.md`(2026-11-21,
RTX PRO 4000 / RTX 2000 Ada). 운영 연결은 G-MODEL 통과 후.

## 공통

- 이미지: `avibe-generator:gpu` (`GENERATOR_WEIGHTS_HASH`·`GENERATOR_MODEL_VERSION`
  빌드 ARG로 스탬프, v2 응답에 에코).
- 토큰: `GENERATOR_API_TOKEN` 필수(외부 도달 가능하므로). Go 워커는
  `GENERATION_BACKEND_TOKEN`에 같은 값(WI-N12).
- 헬스: `GET /healthz` → `modelLoaded: true`가 웜 기준.
- 계약: `POST /generate` (v1/v2, `generator/README.md`).

## A. Pod (개발·실측용)

네트워크 볼륨 방식(`scripts/runpod-gen.sh`)이 있으면 그대로 쓰고,
이미지 방식은 아래와 같다. 이미지 방식은 시작 시 다운로드를 안 해서
네트워크 디스크 로딩(약 70초)보다 콜드 스타트가 짧아야 한다 — 비교
실측은 Pod 1회 기동으로 한다(아래 "콜드 스타트 측정").

```sh
# 1. 이미지 빌드·푸시 (가중치 포함, 레지스트리는 PO 결정)
docker build --target gpu -t <registry>/avibe-generator:gpu \
  --build-arg GENERATOR_WEIGHTS_HASH=<sha256 of checkpoints> generator/
docker push <registry>/avibe-generator:gpu
# 2. Pod 생성: Secure Cloud, EU-RO-1, VRAM 16GB 이상, CUDA >= 12.8,
#    이미지 <registry>/avibe-generator:gpu, 포트 8000/http, 22/tcp,
#    환경 GENERATOR_API_TOKEN=<secret>
# 3. 헬스 대기 후 측정 (아래), 끝나면 즉시 Pod 삭제 (과금)
```

## B. Serverless (운영 후보)

Runpod Serverless 엔드포인트에 같은 이미지를 올린다. 진입점이
`/generate`가 아니라서 어댑터가 필요 없다 — Serverless는 컨테이너의
HTTP 서버를 그대로 호출한다(포트 8000, `runsync`/`run` 래핑은
Runpod 측). 설정:

- 컨테이너 이미지: `<registry>/avibe-generator:gpu`
- 컨테이너 포트: 8000, 시작 명령: 이미지 기본 CMD
- 환경: `GENERATOR_API_TOKEN=<secret>` (Runpod secret)
- GPU: VRAM 16GB 이상 (실측 하한, §C)
- 헬스체크: `GET /healthz`, `modelLoaded: true`
- 스케일: min 0 (콜드 스타트 허용) 또는 min 1 (지연 보장, 비용 별도)

요금·지연 비교는 hourly Pod와 같은 GPU 기종으로 맞추고,
`docs/generation/gpu-options.md` C1 행에 기입한다(가격은 공급사
요금 페이지 캡처 기준, 추정 금지).

## C. 콜드 스타트 측정 방법

```sh
T0=$(date +%s)
# Pod 생성 (또는 serverless cold 호출) …… 서버 시작 ……
until curl -sf http://<host>:8000/healthz | grep -q '"modelLoaded":true'; do
  sleep 2
done
T1=$(date +%s); echo "cold-to-warm: $((T1-T0))s"
# 첫 생성 (모델이 메모리에 올라가는 시간 포함)
time curl -s -X POST http://<host>:8000/generate \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"jobId":"cold1","prompt":"lofi chill hop, warm piano","params":{"durationSec":30,"seed":42}}' \
  -o /tmp/cold1.json -w 'first-generate: %{time_total}s http=%{http_code}\n'
```

비교 기준(2026-11-21 네트워크 볼륨 방식): Pod 생성~서버 시작 78초,
그 직후 첫 생성 71.3초(이후 30초 곡 2.3초). 이미지 방식 측정값이
나오면 여기에 기입한다:

| 방식 | cold-to-warm | 첫 생성(30초) | 웜 30초 |
|---|---|---|---|
| 네트워크 볼륨 (기준) | 78초 | 71.3초 | 2.3초 |
| 이미지 (gpu 타겟) | NOT TESTED | NOT TESTED | NOT TESTED |

측정 1회는 `runpod-gen.sh up` → 측정 → 반드시 `down`.
Pod 켜기·끄기는 PO 또는 Claude가 한다(P6).
