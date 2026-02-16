// Copyright 2026 Marko Milivojevic
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

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

// Acquire creates a lock file in the destination directory.
// Uses O_CREATE|O_EXCL for atomic create-or-fail — no TOCTOU race.
// Returns the lock file path on success, or an error if a lock already exists.
func Acquire(destDir string) (string, error) {
	lockPath := filepath.Join(destDir, ".backup.lock")

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

	// Atomic exclusive creation — kernel-guaranteed to fail if file exists
	f, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		if os.IsExist(err) {
			return "", lockExistsError(lockPath)
		}
		return "", fmt.Errorf("failed to create lock file: %w", err)
	}

	// Write lock info to the exclusively-created file
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		os.Remove(lockPath) // Clean up on write failure
		return "", fmt.Errorf("failed to write lock info: %w", writeErr)
	}
	if closeErr != nil {
		os.Remove(lockPath) // Clean up on close failure
		return "", fmt.Errorf("failed to finalize lock file: %w", closeErr)
	}

	return lockPath, nil
}

// lockExistsError reads the existing lock file and returns a detailed error message.
func lockExistsError(lockPath string) error {
	existingLock, readErr := Read(lockPath)
	if readErr != nil {
		// Lock file exists but we can't read it
		return errors.New(
			fmt.Sprintf("Backup already in progress (lock file exists: %s)", lockPath),
			fmt.Sprintf("If the process is not running, manually remove the lock file with: rm %s", lockPath),
		)
	}

	// Provide detailed error with PID and timestamp
	return errors.New(
		fmt.Sprintf("Backup already in progress (PID %d on %s, started %s)",
			existingLock.PID,
			existingLock.Hostname,
			existingLock.Timestamp.Format("2006-01-02 15:04:05")),
		fmt.Sprintf("Lock file: %s\nIf the process is not running, manually remove the lock file with: rm %s",
			lockPath, lockPath),
	)
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
