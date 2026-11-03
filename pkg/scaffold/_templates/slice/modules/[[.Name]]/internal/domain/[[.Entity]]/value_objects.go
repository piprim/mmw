package [[.Entity]]

import "unicode/utf8"

// LabelMaxLen mirrors the CHECK constraint of [[.Name]].[[.Entity]].label.
const LabelMaxLen = 200

// Label is the human-readable name of a [[.Entity]]: non-empty, at most
// LabelMaxLen runes. Build it with NewLabel on input, ReconstituteLabel on
// values read from storage.
type Label struct {
	value string
}

// NewLabel validates s and returns a Label, or ErrInvalidLabel.
func NewLabel(s string) (Label, error) {
	if s == "" || utf8.RuneCountInString(s) > LabelMaxLen {
		return Label{}, ErrInvalidLabel
	}

	return Label{value: s}, nil
}

// ReconstituteLabel trusts a value read from storage: a stored row must stay
// readable even if the rule changes.
func ReconstituteLabel(s string) Label {
	return Label{value: s}
}

// String returns the primitive representation of the label.
func (l Label) String() string {
	return l.value
}
