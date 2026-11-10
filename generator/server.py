"""AVIBE generation backend reference server (WI-03 setup-only).

Runs on a GPU machine (e.g. your home PC) and serves the HTTP contract the Go
worker expects:

  GET  /healthz  -> {"status": "ok", "modelLoaded": bool, "model": str}
  POST /generate -> {"audioBase64": str, "mimeType": str, "durationMs": int}

Request body: {"jobId": str, "prompt": str, "lyrics": str, "model": str,
"params": object}.

Without model weights installed, /generate returns 503 with a clear message.
To go live, install ACE-Step + weights on the GPU machine and implement
infer() below following the README of the ACE-Step release you install.
"""

from __future__ import annotations

import base64
import io
import os
import wave

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

app = FastAPI(title="avibe-generator")

MODEL_ID = os.environ.get("GENERATOR_MODEL_ID", "ace-step-v1")
MAX_PROMPT_CHARS = 2000
MAX_LYRICS_CHARS = 10000


class GenerateRequest(BaseModel):
    job_id: str = Field(alias="jobId")
    prompt: str = ""
    lyrics: str = ""
    model: str = ""
    params: dict = {}


class GenerateResponse(BaseModel):
    audio_base64: str = Field(serialization_alias="audioBase64")
    mime_type: str = Field(serialization_alias="mimeType")
    duration_ms: int = Field(serialization_alias="durationMs")

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


def wav_duration_ms(raw: bytes) -> int:
    try:
        with wave.open(io.BytesIO(raw), "rb") as fh:
            frames, rate = fh.getnframes(), fh.getframerate() or 1
            return int(frames / rate * 1000)
    except (wave.Error, EOFError):
        return 0


@app.get("/healthz")
def healthz() -> dict:
    return {"status": "ok", "modelLoaded": model_available(), "model": MODEL_ID}


@app.post("/generate")
def generate(req: GenerateRequest) -> GenerateResponse:
    if not req.prompt.strip() or len(req.prompt) > MAX_PROMPT_CHARS:
        raise HTTPException(status_code=400, detail="prompt 1..2000 chars")
    if len(req.lyrics) > MAX_LYRICS_CHARS:
        raise HTTPException(status_code=400, detail="lyrics max 10000 chars")
    try:
        audio = infer(req.prompt, req.lyrics, req.model or MODEL_ID, req.params or {})
    except NotImplementedError as exc:
        raise HTTPException(status_code=503, detail=str(exc)) from exc
    return GenerateResponse(
        audioBase64=base64.b64encode(audio).decode("ascii"),
        mimeType="audio/wav",
        durationMs=wav_duration_ms(audio),
    )
