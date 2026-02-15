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
