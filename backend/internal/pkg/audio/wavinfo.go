package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// Errors for WAV INFO-chunk tagging (P2-1 #22).
var (
	ErrNotWAV       = errors.New("input is not a WAV file")
	ErrRF64NoInfo   = errors.New("RF64 tagging is out of scope")
	ErrWAVTruncated = errors.New("WAV chunk extends past end of input")
)

// wavInfoField renders one INFO sub-chunk: id + size + NUL-terminated text
// (size counts the NUL, not the alignment pad).
func wavInfoField(id, text string) []byte {
	if len(text) > 255 {
		text = text[:255]
	}
	out := new(bytes.Buffer)
	out.WriteString(id)
	_ = binary.Write(out, binary.LittleEndian, uint32(len(text)+1))
	out.WriteString(text)
	out.WriteByte(0)
	if (len(text)+1)%2 == 1 {
		out.WriteByte(0)
	}
	return out.Bytes()
}

// wavChunk spans one top-level RIFF chunk: offset of the id and total
// length covering id+size+data+pad.
type wavChunk struct{ off, total int }

// wavChunks walks the top-level RIFF chunks starting after the 12-byte header.
func wavChunks(data []byte) ([]wavChunk, error) {
	var chunks []wavChunk
	for off := 12; off+8 <= len(data); {
		size := int(binary.LittleEndian.Uint32(data[off+4 : off+8]))
		total := 8 + size
		if size%2 == 1 {
			total++
		}
		if off+total > len(data) {
			return nil, ErrWAVTruncated
		}
		chunks = append(chunks, wavChunk{off: off, total: total})
		off += total
	}
	return chunks, nil
}

// isInfoList reports whether the chunk at data[off:] is a LIST-INFO chunk.
func isInfoList(data []byte, off int) bool {
	return off+12 <= len(data) &&
		string(data[off:off+4]) == "LIST" &&
		string(data[off+8:off+12]) == "INFO"
}

// InjectWAVInfo embeds AI-disclosure metadata (INAM title, IARL archivist,
// ICMT comment) into a WAV file's LIST-INFO chunk, replacing any existing
// INFO chunk. Audio chunks pass through untouched.
func InjectWAVInfo(data []byte, title, artist, comment string) ([]byte, error) {
	if len(data) < 12 || string(data[:4]) == "RF64" {
		if len(data) >= 4 && string(data[:4]) == "RF64" {
			return nil, ErrRF64NoInfo
		}
		return nil, ErrNotWAV
	}
	if string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, ErrNotWAV
	}
	chunks, err := wavChunks(data)
	if err != nil {
		return nil, err
	}
	out := new(bytes.Buffer)
	out.Write(data[:12])
	for _, c := range chunks {
		if isInfoList(data, c.off) {
			continue // replaced below
		}
		out.Write(data[c.off : c.off+c.total])
	}
	body := new(bytes.Buffer)
	body.WriteString("INFO")
	body.Write(wavInfoField("INAM", title))
	body.Write(wavInfoField("IARL", artist))
	body.Write(wavInfoField("ICMT", comment))
	listPayload := body.Bytes()
	out.WriteString("LIST")
	_ = binary.Write(out, binary.LittleEndian, uint32(len(listPayload)))
	out.Write(listPayload)
	if (out.Len()-12)%2 == 1 {
		out.WriteByte(0)
	}
	tagged := out.Bytes()
	binary.LittleEndian.PutUint32(tagged[4:8], uint32(len(tagged)-8))
	return tagged, nil
}

// ParseWAVInfo reads the first LIST-INFO chunk into id->text fields.
func ParseWAVInfo(data []byte) (map[string]string, error) {
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, ErrNotWAV
	}
	chunks, err := wavChunks(data)
	if err != nil {
		return nil, err
	}
	for _, c := range chunks {
		if !isInfoList(data, c.off) {
			continue
		}
		size := int(binary.LittleEndian.Uint32(data[c.off+4 : c.off+8]))
		end := c.off + 8 + size
		fields := map[string]string{}
		for p := c.off + 12; p+8 <= end; {
			fid := string(data[p : p+4])
			fsize := int(binary.LittleEndian.Uint32(data[p+4 : p+8]))
			if p+8+fsize > end {
				break
			}
			text := string(data[p+8 : p+8+fsize])
			text = string(bytes.TrimRight([]byte(text), "\x00"))
			fields[fid] = text
			p += 8 + fsize
			if fsize%2 == 1 {
				p++
			}
		}
		return fields, nil
	}
	return nil, errors.New("WAV has no INFO chunk")
}
