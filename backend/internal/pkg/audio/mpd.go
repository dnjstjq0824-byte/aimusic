// Package audio provides professional MPEG-DASH MPD (Media Presentation Description)
// dynamic manifest generation conforming to ISO/IEC 23009-1 and DASH-IF interoperability standards.
package audio

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Standard DASH XML namespace URIs.
const (
	DASHNamespaceMPD           = "urn:mpeg:dash:schema:mpd:2011"
	DASHNamespaceXSI           = "http://www.w3.org/2001/XMLSchema-instance"
	DASHSchemaLocation         = "urn:mpeg:dash:schema:mpd:2011 DASH-MPD.xsd"
	DASHProfileLive            = "urn:mpeg:dash:profile:isoff-live:2011"
	DASHProfileDASHIF          = "urn:com:dashif:dash264"
	DASHChannelConfigScheme2011 = "urn:mpeg:dash:23003:3:audio_channel_configuration:2011"
	DASHRoleScheme2011         = "urn:mpeg:dash:role:2011"
	DASHProtectionCENC         = "urn:mpeg:dash:mp4protection:2011"
	DASHProtectionWidevine     = "urn:uuid:edef8ba9-79d6-4ace-a3c8-27dcd51d21ed"
	DASHProtectionPlayReady    = "urn:uuid:9a04f079-9840-4286-ab92-e65be0885f95"
	DASHAIDisclosureScheme     = "urn:aimusic:ai-disclosure:2026"
)

// Common DASH validation errors.
var (
	ErrNilStreamSpec       = errors.New("dash stream specification cannot be nil")
	ErrInvalidTrackID      = errors.New("track ID cannot be empty")
	ErrInvalidDuration     = errors.New("audio duration must be positive")
	ErrNoProfilesSpecified = errors.New("at least one audio profile representation must be specified")
	ErrInvalidISODuration  = errors.New("invalid ISO 8601 duration format")
)

// MPD represents the root XML element of an MPEG-DASH manifest.
type MPD struct {
	XMLName                   xml.Name        `xml:"MPD"`
	Xmlns                     string          `xml:"xmlns,attr"`
	XmlnsXsi                  string          `xml:"xmlns:xsi,attr,omitempty"`
	SchemaLocation            string          `xml:"xsi:schemaLocation,attr,omitempty"`
	Profiles                  string          `xml:"profiles,attr"`
	Type                      string          `xml:"type,attr"`
	MediaPresentationDuration string          `xml:"mediaPresentationDuration,attr"`
	MinBufferTime             string          `xml:"minBufferTime,attr"`
	MaxSegmentDuration        string          `xml:"maxSegmentDuration,attr,omitempty"`
	BaseURL                   string          `xml:"BaseURL,omitempty"`
	Period                    []DASHPeriod    `xml:"Period"`
}

// DASHPeriod represents a contiguous presentation time interval.
type DASHPeriod struct {
	ID             string              `xml:"id,attr"`
	Start          string              `xml:"start,attr"`
	Duration       string              `xml:"duration,attr,omitempty"`
	AdaptationSets []DASHAdaptationSet `xml:"AdaptationSet"`
}

// DASHAdaptationSet contains a set of interchangeable audio representations.
type DASHAdaptationSet struct {
	ID                        int                    `xml:"id,attr"`
	ContentType               string                 `xml:"contentType,attr"`
	MimeType                  string                 `xml:"mimeType,attr"`
	Codecs                    string                 `xml:"codecs,attr"`
	AudioSamplingRate         int                    `xml:"audioSamplingRate,attr"`
	Lang                      string                 `xml:"lang,attr,omitempty"`
	AudioChannelConfiguration *DASHAudioChannelConfig `xml:"AudioChannelConfiguration,omitempty"`
	Role                      *DASHRole              `xml:"Role,omitempty"`
	ContentProtections        []DASHContentProtection `xml:"ContentProtection,omitempty"`
	EssentialProperties       []DASHEssentialProperty `xml:"EssentialProperty,omitempty"`
	SegmentTemplate           *DASHSegmentTemplate   `xml:"SegmentTemplate,omitempty"`
	Representations           []DASHRepresentation   `xml:"Representation"`
}

