package handler

import (
	"strings"
	"testing"
)

func TestSafeExportFilename(t *testing.T) {
	got := safeExportFilename("2026-11-26T15:04:05Z")
	if want := "dsp-kit-2026-11-26T15-04-05Z.csv"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if strings.ContainsAny(got, `:/\`) {
		t.Fatalf("filename contains Windows-illegal chars: %q", got)
	}
	if got := safeExportFilename(""); got != "dsp-kit-export.csv" {
		t.Fatalf("empty stamp: got %q", got)
	}
}
