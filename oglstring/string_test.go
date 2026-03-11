package oglstring

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestUnaccentString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple accented characters",
			input:    "café",
			expected: "cafe",
		},
		{
			name:     "multiple accents",
			input:    "résumé",
			expected: "resume",
		},
		{
			name:     "mixed accented and non-accented",
			input:    "Hôtel Montréal",
			expected: "Hotel Montreal",
		},
		{
			name:     "no accents",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "spanish characters",
			input:    "niño señor",
			expected: "nino senor",
		},
		{
			name:     "french characters",
			input:    "être français",
			expected: "etre francais",
		},
		{
			name:     "german characters",
			input:    "über",
			expected: "uber",
		},
		{
			name:     "portuguese characters",
			input:    "ação",
			expected: "acao",
		},
		{
			name:     "numbers and special characters unchanged",
			input:    "test123!@#",
			expected: "test123!@#",
		},
		{
			name:     "uppercase accented characters",
			input:    "CAFÉ",
			expected: "CAFE",
		},
		{
			name:     "mixed case with accents",
			input:    "Crème Brûlée",
			expected: "Creme Brulee",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := UnaccentString(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("UnaccentString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestUnaccentReader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple accented characters",
			input:    "café",
			expected: "cafe",
		},
		{
			name:     "multiple accents",
			input:    "résumé",
			expected: "resume",
		},
		{
			name:     "no accents",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "empty reader",
			input:    "",
			expected: "",
		},
		{
			name:     "large text with accents",
			input:    "Le château était très beau et élégant",
			expected: "Le chateau etait tres beau et elegant",
		},
		{
			name:     "multiline text",
			input:    "première ligne\ndeuxième ligne\ntroisième ligne",
			expected: "premiere ligne\ndeuxieme ligne\ntroisieme ligne",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			result := UnaccentReader(reader)

			var buf bytes.Buffer
			_, err := io.Copy(&buf, result)
			if err != nil {
				t.Fatalf("error reading from transformed reader: %v", err)
			}

			output := buf.String()
			if output != tt.expected {
				t.Errorf("UnaccentReader(%q) = %q, want %q", tt.input, output, tt.expected)
			}
		})
	}
}

func TestUnaccentReader_MultipleReads(t *testing.T) {
	input := "café résumé"
	expected := "cafe resume"

	reader := strings.NewReader(input)
	result := UnaccentReader(reader)

	// Read in chunks to test buffered reading
	buf := make([]byte, 5)
	var output strings.Builder

	for {
		n, err := result.Read(buf)
		if n > 0 {
			output.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("error during chunked reading: %v", err)
		}
	}

	if output.String() != expected {
		t.Errorf("UnaccentReader with chunked reads = %q, want %q", output.String(), expected)
	}
}

func TestNormalizeFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		err      error
	}{
		{
			name:     "simple filename with extension",
			input:    "document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "filename with spaces",
			input:    "my document.txt",
			expected: "my-document.txt",
		},
		{
			name:     "filename with accents",
			input:    "résumé.docx",
			expected: "resume.docx",
		},
		{
			name:     "filename with special characters",
			input:    "file@#$%name!.jpg",
			expected: "file-name-.jpg",
		},
		{
			name:     "filename with multiple dots",
			input:    "my.file.name.tar.gz",
			expected: "my-file-name-tar.gz",
		},
		{
			name:     "filename with underscores and hyphens",
			input:    "my_file-name.txt",
			expected: "my_file-name.txt",
		},
		{
			name:     "filename with numbers",
			input:    "report_2024_01.xlsx",
			expected: "report_2024_01.xlsx",
		},
		{
			name:     "filename with unicode characters",
			input:    "文档.pdf",
			expected: "-.pdf",
		},
		{
			name:     "filename with mixed accents and special chars",
			input:    "Café & Thé.txt",
			expected: "Cafe-The.txt",
		},
		{
			name:     "filename without extension",
			input:    "README",
			expected: "README",
		},
		{
			name:     "empty filename",
			input:    "",
			expected: "",
			err:      errors.New("empty filename"),
		},
		{
			name:     "filename with only extension",
			input:    ".gitignore",
			expected: ".gitignore",
		},
		{
			name:     "filename with path (should be cleaned)",
			input:    "../../../etc/passwd",
			expected: "etc-passwd",
		},
		{
			name:     "filename with dots and spaces",
			input:    "my . file . name.txt",
			expected: "my-file-name.txt",
		},
		{
			name:     "filename with parentheses",
			input:    "file (copy).txt",
			expected: "file-copy-.txt",
		},
		{
			name:     "filename with brackets",
			input:    "file[1].txt",
			expected: "file-1-.txt",
		},
		{
			name:     "uppercase with accents",
			input:    "CAFÉ.PDF",
			expected: "CAFE.PDF",
		},
		{
			name:     "mixed case with special chars and accents",
			input:    "Crème Brûlée Recipe!.md",
			expected: "Creme-Brulee-Recipe-.md",
		},
		{
			name:     "filename with emojis",
			input:    "file😀test.txt",
			expected: "file-test.txt",
		},
		{
			name:     "filename with leading/trailing spaces",
			input:    "  spaced file  .txt",
			expected: "spaced-file.txt",
		},
		{
			name:     "filename with tabs and newlines",
			input:    "file\twith\nnewlines.txt",
			expected: "file-with-newlines.txt",
		},
		{
			name:     "complex accented filename",
			input:    "Ñoño's файл.doc",
			expected: "Nono-s-.doc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeFileName(tt.input)
			if err != nil && tt.err == nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err == nil && tt.err != nil {
				t.Fatalf("expected error: %v", tt.err)
			}

			if result != tt.expected {
				t.Errorf("NormalizeFileName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeFileName_PreservesValidCharacters(t *testing.T) {
	// Test that valid ASCII alphanumeric, underscore, and hyphen are preserved
	input := "Valid_File-Name123.txt"
	expected := "Valid_File-Name123.txt"

	result, err := NormalizeFileName(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != expected {
		t.Errorf("NormalizeFileName(%q) = %q, want %q", input, result, expected)
	}
}

func TestNormalizeFileName_HandlesPathTraversal(t *testing.T) {
	// Test that path traversal attempts are neutralized
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple parent directory",
			input:    "../file.txt",
			expected: "file.txt",
		},
		{
			name:     "multiple parent directories",
			input:    "../../file.txt",
			expected: "file.txt",
		},
		{
			name:     "absolute path",
			input:    "/etc/passwd",
			expected: "etc-passwd",
		},
		{
			name:     "windows path",
			input:    "C:\\Windows\\System32\\file.exe",
			expected: "C-Windows-System32-file.exe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeFileName(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("NormalizeFileName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkUnaccentString(b *testing.B) {
	input := "Le château était très beau et élégant avec des fenêtres magnifiques"

	b.ResetTimer()
	for b.Loop() {
		_, _ = UnaccentString(input)
	}
}

func BenchmarkUnaccentReader(b *testing.B) {
	input := "Le château était très beau et élégant avec des fenêtres magnifiques"

	b.ResetTimer()
	for b.Loop() {
		reader := strings.NewReader(input)
		result := UnaccentReader(reader)
		_, _ = io.ReadAll(result)
	}
}

func BenchmarkNormalizeFileName(b *testing.B) {
	input := "Crème Brûlée Recipe (Final Version) [2024]!.pdf"

	b.ResetTimer()
	for b.Loop() {
		_, _ = NormalizeFileName(input)
	}
}
