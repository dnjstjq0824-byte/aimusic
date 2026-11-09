package transcode

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestMasterVaultManager_SealAndOpenBitPerfect(t *testing.T) {
	masterKey := make([]byte, 32) // AES-256
	_, _ = rand.Read(masterKey)

	vault, err := NewMasterVaultManager(masterKey)
	if err != nil {
		t.Fatalf("NewMasterVaultManager failed: %v", err)
	}

	// Mock pristine 24-bit 96kHz studio WAV master (100KB)
	rawWAV := make([]byte, 100*1024)
	copy(rawWAV[:4], "RIFF")
	copy(rawWAV[8:12], "WAVE")
	_, _ = rand.Read(rawWAV[12:])

	meta := MasterMetadata{
		TrackID:    "track-lossless-001",
		Title:      "Elysian Dreams (Master 24-bit)",
		Artist:     "AVibe Studio Artists",
		SampleRate: 96000,
		Channels:   2,
		BitDepth:   24,
	}

	archive, err := vault.SealMaster(meta, rawWAV, "GLACIER_FLEXIBLE")
	if err != nil {
		t.Fatalf("SealMaster failed: %v", err)
	}

	// Verify encryption
	if bytes.Equal(archive.Ciphertext, rawWAV) {
		t.Fatalf("ciphertext must not match raw WAV bytes")
	}
	if archive.Manifest.OriginalSHA256 == "" {
		t.Errorf("manifest missing OriginalSHA256")
	}

	// Serialize and deserialize
	archiveBytes, err := archive.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	deserialized, err := DeserializeVaultArchive(archiveBytes)
	if err != nil {
		t.Fatalf("DeserializeVaultArchive failed: %v", err)
	}

	// Unseal archive
	unsealedWAV, restoredMeta, err := vault.OpenMaster(deserialized)
	if err != nil {
		t.Fatalf("OpenMaster failed: %v", err)
	}

	// Bit-perfect match verification
	if !bytes.Equal(rawWAV, unsealedWAV) {
		t.Fatalf("DoD violated: unsealed WAV is not bit-perfect identical to original!")
	}
	if restoredMeta.SampleRate != 96000 || restoredMeta.BitDepth != 24 {
		t.Errorf("metadata mismatch: %d Hz, %d bit", restoredMeta.SampleRate, restoredMeta.BitDepth)
	}
}

func TestMasterVaultManager_TamperedDataDetection(t *testing.T) {
	masterKey := make([]byte, 32)
	_, _ = rand.Read(masterKey)

	vault, _ := NewMasterVaultManager(masterKey)
	rawWAV := []byte("RIFF1234WAVEmasteraudiocontent")

	archive, err := vault.SealMaster(MasterMetadata{TrackID: "track-tamper"}, rawWAV, "COLDLINE")
	if err != nil {
		t.Fatalf("SealMaster failed: %v", err)
	}

	// Corrupt ciphertext byte
	archive.Ciphertext[len(archive.Ciphertext)/2] ^= 0xFF

	_, _, err = vault.OpenMaster(archive)
	if err == nil {
		t.Fatalf("expected error on tampered ciphertext, got nil")
	}
}
