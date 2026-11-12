package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func strp(s string) *string { return &s }

func TestTrackPublicViewHidesInternalFields(t *testing.T) {
	full := &Track{
		ID:                   "t1",
		OriginalObjectKey:    "staging/p/sha/master.wav",
		AudioFingerprintHash: strp("fp_123"),
		HLSPrefix:            strp("hls/t1"),
		RejectionReasonKo:    strp("내부 메모"),
		AIPrompt:             strp("secret prompt"),
		IsPromptPublic:       false,
		Status:               TrackReady,
	}
	pub := full.PublicView()
	raw, err := json.Marshal(pub)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	for _, leaked := range []string{"originalObjectKey", "audioFingerprintHash", "hlsPrefix", "rejectionReasonKo", "aiPrompt", "secret prompt", "staging/"} {
		if strings.Contains(body, leaked) {
			t.Fatalf("public view leaked %q: %s", leaked, body)
		}
	}
	if full.OriginalObjectKey == "" || full.AIPrompt == nil {
		t.Fatal("PublicView must not mutate the source track")
	}
}

func TestTrackPublicViewKeepsPublicPrompt(t *testing.T) {
	pub := (&Track{AIPrompt: strp("open prompt"), IsPromptPublic: true}).PublicView()
	if pub.AIPrompt == nil || *pub.AIPrompt != "open prompt" {
		t.Fatalf("public prompt dropped: %+v", pub.AIPrompt)
	}
	if (*Track)(nil).PublicView() != nil {
		t.Fatal("nil track must stay nil")
	}
}
