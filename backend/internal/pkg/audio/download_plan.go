package audio

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
)

// Download streaming plans (WI-D4 B-07): tag only the head of a master and
// stream the remaining bytes. Plans render byte-identical output to the
// whole-file InjectWAVInfo/InjectID3v24 paths while holding ~KBs in memory
// instead of the full master.

// ByteRange is one object span streamed in order. Length < 0 reads to the
// end of the object.
type ByteRange struct {
	Offset int64
	Length int64
}

// DownloadPlan is an ordered download: Prefix bytes, streamed Ranges, Suffix
// bytes. TotalSize is the exact Content-Length.
type DownloadPlan struct {
	ContentType string
	Prefix      []byte
	Ranges      []ByteRange
	Suffix      []byte
	TotalSize   int64
}

// Stream writes the plan to w, opening each range through open. Ranges stay
// streaming: at most one copy buffer is live at a time.
func (p *DownloadPlan) Stream(ctx context.Context, w io.Writer, open func(offset, length int64) (io.ReadCloser, error)) error {
	if len(p.Prefix) > 0 {
		if _, err := w.Write(p.Prefix); err != nil {
			return err
		}
	}
	for _, r := range p.Ranges {
		if err := ctx.Err(); err != nil {
			return err
		}
		rc, err := open(r.Offset, r.Length)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(w, rc)
		closeErr := rc.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if len(p.Suffix) > 0 {
		if _, err := w.Write(p.Suffix); err != nil {
			return err
		}
	}
	return nil
}

// FetchRange reads object bytes for header completion past the head window.
type FetchRange func(offset, length int64) ([]byte, error)

// maxWAVPlannedChunks bounds header walks on hostile files; real masters
// carry dozens of chunks at most.
const maxWAVPlannedChunks = 4096

// buildWAVInfoList renders the disclosure LIST-INFO chunk, byte-identical to
// the chunk InjectWAVInfo appends (INAM title, IARL archivist, ICMT comment).
func buildWAVInfoList(title, artist, comment string) []byte {
	body := new(bytes.Buffer)
	body.WriteString("INFO")
	body.Write(wavInfoField("INAM", title))
	body.Write(wavInfoField("IARL", artist))
	body.Write(wavInfoField("ICMT", comment))
	payload := body.Bytes()
	out := new(bytes.Buffer)
	out.WriteString("LIST")
	_ = binary.Write(out, binary.LittleEndian, uint32(len(payload)))
	out.Write(payload)
	return out.Bytes()
}

// wavHeaderCache pages chunk headers, starting from the head window and
// extending through fetch on demand.
type wavHeaderCache struct {
	buf   []byte
	total int64
	fetch FetchRange
}

func (c *wavHeaderCache) bytesAt(off int64, n int64) ([]byte, error) {
	if off < 0 || n <= 0 || off+n > c.total {
		return nil, fmt.Errorf("%w: header [%d,%d) of %d", ErrWAVTruncated, off, off+n, c.total)
	}
	if off+n > int64(len(c.buf)) {
		got, err := c.fetch(off, n)
		if err != nil {
			return nil, err
		}
		if int64(len(got)) < n {
			return nil, fmt.Errorf("%w: short header fetch", ErrWAVTruncated)
		}
		// Splice the fetched header into the cache (headers never overlap
		// payload the planner keeps, so extension order is irrelevant).
		if off == int64(len(c.buf)) {
			c.buf = append(c.buf, got...)
		} else {
			need := make([]byte, off+n)
			copy(need, c.buf)
			copy(need[off:], got)
			c.buf = need
		}
	}
	return c.buf[off : off+n], nil
}

// PlanWAVDownload plans a tagged WAV download: a patched 12-byte RIFF header,
// streamed kept spans (every old LIST-INFO removed, wherever it sits), and
// the disclosure LIST-INFO suffix. Output is byte-identical to
// InjectWAVInfo over the whole file.
func PlanWAVDownload(head []byte, totalSize int64, title, artist, comment string, fetch FetchRange) (*DownloadPlan, error) {
	if len(head) < 12 || totalSize < 12 {
		return nil, ErrNotWAV
	}
	if string(head[:4]) == "RF64" {
		return nil, ErrRF64NoInfo
	}
	if string(head[:4]) != "RIFF" || string(head[8:12]) != "WAVE" {
		return nil, ErrNotWAV
	}
	cache := &wavHeaderCache{buf: bytes.Clone(head), total: totalSize, fetch: fetch}
	type span struct{ off, total int64 }
	var removed []span
	var off int64 = 12
	chunks := 0
	for off+8 <= totalSize {
		chunks++
		if chunks > maxWAVPlannedChunks {
			return nil, fmt.Errorf("%w: too many chunks", ErrWAVTruncated)
		}
		hdr, err := cache.bytesAt(off, 12)
		if err != nil {
			// A chunk id+size (8B) may remain without the 4 LIST-type bytes;
			// only LIST chunks need the type probe.
			hdr8, err8 := cache.bytesAt(off, 8)
			if err8 != nil {
				return nil, err8
			}
			if string(hdr8[:4]) == "LIST" {
				return nil, err
			}
			hdr = hdr8
		}
		size := int64(binary.LittleEndian.Uint32(hdr[4:8]))
		total := 8 + size
		if size%2 == 1 {
			total++
		}
		if off+total > totalSize {
			return nil, ErrWAVTruncated
		}
		if string(hdr[:4]) == "LIST" && len(hdr) >= 12 && string(hdr[8:12]) == "INFO" {
			removed = append(removed, span{off: off, total: total})
		}
		off += total
	}
	walkEnd := off
	// Kept spans: [12, walkEnd) minus removed INFO spans, in order. Trailing
	// bytes past walkEnd (< 8, no full chunk) are dropped exactly like the
	// whole-file path, which stops its walk at the same bound.
	var ranges []ByteRange
	cur := int64(12)
	kept := int64(0)
	for _, r := range removed {
		if r.off > cur {
			ranges = append(ranges, ByteRange{Offset: cur, Length: r.off - cur})
			kept += r.off - cur
		}
		cur = r.off + r.total
	}
	if walkEnd > cur {
		ranges = append(ranges, ByteRange{Offset: cur, Length: walkEnd - cur})
		kept += walkEnd - cur
	}
	list := buildWAVInfoList(title, artist, comment)
	suffix := list
	if (kept+int64(len(list)))%2 == 1 {
		suffix = append(bytes.Clone(list), 0)
	}
	newTotal := int64(12) + kept + int64(len(suffix))
	prefix := bytes.Clone(head[:12])
	binary.LittleEndian.PutUint32(prefix[4:8], uint32(newTotal-8))
	return &DownloadPlan{
		ContentType: "audio/wav",
		Prefix:      prefix,
		Ranges:      ranges,
		Suffix:      suffix,
		TotalSize:   newTotal,
	}, nil
}

// PlanMP3Download plans a tagged MP3 download: a fresh ID3v2.4 head plus the
// audio stream past any existing leading tag (footer-aware, mirroring
// StripID3v24). Output is byte-identical to InjectID3v24.
func PlanMP3Download(head []byte, totalSize int64, meta *AudioMetadata) (*DownloadPlan, error) {
	if len(head) < 10 {
		return nil, fmt.Errorf("%w: head too short", ErrCorruptID3Header)
	}
	tag, err := BuildID3v24Tag(meta)
	if err != nil {
		return nil, err
	}
	audioOff := int64(0)
	if bytes.Equal(head[:3], ID3v2Identifier) {
		var synch [4]byte
		copy(synch[:], head[6:10])
		payload, err := DecodeSynchsafe(synch)
		if err != nil {
			return nil, fmt.Errorf("decoding tag size: %w", err)
		}
		audioOff = 10 + int64(payload)
		if head[5]&0x10 != 0 {
			audioOff += 10
		}
		if totalSize < audioOff {
			return nil, io.ErrUnexpectedEOF
		}
	}
	return &DownloadPlan{
		ContentType: "audio/mpeg",
		Prefix:      tag,
		Ranges:      []ByteRange{{Offset: audioOff, Length: -1}},
		TotalSize:   totalSize - audioOff + int64(len(tag)),
	}, nil
}

// PlanRawDownload streams an unknown-format master untouched.
func PlanRawDownload(totalSize int64) *DownloadPlan {
	return &DownloadPlan{
		ContentType: "application/octet-stream",
		Ranges:      []ByteRange{{Offset: 0, Length: -1}},
		TotalSize:   totalSize,
	}
}
