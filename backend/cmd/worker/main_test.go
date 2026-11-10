package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSHA256FileStreamsWithoutLoading(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input.bin")
	// "abc" has a well-known SHA-256.
	if err := os.WriteFile(path, []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := sha256File(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got != want {
		t.Fatalf("sha256 = %s, want %s", got, want)
	}
	if _, err := sha256File(filepath.Join(dir, "missing.bin")); err == nil {
		t.Fatal("expected error for missing file")
	}
}
