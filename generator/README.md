# AVIBE generation backend (ACE-Step 1.5)

HTTP server the Go worker calls when `GENERATION_BACKEND_URL` is set.
It runs on a GPU machine. `infer()` drives ACE-Step 1.5 (turbo DiT). It was
verified on a Runpod RTX PRO 4000 on 2026-11-21; the results are in
`docs/generation/poc-runpod-20261121.md`. The watermark is still a stub, so
v2 requests stay at 503.

## Contract

- `GET /healthz` → `{"status":"ok","modelLoaded":bool,"model":str,"contractVersions":[1,2],"weightsHash":str}`
- v1 `POST /generate` with `{"jobId","prompt","lyrics","model","params"}`
  → `{"audioBase64","mimeType","durationMs"}` (WAV bytes, base64)
- v2 (WI-G1) adds `contractVersion:2`, `modelVersion`, `seed`,
  `durationSec` (1..600), `variations` (1..4), `stems`, `watermark:true`
  (forced — `false` is a 400), and `delivery:{"uploadUrl"}` for large
  payloads → `{"contractVersion":2,"audioBase64"|"audioUrl","mimeType",
  "durationMs","modelVersion","seed","weightsHash","safety","watermarkId"}`
  with `safety={"passed":bool,"checks":[...]}`.
- Without the model: `503` with a clear message (the job fails; the server
  never returns fake audio). v2 without a watermark library is also `503`
  (fail closed).
- Env: `GENERATOR_MODEL_VERSION` and `GENERATOR_WEIGHTS_HASH` are echoed back
  on v2 replies.
- Model env: `ACESTEP_ROOT` is the ACE-Step 1.5 checkout; its `checkpoints/`
  holds the weights. `ACESTEP_CONFIG` defaults to `acestep-v15-turbo`, and
  `ACESTEP_INFERENCE_STEPS` defaults to 8. `params` accepts `durationSec`,
  `seed` and `inferenceSteps`.
- Auth: when `GENERATOR_API_TOKEN` is set, every route except `/healthz`
  requires `Authorization: Bearer <token>`; anything else gets a 401. Set it
  whenever the port can be reached from outside, because a Runpod proxy URL is
  public. The Go worker sends it from `GENERATION_BACKEND_TOKEN` (WI-N12);
  a 401 fails the job without retry.

## Bring-up on the GPU machine

ACE-Step 1.5's code and weights are both MIT-licensed. Install on a persistent
disk: on Runpod that is `/workspace`, because the container disk is wiped
when the Pod stops.

```sh
git clone --depth 1 https://github.com/ACE-Step/ACE-Step-1.5.git
cd ACE-Step-1.5 && pip install uv && uv sync     # weights download on first run
# Run this server with that venv (it already has fastapi and uvicorn):
GENERATOR_API_TOKEN=<secret> ACESTEP_ROOT=$PWD \
  .venv/bin/python -m uvicorn server:app --app-dir /path/to/generator --host 127.0.0.1 --port 8000
```

On Runpod the whole stack is already installed on network volume `y7tzkyd0g5`.
Use `bash scripts/runpod-gen.sh up|test|status|sync|down`; nothing gets reinstalled
(`docs/generation/poc-runpod-20261121.md` §6).

Point the backend at it:

```sh
GENERATION_BACKEND_URL=http://<gpu-host>:8000
GENERATION_TIMEOUT_SECONDS=600   # default
```

Submit with `POST /api/v1/partner/generations` and poll
`GET /api/v1/partner/generations/:id`. On `SUCCEEDED`, `outputObjectKey`
holds the staged WAV. Run the normal PreCheck/Finalize flow with its SHA-256
to register the track. Production use waits for G-MODEL: watermark,
similarity and the rest of the readiness plan.

## GPU floor

ACE-Step 1.5 turbo ran comfortably on a 24GB GPU. Per the upstream README,
2B turbo fits in 8–16GB, and XL needs 20GB or more without offload. Measured
warm with turbo, a 60s song took about 2.0s and a 180s song about 4.8s. The
dev box here (GTX 1630 4GB) is not a target.
