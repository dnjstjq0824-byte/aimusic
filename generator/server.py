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

Without model weights installed, /generate returns 503 with a clear message.
To go live, install ACE-Step + weights on the GPU machine and implement
infer() below following the README of the ACE-Step release you install.
"""

from __future__ import annotations

import base64
import io
import os
import urllib.request
import wave

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

app = FastAPI(title="avibe-generator")

MODEL_ID = os.environ.get("GENERATOR_MODEL_ID", "ace-step-v1")
MODEL_VERSION = os.environ.get("GENERATOR_MODEL_VERSION", "")
WEIGHTS_HASH = os.environ.get("GENERATOR_WEIGHTS_HASH", "")
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
    mime_type: str = Field(serialization_alias="mimeType")
    duration_ms: int = Field(serialization_alias="durationMs")
    model_version: str | None = Field(default=None, alias="modelVersion")
    seed: int | None = None
    weights_hash: str | None = Field(default=None, alias="weightsHash")
    safety: Safety | None = None
    watermark_id: str | None = Field(default=None, alias="watermarkId")

    model_config = {"populate_by_name": True}


def model_available() -> bool:
    """True when the optional ACE-Step stack is importable.

    Kept as a function (not import-time) so /healthz can report honestly.
    """
    try:
        import acestep  # noqa: F401  pylint: disable=unused-import
    except ImportError:
        return os.environ.get("GENERATOR_MODEL_DIR", "") != "" and os.path.isdir(
            os.environ["GENERATOR_MODEL_DIR"]
        )
    return True


def infer(prompt: str, lyrics: str, model: str, params: dict) -> bytes:
    """Run the music model and return WAV bytes.

    SETUP-ONLY STUB: implement on the GPU machine. Suggested shape:

        from acestep import AceStepPipeline  # API of your installed release
        pipe = get_cached_pipeline(model, os.environ["GENERATOR_MODEL_DIR"])
        wav_bytes = pipe.generate(prompt, lyrics=lyrics, **params)

    Match the exact call signature to the ACE-Step README of the version you
    install; do not guess it here.
    """
    raise NotImplementedError(
        "ACE-Step weights are not installed on this machine. Set GENERATOR_MODEL_DIR "
        "and implement generator/server.py infer() on the GPU host (see generator/README.md)."
    )


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


@app.post("/generate")
def generate(req: GenerateRequest) -> GenerateResponse:
    version = validate_request(req)
    try:
        audio = infer(req.prompt, req.lyrics, req.model or MODEL_ID, req.params or {})
    except NotImplementedError as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
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
