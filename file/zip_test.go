package file

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestAddFileToZip(t *testing.T) {
	t.Run("adds file to zip archive", func(t *testing.T) {
		// Create temp file with content
		tmpDir, err := os.MkdirTemp("", "test-addfiletozip-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		testContent := []byte("test file content")
		testFilePath := filepath.Join(tmpDir, "testfile.txt")
		err = os.WriteFile(testFilePath, testContent, 0644)
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		// Create zip file
		zipPath := filepath.Join(tmpDir, "test.zip")
		zipFile, err := os.Create(zipPath)
		if err != nil {
			t.Fatalf("failed to create zip file: %v", err)
		}
		defer zipFile.Close()

		zipWriter := zip.NewWriter(zipFile)

		// Add file to zip
		err = addFileToZip(zipWriter, testFilePath)
		if err != nil {
			t.Fatalf("addFileToZip() returned error: %v", err)
		}

		err = zipWriter.Close()
		if err != nil {
			t.Fatalf("failed to close zip writer: %v", err)
		}

		// Verify zip content
		zipReader, err := zip.OpenReader(zipPath)
		if err != nil {
			t.Fatalf("failed to open zip for reading: %v", err)
		}
		defer zipReader.Close()

		if len(zipReader.File) != 1 {
			t.Fatalf("expected 1 file in zip, got %d", len(zipReader.File))
		}

		// Check file name (should be base name only)
		if zipReader.File[0].Name != "testfile.txt" {
			t.Errorf("expected file name 'testfile.txt', got '%s'", zipReader.File[0].Name)
		}

		// Check content
		rc, err := zipReader.File[0].Open()
		if err != nil {
			t.Fatalf("failed to open file in zip: %v", err)
		}
		defer rc.Close()

		var buf bytes.Buffer
		_, err = buf.ReadFrom(rc)
		if err != nil {
			t.Fatalf("failed to read file from zip: %v", err)
		}

		if !bytes.Equal(buf.Bytes(), testContent) {
			t.Errorf("content mismatch: expected %s, got %s", testContent, buf.Bytes())
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-addfiletozip-noexist-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		zipPath := filepath.Join(tmpDir, "test.zip")
		zipFile, err := os.Create(zipPath)
		if err != nil {
			t.Fatalf("failed to create zip file: %v", err)
		}
		defer zipFile.Close()

		zipWriter := zip.NewWriter(zipFile)
		defer zipWriter.Close()

		err = addFileToZip(zipWriter, "/nonexistent/file.txt")
		if err == nil {
			t.Error("addFileToZip() should return error for non-existent file")
		}
	})

	t.Run("adds multiple files to zip", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-addfiletozip-multi-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create test files
		files := map[string][]byte{
			"file1.txt": []byte("content 1"),
			"file2.txt": []byte("content 2"),
			"file3.txt": []byte("content 3"),
		}

		var filePaths []string
		for name, content := range files {
			path := filepath.Join(tmpDir, name)
			err = os.WriteFile(path, content, 0644)
			if err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}
			filePaths = append(filePaths, path)
		}

		// Create zip
		zipPath := filepath.Join(tmpDir, "test.zip")
		zipFile, err := os.Create(zipPath)
		if err != nil {
			t.Fatalf("failed to create zip file: %v", err)
		}

		zipWriter := zip.NewWriter(zipFile)

		for _, path := range filePaths {
			err = addFileToZip(zipWriter, path)
			if err != nil {
				t.Fatalf("addFileToZip() returned error: %v", err)
			}
		}

		zipWriter.Close()
		zipFile.Close()

		// Verify
		zipReader, err := zip.OpenReader(zipPath)
		if err != nil {
			t.Fatalf("failed to open zip for reading: %v", err)
		}
		defer zipReader.Close()

		if len(zipReader.File) != 3 {
			t.Errorf("expected 3 files in zip, got %d", len(zipReader.File))
		}
	})

	t.Run("preserves file content integrity", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-addfiletozip-integrity-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create file with binary content
		binaryContent := make([]byte, 1024)
		for i := range binaryContent {
			binaryContent[i] = byte(i % 256)
		}

		testFilePath := filepath.Join(tmpDir, "binary.dat")
		err = os.WriteFile(testFilePath, binaryContent, 0644)
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		// Create zip
		zipPath := filepath.Join(tmpDir, "test.zip")
		zipFile, err := os.Create(zipPath)
		if err != nil {
			t.Fatalf("failed to create zip file: %v", err)
		}

		zipWriter := zip.NewWriter(zipFile)
		err = addFileToZip(zipWriter, testFilePath)
		if err != nil {
			t.Fatalf("addFileToZip() returned error: %v", err)
		}

		zipWriter.Close()
		zipFile.Close()

		// Verify content integrity
		zipReader, err := zip.OpenReader(zipPath)
		if err != nil {
			t.Fatalf("failed to open zip for reading: %v", err)
		}
		defer zipReader.Close()

		rc, err := zipReader.File[0].Open()
		if err != nil {
			t.Fatalf("failed to open file in zip: %v", err)
		}
		defer rc.Close()

		var buf bytes.Buffer
		_, err = buf.ReadFrom(rc)
		if err != nil {
			t.Fatalf("failed to read file from zip: %v", err)
		}

		if !bytes.Equal(buf.Bytes(), binaryContent) {
			t.Error("binary content was corrupted during zip")
		}
	})

	t.Run("uses deflate compression", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-addfiletozip-compress-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		testFilePath := filepath.Join(tmpDir, "test.txt")
		err = os.WriteFile(testFilePath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		zipPath := filepath.Join(tmpDir, "test.zip")
		zipFile, err := os.Create(zipPath)
		if err != nil {
			t.Fatalf("failed to create zip file: %v", err)
		}

		zipWriter := zip.NewWriter(zipFile)
		err = addFileToZip(zipWriter, testFilePath)
		if err != nil {
			t.Fatalf("addFileToZip() returned error: %v", err)
		}

		zipWriter.Close()
		zipFile.Close()

		// Verify compression method
		zipReader, err := zip.OpenReader(zipPath)
		if err != nil {
			t.Fatalf("failed to open zip for reading: %v", err)
		}
		defer zipReader.Close()

		if zipReader.File[0].Method != zip.Deflate {
			t.Errorf("expected Deflate compression, got %d", zipReader.File[0].Method)
		}
	})
}

