"""AVIBE generation backend reference server (WI-03 setup, WI-G1 contract v2).

Runs on a GPU machine (e.g. your home PC) and serves the HTTP contract the Go
worker expects:

  GET  /healthz  -> {"status": "ok", "modelLoaded": bool, "model": str,
                     "contractVersions": [1, 2], "weightsHash": str}
  POST /generate -> v1 {"audioBase64", "mimeType", "durationMs"} or
                     v2 {"contractVersion": 2, "audioBase64"|"audioUrl",
                         "mimeType", "durationMs", "modelVersion", "seed",
                         "weightsHash", "safety", "watermarkId"}

Request body (v1 clients send the original five keys and get a v1 reply;
v2 clients add contractVersion 2 plus modelVersion/seed/durationSec/
variations/stems/watermark/delivery):

  {"jobId": str, "prompt": str, "lyrics": str, "model": str,
   "contractVersion": 1|2, "modelVersion": str, "seed": int,
   "durationSec": int, "variations": int, "stems": [str],
   "watermark": bool, "params": object,
   "delivery": {"uploadUrl": str}}

v2 forces watermark:true (a v2 request with watermark:false is rejected).
Large payloads: when delivery.uploadUrl is set, the server PUTs the WAV
there and answers with audioUrl instead of inline base64.

Model: ACE-Step 1.5 (https://github.com/ACE-Step/ACE-Step-1.5, code MIT).
Set ACESTEP_ROOT to an installed checkout (its `checkpoints/` holds the
weights; first run downloads them) and run this server with that checkout's
venv. Without it, /generate returns 503 with a clear message.
Measured 2026-11-21 on Runpod RTX PRO 4000 (24GB): turbo DiT, warm,
60s song ~2.0s, 180s song ~4.8s (see docs/generation/poc-runpod-20261121.md).

Auth: when GENERATOR_API_TOKEN is set, every request except /healthz must send
`Authorization: Bearer <token>` (the Runpod proxy URL is public).
"""

from __future__ import annotations

import base64
import hmac
import io
import os
import tempfile
import threading
import urllib.request
import wave

from fastapi import FastAPI, HTTPException, Request
from pydantic import BaseModel, Field

app = FastAPI(title="avibe-generator")

MODEL_ID = os.environ.get("GENERATOR_MODEL_ID", "ace-step-v1.5-turbo")
MODEL_VERSION = os.environ.get("GENERATOR_MODEL_VERSION", "")
WEIGHTS_HASH = os.environ.get("GENERATOR_WEIGHTS_HASH", "")
API_TOKEN = os.environ.get("GENERATOR_API_TOKEN", "")
ACESTEP_ROOT = os.environ.get("ACESTEP_ROOT", "")
ACESTEP_CONFIG = os.environ.get("ACESTEP_CONFIG", "acestep-v15-turbo")
DEFAULT_STEPS = int(os.environ.get("ACESTEP_INFERENCE_STEPS", "8"))  # turbo default
MAX_PROMPT_CHARS = 2000
MAX_LYRICS_CHARS = 10000
MAX_DURATION_SEC = 600
MAX_VARIATIONS = 4


class Delivery(BaseModel):
    upload_url: str = Field(default="", alias="uploadUrl")

    model_config = {"populate_by_name": True}


class GenerateRequest(BaseModel):
    contract_version: int = Field(default=1, alias="contractVersion")
    job_id: str = Field(alias="jobId")
    prompt: str = ""
    lyrics: str = ""
    model: str = ""
    model_version: str = Field(default="", alias="modelVersion")
    seed: int | None = None
    duration_sec: int | None = Field(default=None, alias="durationSec")
    variations: int = 1
    stems: list[str] = []
    watermark: bool | None = None
    params: dict = {}
    delivery: Delivery | None = None

    model_config = {"populate_by_name": True}


class Safety(BaseModel):
    passed: bool
    checks: list[str] = []

    model_config = {"populate_by_name": True}


class GenerateResponse(BaseModel):
    contract_version: int | None = Field(default=None, alias="contractVersion")
    audio_base64: str | None = Field(default=None, alias="audioBase64")
    audio_url: str | None = Field(default=None, alias="audioUrl")
    # alias (not serialization_alias): the handlers build this model with the
    # camelCase names, which failed validation before 2026-11-21.
    mime_type: str = Field(alias="mimeType")
    duration_ms: int = Field(alias="durationMs")
    model_version: str | None = Field(default=None, alias="modelVersion")
    seed: int | None = None
    weights_hash: str | None = Field(default=None, alias="weightsHash")
    safety: Safety | None = None
    watermark_id: str | None = Field(default=None, alias="watermarkId")

    model_config = {"populate_by_name": True}


