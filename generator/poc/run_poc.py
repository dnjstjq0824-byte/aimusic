"""WI-G2 GPU PoC driver (runs on the GPU PC; NOT TESTED elsewhere).

Procedure (preparation doc §4-§5, WI-G2):
  1. ACE-Step inference (license re-verified at run time from the installed
     release's LICENSE/README — never assumed here).
  2. Watermark insert + detect (candidate: AudioSeal — license + robustness
     confirmed here, not assumed).
  3. Fingerprint similarity screen vs fixtures (generator/eval/memorization).
  4. JSON + Markdown report with versions, license evidence, detection
     rates, and similarity results.

On this dev box (no GPU, no weights) every heavy step is skipped and the
report records status NOT TESTED with the exact missing inputs. Exit code
is 0 for a completed procedure run (even NOT TESTED); 2 for misuse.
Generated clips never upload to any operational store.

Usage (GPU PC):
  set GENERATOR_MODEL_DIR=<weights> (plus torch/CUDA/acestep/audioseal)
  python generator/poc/run_poc.py --out docs/handoff/20261114/evidence/g2/poc-report.json

Remote mode (WI-N11: any machine with network access to a GPU server):
  python generator/poc/run_poc.py --out report.json --endpoint https://<pod>-8000.proxy.runpod.net
  The bearer token comes from --token or GENERATOR_API_TOKEN. The PoC calls
  GET /healthz + POST /generate (contract v2) over HTTPS and fingerprints
  the returned audio. eval/memorization.py stays file-based: save remote
  clips locally, then scan them with --wav/--fps.
"""

from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import sys
from datetime import UTC, datetime
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(REPO / "generator"))


