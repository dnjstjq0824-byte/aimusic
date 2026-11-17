"""WI-G4 automatic-metric runner skeleton (stdlib only).

Runs prompt-alignment / audio-distance style metrics over a clip list and
writes a JSON report the model card can cite (G-MODEL items 3-5).

Backends:
  synthetic  deterministic pseudo-metrics from clip metadata. Proves harness
             wiring offline. NEVER use for a G-MODEL decision (labeled).
  fad/clap   real metric hooks for the GPU PC. Fail with install instructions
             when torch/weights are missing; implement against the release
             README of the metric package you install.

Clip list (JSON): {"clips": [{"id": str, "prompt": str, "audioPath": str}]}
Report (JSON): {"runId", "modelVersion", "backend", "metrics": {...},
                "clips": [{"id", "prompt", "scores": {...}}]}

Usage:
  python run_metrics.py --clips clips.json --out report.json --backend synthetic
  python run_metrics.py --self-test
"""

from __future__ import annotations

import argparse
import hashlib
import json
import sys
import tempfile
from datetime import datetime, timezone
from pathlib import Path


def pseudo_score(*parts: str) -> float:
    """Deterministic [0,1) score from strings (synthetic backend only)."""
    digest = hashlib.sha256("|".join(parts).encode("utf-8")).digest()
    return int.from_bytes(digest[:8], "big") / 2**64


def run_synthetic(clips: list[dict], model_version: str) -> dict:
    scored = []
    for clip in clips:
        cid, prompt = clip["id"], clip.get("prompt", "")
        scored.append({
            "id": cid,
            "prompt": prompt,
            "scores": {
                "promptAlignment": round(pseudo_score(model_version, cid, prompt), 4),
                "audioDistance": round(pseudo_score(model_version, cid, "fad") * 50, 4),
            },
        })
    n = max(len(scored), 1)
    return {
        "clips": scored,
        "metrics": {
            "meanPromptAlignment": round(sum(c["scores"]["promptAlignment"] for c in scored) / n, 4),
            "meanAudioDistance": round(sum(c["scores"]["audioDistance"] for c in scored) / n, 4),
            "n": len(scored),
        },
    }


def need_gpu_backend(name: str) -> None:
    raise SystemExit(
        f"backend {name!r} needs the GPU PC stack (torch + metric weights); "
        "install per that package's release README and re-run with --backend "
        f"{name}. This machine ran nothing."
    )


def run(clips_path: Path, out_path: Path, backend: str, model_version: str) -> dict:
    clips_doc = json.loads(clips_path.read_text(encoding="utf-8"))
    clips = clips_doc.get("clips", [])
    if not clips or any("id" not in c for c in clips):
        raise SystemExit("clips file must hold {\"clips\": [{\"id\", ...}]}")
    if backend == "synthetic":
        body = run_synthetic(clips, model_version)
    elif backend in ("fad", "clap"):
        need_gpu_backend(backend)
    else:
        raise SystemExit(f"unknown backend {backend!r} (want synthetic|fad|clap)")
    report = {
        "runId": hashlib.sha256(f"{model_version}|{len(clips)}".encode()).hexdigest()[:16],
        "modelVersion": model_version,
        "backend": backend,
        "syntheticWarning": backend == "synthetic",
        "createdAt": datetime.now(timezone.utc).isoformat(),
        **body,
    }
    out_path.write_text(json.dumps(report, indent=2, ensure_ascii=False), encoding="utf-8")
    return report


def self_test() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        clips = Path(tmp) / "clips.json"
        out = Path(tmp) / "report.json"
        clips.write_text(json.dumps({"clips": [
            {"id": "c1", "prompt": "calm piano", "audioPath": "c1.wav"},
            {"id": "c2", "prompt": "lofi drums", "audioPath": "c2.wav"},
        ]}), encoding="utf-8")
        first = run(clips, out, "synthetic", "test-v1")
        second = run(clips, out, "synthetic", "test-v1")
        first.pop("createdAt")
        second.pop("createdAt")
        assert first == second, "synthetic backend must be deterministic"
        assert first["metrics"]["n"] == 2, first["metrics"]
        assert first["syntheticWarning"] is True
        assert 0 <= first["metrics"]["meanPromptAlignment"] < 1
        print("run_metrics self-test: PASS (deterministic, n=2)")


def main(argv: list[str]) -> None:
    ap = argparse.ArgumentParser(description="WI-G4 automatic-metric runner")
    ap.add_argument("--clips", type=Path, help="clip list JSON")
    ap.add_argument("--out", type=Path, help="report JSON to write")
    ap.add_argument("--backend", default="synthetic", help="synthetic|fad|clap")
    ap.add_argument("--model-version", default="", help="registry version under test")
    ap.add_argument("--self-test", action="store_true")
    args = ap.parse_args(argv)
    if args.self_test:
        self_test()
        return
    if not args.clips or not args.out:
        ap.error("--clips and --out are required (or --self-test)")
    report = run(args.clips, args.out, args.backend, args.model_version)
    print(json.dumps(report["metrics"], indent=2))


if __name__ == "__main__":
    main(sys.argv[1:])
