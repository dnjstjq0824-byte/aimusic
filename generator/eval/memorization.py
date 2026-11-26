"""WI-G4 memorization-test skeleton (stdlib only).

Screens a candidate clip against a fingerprint list (training-set /
catalog excerpts) and reports the maximum similarity in [0,1]. The worker
graduates this into the "similarity" leg of the AVIBE gate once thresholds
are calibrated on real data (G-MODEL item 5).

Current fingerprint: block-averaged PCM sign hash over mono 16-bit WAV.
Catches exact and near-exact reuse; robust to nothing else. The spectral
fingerprint (fingerprint_spectral) is stronger: 16 time blocks x 16
log-spaced frequency bands from a Hann-windowed FFT, bits set where a
band reaches the frame mean (gain-invariant, degrades gracefully under
small edits). It stays stdlib-only (no numpy) so CI and laptops run
it; the documented GPU-PC upgrade is a Shazam-style peak-pair scheme over
a reference corpus. PCM remains the cheap prefilter.

Fingerprint list (JSON): {"fingerprints": [{"id": str, "hash": hex64}]}
Usage:
  python memorization.py --wav candidate.wav --fps fps.json [--spectral]
  python memorization.py --self-test
"""

from __future__ import annotations

import argparse
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


SPEC_BLOCKS = 16
SPEC_BANDS = 16
SPEC_FFT = 2048


def _fft(re: list[float], im: list[float]) -> None:
    """In-place iterative radix-2 FFT (len must be a power of two)."""
    n = len(re)
    j = 0
    for i in range(1, n):
        bit = n >> 1
        while j & bit:
            j ^= bit
            bit >>= 1
        j ^= bit
        if i < j:
            re[i], re[j] = re[j], re[i]
            im[i], im[j] = im[j], im[i]
    length = 2
    while length <= n:
        ang = -2.0 * math.pi / length
        wlen_re, wlen_im = math.cos(ang), math.sin(ang)
        half = length // 2
        for i in range(0, n, length):
            w_re, w_im = 1.0, 0.0
            for k in range(half):
                u_re, u_im = re[i + k], im[i + k]
                a_re, a_im = re[i + k + half], im[i + k + half]
                v_re = a_re * w_re - a_im * w_im
                v_im = a_re * w_im + a_im * w_re
                re[i + k], im[i + k] = u_re + v_re, u_im + v_im
                re[i + k + half], im[i + k + half] = u_re - v_re, u_im - v_im
                w_re, w_im = (w_re * wlen_re - w_im * wlen_im,
                              w_re * wlen_im + w_im * wlen_re)
        length <<= 1


def _band_edges(n_bands: int, n_bins: int) -> list[int]:
    """Log-spaced, strictly increasing bin edges from 1 to n_bins."""
    edges = [1]
    for i in range(1, n_bands + 1):
        edges.append(max(round(n_bins ** (i / n_bands)), edges[-1] + 1))
    edges[-1] = min(edges[-1], n_bins)
    return edges


