package audio

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// P2-1 (#22): downloads of WAV masters carry the AI disclosure inside a
// LIST-INFO chunk (INAM/ICMT), keeping the file playable.

func buildTestWAV(t *testing.T, samples []int16) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(0)) // patched below
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1)) // PCM
	_ = binary.Write(buf, binary.LittleEndian, uint16(1)) // mono
	_ = binary.Write(buf, binary.LittleEndian, uint32(8000))
	_ = binary.Write(buf, binary.LittleEndian, uint32(16000))
	_ = binary.Write(buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(len(samples)*2))
	for _, s := range samples {
		_ = binary.Write(buf, binary.LittleEndian, s)
	}
	raw := buf.Bytes()
	binary.LittleEndian.PutUint32(raw[4:8], uint32(len(raw)-8))
	return raw
}

func TestInjectWAVInfoRoundTrip(t *testing.T) {
	raw := buildTestWAV(t, []int16{0, 1000, -1000, 32767})
	tagged, err := InjectWAVInfo(raw, "Neon Test", "AVIBE", "AI-generated music. Verify: https://avibemusic.com")
	if err != nil {
		t.Fatalf("InjectWAVInfo: %v", err)
	}
	info, err := ParseWAVInfo(tagged)
	if err != nil {
		t.Fatalf("ParseWAVInfo: %v", err)
	}
	if info["INAM"] != "Neon Test" || info["IARL"] != "AVIBE" {
		t.Fatalf("info = %v", info)
	}
	if got := info["ICMT"]; got == "" || len(got) > 256 {
		t.Fatalf("ICMT = %q", got)
	}
	// Audio payload survives byte-identical.
	head := tagged[:12]
	if string(head[:4]) != "RIFF" || string(head[8:12]) != "WAVE" {
		t.Fatalf("RIFF header broken: %q", head)
	}
	if int(binary.LittleEndian.Uint32(tagged[4:8])) != len(tagged)-8 {
		t.Fatalf("RIFF size mismatch: %d vs %d", binary.LittleEndian.Uint32(tagged[4:8]), len(tagged)-8)
	}
	for _, chunk := range []string{"fmt ", "data"} {
		if !bytes.Contains(tagged, []byte(chunk)) {
			t.Fatalf("chunk %q lost", chunk)
		}
	}
	// Re-injection replaces the INFO chunk instead of stacking it.
	twice, err := InjectWAVInfo(tagged, "Second", "AVIBE", "again")
	if err != nil {
		t.Fatalf("second InjectWAVInfo: %v", err)
	}
	if n := bytes.Count(twice, []byte("LIST")); n != 1 {
		t.Fatalf("LIST chunks = %d, want 1", n)
	}
	info2, err := ParseWAVInfo(twice)
	if err != nil || info2["INAM"] != "Second" {
		t.Fatalf("re-inject info = %v err = %v", info2, err)
	}
}

func TestInjectWAVInfoRejects(t *testing.T) {
	if _, err := InjectWAVInfo([]byte("too short"), "t", "a", "c"); err == nil {
		t.Fatalf("short input must fail")
	}
	if _, err := InjectWAVInfo(bytes.Repeat([]byte("x"), 64), "t", "a", "c"); err == nil {
		t.Fatalf("non-RIFF input must fail")
	}
	rf64 := buildTestWAV(t, []int16{1})
	copy(rf64[:4], "RF64")
	if _, err := InjectWAVInfo(rf64, "t", "a", "c"); err == nil {
		t.Fatalf("RF64 must fail (out of scope)")
	}
}
