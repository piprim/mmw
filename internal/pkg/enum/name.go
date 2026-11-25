// Package enum generates go-enum source files from the rows of lookup tables.
package enum

import (
	"errors"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	// ErrEmptyName is returned when a label has no usable character.
	ErrEmptyName = errors.New("label gives an empty name")
	// ErrLeadingDigit is returned when the cleaned label starts with a digit.
	ErrLeadingDigit = errors.New("name starts with a digit")
)

var (
	ligatures = strings.NewReplacer("œ", "oe", "æ", "ae")
	separator = regexp.MustCompile(`[^a-z0-9]+`)
	// camelCase boundaries: agendaGoogle -> agenda_Google, DDVProposition -> DDV_Proposition.
	lowerUpper   = regexp.MustCompile(`([\p{Ll}0-9])(\p{Lu})`)
	acronymUpper = regexp.MustCompile(`(\p{Lu})(\p{Lu}\p{Ll})`)
)

// CleanName turns a label into a go-enum value name: snake_case, ending in
// "_ID" so that go-enum gives the constant the Go spelling (AgendaGoogleID).
func CleanName(label string) (string, error) {
	s := lowerUpper.ReplaceAllString(label, "${1}_${2}")
	s = acronymUpper.ReplaceAllString(s, "${1}_${2}")
	s = ligatures.Replace(strings.ToLower(s))
	// NFD splits an accented letter into the letter and its mark; drop the marks.
	s = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}

		return r
	}, norm.NFD.String(s))
	s = strings.Trim(separator.ReplaceAllString(s, "_"), "_")

	if s == "" {
		return "", ErrEmptyName
	}
	if s[0] >= '0' && s[0] <= '9' {
		return "", ErrLeadingDigit
	}

	return s + "_ID", nil
}