@app.middleware("http")
async def require_token(request: Request, call_next):
    """Bearer-token gate (fail closed once GENERATOR_API_TOKEN is set)."""
    if API_TOKEN and request.url.path != "/healthz":
        sent = request.headers.get("authorization", "")
        expected = f"Bearer {API_TOKEN}"
        if not hmac.compare_digest(sent.encode(), expected.encode()):
            from fastapi.responses import JSONResponse

            return JSONResponse(status_code=401, content={"detail": "unauthorized"})
    return await call_next(request)


def model_available() -> bool:
    """True when an ACE-Step 1.5 checkout with weights is configured and importable.

    Kept as a function (not import-time) so /healthz can report honestly.
    """
    if not ACESTEP_ROOT or not os.path.isdir(os.path.join(ACESTEP_ROOT, "checkpoints")):
        return False
    try:
        import acestep  # noqa: F401  pylint: disable=unused-import
    except ImportError:
        return False
    return True


_handlers: tuple | None = None
_init_lock = threading.Lock()
_gen_lock = threading.Lock()  # one generation at a time per GPU


def _get_handlers() -> tuple:
    """Load the ACE-Step DiT once (lazy, so /healthz stays fast)."""
    global _handlers  # noqa: PLW0603
    with _init_lock:
        if _handlers is not None:
            return _handlers
        if not model_available():
            raise NotImplementedError(
                "ACE-Step 1.5 is not installed here: set ACESTEP_ROOT to a checkout with "
                "checkpoints/ and run with its venv (see generator/README.md)."
            )
        from acestep.handler import AceStepHandler  # pylint: disable=import-outside-toplevel
        from acestep.llm_inference import LLMHandler  # pylint: disable=import-outside-toplevel

        try:
            import flash_attn  # noqa: F401  pylint: disable=unused-import,import-outside-toplevel

            use_flash = True
        except ImportError:
            use_flash = False
        dit = AceStepHandler()
        status, ok = dit.initialize_service(
            project_root=ACESTEP_ROOT,
            config_path=ACESTEP_CONFIG,
            device="auto",
            use_flash_attention=use_flash,
            compile_model=False,
            offload_to_cpu=False,
            offload_dit_to_cpu=False,
            quantization=None,
        )
        if not ok:
            raise RuntimeError(f"ACE-Step DiT init failed: {status}")
        # The 5Hz LM (thinking / CoT metadata) is not loaded: text2music runs
        # DiT-only. Enabling it is a later WI with its own measurement.
        _handlers = (dit, LLMHandler())
        return _handlers


def infer(prompt: str, lyrics: str, model: str, params: dict) -> bytes:
    """Run ACE-Step 1.5 text2music and return WAV bytes.

    params: optional "durationSec" (else the model picks), "seed" (else random),
    "inferenceSteps" (default ACESTEP_INFERENCE_STEPS). `model` is recorded by
    the caller; the loaded config is ACESTEP_CONFIG.
    """
    dit, llm = _get_handlers()
    from acestep.inference import (  # pylint: disable=import-outside-toplevel
        GenerationConfig,
        GenerationParams,
        generate_music,
    )

    seed = params.get("seed")
    duration = params.get("durationSec")
    text = (lyrics or "").strip()
    gen_params = GenerationParams(
        task_type="text2music",
        caption=prompt,
        lyrics=text or "[Instrumental]",
        instrumental=not text,
        duration=float(duration) if duration else -1,
        inference_steps=int(params.get("inferenceSteps") or DEFAULT_STEPS),
        seed=int(seed) if seed is not None else -1,
        thinking=False,
        use_cot_metas=False,
        use_cot_caption=False,
        use_cot_language=False,
    )
    gen_config = GenerationConfig(
        batch_size=1,
        use_random_seed=seed is None,
        seeds=[int(seed)] if seed is not None else None,
        audio_format="wav",
    )
    with _gen_lock, tempfile.TemporaryDirectory() as tmp:
        result = generate_music(dit, llm, gen_params, gen_config, save_dir=tmp)
        if not result.success or not result.audios:
            raise RuntimeError(result.error or result.status_message or "ACE-Step generation failed")
        path = result.audios[0].get("path") or ""
        if not path or not os.path.isfile(path):
            raise RuntimeError("ACE-Step returned no audio file")
        with open(path, "rb") as fh:
            return fh.read()


