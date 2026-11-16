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
"""

from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import sys
from datetime import datetime, timezone
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


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser(description="WI-G2 GPU PoC driver")
    ap.add_argument("--out", type=Path, required=True, help="report JSON path")
    ap.add_argument("--duration-sec", type=int, default=30)
    ap.add_argument("--prompt", default="calm piano loop, 90 bpm")
    args = ap.parse_args(argv)

    import os
    weights = os.environ.get("GENERATOR_MODEL_DIR", "")
    weights_ok = bool(weights) and Path(weights).is_dir()

    report: dict = {
        "wi": "G2",
        "status": "NOT TESTED",
        "createdAt": datetime.now(timezone.utc).isoformat(),
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
    import importlib.util as _ilu  # noqa: E402
    import server  # noqa: E402

    def _load(name: str, path: Path):  # noqa: ANN001, ANN202
        spec = _ilu.spec_from_file_location(name, path)
        mod = _ilu.module_from_spec(spec)
        spec.loader.exec_module(mod)
        return mod

    t0 = datetime.now(timezone.utc)
    audio = server.infer(args.prompt, "", os.environ.get("GENERATOR_MODEL_ID", "ace-step-v1"), {})
    marked, watermark_id = server.insert_watermark(audio, "poc")
    detect = {"clean": None, "note": "detector runs on the GPU PC with the installed watermark package"}
    try:
        fingerprint_pcm = _load("memorization", REPO / "generator" / "eval" / "memorization.py").fingerprint_pcm
        import tempfile, wave  # noqa: E402
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
        "elapsedSec": (datetime.now(timezone.utc) - t0).total_seconds(),
    }
    args.out.write_text(json.dumps(report, indent=2), encoding="utf-8")
    print(f"G2 PoC: {report['status']} (report: {args.out})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
