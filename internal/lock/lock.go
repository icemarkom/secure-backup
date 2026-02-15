package lock

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/icemarkom/secure-backup/internal/errors"
)

// LockInfo represents the contents of a lock file
type LockInfo struct {
	PID       int       `json:"pid"`
	Hostname  string    `json:"hostname"`
	Timestamp time.Time `json:"timestamp"`
}

// Acquire creates a lock file in the destination directory
// Returns the lock file path on success, or an error if a lock already exists
func Acquire(destDir string) (string, error) {
	lockPath := filepath.Join(destDir, ".backup.lock")

	// Check if lock file already exists
	if _, err := os.Stat(lockPath); err == nil {
		// Lock exists - read it to get details for error message
		existingLock, readErr := Read(lockPath)
		if readErr != nil {
			// Lock file exists but we can't read it
			return "", errors.New(
				fmt.Sprintf("Backup already in progress (lock file exists: %s)", lockPath),
				fmt.Sprintf("If the process is not running, manually remove the lock file with: rm %s", lockPath),
			)
		}

		// Provide detailed error with PID and timestamp
		return "", errors.New(
			fmt.Sprintf("Backup already in progress (PID %d on %s, started %s)",
				existingLock.PID,
				existingLock.Hostname,
				existingLock.Timestamp.Format("2006-01-02 15:04:05")),
			fmt.Sprintf("Lock file: %s\nIf the process is not running, manually remove the lock file with: rm %s",
				lockPath, lockPath),
		)
	}

	// Create lock info
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	lockInfo := LockInfo{
		PID:       os.Getpid(),
		Hostname:  hostname,
		Timestamp: time.Now().UTC(),
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(lockInfo, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize lock info: %w", err)
	}

	// Write to temp file first for atomic operation
	tmpPath := lockPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write lock file: %w", err)
	}

	// Atomic rename to final path
	if err := os.Rename(tmpPath, lockPath); err != nil {
		os.Remove(tmpPath) // Clean up temp file on failure
		return "", fmt.Errorf("failed to create lock file: %w", err)
	}

	return lockPath, nil
}

// Release removes the lock file
// Does not return an error if the lock file doesn't exist (graceful cleanup)
func Release(lockPath string) error {
	err := os.Remove(lockPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}
	return nil
}

// Read reads and deserializes a lock file
func Read(lockPath string) (*LockInfo, error) {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("lock file not found: %s", lockPath)
		}
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	var lockInfo LockInfo
	if err := json.Unmarshal(data, &lockInfo); err != nil {
		return nil, fmt.Errorf("failed to parse lock file (corrupted?): %w", err)
	}

	return &lockInfo, nil
}