// DASHAudioChannelConfig defines the spatial channel layout (e.g. 2 for stereo).
type DASHAudioChannelConfig struct {
	SchemeIDURI string `xml:"schemeIdUri,attr"`
	Value       string `xml:"value,attr"`
}

// DASHRole defines the semantic role of the adaptation set (e.g. "main").
type DASHRole struct {
	SchemeIDURI string `xml:"schemeIdUri,attr"`
	Value       string `xml:"value,attr"`
}

// DASHContentProtection defines DRM and CENC encryption parameters.
type DASHContentProtection struct {
	SchemeIDURI string `xml:"schemeIdUri,attr"`
	Value       string `xml:"value,attr,omitempty"`
	DefaultKID  string `xml:"cenc:default_KID,attr,omitempty"`
	PSSH        string `xml:"cenc:pssh,omitempty"`
}

// DASHEssentialProperty communicates essential playback constraints or regulatory disclosures.
type DASHEssentialProperty struct {
	SchemeIDURI string `xml:"schemeIdUri,attr"`
	Value       string `xml:"value,attr"`
}

// DASHSegmentTemplate describes segment addressing via variable substitution.
type DASHSegmentTemplate struct {
	Timescale       int                  `xml:"timescale,attr"`
	Initialization  string               `xml:"initialization,attr"`
	Media           string               `xml:"media,attr"`
	StartNumber     int                  `xml:"startNumber,attr"`
	Duration        int                  `xml:"duration,attr,omitempty"`
	SegmentTimeline *DASHSegmentTimeline `xml:"SegmentTimeline,omitempty"`
}

// DASHSegmentTimeline specifies explicit timing for variable duration segments.
type DASHSegmentTimeline struct {
	Segments []DASHTimelineSegment `xml:"S"`
}

// DASHTimelineSegment describes an individual segment timing in timescale ticks.
type DASHTimelineSegment struct {
	T *int64 `xml:"t,attr,omitempty"`
	D int64  `xml:"d,attr"`
	R *int   `xml:"r,attr,omitempty"`
}

// DASHRepresentation describes an individual bitrate variant stream.
type DASHRepresentation struct {
	ID                string               `xml:"id,attr"`
	Bandwidth         int                  `xml:"bandwidth,attr"`
	Codecs            string               `xml:"codecs,attr,omitempty"`
	AudioSamplingRate int                  `xml:"audioSamplingRate,attr,omitempty"`
	SegmentTemplate   *DASHSegmentTemplate `xml:"SegmentTemplate,omitempty"`
}

// DASHStreamProfile describes an audio bitrate encoding tier.
type DASHStreamProfile struct {
	ID          string `json:"id"`
	BitrateKbps int    `json:"bitrate_kbps"`
	Codec       string `json:"codec"`
	SampleRate  int    `json:"sample_rate"`
	Channels    int    `json:"channels"`
}

// DefaultStreamingAudioProfiles returns standard 5-tier audio profiles for adaptive streaming.
func DefaultStreamingAudioProfiles() []DASHStreamProfile {
	return []DASHStreamProfile{
		{ID: "audio_aac_64k", BitrateKbps: 64, Codec: "mp4a.40.2", SampleRate: 48000, Channels: 2},
		{ID: "audio_aac_128k", BitrateKbps: 128, Codec: "mp4a.40.2", SampleRate: 48000, Channels: 2},
		{ID: "audio_aac_192k", BitrateKbps: 192, Codec: "mp4a.40.2", SampleRate: 48000, Channels: 2},
		{ID: "audio_opus_256k", BitrateKbps: 256, Codec: "opus", SampleRate: 48000, Channels: 2},
		{ID: "audio_aac_320k", BitrateKbps: 320, Codec: "mp4a.40.2", SampleRate: 48000, Channels: 2},
	}
}

// DASHStreamSpec provides configuration for building a DASH dynamic manifest.
type DASHStreamSpec struct {
	TrackID            string
	Title              string
	Artist             string
	DurationSec        float64
	SegmentDurationSec float64
	BaseURL            string
	Profiles           []DASHStreamProfile
	AIProvider         string
	AIModelVersion     string
	AIGenerationType   string
	ISRC               string
	EnableDRM          bool
	DRMKeyID           string
	UseTimeline        bool
}