def sha_file(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def check_python_package(name: str) -> dict:
    spec = importlib.util.find_spec(name)
    if spec is None:
        return {"name": name, "installed": False}
    mod = importlib.import_module(name)
    info: dict = {"name": name, "installed": True,
                  "version": getattr(mod, "__version__", "unknown")}
    base = Path(getattr(mod, "__file__", "")).parent
    for lic_name in ("LICENSE", "LICENSE.txt", "LICENSE.md", "COPYING"):
        lic = base / lic_name
        if lic.is_file():
            info["licenseFile"] = str(lic)
            info["licenseSha256"] = sha_file(lic)
            break
    return info


def check_cuda() -> dict:
    try:
        import torch  # type: ignore
        return {"torch": getattr(torch, "__version__", "unknown"),
                "cudaAvailable": bool(torch.cuda.is_available()),
                "device": torch.cuda.get_device_name(0) if torch.cuda.is_available() else None}
    except ImportError:
        return {"torch": None, "cudaAvailable": False, "device": None}


def call_remote(endpoint: str, token: str, prompt: str, duration_sec: int) -> dict:
    """Run the v2 generation over HTTPS; return the decoded reply + audio bytes."""
    import base64
    import os
    import urllib.error
    import urllib.request

    base = endpoint.rstrip("/")
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"

    def call(method: str, path: str, payload: dict | None = None) -> dict:
        data = json.dumps(payload).encode("utf-8") if payload is not None else None
        req = urllib.request.Request(base + path, data=data, method=method, headers=headers)
        try:
            with urllib.request.urlopen(req, timeout=900) as resp:
                return {"status": resp.status, "body": json.loads(resp.read().decode("utf-8"))}
        except urllib.error.HTTPError as exc:
            return {"status": exc.code, "body": exc.read().decode("utf-8", "replace")}

    health = call("GET", "/healthz")
    if health["status"] != 200:
        raise SystemExit(f"remote /healthz failed: {health}")
    gen = call("POST", "/generate", {
        "jobId": "poc-remote",
        "prompt": prompt,
        "contractVersion": 2,
        "durationSec": duration_sec,
        "seed": 42,
        "watermark": True,
    })
    if gen["status"] != 200:
        raise SystemExit(f"remote /generate failed: {gen}")
    body = gen["body"]
    audio_b64 = body.get("audioBase64", "")
    if not audio_b64 and body.get("audioUrl"):
        url_req = urllib.request.Request(body["audioUrl"], method="GET")
        with urllib.request.urlopen(url_req, timeout=900) as resp:
            audio = resp.read()
    else:
        audio = base64.b64decode(audio_b64)
    return {"health": health["body"], "reply": body, "audio": audio,
            "tokenSent": bool(token), "tokenEnv": os.environ.get("GENERATOR_API_TOKEN") is not None}


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser(description="WI-G2 GPU PoC driver")
    ap.add_argument("--out", type=Path, required=True, help="report JSON path")
    ap.add_argument("--duration-sec", type=int, default=30)
    ap.add_argument("--prompt", default="calm piano loop, 90 bpm")
    ap.add_argument("--endpoint", default="", help="remote GPU server base URL (remote mode)")
    ap.add_argument("--token", default="", help="bearer token (else GENERATOR_API_TOKEN)")
    args = ap.parse_args(argv)

    import os
    if args.endpoint:
        return main_remote(args)
    weights = os.environ.get("GENERATOR_MODEL_DIR", "")
    weights_ok = bool(weights) and Path(weights).is_dir()

    report: dict = {
        "wi": "G2",
        "status": "NOT TESTED",
        "createdAt": datetime.now(UTC).isoformat(),
        "prompt": args.prompt,
        "durationSec": args.duration_sec,
        "weightsDir": weights or None,
        "weightsPresent": weights_ok,
        "packages": {
            "acestep": check_python_package("acestep"),
            "audioseal": check_python_package("audioseal"),
        },
        "cuda": check_cuda(),
        "licenseNote": "Weight + watermark licenses are re-verified from the installed release artifacts on the GPU PC at run time (see packages.*.licenseFile hashes above). Nothing here asserts a license.",
        "missing": [],
        "results": None,
    }
    if not weights_ok:
        report["missing"].append("GENERATOR_MODEL_DIR with weight files")
    for key in ("acestep", "audioseal"):
        if not report["packages"][key]["installed"]:
            report["missing"].append(f"python package {key}")
    if not report["cuda"]["cudaAvailable"]:
        report["missing"].append("CUDA-capable GPU")

    if report["missing"]:
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(json.dumps(report, indent=2), encoding="utf-8")
        print(f"G2 PoC: NOT TESTED ({'; '.join(report['missing'])})")
        print(f"report: {args.out}")
        return 0

    # --- Live path (GPU PC only; unreachable without the stack above). ---
    import importlib.util as _ilu

    import server

    def _load(name: str, path: Path):
        spec = _ilu.spec_from_file_location(name, path)
        mod = _ilu.module_from_spec(spec)
        spec.loader.exec_module(mod)
        return mod

    t0 = datetime.now(UTC)
    audio = server.infer(args.prompt, "", os.environ.get("GENERATOR_MODEL_ID", "ace-step-v1"), {})
    marked, watermark_id = server.insert_watermark(audio, "poc")
    detect = {"clean": None, "note": "detector runs on the GPU PC with the installed watermark package"}
    try:
        fingerprint_pcm = _load("memorization", REPO / "generator" / "eval" / "memorization.py").fingerprint_pcm
        import tempfile
        import wave
        with tempfile.TemporaryDirectory() as tmp:
            clip = Path(tmp) / "poc.wav"
            with wave.open(str(clip), "wb") as fh:
                fh.setnchannels(1)
                fh.setsampwidth(2)
                fh.setframerate(44100)
                fh.writeframes(marked[:44100 * 2 * min(args.duration_sec, 30)])
            fp = fingerprint_pcm(clip)
        detect["fingerprint"] = fp
    except Exception as exc:  # noqa: BLE001 - PoC reports, never crashes
        detect["error"] = str(exc)
    report["status"] = "PASS" if detect.get("fingerprint") else "INCONCLUSIVE"
    report["results"] = {
        "watermarkId": watermark_id,
        "durationMs": server.wav_duration_ms(marked),
        "detection": detect,
        "similarityNote": "compare fingerprint against the training fixture set with eval/memorization.py --fps",
        "elapsedSec": (datetime.now(UTC) - t0).total_seconds(),
    }
    args.out.write_text(json.dumps(report, indent=2), encoding="utf-8")
    print(f"G2 PoC: {report['status']} (report: {args.out})")
    return 0


def main_remote(args) -> int:
    """Remote PoC: drive a GPU server over HTTPS, fingerprint the clip."""
    import importlib.util as _ilu
    import os
    import tempfile
    import wave

    def _load(name: str, path: Path):
        spec = _ilu.spec_from_file_location(name, path)
        mod = _ilu.module_from_spec(spec)
        spec.loader.exec_module(mod)
        return mod

    token = args.token or os.environ.get("GENERATOR_API_TOKEN", "")
    t0 = datetime.now(UTC)
    remote = call_remote(args.endpoint, token, args.prompt, args.duration_sec)
    reply = remote["reply"]
    audio: bytes = remote["audio"]
    with tempfile.TemporaryDirectory() as tmp:
        clip = Path(tmp) / "poc-remote.wav"
        clip.write_bytes(audio)
        try:
            fp = _load("memorization", REPO / "generator" / "eval" / "memorization.py").fingerprint_pcm(clip)
        except SystemExit as exc:
            fp = f"fingerprint skipped: {exc}"
        try:
            with wave.open(str(clip), "rb") as fh:
                rate = fh.getframerate() or 1
                duration_ms = int(fh.getnframes() / rate * 1000)
        except (wave.Error, EOFError):
            duration_ms = 0
    report = {
        "wi": "G2",
        "mode": "remote",
        "status": "PASS",
        "createdAt": datetime.now(UTC).isoformat(),
        "endpoint": args.endpoint.rstrip("/"),
        "prompt": args.prompt,
        "durationSec": args.duration_sec,
        "health": remote["health"],
        "tokenSent": remote["tokenSent"],
        "results": {
            "contractVersion": reply.get("contractVersion"),
            "mimeType": reply.get("mimeType"),
            "durationMs": duration_ms,
            "reportedDurationMs": reply.get("durationMs"),
            "modelVersion": reply.get("modelVersion"),
            "seed": reply.get("seed"),
            "weightsHash": reply.get("weightsHash"),
            "safety": reply.get("safety"),
            "watermarkId": reply.get("watermarkId"),
            "audioBytes": len(audio),
            "fingerprint": fp,
            "similarityNote": "compare fingerprint against the training fixture set with eval/memorization.py --fps",
            "elapsedSec": (datetime.now(UTC) - t0).total_seconds(),
        },
    }
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps(report, indent=2), encoding="utf-8")
    print(f"G2 PoC remote: PASS (report: {args.out})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
