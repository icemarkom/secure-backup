package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// Manifest represents metadata and integrity information for a backup
type Manifest struct {
	CreatedAt         time.Time `json:"created_at"`
	CreatedBy         CreatedBy `json:"created_by"`
	SourcePath        string    `json:"source_path"`
	BackupFile        string    `json:"backup_file"`
	Compression       string    `json:"compression"`
	Encryption        string    `json:"encryption"`
	ChecksumAlgorithm string    `json:"checksum_algorithm"`
	ChecksumValue     string    `json:"checksum_value"`
	SizeBytes         int64     `json:"size_bytes"`
}

// CreatedBy holds information about the tool that created the backup
type CreatedBy struct {
	Tool     string `json:"tool"`
	Version  string `json:"version"`
	Hostname string `json:"hostname"`
}

// New creates a new manifest with the given parameters
func New(sourcePath, backupFile, version string) (*Manifest, error) {
	hostname, err := os.Hostname()
	if err != nil {
		// Graceful fallback
		hostname = "unknown"
	}

	return &Manifest{
		CreatedAt:         time.Now().UTC(),
		SourcePath:        sourcePath,
		BackupFile:        backupFile,
		Compression:       "gzip",
		Encryption:        "gpg",
		ChecksumAlgorithm: "sha256",
		ChecksumValue:     "", // Set later via ComputeChecksum
		SizeBytes:         0,  // Set later
		CreatedBy: CreatedBy{
			Tool:     "secure-backup",
			Version:  version,
			Hostname: hostname,
		},
	}, nil
}

// Write serializes the manifest to a JSON file with indentation using atomic write
func (m *Manifest) Write(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Write to temp file first for atomic operation
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}

	// Atomic rename to final path
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // Clean up temp file on failure
		return fmt.Errorf("failed to finalize manifest file: %w", err)
	}

	return nil
}

// Read deserializes a manifest from a JSON file
func Read(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &m, nil
}

// Validate checks that all required fields are present
func (m *Manifest) Validate() error {
	if m.SourcePath == "" {
		return fmt.Errorf("manifest missing source_path")
	}
	if m.BackupFile == "" {
		return fmt.Errorf("manifest missing backup_file")
	}
	if m.ChecksumValue == "" {
		return fmt.Errorf("manifest missing checksum_value")
	}
	if m.ChecksumAlgorithm == "" {
		return fmt.Errorf("manifest missing checksum_algorithm")
	}
	if m.CreatedBy.Tool == "" {
		return fmt.Errorf("manifest missing created_by.tool")
	}
	if m.CreatedBy.Version == "" {
		return fmt.Errorf("manifest missing created_by.version")
	}
	return nil
}

// ComputeChecksum calculates the SHA256 checksum of a file using streaming
func ComputeChecksum(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to compute checksum: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// ValidateChecksum verifies that the backup file's checksum matches the manifest
func (m *Manifest) ValidateChecksum(backupFilePath string) error {
	actualChecksum, err := ComputeChecksum(backupFilePath)
	if err != nil {
		return fmt.Errorf("failed to compute file checksum: %w", err)
	}

	if actualChecksum != m.ChecksumValue {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", m.ChecksumValue, actualChecksum)
	}

	return nil
}
