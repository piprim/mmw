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

func TestStructArgs_basic(t *testing.T) {
	snap := testSnap{ID: "abc", Name: "foo", Score: 42}
	args := StructArgs(snap)
	assert.Equal(t, map[string]any{"id": "abc", "name": "foo", "score": 42}, args)
}

func TestStructArgs_tagOptions(t *testing.T) {
	snap := testSnapWithOptions{ID: "x", Skip: "ignored", NoTag: "also ignored"}
	args := StructArgs(snap)
	assert.Equal(t, map[string]any{"id": "x"}, args)
}

func TestStructArgs_nilPointer(t *testing.T) {
	type withPtr struct {
		Val *time.Time `db:"val"`
	}
	args := StructArgs(withPtr{Val: nil})
	require.Contains(t, args, "val")
	assert.Nil(t, args["val"])
}

func TestStructArgs_pointerDereference(t *testing.T) {
	snap := testSnap{ID: "abc", Name: "foo", Score: 42}
	args := StructArgs(&snap)
	assert.Equal(t, map[string]any{"id": "abc", "name": "foo", "score": 42}, args)
}
