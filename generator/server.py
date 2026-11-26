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
# Watermark (WI-N13): AudioSeal (code MIT, weights MIT). Model cards load
# lazily so /healthz stays fast; missing package/GPU keeps v2 at 503.
AUDIOSEAL_GEN = os.environ.get("AUDIOSEAL_GENERATOR", "audioseal_wm_16bits")
AUDIOSEAL_DET = os.environ.get("AUDIOSEAL_DETECTOR", "audioseal_detector_16bits")
AUDIOSEAL_SR = 16000
WATERMARK_PROB_MIN = 0.5  # detector presence threshold (measured 1.0/0.0 split)
# Similarity screen (WI-N13): fingerprint list JSON
# {"fingerprints": [{"id": str, "hash": hex64}]} (eval/memorization.py).
# Empty = no references yet; the check passes vacuously and the safety
# verdict records "similarity" only when references were actually compared.
REFERENCE_FPS_PATH = os.environ.get("GENERATOR_REFERENCE_FPS", "")
SIMILARITY_MAX = float(os.environ.get("GENERATOR_SIMILARITY_MAX", "0.90"))


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
    """True when an ACE-Step 1.5 checkout is configured and importable.

    Weights live in ACESTEP_ROOT/checkpoints and are downloaded by ACE-Step on
    the first generation (several minutes), so they are not required here.
    Kept as a function (not import-time) so /healthz can report honestly.
    """
    if not ACESTEP_ROOT or not os.path.isdir(os.path.join(ACESTEP_ROOT, "acestep")):
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
    global _handlers
    with _init_lock:
        if _handlers is not None:
            return _handlers
        if not model_available():
            raise NotImplementedError(
                "ACE-Step 1.5 is not installed here: set ACESTEP_ROOT to an ACE-Step 1.5 checkout "
                "and run with its venv (see generator/README.md)."
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


_watermark_models: tuple | None = None
_watermark_lock = threading.Lock()


def _get_watermark_models() -> tuple:
    """Load the AudioSeal generator + detector once (lazy)."""
    global _watermark_models
    with _watermark_lock:
        if _watermark_models is not None:
            return _watermark_models
        try:
            from audioseal import AudioSeal  # pylint: disable=import-outside-toplevel
        except ImportError as exc:
            raise NotImplementedError(
                "audioseal is not installed here: pip install audioseal "
                "(torch + the MIT model cards) on the GPU machine."
            ) from exc
        gen = AudioSeal.load_generator(AUDIOSEAL_GEN)
        det = AudioSeal.load_detector(AUDIOSEAL_DET)
        try:
            import torch  # pylint: disable=import-outside-toplevel

            if torch.cuda.is_available():
                gen = gen.cuda()
                det = det.cuda()
        except ImportError:
            pass
        _watermark_models = (gen, det)
        return _watermark_models


def _resample_mono(raw_wav: bytes, target_sr: int) -> tuple[list[float], int]:
    """Decode WAV bytes to mono float samples; resample to target_sr."""
    import struct  # pylint: disable=import-outside-toplevel

    with wave.open(io.BytesIO(raw_wav), "rb") as fh:
        n_channels, width, rate, n_frames = (
            fh.getnchannels(),
            fh.getsampwidth(),
            fh.getframerate(),
            fh.getnframes(),
        )
        raw = fh.readframes(n_frames)
    if width == 2:
        samples = struct.unpack(f"<{len(raw) // 2}h", raw)
        scale = 32768.0
    elif width == 4:
        samples = struct.unpack(f"<{len(raw) // 4}i", raw)
        scale = 2147483648.0
    else:
        raise ValueError(f"watermark needs 16/32-bit WAV, got {width * 8}-bit")
    mono = [sum(samples[i:i + n_channels]) / (n_channels * scale) for i in range(0, len(samples), n_channels)]
    if rate == target_sr or not mono:
        return mono, rate
    # Linear resample (stdlib-only; quality is fine for a watermark carrier).
    ratio = target_sr / rate
    out_len = max(int(len(mono) * ratio), 1)
    out = [mono[min(int(i / ratio), len(mono) - 1)] for i in range(out_len)]
    return out, target_sr


def _decode_wav(raw_wav: bytes) -> tuple[list[list[float]], int, int]:
    """Decode WAV bytes to per-channel float samples.

    Returns (channels, rate, width); each channel holds n_frames floats in
    [-1, 1]. Only 16/32-bit PCM is supported (same floor as _resample_mono).
    """
    import struct  # pylint: disable=import-outside-toplevel

    with wave.open(io.BytesIO(raw_wav), "rb") as fh:
        n_channels, width, rate, n_frames = (
            fh.getnchannels(),
            fh.getsampwidth(),
            fh.getframerate(),
            fh.getnframes(),
        )
        raw = fh.readframes(n_frames)
    if width == 2:
        samples = struct.unpack(f"<{len(raw) // 2}h", raw)
        scale = 32768.0
    elif width == 4:
        samples = struct.unpack(f"<{len(raw) // 4}i", raw)
        scale = 2147483648.0
    else:
        raise ValueError(f"watermark needs 16/32-bit WAV, got {width * 8}-bit")
    channels = [[v / scale for v in samples[ch::n_channels]] for ch in range(n_channels)]
    return channels, rate, width


def _resample_linear(samples: list[float], out_len: int) -> list[float]:
    """Resample to exactly out_len samples (true linear interpolation)."""
    if out_len <= 0:
        return []
    if len(samples) <= 1:
        return [samples[0] if samples else 0.0] * out_len
    if len(samples) == out_len:
        return list(samples)
    step = (len(samples) - 1) / (out_len - 1)
    out: list[float] = []
    for i in range(out_len):
        pos = i * step
        lo = int(pos)
        hi = min(lo + 1, len(samples) - 1)
        frac = pos - lo
        out.append(samples[lo] * (1.0 - frac) + samples[hi] * frac)
    return out


def _encode_wav(channels: list[list[float]], rate: int, width: int) -> bytes:
    """Encode per-channel float samples to WAV bytes (clipped to [-1, 1])."""
    import struct  # pylint: disable=import-outside-toplevel

    if width == 2:
        scale, fmt = 32767, "h"
    elif width == 4:
        scale, fmt = 2147483647, "i"
    else:
        raise ValueError(f"watermark needs 16/32-bit WAV, got {width * 8}-bit")
    n_channels = len(channels)
    n_frames = len(channels[0]) if channels else 0
    # Chunked pack: one struct.pack call per 1M samples keeps peak memory
    # bounded on long clips (180s stereo 44.1k is ~16M samples).
    pcm_parts: list[bytes] = []
    chunk: list[int] = []
    for i in range(n_frames):
        for ch in range(n_channels):
            s = channels[ch][i]
            if s > 1.0:
                s = 1.0
            elif s < -1.0:
                s = -1.0
            chunk.append(int(s * scale))
            if len(chunk) >= 1_000_000:
                pcm_parts.append(struct.pack(f"<{len(chunk)}{fmt}", *chunk))
                chunk = []
    if chunk:
        pcm_parts.append(struct.pack(f"<{len(chunk)}{fmt}", *chunk))
    buf = io.BytesIO()
    with wave.open(buf, "wb") as fh:
        fh.setnchannels(n_channels)
        fh.setsampwidth(width)
        fh.setframerate(rate)
        fh.writeframes(b"".join(pcm_parts))
    return buf.getvalue()


def insert_watermark(raw_wav: bytes, job_id: str) -> tuple[bytes, str]:
    """Insert the inaudible provenance watermark; return (wav, watermarkId).

    AudioSeal (WI-N13, code + weights MIT). Only the watermark *residual*
    is computed at 16kHz mono; it is upsampled to the clip's native rate
    and added to the original full-quality channels, so the returned WAV
    keeps the input's sample rate, channel count, and bit depth (C-FIX-1:
    no more 16kHz mono downsample). The residual is added equally to every
    channel, so the detector (which downmixes to mono) still sees the full
    mark. The message bits are random per job and the returned watermarkId
    is a server-generated `wm-<job>-<rand>` handle (message bits alone are
    not bit-perfect, so the ID is tracked out-of-band, not decoded from
    audio). Raises NotImplementedError when audioseal is missing (v2 stays
    503, fail closed).
    """
    import secrets  # pylint: disable=import-outside-toplevel

    gen, _ = _get_watermark_models()
    import torch  # pylint: disable=import-outside-toplevel

    channels, rate, width = _decode_wav(raw_wav)
    n_frames = len(channels[0]) if channels else 0
    mono, _ = _resample_mono(raw_wav, AUDIOSEAL_SR)
    audio = torch.tensor(mono, dtype=torch.float32).unsqueeze(0).unsqueeze(0)
    message = torch.randint(0, 2, (1, 16))
    device = next(gen.parameters()).device
    with torch.no_grad():
        residual = gen.get_watermark(audio.to(device), sample_rate=AUDIOSEAL_SR, message=message.to(device))
    wm_16k = residual.squeeze().detach().cpu().tolist()
    wm_native = _resample_linear(wm_16k, n_frames)
    marked = [[s + w for s, w in zip(ch, wm_native)] for ch in channels]
    watermark_id = f"wm-{job_id}-{secrets.token_hex(4)}"
    return _encode_wav(marked, rate, width), watermark_id


def detect_watermark(raw_wav: bytes) -> tuple[bool, float]:
    """Return (present, probability) for the AudioSeal mark in raw_wav."""
    _, det = _get_watermark_models()
    import torch  # pylint: disable=import-outside-toplevel

    mono, _ = _resample_mono(raw_wav, AUDIOSEAL_SR)
    audio = torch.tensor(mono, dtype=torch.float32).unsqueeze(0).unsqueeze(0)
    device = next(det.parameters()).device
    with torch.no_grad():
        result, _ = det.detect_watermark(audio.to(device), AUDIOSEAL_SR)
    prob = float(result.reshape(-1)[0].detach().cpu())
    return prob >= WATERMARK_PROB_MIN, prob


def check_similarity(raw_wav: bytes) -> tuple[bool, dict]:
    """Screen raw_wav against the reference fingerprint list.

    Returns (passed, detail). With no reference file configured, the
    check passes vacuously and detail.compared is 0 — the caller must
    NOT claim "similarity" in that case (AVIBE gate stays fail-closed
    until references exist). Otherwise passed = best < SIMILARITY_MAX.
    """
    import json  # pylint: disable=import-outside-toplevel

    if not REFERENCE_FPS_PATH:
        return True, {"compared": 0, "best": None, "note": "no reference list configured"}
    try:
        with open(REFERENCE_FPS_PATH, encoding="utf-8") as fh:
            refs = json.load(fh).get("fingerprints", [])
    except (OSError, ValueError) as exc:
        return False, {"compared": 0, "best": None, "error": str(exc)}
    try:
        channels, _, _ = _decode_wav(raw_wav)
    except (wave.Error, EOFError, ValueError) as exc:
        return False, {"compared": 0, "best": None, "error": str(exc)}
    n_channels = len(channels)
    n_frames = len(channels[0]) if channels else 0
    mono = [sum(channels[ch][i] for ch in range(n_channels)) / n_channels for i in range(n_frames)]
    blocks, bits, per = 256, 0, max(len(mono) // 256, 1)
    for b in range(blocks):
        block = mono[b * per:(b + 1) * per] or [0.0]
        if sum(block) / len(block) >= 0:
            bits |= 1 << b
    want = f"{bits:064x}"
    best = {"id": None, "similarity": 0.0}
    for fp in refs:
        try:
            sim = (blocks - (int(want, 16) ^ int(fp["hash"], 16)).bit_count()) / blocks
        except (KeyError, ValueError):
            continue
        if sim > best["similarity"]:
            best = {"id": fp.get("id"), "similarity": round(sim, 4)}
    passed = best["similarity"] < SIMILARITY_MAX
    return passed, {"compared": len(refs), "best": best, "hash": want,
                    "maxAllowed": SIMILARITY_MAX,
                    "fpNote": "cheap PCM prefilter (eval/memorization.py); spectral upgrade is the documented next step"}


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
    # Similarity screen (WI-N13): fail the job when the clip is too close
    # to a reference. "similarity" is claimed only when references were
    # actually compared (else the AVIBE gate stays fail-closed).
    sim_passed, sim_detail = check_similarity(marked)
    checks = ["prompt", "watermark"]
    if sim_detail.get("compared"):
        checks.append("similarity")
    safety = Safety(passed=bool(sim_passed), checks=checks)
    duration_ms = wav_duration_ms(marked)
    if req.delivery and req.delivery.upload_url:
        try:
            put_bytes(req.delivery.upload_url, marked, "audio/wav")
        except Exception as exc:
            raise HTTPException(status_code=502, detail=f"presigned upload failed: {exc}") from exc
        return GenerateResponse(
            contractVersion=2,
            audioUrl=req.delivery.upload_url,
            mimeType="audio/wav",
            durationMs=duration_ms,
            modelVersion=req.model_version or MODEL_VERSION,
            seed=req.seed,
            weightsHash=WEIGHTS_HASH,
            safety=safety,
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
        safety=safety,
        watermarkId=watermark_id,
    )