def _frame_energies(frame: list[float], edges: list[int]) -> list[float]:
    n = SPEC_FFT
    if len(frame) >= n:
        step = len(frame) / n
        frame = [sum(frame[int(i * step):int((i + 1) * step)] or [0.0]) /
                 max(len(frame[int(i * step):int((i + 1) * step)]), 1)
                 for i in range(n)]
    else:
        frame = list(frame) + [0.0] * (n - len(frame))
    hann = [0.5 * (1.0 - math.cos(2.0 * math.pi * i / (n - 1))) for i in range(n)]
    re = [s * w for s, w in zip(frame, hann)]
    im = [0.0] * n
    _fft(re, im)
    power = [re[i] * re[i] + im[i] * im[i] for i in range(n // 2 + 1)]
    return [sum(power[edges[b]:edges[b + 1]]) for b in range(len(edges) - 1)]


def fingerprint_spectral(path: Path) -> str:
    """Spectral hash: 16 time blocks x 16 log-spaced bands.

    Bit (t, b) is set when band b carries at least the frame's mean
    energy, so uniform gain changes flip no bits while different spectra
    land far apart. Same 256-bit hex format as fingerprint_pcm.
    """
    mono, _ = read_mono_pcm16(path)
    if not mono:
        raise SystemExit(f"{path}: empty audio")
    edges = _band_edges(SPEC_BANDS, SPEC_FFT // 2)
    per = max(len(mono) // SPEC_BLOCKS, 1)
    bits = 0
    for t in range(SPEC_BLOCKS):
        block = mono[t * per:(t + 1) * per] or [0.0]
        energies = _frame_energies(block, edges)
        mean = sum(energies) / len(energies)
        for b, e in enumerate(energies):
            if e >= mean:
                bits |= 1 << (t * SPEC_BANDS + b)
    return f"{bits:064x}"


def similarity_hex(a: str, b: str) -> float:
    """Fraction of equal bits between two 256-bit hex hashes."""
    xa, xb = int(a, 16), int(b, 16)
    return (BLOCKS - (xa ^ xb).bit_count()) / BLOCKS


def scan(candidate: Path, fps_path: Path, spectral: bool = False) -> dict:
    want = fingerprint_spectral(candidate) if spectral else fingerprint_pcm(candidate)
    fps = json.loads(fps_path.read_text(encoding="utf-8")).get("fingerprints", [])
    best = {"id": None, "similarity": 0.0}
    for fp in fps:
        sim = similarity_hex(want, fp["hash"])
        if sim > best["similarity"]:
            best = {"id": fp["id"], "similarity": round(sim, 4)}
    return {"candidate": str(candidate), "hash": want, "best": best,
            "compared": len(fps), "method": "spectral" if spectral else "pcm"}


def write_tone(path: Path, freq: float, seconds: float = 1.0, rate: int = 8000,
               amp: float = 12000) -> None:
    frames = b"".join(
        struct.pack("<h", int(amp * math.sin(2 * math.pi * freq * i / rate)))
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
        hs = fingerprint_spectral(d / "a.wav")
        assert fingerprint_spectral(d / "a.wav") == hs, "spectral must be stable"
        assert similarity_hex(hs, hs) == 1.0
        hb_s = fingerprint_spectral(d / "b.wav")
        assert similarity_hex(hs, hb_s) < 0.95, "different spectra must separate"
        write_tone(d / "half.wav", 440.0, amp=6000)
        assert similarity_hex(hs, fingerprint_spectral(d / "half.wav")) == 1.0, \
            "uniform gain must flip no bits"
        fps_s = d / "fps_spectral.json"
        fps_s.write_text(json.dumps({"fingerprints": [
            {"id": "train-a", "hash": hs},
            {"id": "train-b", "hash": hb_s},
        ]}), encoding="utf-8")
        res_s = scan(d / "a.wav", fps_s, spectral=True)
        assert res_s["best"]["id"] == "train-a" and res_s["best"]["similarity"] == 1.0, res_s
        assert res_s["method"] == "spectral"
        print("memorization spectral self-test: PASS (stable, gain-invariant, separates)")


def main(argv: list[str]) -> None:
    ap = argparse.ArgumentParser(description="WI-G4 memorization test")
    ap.add_argument("--wav", type=Path, help="candidate 16-bit WAV")
    ap.add_argument("--fps", type=Path, help="fingerprint list JSON")
    ap.add_argument("--spectral", action="store_true",
                    help="use the spectral hash (fps must hold spectral hashes)")
    ap.add_argument("--self-test", action="store_true")
    args = ap.parse_args(argv)
    if args.self_test:
        self_test()
        return
    if not args.wav or not args.fps:
        ap.error("--wav and --fps are required (or --self-test)")
    print(json.dumps(scan(args.wav, args.fps, spectral=args.spectral), indent=2))


if __name__ == "__main__":
    main(sys.argv[1:])
