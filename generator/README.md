# AVIBE generation backend (WI-03, setup-only)

HTTP server the Go worker calls when `GENERATION_BACKEND_URL` is set.
Runs on a GPU machine (your home PC). Nothing here runs a model by itself.

## Contract

- `GET /healthz` → `{"status":"ok","modelLoaded":bool,"model":str}`
- `POST /generate` with `{"jobId","prompt","lyrics","model","params"}`
  → `{"audioBase64","mimeType","durationMs"}` (WAV bytes, base64)
- Without weights: `503` with a clear message (job fails, never fake audio).

## Bring-up on the GPU machine

```sh
cd generator
python -m venv .venv
source .venv/bin/activate        # Windows: .venv\Scripts\activate
pip install -r requirements.txt
# then install torch + CUDA + the acestep release you chose,
# set GENERATOR_MODEL_DIR to the weights, and implement infer() in server.py
# following that release's README.
uvicorn server:app --host 0.0.0.0 --port 8000
```

Point the backend at it:

```sh
GENERATION_BACKEND_URL=http://<gpu-host>:8000
GENERATION_TIMEOUT_SECONDS=600   # default
```

Submit via `POST /api/v1/partner/generations`, poll
`GET /api/v1/partner/generations/:id`. On `SUCCEEDED`, `outputObjectKey` holds
the staged WAV — run the normal PreCheck/Finalize flow with its SHA-256 to
register the track.

## GPU floor

ACE-Step class models need ~8GB+ VRAM and a CUDA-capable GPU. The dev box here
(GTX 1630 4GB) cannot host it — that is why this directory is setup-only.
