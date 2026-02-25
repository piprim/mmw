package file

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestNewLockFile(t *testing.T) {
	t.Run("wraps os.File correctly", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-lockfile-*")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		lockFile := NewLockFile(tmpFile)
		if lockFile == nil {
			t.Fatal("NewLockFile() returned nil")
		}
		if lockFile.File != tmpFile {
			t.Error("NewLockFile() did not wrap the file correctly")
		}
	})
}

func TestOpenLockFile(t *testing.T) {
	t.Run("opens existing file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-openlockfile-*")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		tmpFile.Close()
		defer os.Remove(tmpFile.Name())

		lockFile, err := OpenLockFile(tmpFile.Name(), 0644)
		if err != nil {
			t.Fatalf("OpenLockFile() returned error: %v", err)
		}
		defer lockFile.Close()

		if lockFile == nil {
			t.Error("OpenLockFile() returned nil lock file")
		}
	})

	t.Run("creates new file if not exists", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-openlockfile-new-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		newFile := filepath.Join(tmpDir, "newlock.pid")
		lockFile, err := OpenLockFile(newFile, 0644)
		if err != nil {
			t.Fatalf("OpenLockFile() returned error: %v", err)
		}
		defer lockFile.Close()

		if !Exists(newFile) {
			t.Error("OpenLockFile() did not create the file")
		}
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
		_, err := OpenLockFile("/nonexistent/dir/file.pid", 0644)
		if err == nil {
			t.Error("OpenLockFile() should return error for invalid path")
		}
	})
}

func TestLockFile_LockUnlock(t *testing.T) {
	t.Run("lock and unlock succeed", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-lock-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		lockPath := filepath.Join(tmpDir, "test.lock")
		lockFile, err := OpenLockFile(lockPath, 0644)
		if err != nil {
			t.Fatalf("OpenLockFile() returned error: %v", err)
		}
		defer lockFile.Close()

		// Lock should succeed
		err = lockFile.Lock()
		if err != nil {
			t.Fatalf("Lock() returned error: %v", err)
		}

		// Unlock should succeed
		err = lockFile.Unlock()
		if err != nil {
			t.Fatalf("Unlock() returned error: %v", err)
		}
	})

	t.Run("second lock on same file fails", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-lock-conflict-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		lockPath := filepath.Join(tmpDir, "test.lock")

		// First lock
		lockFile1, err := OpenLockFile(lockPath, 0644)
		if err != nil {
			t.Fatalf("OpenLockFile() returned error: %v", err)
		}
		defer lockFile1.Close()

		err = lockFile1.Lock()
		if err != nil {
			t.Fatalf("first Lock() returned error: %v", err)
		}

		// Second lock should fail (non-blocking)
		lockFile2, err := OpenLockFile(lockPath, 0644)
		if err != nil {
			t.Fatalf("second OpenLockFile() returned error: %v", err)
		}
		defer lockFile2.Close()

		err = lockFile2.Lock()
		if err == nil {
			t.Error("second Lock() should fail when file is already locked")
		}

		// Cleanup: unlock first file
		lockFile1.Unlock()
	})
}

func TestLockFile_WritePid(t *testing.T) {
	t.Run("writes current process pid", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-writepid-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		pidPath := filepath.Join(tmpDir, "test.pid")
		lockFile, err := OpenLockFile(pidPath, 0644)
		if err != nil {
			t.Fatalf("OpenLockFile() returned error: %v", err)
		}
		defer lockFile.Close()

		err = lockFile.WritePid()
		if err != nil {
			t.Fatalf("WritePid() returned error: %v", err)
		}

		// Verify written content
		content, err := os.ReadFile(pidPath)
		if err != nil {
			t.Fatalf("failed to read pid file: %v", err)
		}

		expectedPid := strconv.Itoa(os.Getpid())
		if string(content) != expectedPid {
			t.Errorf("expected pid %s, got %s", expectedPid, string(content))
		}
	})

	t.Run("overwrites existing content", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-writepid-overwrite-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		pidPath := filepath.Join(tmpDir, "test.pid")

		// Write initial content
		err = os.WriteFile(pidPath, []byte("999999999"), 0644)
		if err != nil {
			t.Fatalf("failed to write initial content: %v", err)
		}

		lockFile, err := OpenLockFile(pidPath, 0644)
		if err != nil {
			t.Fatalf("OpenLockFile() returned error: %v", err)
		}
		defer lockFile.Close()

		err = lockFile.WritePid()
		if err != nil {
			t.Fatalf("WritePid() returned error: %v", err)
		}

		// Verify content is overwritten (truncated to current pid)
		content, err := os.ReadFile(pidPath)
		if err != nil {
			t.Fatalf("failed to read pid file: %v", err)
		}

		expectedPid := strconv.Itoa(os.Getpid())
		if string(content) != expectedPid {
			t.Errorf("expected pid %s, got %s", expectedPid, string(content))
		}
	})
}

