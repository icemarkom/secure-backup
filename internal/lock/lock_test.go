package lock

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcquire_Success(t *testing.T) {
	tmpDir := t.TempDir()

	lockPath, err := Acquire(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, ".backup.lock"), lockPath)

	// Verify lock file exists
	_, err = os.Stat(lockPath)
	assert.NoError(t, err, "lock file should exist")

	// Verify lock file contents
	data, err := os.ReadFile(lockPath)
	require.NoError(t, err)

	var lockInfo LockInfo
	err = json.Unmarshal(data, &lockInfo)
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, os.Getpid(), lockInfo.PID)
	assert.NotEmpty(t, lockInfo.Hostname)
	assert.WithinDuration(t, time.Now().UTC(), lockInfo.Timestamp, 2*time.Second)
}

func TestAcquire_LockExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first lock
	lockPath1, err := Acquire(tmpDir)
	require.NoError(t, err)
	defer os.Remove(lockPath1)

	// Try to acquire again - should fail
	lockPath2, err := Acquire(tmpDir)
	assert.Error(t, err)
	assert.Empty(t, lockPath2)
	assert.Contains(t, err.Error(), "Backup already in progress")
	assert.Contains(t, err.Error(), fmt.Sprintf("PID %d", os.Getpid()))
}

func TestAcquire_LockExistsButCorrupted(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, ".backup.lock")

	// Create corrupted lock file
	err := os.WriteFile(lockPath, []byte("invalid json {{{"), 0644)
	require.NoError(t, err)

	// Try to acquire - should fail with generic message
	acquiredPath, err := Acquire(tmpDir)
	assert.Error(t, err)
	assert.Empty(t, acquiredPath)
	assert.Contains(t, err.Error(), "already in progress")
	assert.Contains(t, err.Error(), "manually remove")
}

func TestAcquire_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()

	lockPath, err := Acquire(tmpDir)
	require.NoError(t, err)
	defer os.Remove(lockPath)

	// Verify no .tmp file exists
	tmpPath := lockPath + ".tmp"
	_, err = os.Stat(tmpPath)
	assert.True(t, os.IsNotExist(err), ".tmp file should not exist after successful acquire")

	// Verify no .tmp files in directory
	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)

	for _, file := range files {
		assert.False(t, strings.HasSuffix(file.Name(), ".tmp"),
			"no .tmp files should remain in directory: found %s", file.Name())
	}
}

func TestRelease_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Acquire lock
	lockPath, err := Acquire(tmpDir)
	require.NoError(t, err)

	// Verify lock exists
	_, err = os.Stat(lockPath)
	assert.NoError(t, err)

	// Release lock
	err = Release(lockPath)
	assert.NoError(t, err)

	// Verify lock is gone
	_, err = os.Stat(lockPath)
	assert.True(t, os.IsNotExist(err), "lock file should be removed")
}

func TestRelease_AlreadyRemoved(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, ".backup.lock")

	// Release non-existent lock - should not error
	err := Release(lockPath)
	assert.NoError(t, err, "releasing non-existent lock should be graceful")
}

func TestRelease_PermissionDenied(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir := t.TempDir()

	// Acquire lock
	lockPath, err := Acquire(tmpDir)
	require.NoError(t, err)

	// Make directory read-only
	err = os.Chmod(tmpDir, 0555)
	require.NoError(t, err)
	defer os.Chmod(tmpDir, 0755) // Restore permissions for cleanup

	// Try to release - should fail
	err = Release(lockPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove lock file")
}

func TestRead_Valid(t *testing.T) {
	tmpDir := t.TempDir()

	// Acquire lock
	lockPath, err := Acquire(tmpDir)
	require.NoError(t, err)
	defer os.Remove(lockPath)

	// Read lock
	lockInfo, err := Read(lockPath)
	require.NoError(t, err)
	assert.NotNil(t, lockInfo)

	// Verify fields
	assert.Equal(t, os.Getpid(), lockInfo.PID)
	assert.NotEmpty(t, lockInfo.Hostname)
	assert.WithinDuration(t, time.Now().UTC(), lockInfo.Timestamp, 2*time.Second)
}

func TestRead_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, ".backup.lock")

	// Create invalid JSON
	err := os.WriteFile(lockPath, []byte("not valid json {{{"), 0644)
	require.NoError(t, err)

	// Read should fail
	lockInfo, err := Read(lockPath)
	assert.Error(t, err)
	assert.Nil(t, lockInfo)
	assert.Contains(t, err.Error(), "failed to parse lock file")
}

func TestRead_NonexistentFile(t *testing.T) {
	lockPath := "/nonexistent/path/.backup.lock"

	lockInfo, err := Read(lockPath)
	assert.Error(t, err)
	assert.Nil(t, lockInfo)
	assert.Contains(t, err.Error(), "lock file not found")
}

func TestLockInfo_JSONRoundTrip(t *testing.T) {
	original := LockInfo{
		PID:       12345,
		Hostname:  "test-server",
		Timestamp: time.Now().UTC().Round(time.Second), // Round to avoid nanosecond differences
	}

	// Marshal
	data, err := json.MarshalIndent(original, "", "  ")
	require.NoError(t, err)

	// Unmarshal
	var decoded LockInfo
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.PID, decoded.PID)
	assert.Equal(t, original.Hostname, decoded.Hostname)
	assert.True(t, original.Timestamp.Equal(decoded.Timestamp))
}

func TestAcquireReleaseCycle(t *testing.T) {
	tmpDir := t.TempDir()

	// Acquire
	lockPath, err := Acquire(tmpDir)
	require.NoError(t, err)

	// Should not be able to acquire again
	_, err = Acquire(tmpDir)
	assert.Error(t, err)

	// Release
	err = Release(lockPath)
	require.NoError(t, err)

	// Should be able to acquire again now
	lockPath2, err := Acquire(tmpDir)
	require.NoError(t, err)
	defer os.Remove(lockPath2)

	assert.Equal(t, lockPath, lockPath2, "lock path should be the same")
}
