package audio

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"testing"
)

// wavChunkBin renders one RIFF chunk: id + size + payload + alignment pad.
func wavChunkBin(id string, payload []byte) []byte {
	out := new(bytes.Buffer)
	out.WriteString(id)
	_ = binary.Write(out, binary.LittleEndian, uint32(len(payload)))
	out.Write(payload)
	if len(payload)%2 == 1 {
		out.WriteByte(0)
	}
	return out.Bytes()
}

// wavFixtureBin assembles a WAV file from top-level chunks with a patched size.
func wavFixtureBin(chunks ...[]byte) []byte {
	out := new(bytes.Buffer)
	out.WriteString("RIFF")
	_ = binary.Write(out, binary.LittleEndian, uint32(0))
	out.WriteString("WAVE")
	for _, c := range chunks {
		out.Write(c)
	}
	b := out.Bytes()
	binary.LittleEndian.PutUint32(b[4:8], uint32(len(b)-8))
	return b
}

func wavInfoListBin(title string) []byte {
	field := new(bytes.Buffer)
	field.WriteString("INAM")
	_ = binary.Write(field, binary.LittleEndian, uint32(len(title)+1))
	field.WriteString(title)
	field.WriteByte(0)
	if (len(title)+1)%2 == 1 {
		field.WriteByte(0)
	}
	payload := append([]byte("INFO"), field.Bytes()...)
	return wavChunkBin("LIST", payload)
}

func fixtureFetch(t *testing.T, data []byte) FetchRange {
	t.Helper()
	return func(offset, length int64) ([]byte, error) {
		if offset < 0 || length <= 0 || offset+length > int64(len(data)) {
			return nil, fmt.Errorf("fetch [%d,%d) outside %d bytes", offset, offset+length, len(data))
		}
		return bytes.Clone(data[offset : offset+length]), nil
	}
}

func renderPlan(t *testing.T, p *DownloadPlan, data []byte) []byte {
	t.Helper()
	open := func(offset, length int64) (io.ReadCloser, error) {
		end := int64(len(data))
		if length >= 0 {
			end = offset + length
		}
		if offset < 0 || end > int64(len(data)) || end < offset {
			return nil, fmt.Errorf("range [%d,%d) outside %d bytes", offset, end, len(data))
		}
		return io.NopCloser(bytes.NewReader(data[offset:end])), nil
	}
	var buf bytes.Buffer
	if err := p.Stream(context.Background(), &buf, open); err != nil {
		t.Fatalf("Stream: %v", err)
	}
	return buf.Bytes()
}

func TestPlanWAVMatchesWholeFileInjection(t *testing.T) {
	fmtChunk := wavChunkBin("fmt ", make([]byte, 16))
	dataEven := wavChunkBin("data", bytes.Repeat([]byte{0x11}, 64))
	dataOdd := wavChunkBin("data", bytes.Repeat([]byte{0x22}, 63))
	oldInfo := wavInfoListBin("Old Title")
	cases := []struct {
		name string
		file []byte
	}{
		{"no info even data", wavFixtureBin(fmtChunk, dataEven)},
		{"no info odd data", wavFixtureBin(fmtChunk, dataOdd)},
		{"info before data", wavFixtureBin(fmtChunk, oldInfo, dataEven)},
		{"info after data", wavFixtureBin(fmtChunk, dataEven, oldInfo)},
		{"info only plus data", wavFixtureBin(oldInfo, fmtChunk, dataOdd)},
	}
	for _, tc := range cases {
		want, err := InjectWAVInfo(tc.file, "Neon", "AVIBE", "disclosure")
		if err != nil {
			t.Fatalf("%s: InjectWAVInfo: %v", tc.name, err)
		}
		plan, err := PlanWAVDownload(tc.file, int64(len(tc.file)), "Neon", "AVIBE", "disclosure", fixtureFetch(t, tc.file))
		if err != nil {
			t.Fatalf("%s: PlanWAVDownload: %v", tc.name, err)
		}
		if plan.ContentType != "audio/wav" {
			t.Errorf("%s: content type = %q", tc.name, plan.ContentType)
		}
		got := renderPlan(t, plan, tc.file)
		if !bytes.Equal(got, want) {
			t.Errorf("%s: streamed %d bytes differ from whole-file %d bytes", tc.name, len(got), len(want))
		}
		if int64(len(got)) != plan.TotalSize {
			t.Errorf("%s: TotalSize %d != rendered %d", tc.name, plan.TotalSize, len(got))
		}
		if len(got) > 0 && len(plan.Prefix) > 64<<10 {
			t.Errorf("%s: prefix %d bytes exceeds head budget", tc.name, len(plan.Prefix))
		}
	}
}

func TestPlanWAVCompletesHeadersPastHead(t *testing.T) {
	fmtChunk := wavChunkBin("fmt ", make([]byte, 16))
	data := wavChunkBin("data", bytes.Repeat([]byte{0x33}, 4096))
	file := wavFixtureBin(fmtChunk, wavInfoListBin("Old"), data)
	want, err := InjectWAVInfo(file, "T", "AVIBE", "D")
	if err != nil {
		t.Fatalf("InjectWAVInfo: %v", err)
	}
	// Only the first 20 bytes are handed over; the planner must page the
	// remaining headers through fetch.
	plan, err := PlanWAVDownload(file[:20], int64(len(file)), "T", "AVIBE", "D", fixtureFetch(t, file))
	if err != nil {
		t.Fatalf("PlanWAVDownload: %v", err)
	}
	if got := renderPlan(t, plan, file); !bytes.Equal(got, want) {
		t.Fatalf("paged plan output differs (%d vs %d bytes)", len(got), len(want))
	}
}

