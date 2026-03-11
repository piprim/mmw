package oglstring

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var isMn = runes.Predicate(func(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
})

var unaccentTransformer = transform.Chain(norm.NFD, runes.Remove(isMn), norm.NFC)

// UnaccentString removes accents from a string using Unicode NFD normalization.
// It converts accented characters to their base form (e.g., "café" → "cafe").
//
// Example:
//
//	result, _ := UnaccentString("résumé")  // Returns: "resume"
//	result, _ := UnaccentString("Crème Brûlée")  // Returns: "Creme Brulee"
func UnaccentString(s string) (string, error) {
	result, _, err := transform.String(unaccentTransformer, s)
	if err != nil {
		return "", fmt.Errorf("failed to unaccent string: %w", err)
	}

	return result, nil
}

// UnaccentReader wraps a reader to remove accents from the stream.
// Useful for processing large text files without loading them into memory.
// The transformation is performed using Unicode NFD normalization.
//
// Example:
//
//	reader := strings.NewReader("Café français")
//	unaccentedReader := UnaccentReader(reader)
//	// Reading from unaccentedReader yields: "Cafe francais"
func UnaccentReader(r io.Reader) io.Reader {
	return transform.NewReader(r, unaccentTransformer)
}

// NormalizeFileName returns a normalized and sanitized filename from a string.
//
// The normalization process:
//   - Removes accents using Unicode NFD normalization (café → cafe)
//   - Replaces non-ASCII and special characters with hyphens
//   - Preserves alphanumeric characters, underscores, and hyphens
//   - Strips path traversal attempts (../, leading /)
//   - Collapses multiple consecutive hyphens into a single hyphen
//   - Preserves the file extension
//
// Returns an error if the name is empty.
//
// Example:
//
//	NormalizeFileName("Café & Thé (2024).pdf") // Returns: "Cafe-The-2024-.pdf"
//	NormalizeFileName("../../../etc/passwd")    // Returns: "etc-passwd"
func NormalizeFileName(name string) (string, error) {
	if name == "" {
		return "", errors.New("empty name")
	}

	name, err := UnaccentString(name)
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(name)
	nameSExt := strings.TrimSuffix(name, ext)
	if nameSExt != "" {
		nameSExt = strings.Trim(nameSExt, " ")
		nameSExt = filepath.Clean(strings.ReplaceAll(nameSExt, "..", ""))
		nameSExt = strings.TrimLeft(nameSExt, "/")
		nameSExt = strings.TrimRight(nameSExt, "/")
		nameR := regexp.MustCompile(`[^a-zA-Z0-9_\-]`)
		nameSExt = nameR.ReplaceAllString(nameSExt, "-")
		nameR = regexp.MustCompile(`-{2,}`)
		nameSExt = nameR.ReplaceAllString(nameSExt, "-")
	}

	return nameSExt + ext, nil
}
