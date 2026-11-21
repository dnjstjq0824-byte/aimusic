# AVIBE generation server image (WI-N11).
#
# Two targets:
#   server - python-slim + server.py only (no model; /generate answers 503).
#            Local verify: docker build --target server . && run, no GPU.
#   gpu    - runpod torch base + ACE-Step 1.5 @ ca1e85f + weights + server.
#            Pod and Serverless use this target. Weights bake in at build
#            time (DOWNLOAD_WEIGHTS=1, default) so cold start skips the
#            ~70s network-disk model load measured on 2026-11-21.
#
# Build args:
#   ACESTEP_REF          ACE-Step 1.5 commit (default ca1e85f, the PoC ref)
#   DOWNLOAD_WEIGHTS     1 = bake turbo weights into the image (default);
#                        0 = download on first generation (slimmer image)
#   GENERATOR_MODEL_VERSION / GENERATOR_WEIGHTS_HASH echoed on v2 replies
#
# Run:
#   docker run -p 8000:8000 -e GENERATOR_API_TOKEN=<secret> avibe-generator:gpu
# Health: GET /healthz -> {"status":"ok","modelLoaded":bool,...}
ARG ACESTEP_REF=ca1e85f

FROM python:3.12-slim AS server
WORKDIR /app
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt
COPY server.py ./
COPY eval/memorization.py ./eval_memo.py
EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
  CMD python -c "import urllib.request,sys;urllib.request.urlopen('http://127.0.0.1:8000/healthz',timeout=8);sys.exit(0)"
CMD ["python", "-m", "uvicorn", "server:app", "--host", "0.0.0.0", "--port", "8000"]

FROM runpod/pytorch:1.0.2-cu1281-torch280-ubuntu2404 AS gpu
ARG ACESTEP_REF
ARG DOWNLOAD_WEIGHTS=1
ARG GENERATOR_MODEL_VERSION=acestep-v15-turbo
ARG GENERATOR_WEIGHTS_HASH=""
ENV ACESTEP_ROOT=/opt/acestep \
    ACESTEP_CONFIG=acestep-v15-turbo \
    ACESTEP_INFERENCE_STEPS=8 \
    GENERATOR_MODEL_ID=ace-step-v1.5-turbo \
    GENERATOR_MODEL_VERSION=${GENERATOR_MODEL_VERSION} \
    GENERATOR_WEIGHTS_HASH=${GENERATOR_WEIGHTS_HASH} \
    DEBIAN_FRONTEND=noninteractive
WORKDIR /app
# ACE-Step 1.5 (code MIT) at the pinned PoC commit.
RUN apt-get update && apt-get install -y --no-install-recommends git curl \
 && rm -rf /var/lib/apt/lists/* \
 && git clone https://github.com/ACE-Step/ACE-Step-1.5.git /opt/acestep \
 && git -C /opt/acestep checkout "${ACESTEP_REF}" \
 && pip install -q uv \
 && cd /opt/acestep && UV_LINK_MODE=copy uv sync --no-dev
COPY requirements.txt ./
RUN /opt/acestep/.venv/bin/pip install --no-cache-dir -r requirements.txt
COPY server.py ./
COPY eval/memorization.py ./eval_memo.py
# Bake the turbo weights so the first request skips the download
# (single forward pass, output discarded; ~9.5GB into checkpoints/).
RUN if [ "$DOWNLOAD_WEIGHTS" = "1" ]; then \
      cd /opt/acestep && .venv/bin/python -c \
        "from acestep.handler import AceStepHandler; h=AceStepHandler(); \
         s,ok=h.initialize_service(project_root='/opt/acestep',config_path='acestep-v15-turbo',device='auto',use_flash_attention=False,compile_model=False,offload_to_cpu=False,offload_dit_to_cpu=False,quantization=None); \
         assert ok, s; print('weights baked:', s)"; \
    fi
EXPOSE 8000
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
  CMD /opt/acestep/.venv/bin/python -c "import urllib.request,sys;urllib.request.urlopen('http://127.0.0.1:8000/healthz',timeout=8);sys.exit(0)"
CMD ["/opt/acestep/.venv/bin/python", "-m", "uvicorn", "server:app", "--host", "0.0.0.0", "--port", "8000"]
