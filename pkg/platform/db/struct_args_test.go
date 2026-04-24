package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testSnap struct {
	ID    string `db:"id"`
	Name  string `db:"name"`
	Score int    `db:"score"`
}

type testSnapWithOptions struct {
	ID    string `db:"id,omitempty"`
	Skip  string `db:"-"`
	NoTag string
}

func TestStructArgs(t *testing.T) {
	t.Run("maps tagged fields to name→value pairs", func(t *testing.T) {
		snap := testSnap{ID: "abc", Name: "foo", Score: 42}
		assert.Equal(t, map[string]any{"id": "abc", "name": "foo", "score": 42}, StructArgs(snap))
	})

	t.Run("omits fields tagged with dash or lacking db tag", func(t *testing.T) {
		snap := testSnapWithOptions{ID: "x", Skip: "ignored", NoTag: "also ignored"}
		assert.Equal(t, map[string]any{"id": "x"}, StructArgs(snap))
	})

	t.Run("includes nil pointer field", func(t *testing.T) {
		type withPtr struct {
			Val *time.Time `db:"val"`
		}
		args := StructArgs(withPtr{Val: nil})
		require.Contains(t, args, "val")
		assert.Nil(t, args["val"])
	})

	t.Run("dereferences pointer receiver", func(t *testing.T) {
		snap := testSnap{ID: "abc", Name: "foo", Score: 42}
		assert.Equal(t, map[string]any{"id": "abc", "name": "foo", "score": 42}, StructArgs(&snap))
	})
}