func TestPlanWAVRejects(t *testing.T) {
	file := wavFixtureBin(wavChunkBin("fmt ", make([]byte, 16)), wavChunkBin("data", []byte{1, 2, 3, 4}))
	fetch := fixtureFetch(t, file)
	for name, head := range map[string][]byte{
		"short":   file[:8],
		"not wav": bytes.Clone(file),
	} {
		if name == "not wav" {
			copy(head[:4], "XXXX")
		}
		if _, err := PlanWAVDownload(head, int64(len(file)), "T", "A", "D", fetch); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if _, err := PlanWAVDownload([]byte("RF64____WAVE"), 12, "T", "A", "D", fetch); err == nil {
		t.Errorf("RF64 accepted")
	}
	// Truncated chunk (declares past EOF) errors like the whole-file path.
	bad := wavFixtureBin(wavChunkBin("fmt ", make([]byte, 16)))
	binary.LittleEndian.PutUint32(bad[16:20], uint32(1<<30))
	if _, err := PlanWAVDownload(bad, int64(len(bad)), "T", "A", "D", fixtureFetch(t, bad)); err == nil {
		t.Errorf("truncated chunk accepted")
	}
}

func synchsafeBin(n int) []byte {
	return []byte{byte(n >> 21), byte(n>>14) & 0x7f, byte(n>>7) & 0x7f, byte(n) & 0x7f}
}

func id3FixtureBin(payloadLen int, footer bool, audioLen int) []byte {
	flags := byte(0)
	if footer {
		flags = 0x10
	}
	out := new(bytes.Buffer)
	out.WriteString("ID3")
	out.Write([]byte{0x04, 0x00, flags})
	out.Write(synchsafeBin(payloadLen))
	out.Write(bytes.Repeat([]byte{0x55}, payloadLen))
	if footer {
		out.WriteString("3DI")
		out.Write([]byte{0x04, 0x00, flags})
		out.Write(synchsafeBin(payloadLen))
	}
	out.Write(bytes.Repeat([]byte{0xAA}, audioLen))
	return out.Bytes()
}

func testDLMeta() *AudioMetadata {
	return &AudioMetadata{Title: "Neon", Artist: "AVIBE", Comment: "disclosure", AIProvider: "SUNO"}
}

func TestPlanMP3MatchesWholeFileInjection(t *testing.T) {
	frame := append([]byte{0xFF, 0xFB, 0x90, 0x00}, bytes.Repeat([]byte{0xBB}, 100)...)
	cases := []struct {
		name string
		file []byte
	}{
		{"no tag", frame},
		{"tag no footer", id3FixtureBin(64, false, 128)},
		{"tag with footer", id3FixtureBin(64, true, 128)},
		{"empty tag", id3FixtureBin(0, false, 64)},
	}
	for _, tc := range cases {
		want, err := InjectID3v24(tc.file, testDLMeta())
		if err != nil {
			t.Fatalf("%s: InjectID3v24: %v", tc.name, err)
		}
		plan, err := PlanMP3Download(tc.file, int64(len(tc.file)), testDLMeta())
		if err != nil {
			t.Fatalf("%s: PlanMP3Download: %v", tc.name, err)
		}
		if plan.ContentType != "audio/mpeg" {
			t.Errorf("%s: content type = %q", tc.name, plan.ContentType)
		}
		got := renderPlan(t, plan, tc.file)
		if !bytes.Equal(got, want) {
			t.Errorf("%s: streamed %d bytes differ from whole-file %d bytes", tc.name, len(got), len(want))
		}
		if int64(len(got)) != plan.TotalSize {
			t.Errorf("%s: TotalSize %d != rendered %d", tc.name, plan.TotalSize, len(got))
		}
	}
}

func TestPlanMP3Rejects(t *testing.T) {
	if _, err := PlanMP3Download([]byte("ID3\x04\x00"), 8, testDLMeta()); err == nil {
		t.Errorf("short head accepted")
	}
	// Tag declares more than the file holds.
	bad := id3FixtureBin(64, false, 16)
	if _, err := PlanMP3Download(bad, 20, testDLMeta()); err == nil {
		t.Errorf("overlong tag accepted")
	}
	if _, err := PlanMP3Download(make([]byte, 64), 64, nil); err == nil {
		t.Errorf("nil meta accepted")
	}
}

func TestPlanRawStreamsWholeObject(t *testing.T) {
	data := []byte(strings.Repeat("0123456789abcdef", 64))
	plan := PlanRawDownload(int64(len(data)))
	if plan.ContentType != "application/octet-stream" || plan.TotalSize != int64(len(data)) {
		t.Fatalf("raw plan = %+v", plan)
	}
	if got := renderPlan(t, plan, data); !bytes.Equal(got, data) {
		t.Fatalf("raw stream differs")
	}
}

func TestPlanStreamPropagatesRangeErrors(t *testing.T) {
	plan := PlanRawDownload(100)
	open := func(offset, length int64) (io.ReadCloser, error) {
		return nil, fmt.Errorf("range store down")
	}
	if err := plan.Stream(context.Background(), io.Discard, open); err == nil {
		t.Fatalf("range error swallowed")
	}
}