func TestZipFiles(t *testing.T) {
	t.Run("writes valid zip with single file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-zipfiles-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		testContent := []byte("test content for zip")
		testFilePath := filepath.Join(tmpDir, "test.txt")
		err = os.WriteFile(testFilePath, testContent, 0644)
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		var buf bytes.Buffer
		err = ZipFiles(&buf, []string{testFilePath})
		if err != nil {
			t.Fatalf("ZipFiles() returned error: %v", err)
		}

		if buf.Len() == 0 {
			t.Fatal("ZipFiles() wrote empty bytes")
		}

		// Verify the zip is valid by reading it
		zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		if err != nil {
			t.Fatalf("failed to read zip: %v", err)
		}

		if len(zipReader.File) != 1 {
			t.Fatalf("expected 1 file in zip, got %d", len(zipReader.File))
		}

		if zipReader.File[0].Name != "test.txt" {
			t.Errorf("expected file name 'test.txt', got '%s'", zipReader.File[0].Name)
		}

		// Verify content
		rc, err := zipReader.File[0].Open()
		if err != nil {
			t.Fatalf("failed to open file in zip: %v", err)
		}
		defer rc.Close()

		var contentBuf bytes.Buffer
		_, err = contentBuf.ReadFrom(rc)
		if err != nil {
			t.Fatalf("failed to read file from zip: %v", err)
		}

		if !bytes.Equal(contentBuf.Bytes(), testContent) {
			t.Errorf("content mismatch: expected %s, got %s", testContent, contentBuf.Bytes())
		}
	})

	t.Run("writes valid zip with multiple files", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-zipfiles-multi-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		files := map[string][]byte{
			"file1.txt": []byte("content one"),
			"file2.txt": []byte("content two"),
			"file3.txt": []byte("content three"),
		}

		var filePaths []string
		for name, content := range files {
			path := filepath.Join(tmpDir, name)
			err = os.WriteFile(path, content, 0644)
			if err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}
			filePaths = append(filePaths, path)
		}

		var buf bytes.Buffer
		err = ZipFiles(&buf, filePaths)
		if err != nil {
			t.Fatalf("ZipFiles() returned error: %v", err)
		}

		if buf.Len() == 0 {
			t.Fatal("ZipFiles() wrote empty bytes")
		}

		// Verify the zip contains all files
		zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		if err != nil {
			t.Fatalf("failed to read zip: %v", err)
		}

		if len(zipReader.File) != 3 {
			t.Errorf("expected 3 files in zip, got %d", len(zipReader.File))
		}
	})

	t.Run("writes to file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-zipfiles-tofile-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create source file
		testContent := []byte("file content")
		srcPath := filepath.Join(tmpDir, "source.txt")
		err = os.WriteFile(srcPath, testContent, 0644)
		if err != nil {
			t.Fatalf("failed to write source file: %v", err)
		}

		// Write zip to file
		zipPath := filepath.Join(tmpDir, "output.zip")
		f, err := os.Create(zipPath)
		if err != nil {
			t.Fatalf("failed to create zip file: %v", err)
		}

		err = ZipFiles(f, []string{srcPath})
		f.Close()
		if err != nil {
			t.Fatalf("ZipFiles() returned error: %v", err)
		}

		// Verify the zip file
		zipReader, err := zip.OpenReader(zipPath)
		if err != nil {
			t.Fatalf("failed to open zip file: %v", err)
		}
		defer zipReader.Close()

		if len(zipReader.File) != 1 {
			t.Errorf("expected 1 file in zip, got %d", len(zipReader.File))
		}
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		var buf bytes.Buffer
		err := ZipFiles(&buf, []string{"/nonexistent/file.txt"})
		if err == nil {
			t.Error("ZipFiles() should return error for non-existent file")
		}
	})

	t.Run("handles empty file list", func(t *testing.T) {
		var buf bytes.Buffer
		err := ZipFiles(&buf, []string{})
		if err != nil {
			t.Fatalf("ZipFiles() with empty list returned error: %v", err)
		}

		// Empty zip should still be valid
		if buf.Len() == 0 {
			t.Fatal("ZipFiles() wrote empty bytes for empty list")
		}

		// Verify it's a valid empty zip
		zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		if err != nil {
			t.Fatalf("failed to read empty zip: %v", err)
		}

		if len(zipReader.File) != 0 {
			t.Errorf("expected 0 files in empty zip, got %d", len(zipReader.File))
		}
	})

	t.Run("preserves binary content", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-zipfiles-binary-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create binary content
		binaryContent := make([]byte, 1024)
		for i := range binaryContent {
			binaryContent[i] = byte(i % 256)
		}

		testFilePath := filepath.Join(tmpDir, "binary.dat")
		err = os.WriteFile(testFilePath, binaryContent, 0644)
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		var buf bytes.Buffer
		err = ZipFiles(&buf, []string{testFilePath})
		if err != nil {
			t.Fatalf("ZipFiles() returned error: %v", err)
		}

		// Verify content integrity
		zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		if err != nil {
			t.Fatalf("failed to read zip: %v", err)
		}

		rc, err := zipReader.File[0].Open()
		if err != nil {
			t.Fatalf("failed to open file in zip: %v", err)
		}
		defer rc.Close()

		var contentBuf bytes.Buffer
		_, err = contentBuf.ReadFrom(rc)
		if err != nil {
			t.Fatalf("failed to read file from zip: %v", err)
		}

		if !bytes.Equal(contentBuf.Bytes(), binaryContent) {
			t.Error("binary content was corrupted during zip")
		}
	})
}
