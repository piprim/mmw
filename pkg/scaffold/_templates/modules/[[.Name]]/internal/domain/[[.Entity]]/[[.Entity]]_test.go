[[if .Database -]]
package [[.Entity]]_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

func TestParseID(t *testing.T) {
	tests := []struct {
		name    string
		input   int64
		want    [[.Entity]].ID
		wantErr error
	}{
		{name: "positive id", input: 42, want: [[.Entity]].ID(42)},
		{name: "zero id", input: 0, wantErr: [[.Entity]].ErrInvalidID},
		{name: "negative id", input: -7, wantErr: [[.Entity]].ErrInvalidID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := [[.Entity]].ParseID(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ParseID(%d) error = %v, want %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ParseID(%d) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestID_Formats(t *testing.T) {
	t.Run("Int64 returns the underlying value", func(t *testing.T) {
		if got := [[.Entity]].ID(15).Int64(); got != 15 {
			t.Errorf("Int64() = %d, want 15", got)
		}
	})

	t.Run("String returns decimal text", func(t *testing.T) {
		if got := [[.Entity]].ID(15).String(); got != "15" {
			t.Errorf("String() = %q, want \"15\"", got)
		}
	})
}

func TestReconstitute_RoundTrip(t *testing.T) {
	now := time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)
	snap := [[.Entity]].Snapshot{ID: 7, Label: "Première", CreatedAt: now, UpdatedAt: now}
	a := [[.Entity]].Reconstitute(&snap)

	t.Run("exposes the stored state through getters", func(t *testing.T) {
		if a.ID() != 7 || a.Label().String() != "Première" || !a.CreatedAt().Equal(now) || !a.UpdatedAt().Equal(now) {
			t.Errorf("getters = %d/%q/%v/%v", a.ID(), a.Label(), a.CreatedAt(), a.UpdatedAt())
		}
	})

	t.Run("snapshot equals the input", func(t *testing.T) {
		if got := a.Snapshot(); got != snap {
			t.Errorf("Snapshot() = %+v, want %+v", got, snap)
		}
	})

	t.Run("starts with no event", func(t *testing.T) {
		if len(a.Events()) != 0 {
			t.Errorf("Events() = %d, want 0", len(a.Events()))
		}
	})
}

func TestDelete_RecordsOneEvent(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	a := [[.Entity]].Reconstitute(&[[.Entity]].Snapshot{ID: 7, Label: "Première"})
	a.Delete(userID)

	t.Run("records exactly one Deleted event", func(t *testing.T) {
		if len(a.Events()) != 1 {
			t.Fatalf("Events() = %d, want 1", len(a.Events()))
		}
		if a.Events()[0].EventType() != [[.Entity]].EventTypeDeleted {
			t.Errorf("EventType() = %q, want %q", a.Events()[0].EventType(), [[.Entity]].EventTypeDeleted)
		}
	})

	t.Run("the event carries the aggregate ID, the user and the label", func(t *testing.T) {
		ev, ok := a.Events()[0].(*[[.Entity]].Deleted)
		if !ok {
			t.Fatalf("event type = %T, want *[[.Entity]].Deleted", a.Events()[0])
		}
		if ev.GetAggregateID() != "7" || ev.GetUserID() != userID || ev.Label != "Première" {
			t.Errorf("event = %+v", ev)
		}
		if ev.GetOccurredAt().IsZero() {
			t.Error("OccurredAt is zero")
		}
	})

	t.Run("ClearEvents empties the list", func(t *testing.T) {
		a.ClearEvents()
		if len(a.Events()) != 0 {
			t.Errorf("Events() = %d after ClearEvents, want 0", len(a.Events()))
		}
	})
}
[[- else -]]
package [[.Entity]]_test

import (
	"testing"

	"github.com/google/uuid"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

func TestNew(t *testing.T) {
	a := [[.Entity]].New([[.Entity]].ReconstituteLabel("Bonjour"))

	t.Run("exposes the label", func(t *testing.T) {
		if a.Label().String() != "Bonjour" {
			t.Errorf("Label() = %q", a.Label())
		}
	})

	t.Run("starts with no event", func(t *testing.T) {
		if len(a.Events()) != 0 {
			t.Errorf("Events() = %d, want 0", len(a.Events()))
		}
	})
}

func TestMarkSent_RecordsOneEvent(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	a := [[.Entity]].New([[.Entity]].ReconstituteLabel("Bonjour"))
	a.MarkSent(userID)

	t.Run("records exactly one Sent event", func(t *testing.T) {
		if len(a.Events()) != 1 || a.Events()[0].EventType() != [[.Entity]].EventTypeSent {
			t.Fatalf("Events() = %v, want one Sent", a.Events())
		}
	})

	t.Run("the event carries the label as aggregate ID and the user", func(t *testing.T) {
		ev, ok := a.Events()[0].(*[[.Entity]].Sent)
		if !ok {
			t.Fatalf("event type = %T", a.Events()[0])
		}
		if ev.GetAggregateID() != "Bonjour" || ev.GetUserID() != userID || ev.Label != "Bonjour" || ev.GetOccurredAt().IsZero() {
			t.Errorf("event = %+v", ev)
		}
	})

	t.Run("ClearEvents empties the list", func(t *testing.T) {
		a.ClearEvents()
		if len(a.Events()) != 0 {
			t.Errorf("Events() = %d after ClearEvents, want 0", len(a.Events()))
		}
	})
}
[[- end]]