def insert_watermark(raw_wav: bytes, job_id: str) -> tuple[bytes, str]:
    """Insert the inaudible provenance watermark; return (wav, watermarkId).

    SETUP-ONLY STUB: implement on the GPU machine with the watermark library
    chosen by the WI-G2 PoC (candidate: AudioSeal — license + robustness to
    confirm there). Until then, watermarking is unavailable and v2 requests
    fail closed with 503 below.
    """
    raise NotImplementedError(
        "watermark library not installed on this machine "
        "(WI-G2 PoC selects it; see docs/generation/gpu-options.md and generator/poc/run_poc.py)"
    )


def wav_duration_ms(raw: bytes) -> int:
    try:
        with wave.open(io.BytesIO(raw), "rb") as fh:
            frames, rate = fh.getnframes(), fh.getframerate() or 1
            return int(frames / rate * 1000)
    except (wave.Error, EOFError):
        return 0


def put_bytes(url: str, data: bytes, content_type: str) -> None:
    req = urllib.request.Request(url, data=data, method="PUT")
    req.add_header("Content-Type", content_type)
    with urllib.request.urlopen(req, timeout=600) as resp:
        if resp.status not in (200, 201, 204):
            raise RuntimeError(f"upload to presigned URL failed: HTTP {resp.status}")


def validate_request(req: GenerateRequest) -> int:
    """Shared v1/v2 validation; returns the effective contract version."""
    version = req.contract_version or 1
    if version not in (1, 2):
        raise HTTPException(status_code=400, detail="contractVersion must be 1 or 2")
    if not req.prompt.strip() or len(req.prompt) > MAX_PROMPT_CHARS:
        raise HTTPException(status_code=400, detail="prompt 1..2000 chars")
    if len(req.lyrics) > MAX_LYRICS_CHARS:
        raise HTTPException(status_code=400, detail="lyrics max 10000 chars")
    if version == 2:
        if req.watermark is False:
            raise HTTPException(status_code=400, detail="contract v2 forces watermark:true")
        if req.duration_sec is not None and not 1 <= req.duration_sec <= MAX_DURATION_SEC:
            raise HTTPException(status_code=400, detail="durationSec 1..600")
        if not 1 <= req.variations <= MAX_VARIATIONS:
            raise HTTPException(status_code=400, detail="variations 1..4")
        if any(not s.strip() for s in req.stems):
            raise HTTPException(status_code=400, detail="stems must be non-empty names")
        if req.seed is not None and req.seed < 0:
            raise HTTPException(status_code=400, detail="seed must be >= 0")
    return version


@app.get("/healthz")
def healthz() -> dict:
    return {
        "status": "ok",
        "modelLoaded": model_available(),
        "model": MODEL_ID,
        "contractVersions": [1, 2],
        "weightsHash": WEIGHTS_HASH,
    }


@app.post("/generate", response_model_exclude_none=True)
def generate(req: GenerateRequest) -> GenerateResponse:
    version = validate_request(req)
    params = dict(req.params or {})
    if req.duration_sec is not None:
        params["durationSec"] = req.duration_sec
    if req.seed is not None:
        params["seed"] = req.seed
    try:
        audio = infer(req.prompt, req.lyrics, req.model or MODEL_ID, params)
    except NotImplementedError as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    except RuntimeError as exc:
        raise HTTPException(status_code=500, detail=str(exc)) from exc
    if version == 1:
        return GenerateResponse(
            audioBase64=base64.b64encode(audio).decode("ascii"),
            mimeType="audio/wav",
            durationMs=wav_duration_ms(audio),
        )
    try:
        marked, watermark_id = insert_watermark(audio, req.job_id)
    except NotImplementedError as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    duration_ms = wav_duration_ms(marked)
    if req.delivery and req.delivery.upload_url:
        try:
            put_bytes(req.delivery.upload_url, marked, "audio/wav")
        except Exception as exc:  # noqa: BLE001 - surface any upload failure
            raise HTTPException(status_code=502, detail=f"presigned upload failed: {exc}") from exc
        return GenerateResponse(
            contractVersion=2,
            audioUrl=req.delivery.upload_url,
            mimeType="audio/wav",
            durationMs=duration_ms,
            modelVersion=req.model_version or MODEL_VERSION,
            seed=req.seed,
            weightsHash=WEIGHTS_HASH,
            safety=Safety(passed=True, checks=["prompt", "watermark"]),
            watermarkId=watermark_id,
        )
    return GenerateResponse(
        contractVersion=2,
        audioBase64=base64.b64encode(marked).decode("ascii"),
        mimeType="audio/wav",
        durationMs=duration_ms,
        modelVersion=req.model_version or MODEL_VERSION,
        seed=req.seed,
        weightsHash=WEIGHTS_HASH,
        safety=Safety(passed=True, checks=["prompt", "watermark"]),
        watermarkId=watermark_id,
    )
