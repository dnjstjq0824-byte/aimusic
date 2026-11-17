"""WI-G4 memorization-test skeleton (stdlib only).

Screens a candidate clip against a fingerprint list (training-set /
catalog excerpts) and reports the maximum similarity in [0,1]. The worker
graduates this into the "similarity" leg of the AVIBE gate once thresholds
are calibrated on real data (G-MODEL item 5).

Current fingerprint: block-averaged PCM sign hash over mono 16-bit WAV.
Catches exact and near-exact reuse; robust to nothing else. A spectral
fingerprint (numpy + a peak-hash scheme) is the documented GPU-PC upgrade
point — implement it in fingerprint_spectral() and keep this one as the
cheap prefilter.

Fingerprint list (JSON): {"fingerprints": [{"id": str, "hash": hex64}]}
Usage:
  python memorization.py --wav candidate.wav --fps fps.json
  python memorization.py --self-test
"""

from __future__ import annotations

import argparse
import hashlib
import json
import math
import struct
import sys
import tempfile
import wave
from pathlib import Path

BLOCKS = 256


def read_mono_pcm16(path: Path) -> tuple[list[float], int]:
    with wave.open(str(path), "rb") as fh:
        if fh.getsampwidth() != 2:
            raise SystemExit(f"{path}: want 16-bit WAV, got {fh.getsampwidth() * 8}-bit")
        n_channels, rate, n_frames = fh.getnchannels(), fh.getframerate(), fh.getnframes()
        raw = fh.readframes(n_frames)
    samples = struct.unpack(f"<{len(raw) // 2}h", raw)
    mono: list[float] = []
    for i in range(0, len(samples), n_channels):
        mono.append(sum(samples[i:i + n_channels]) / (n_channels * 32768.0))
    return mono, rate


def fingerprint_pcm(path: Path) -> str:
    """Cheap prefilter hash: sign of block means -> 256 bits -> hex."""
    mono, _ = read_mono_pcm16(path)
    if not mono:
        raise SystemExit(f"{path}: empty audio")
    per = max(len(mono) // BLOCKS, 1)
    bits = 0
    for b in range(BLOCKS):
        block = mono[b * per:(b + 1) * per] or [0.0]
        if sum(block) / len(block) >= 0:
            bits |= 1 << b
    return f"{bits:064x}"


def fingerprint_spectral(path: Path) -> str:  # noqa: ARG001
    """GPU-PC upgrade point: robust spectral peak hash (not implemented)."""
    raise SystemExit(
        "spectral fingerprint needs the GPU PC stack (numpy + peak-hash code); "
        "implement fingerprint_spectral() there. This machine ran nothing."
    )


def similarity_hex(a: str, b: str) -> float:
    """Fraction of equal bits between two 256-bit hex hashes."""
    xa, xb = int(a, 16), int(b, 16)
    return (BLOCKS - bin(xa ^ xb).count("1")) / BLOCKS


def scan(candidate: Path, fps_path: Path) -> dict:
    want = fingerprint_pcm(candidate)
    fps = json.loads(fps_path.read_text(encoding="utf-8")).get("fingerprints", [])
    best = {"id": None, "similarity": 0.0}
    for fp in fps:
        sim = similarity_hex(want, fp["hash"])
        if sim > best["similarity"]:
            best = {"id": fp["id"], "similarity": round(sim, 4)}
    return {"candidate": str(candidate), "hash": want, "best": best, "compared": len(fps)}


def write_tone(path: Path, freq: float, seconds: float = 1.0, rate: int = 8000) -> None:
    frames = b"".join(
        struct.pack("<h", int(12000 * math.sin(2 * math.pi * freq * i / rate)))
        for i in range(int(seconds * rate))
    )
    with wave.open(str(path), "wb") as fh:
        fh.setnchannels(1)
        fh.setsampwidth(2)
        fh.setframerate(rate)
        fh.writeframes(frames)


def self_test() -> None:
    with tempfile.TemporaryDirectory() as tmp:
        d = Path(tmp)
        write_tone(d / "a.wav", 440.0)
        write_tone(d / "b.wav", 880.0)
        ha = fingerprint_pcm(d / "a.wav")
        assert fingerprint_pcm(d / "a.wav") == ha, "fingerprint must be stable"
        assert similarity_hex(ha, ha) == 1.0
        hb = fingerprint_pcm(d / "b.wav")
        assert similarity_hex(ha, hb) < 1.0, "different tones must differ"
        fps = d / "fps.json"
        fps.write_text(json.dumps({"fingerprints": [
            {"id": "train-a", "hash": ha},
            {"id": "train-b", "hash": hb},
        ]}), encoding="utf-8")
        res = scan(d / "a.wav", fps)
        assert res["best"]["id"] == "train-a" and res["best"]["similarity"] == 1.0, res
        assert res["compared"] == 2
        print("memorization self-test: PASS (stable, exact-match 1.0, compared=2)")


def main(argv: list[str]) -> None:
    ap = argparse.ArgumentParser(description="WI-G4 memorization test")
    ap.add_argument("--wav", type=Path, help="candidate 16-bit WAV")
    ap.add_argument("--fps", type=Path, help="fingerprint list JSON")
    ap.add_argument("--self-test", action="store_true")
    args = ap.parse_args(argv)
    if args.self_test:
        self_test()
        return
    if not args.wav or not args.fps:
        ap.error("--wav and --fps are required (or --self-test)")
    print(json.dumps(scan(args.wav, args.fps), indent=2))


if __name__ == "__main__":
    main(sys.argv[1:])