// FormatISODuration converts duration in seconds to ISO 8601 duration string (e.g. PT3M45.500S).
func FormatISODuration(totalSeconds float64) string {
	if totalSeconds <= 0 {
		return "PT0S"
	}

	hours := int(totalSeconds / 3600.0)
	remain := totalSeconds - float64(hours*3600)
	minutes := int(remain / 60.0)
	seconds := remain - float64(minutes*60)

	var sb strings.Builder
	sb.WriteString("PT")
	if hours > 0 {
		sb.WriteString(fmt.Sprintf("%dH", hours))
	}
	if minutes > 0 {
		sb.WriteString(fmt.Sprintf("%dM", minutes))
	}

	// Format seconds with millisecond precision
	if seconds > 0 || (hours == 0 && minutes == 0) {
		rounded := math.Round(seconds*1000.0) / 1000.0
		if math.Abs(rounded-math.Round(rounded)) < 1e-4 {
			sb.WriteString(fmt.Sprintf("%.0fS", rounded))
		} else {
			sb.WriteString(fmt.Sprintf("%.3fS", rounded))
		}
	}

	return sb.String()
}

var isoDurationRegex = regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:([\d.]+)S)?$`)

// ParseISODuration parses an ISO 8601 duration string into seconds.
func ParseISODuration(iso string) (float64, error) {
	if iso == "" {
		return 0, ErrInvalidISODuration
	}
	matches := isoDurationRegex.FindStringSubmatch(iso)
	if matches == nil {
		return 0, ErrInvalidISODuration
	}

	var totalSec float64
	if matches[1] != "" {
		h, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return 0, err
		}
		totalSec += h * 3600.0
	}
	if matches[2] != "" {
		m, err := strconv.ParseFloat(matches[2], 64)
		if err != nil {
			return 0, err
		}
		totalSec += m * 60.0
	}
	if matches[3] != "" {
		s, err := strconv.ParseFloat(matches[3], 64)
		if err != nil {
			return 0, err
		}
		totalSec += s
	}

	return totalSec, nil
}

// BuildDASHManifest constructs a compliant MPEG-DASH MPD manifest byte slice from a stream specification.
func BuildDASHManifest(spec *DASHStreamSpec) ([]byte, error) {
	if spec == nil {
		return nil, ErrNilStreamSpec
	}
	if strings.TrimSpace(spec.TrackID) == "" {
		return nil, ErrInvalidTrackID
	}
	if spec.DurationSec <= 0 {
		return nil, ErrInvalidDuration
	}

	profiles := spec.Profiles
	if len(profiles) == 0 {
		profiles = DefaultStreamingAudioProfiles()
	}

	segDuration := spec.SegmentDurationSec
	if segDuration <= 0 {
		segDuration = 6.0 // standard 6-second audio segment
	}

	timescale := 48000
	ticksPerSegment := int(math.Round(segDuration * float64(timescale)))
	totalTicks := int64(math.Round(spec.DurationSec * float64(timescale)))

	// Adaptation Set definition
	adaptationSet := DASHAdaptationSet{
		ID:                1,
		ContentType:       "audio",
		MimeType:          "audio/mp4",
		Codecs:            profiles[0].Codec,
		AudioSamplingRate: profiles[0].SampleRate,
		Lang:              "und",
		AudioChannelConfiguration: &DASHAudioChannelConfig{
			SchemeIDURI: DASHChannelConfigScheme2011,
			Value:       strconv.Itoa(profiles[0].Channels),
		},
		Role: &DASHRole{
			SchemeIDURI: DASHRoleScheme2011,
			Value:       "main",
		},
	}

	// AI Disclosure Compliance (Art. 31)
	if spec.AIProvider != "" || spec.AIGenerationType != "" {
		disclosureValue := fmt.Sprintf("type=%s;provider=%s", spec.AIGenerationType, spec.AIProvider)
		if spec.AIModelVersion != "" {
			disclosureValue += fmt.Sprintf(";version=%s", spec.AIModelVersion)
		}
		if spec.ISRC != "" {
			disclosureValue += fmt.Sprintf(";isrc=%s", spec.ISRC)
		}
		adaptationSet.EssentialProperties = append(adaptationSet.EssentialProperties, DASHEssentialProperty{
			SchemeIDURI: DASHAIDisclosureScheme,
			Value:       disclosureValue,
		})
	}

	// DRM Content Protection
	if spec.EnableDRM && spec.DRMKeyID != "" {
		adaptationSet.ContentProtections = append(adaptationSet.ContentProtections,
			DASHContentProtection{
				SchemeIDURI: DASHProtectionCENC,
				Value:       "cenc",
				DefaultKID:  spec.DRMKeyID,
			},
			DASHContentProtection{
				SchemeIDURI: DASHProtectionWidevine,
			},
			DASHContentProtection{
				SchemeIDURI: DASHProtectionPlayReady,
			},
		)
	}

	// Segment Template
	template := &DASHSegmentTemplate{
		Timescale:      timescale,
		Initialization: "$RepresentationID$/init.mp4",
		Media:          "$RepresentationID$/chunk-$Number%05d$.m4s",
		StartNumber:    1,
	}

	if spec.UseTimeline {
		timeline := &DASHSegmentTimeline{}
		fullSegments := int(totalTicks / int64(ticksPerSegment))
		remainderTicks := totalTicks % int64(ticksPerSegment)

		if fullSegments > 0 {
			repeatCount := fullSegments - 1
			startTick := int64(0)
			seg := DASHTimelineSegment{
				T: &startTick,
				D: int64(ticksPerSegment),
			}
			if repeatCount > 0 {
				seg.R = &repeatCount
			}
			timeline.Segments = append(timeline.Segments, seg)
		}

		if remainderTicks > 0 {
			timeline.Segments = append(timeline.Segments, DASHTimelineSegment{
				D: remainderTicks,
			})
		}
		template.SegmentTimeline = timeline
	} else {
		template.Duration = ticksPerSegment
	}

	adaptationSet.SegmentTemplate = template

	// Representations
	for _, p := range profiles {
		bandwidth := p.BitrateKbps * 1000
		rep := DASHRepresentation{
			ID:                p.ID,
			Bandwidth:         bandwidth,
			Codecs:            p.Codec,
			AudioSamplingRate: p.SampleRate,
		}
		adaptationSet.Representations = append(adaptationSet.Representations, rep)
	}

	periodDuration := FormatISODuration(spec.DurationSec)
	period := DASHPeriod{
		ID:             "period_0",
		Start:          "PT0S",
		Duration:       periodDuration,
		AdaptationSets: []DASHAdaptationSet{adaptationSet},
	}

	mpd := MPD{
		Xmlns:                     DASHNamespaceMPD,
		XmlnsXsi:                  DASHNamespaceXSI,
		SchemaLocation:            DASHSchemaLocation,
		Profiles:                  fmt.Sprintf("%s,%s", DASHProfileLive, DASHProfileDASHIF),
		Type:                      "static",
		MediaPresentationDuration: periodDuration,
		MinBufferTime:             FormatISODuration(2.0),
		MaxSegmentDuration:        FormatISODuration(segDuration + 0.5),
		BaseURL:                   spec.BaseURL,
		Period:                    []DASHPeriod{period},
	}

	var buf bytes.Buffer
	buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")

	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")
	if err := encoder.Encode(mpd); err != nil {
		return nil, fmt.Errorf("encoding mpd xml: %w", err)
	}
	buf.WriteString("\n")

	return buf.Bytes(), nil
}

// ValidateDASHManifest parses an MPD XML byte slice and checks conformance.
func ValidateDASHManifest(manifestXML []byte) (*MPD, error) {
	if len(manifestXML) == 0 {
		return nil, errors.New("empty manifest xml")
	}

	var mpd MPD
	decoder := xml.NewDecoder(bytes.NewReader(manifestXML))
	if err := decoder.Decode(&mpd); err != nil {
		return nil, fmt.Errorf("xml decode validation failed: %w", err)
	}

	if mpd.Type != "static" && mpd.Type != "dynamic" {
		return nil, fmt.Errorf("invalid mpd type: %s", mpd.Type)
	}
	if mpd.MediaPresentationDuration == "" {
		return nil, errors.New("missing mediaPresentationDuration attribute")
	}
	if len(mpd.Period) == 0 {
		return nil, errors.New("mpd must contain at least one period")
	}

	totalReps := 0
	for _, p := range mpd.Period {
		for _, as := range p.AdaptationSets {
			totalReps += len(as.Representations)
		}
	}
	if totalReps == 0 {
		return nil, errors.New("no representations found in adaptation sets")
	}

	return &mpd, nil
}
