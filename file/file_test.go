package file

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestExists(t *testing.T) {
	t.Run("returns true for existing file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-exists-*")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		if !Exists(tmpFile.Name()) {
			t.Error("Exists() returned false for existing file")
		}
	})

	t.Run("returns true for existing directory", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-exists-dir-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		if !Exists(tmpDir) {
			t.Error("Exists() returned false for existing directory")
		}
	})

	t.Run("returns false for non-existent path", func(t *testing.T) {
		if Exists("/this/path/should/not/exist/ever") {
			t.Error("Exists() returned true for non-existent path")
		}
	})

	t.Run("returns false for empty path", func(t *testing.T) {
		if Exists("") {
			t.Error("Exists() returned true for empty path")
		}
	})
}

func TestExistsFS(t *testing.T) {
	t.Run("returns true for existing file in FS", func(t *testing.T) {
		fsys := fstest.MapFS{
			"testfile.txt": &fstest.MapFile{Data: []byte("content")},
		}

		if !ExistsFS(fsys, "testfile.txt") {
			t.Error("ExistsFS() returned false for existing file")
		}
	})

	t.Run("returns true for existing directory in FS", func(t *testing.T) {
		fsys := fstest.MapFS{
			"subdir/file.txt": &fstest.MapFile{Data: []byte("content")},
		}

		if !ExistsFS(fsys, "subdir") {
			t.Error("ExistsFS() returned false for existing directory")
		}
	})

	t.Run("returns false for non-existent path in FS", func(t *testing.T) {
		fsys := fstest.MapFS{}

		if ExistsFS(fsys, "nonexistent.txt") {
			t.Error("ExistsFS() returned true for non-existent path")
		}
	})
}

func TestIsDir(t *testing.T) {
	t.Run("returns true for directory", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-isdir-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		isDir, err := IsDir(tmpDir)
		if err != nil {
			t.Fatalf("IsDir() returned unexpected error: %v", err)
		}
		if !isDir {
			t.Error("IsDir() returned false for directory")
		}
	})

	t.Run("returns false for regular file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-isdir-file-*")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		isDir, err := IsDir(tmpFile.Name())
		if err != nil {
			t.Fatalf("IsDir() returned unexpected error: %v", err)
		}
		if isDir {
			t.Error("IsDir() returned true for regular file")
		}
	})

	t.Run("returns error for non-existent path", func(t *testing.T) {
		_, err := IsDir("/this/path/should/not/exist")
		if err == nil {
			t.Error("IsDir() should return error for non-existent path")
		}
	})
}

func TestEnsureNotEmpty(t *testing.T) {
	t.Run("returns nil for non-empty string", func(t *testing.T) {
		err := ensureNotEmpty("test")
		if err != nil {
			t.Errorf("ensureNotEmpty() returned error for non-empty string: %v", err)
		}
	})

	t.Run("returns error for empty string", func(t *testing.T) {
		err := ensureNotEmpty("")
		if err == nil {
			t.Error("ensureNotEmpty() should return error for empty string")
		}
		if err.Error() != "empty string given" {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestCreateTargetDirIfNotExists(t *testing.T) {
	t.Run("creates parent directory for file path", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-createtarget-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		targetPath := filepath.Join(tmpDir, "newdir", "file.txt")
		err = CreateTargetDirIfNotExists(targetPath)
		if err != nil {
			t.Fatalf("CreateTargetDirIfNotExists() returned error: %v", err)
		}

		parentDir := filepath.Dir(targetPath)
		if !Exists(parentDir) {
			t.Error("parent directory was not created")
		}
	})

	t.Run("creates directory when path ends with slash", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-createtarget-slash-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		targetPath := filepath.Join(tmpDir, "newdir") + "/"
		err = CreateTargetDirIfNotExists(targetPath)
		if err != nil {
			t.Fatalf("CreateTargetDirIfNotExists() returned error: %v", err)
		}

		// Remove trailing slash for existence check
		dirPath := targetPath[:len(targetPath)-1]
		if !Exists(dirPath) {
			t.Error("directory was not created")
		}
	})

	t.Run("returns error for empty path", func(t *testing.T) {
		err := CreateTargetDirIfNotExists("")
		if err == nil {
			t.Error("CreateTargetDirIfNotExists() should return error for empty path")
		}
	})

	t.Run("succeeds when directory already exists", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-createtarget-exists-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		targetPath := filepath.Join(tmpDir, "file.txt")
		err = CreateTargetDirIfNotExists(targetPath)
		if err != nil {
			t.Errorf("CreateTargetDirIfNotExists() returned error for existing dir: %v", err)
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-createtarget-nested-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		targetPath := filepath.Join(tmpDir, "a", "b", "c", "file.txt")
		err = CreateTargetDirIfNotExists(targetPath)
		if err != nil {
			t.Fatalf("CreateTargetDirIfNotExists() returned error: %v", err)
		}

		parentDir := filepath.Dir(targetPath)
		if !Exists(parentDir) {
			t.Error("nested directories were not created")
		}
	})
}

func TestCreateDirIfNotExists(t *testing.T) {
	t.Run("creates directory when it does not exist", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-createdir-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		newDir := filepath.Join(tmpDir, "newdir")
		err = CreateDirIfNotExists(newDir)
		if err != nil {
			t.Fatalf("CreateDirIfNotExists() returned error: %v", err)
		}

		if !Exists(newDir) {
			t.Error("directory was not created")
		}

		isDir, err := IsDir(newDir)
		if err != nil {
			t.Fatalf("IsDir() returned error: %v", err)
		}
		if !isDir {
			t.Error("created path is not a directory")
		}
	})

	t.Run("succeeds when directory already exists", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-createdir-exists-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		err = CreateDirIfNotExists(tmpDir)
		if err != nil {
			t.Errorf("CreateDirIfNotExists() returned error for existing dir: %v", err)
		}
	})

	t.Run("returns error for empty path", func(t *testing.T) {
		err := CreateDirIfNotExists("")
		if err == nil {
			t.Error("CreateDirIfNotExists() should return error for empty path")
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-createdir-nested-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		nestedDir := filepath.Join(tmpDir, "a", "b", "c")
		err = CreateDirIfNotExists(nestedDir)
		if err != nil {
			t.Fatalf("CreateDirIfNotExists() returned error: %v", err)
		}

		if !Exists(nestedDir) {
			t.Error("nested directories were not created")
		}
	})

	t.Run("creates directory with correct permissions", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-createdir-perms-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		newDir := filepath.Join(tmpDir, "newdir")
		err = CreateDirIfNotExists(newDir)
		if err != nil {
			t.Fatalf("CreateDirIfNotExists() returned error: %v", err)
		}

		info, err := os.Stat(newDir)
		if err != nil {
			t.Fatalf("failed to stat directory: %v", err)
		}

		// Check that directory has expected permissions (0775)
		// Note: actual permissions may be affected by umask
		perm := info.Mode().Perm()
		if perm&fs.FileMode(0700) != fs.FileMode(0700) {
			t.Errorf("directory does not have owner rwx permissions: %o", perm)
		}
	})
}