func TestLockFile_ReadPid(t *testing.T) {
	t.Run("reads pid from file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-readpid-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		pidPath := filepath.Join(tmpDir, "test.pid")
		expectedPid := 12345

		// Write a pid to the file
		err = os.WriteFile(pidPath, []byte(strconv.Itoa(expectedPid)), 0644)
		if err != nil {
			t.Fatalf("failed to write pid file: %v", err)
		}

		lockFile, err := OpenLockFile(pidPath, 0644)
		if err != nil {
			t.Fatalf("OpenLockFile() returned error: %v", err)
		}
		defer lockFile.Close()

		pid, err := lockFile.ReadPid()
		if err != nil {
			t.Fatalf("ReadPid() returned error: %v", err)
		}

		if pid != expectedPid {
			t.Errorf("expected pid %d, got %d", expectedPid, pid)
		}
	})

	t.Run("returns error for empty file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-readpid-empty-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		pidPath := filepath.Join(tmpDir, "test.pid")

		// Create empty file
		err = os.WriteFile(pidPath, []byte{}, 0644)
		if err != nil {
			t.Fatalf("failed to create empty file: %v", err)
		}

		lockFile, err := OpenLockFile(pidPath, 0644)
		if err != nil {
			t.Fatalf("OpenLockFile() returned error: %v", err)
		}
		defer lockFile.Close()

		_, err = lockFile.ReadPid()
		if err == nil {
			t.Error("ReadPid() should return error for empty file")
		}
	})

	t.Run("returns error for non-numeric content", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-readpid-invalid-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		pidPath := filepath.Join(tmpDir, "test.pid")

		// Write non-numeric content
		err = os.WriteFile(pidPath, []byte("not-a-pid"), 0644)
		if err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		lockFile, err := OpenLockFile(pidPath, 0644)
		if err != nil {
			t.Fatalf("OpenLockFile() returned error: %v", err)
		}
		defer lockFile.Close()

		_, err = lockFile.ReadPid()
		if err == nil {
			t.Error("ReadPid() should return error for non-numeric content")
		}
	})
}

func TestReadPidFile(t *testing.T) {
	t.Run("reads pid from file by name", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-readpidfile-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		pidPath := filepath.Join(tmpDir, "test.pid")
		expectedPid := 54321

		err = os.WriteFile(pidPath, []byte(strconv.Itoa(expectedPid)), 0644)
		if err != nil {
			t.Fatalf("failed to write pid file: %v", err)
		}

		pid, err := ReadPidFile(pidPath)
		if err != nil {
			t.Fatalf("ReadPidFile() returned error: %v", err)
		}

		if pid != expectedPid {
			t.Errorf("expected pid %d, got %d", expectedPid, pid)
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := ReadPidFile("/nonexistent/path/file.pid")
		if err == nil {
			t.Error("ReadPidFile() should return error for non-existent file")
		}
	})
}

func TestCreatePidFile(t *testing.T) {
	t.Run("creates and locks pid file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-createpidfile-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		pidPath := filepath.Join(tmpDir, "test.pid")
		lockFile, err := CreatePidFile(pidPath, 0644)
		if err != nil {
			t.Fatalf("CreatePidFile() returned error: %v", err)
		}
		defer lockFile.Remove()

		// Verify file exists
		if !Exists(pidPath) {
			t.Error("pid file was not created")
		}

		// Verify pid content
		content, err := os.ReadFile(pidPath)
		if err != nil {
			t.Fatalf("failed to read pid file: %v", err)
		}

		expectedPid := strconv.Itoa(os.Getpid())
		if string(content) != expectedPid {
			t.Errorf("expected pid %s, got %s", expectedPid, string(content))
		}

		// Verify file is locked (second lock should fail)
		lockFile2, err := OpenLockFile(pidPath, 0644)
		if err != nil {
			t.Fatalf("OpenLockFile() returned error: %v", err)
		}
		defer lockFile2.Close()

		err = lockFile2.Lock()
		if err == nil {
			t.Error("file should be locked after CreatePidFile()")
		}
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
		_, err := CreatePidFile("/nonexistent/dir/test.pid", 0644)
		if err == nil {
			t.Error("CreatePidFile() should return error for invalid path")
		}
	})
}

func TestLockFile_Remove(t *testing.T) {
	t.Run("removes locked file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-remove-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		pidPath := filepath.Join(tmpDir, "test.pid")
		lockFile, err := CreatePidFile(pidPath, 0644)
		if err != nil {
			t.Fatalf("CreatePidFile() returned error: %v", err)
		}

		err = lockFile.Remove()
		if err != nil {
			t.Fatalf("Remove() returned error: %v", err)
		}

		if Exists(pidPath) {
			t.Error("file should be removed after Remove()")
		}
	})

	t.Run("returns error for nil file", func(t *testing.T) {
		var lockFile *LockFile
		err := lockFile.Remove()
		if err == nil {
			t.Error("Remove() should return error for nil file")
		}
		if err != os.ErrInvalid {
			t.Errorf("expected os.ErrInvalid, got %v", err)
		}
	})
}

func TestGetFdName(t *testing.T) {
	t.Run("returns file name for valid fd", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-getfdname-*")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		defer tmpFile.Close()

		name, err := GetFdName(tmpFile.Fd())
		if err != nil {
			t.Fatalf("GetFdName() returned error: %v", err)
		}

		if name != tmpFile.Name() {
			t.Errorf("expected name %s, got %s", tmpFile.Name(), name)
		}
	})

	t.Run("returns error for invalid fd", func(t *testing.T) {
		_, err := GetFdName(999999)
		if err == nil {
			t.Error("GetFdName() should return error for invalid fd")
		}
	})
}
