// Package transcode provides studio master lossless audio vault archiving,
// AES-256-GCM authenticated encryption, SHA-256 provenance tagging, and cold storage preservation.
package transcode

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidMasterKey  = errors.New("master vault encryption key must be 32 bytes (AES-256)")
	ErrIntegrityMismatch = errors.New("unsealed master WAV SHA-256 checksum does not match vault manifest")
	ErrCorruptVaultData  = errors.New("corrupt or tampered master vault archive")
)

// MasterMetadata holds technical provenance properties for the lossless master.
type MasterMetadata struct {
	TrackID          string    `json:"track_id"`
	Title            string    `json:"title"`
	Artist           string    `json:"artist"`
	SampleRate       int       `json:"sample_rate"`
	Channels         int       `json:"channels"`
	BitDepth         int       `json:"bit_depth"`
	OriginalSHA256   string    `json:"original_sha256"`
	OriginalByteSize int64     `json:"original_byte_size"`
	ArchivedAt       time.Time `json:"archived_at"`
	StorageClass     string    `json:"storage_class"` // e.g. "GLACIER_DEEP_ARCHIVE", "COLDLINE"
}

// VaultArchive encapsulates the encrypted payload, nonce, and plaintext manifest header.
type VaultArchive struct {
	Manifest   MasterMetadata `json:"manifest"`
	NonceHex   string         `json:"nonce_hex"`
	Ciphertext []byte         `json:"ciphertext"` // Encrypted with AES-256-GCM
}

// MasterVaultManager manages sealing and opening of pristine lossless audio masters.
type MasterVaultManager struct {
	masterKey []byte
}

// NewMasterVaultManager creates a vault manager using a 256-bit AES master key.
func NewMasterVaultManager(masterKey []byte) (*MasterVaultManager, error) {
	if len(masterKey) != 32 {
		return nil, ErrInvalidMasterKey
	}
	return &MasterVaultManager{masterKey: masterKey}, nil
}

// SealMaster computes provenance hashes, encrypts the raw lossless WAV bytes with AES-256-GCM,
// and packages the archive ready for cold storage deposition.
func (mvm *MasterVaultManager) SealMaster(meta MasterMetadata, rawWAV []byte, storageClass string) (*VaultArchive, error) {
	if len(rawWAV) == 0 {
		return nil, errors.New("master WAV data is empty")
	}
	if storageClass == "" {
		storageClass = "GLACIER"
	}

	// Compute original bit-perfect SHA-256 hash
	h := sha256.Sum256(rawWAV)
	origHash := hex.EncodeToString(h[:])

	meta.OriginalSHA256 = origHash
	meta.OriginalByteSize = int64(len(rawWAV))
	meta.ArchivedAt = time.Now().UTC()
	meta.StorageClass = storageClass

	// AES-256-GCM Authenticated Encryption
	block, err := aes.NewCipher(mvm.masterKey)
	if err != nil {
		return nil, fmt.Errorf("aes init failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm init failed: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed generating nonce: %w", err)
	}

	// Seal: ciphertext includes 16-byte authentication tag
	manifestBytes, _ := json.Marshal(meta)
	ciphertext := gcm.Seal(nil, nonce, rawWAV, manifestBytes)

	return &VaultArchive{
		Manifest:   meta,
		NonceHex:   hex.EncodeToString(nonce),
		Ciphertext: ciphertext,
	}, nil
}

// OpenMaster unseals the cold archive, validates GCM authentication tag, decrypts WAV bytes,
// and verifies that the decrypted WAV matches the original SHA-256 checksum bit-for-bit.
func (mvm *MasterVaultManager) OpenMaster(archive *VaultArchive) ([]byte, *MasterMetadata, error) {
	if archive == nil || len(archive.Ciphertext) == 0 {
		return nil, nil, ErrCorruptVaultData
	}

	nonce, err := hex.DecodeString(archive.NonceHex)
	if err != nil {
		return nil, nil, ErrCorruptVaultData
	}

	block, err := aes.NewCipher(mvm.masterKey)
	if err != nil {
		return nil, nil, fmt.Errorf("aes init failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("gcm init failed: %w", err)
	}

	manifestBytes, _ := json.Marshal(archive.Manifest)
	plaintext, err := gcm.Open(nil, nonce, archive.Ciphertext, manifestBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: authentication failed: %v", ErrCorruptVaultData, err)
	}

	// Verify SHA-256 integrity against sealed manifest
	h := sha256.Sum256(plaintext)
	unsealedHash := hex.EncodeToString(h[:])
	if unsealedHash != archive.Manifest.OriginalSHA256 {
		return nil, nil, ErrIntegrityMismatch
	}

	return plaintext, &archive.Manifest, nil
}

// Serialize serializes the vault archive for S3 storage deposition.
func (va *VaultArchive) Serialize() ([]byte, error) {
	return json.Marshal(va)
}

// DeserializeVaultArchive reconstructs a vault archive from storage bytes.
func DeserializeVaultArchive(data []byte) (*VaultArchive, error) {
	var va VaultArchive
	if err := json.Unmarshal(data, &va); err != nil {
		return nil, fmt.Errorf("failed to deserialize vault archive: %w", err)
	}
	return &va, nil
}
