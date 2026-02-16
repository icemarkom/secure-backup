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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeferReleasesLockOnError demonstrates that defer properly releases the lock
// even when the function returns an error (simulating runBackup error paths)
func TestDeferReleasesLockOnError(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate the runBackup function with an error path
	simulateBackupWithError := func(destDir string) error {
		// Acquire lock
		lockPath, err := Acquire(destDir)
		if err != nil {
			return err
		}
		defer Release(lockPath) // This should run even on error

		// Simulate an error happening (like compressor creation failure)
		return assert.AnError // Return error - defer should still run
	}

	// Call the simulated function
	err := simulateBackupWithError(tmpDir)
	assert.Error(t, err, "simulated error should be returned")

	// Verify lock was released despite the error
	lockPath := filepath.Join(tmpDir, ".backup.lock")
	_, err = os.Stat(lockPath)
	assert.True(t, os.IsNotExist(err), "lock file should be removed by defer even on error")
}

// TestDeferReleasesLockOnSuccess verifies defer also works on success
func TestDeferReleasesLockOnSuccess(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate the runBackup function with success
	simulateBackupSuccess := func(destDir string) error {
		lockPath, err := Acquire(destDir)
		if err != nil {
			return err
		}
		defer Release(lockPath)

		// Simulate successful backup
		return nil
	}

	// Call the simulated function
	err := simulateBackupSuccess(tmpDir)
	require.NoError(t, err)

	// Verify lock was released
	lockPath := filepath.Join(tmpDir, ".backup.lock")
	_, err = os.Stat(lockPath)
	assert.True(t, os.IsNotExist(err), "lock file should be removed by defer on success")
}
