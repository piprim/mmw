package scaffold

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunStep(t *testing.T) {
	t.Run("streams the output and returns nil on success", func(t *testing.T) {
		var out, errOut bytes.Buffer
		err := runStep(context.Background(), &out, &errOut, t.TempDir(), "echo", "sh", "-c", "echo hello")
		require.NoError(t, err)
		assert.Equal(t, "hello\n", out.String())
	})

	t.Run("names the step when the command fails", func(t *testing.T) {
		var out, errOut bytes.Buffer
		err := runStep(context.Background(), &out, &errOut, t.TempDir(), "go mod tidy", "sh", "-c", "exit 3")
		assert.ErrorContains(t, err, "go mod tidy")
	})

	t.Run("runs in the given directory", func(t *testing.T) {
		dir := t.TempDir()
		var out, errOut bytes.Buffer
		require.NoError(t, runStep(context.Background(), &out, &errOut, dir, "pwd", "pwd"))
		assert.Contains(t, out.String(), dir)
	})
}

func TestRunSteps(t *testing.T) {
	t.Run("a failing step names itself and lists the remaining commands", func(t *testing.T) {
		dir := t.TempDir()
		var out, errOut bytes.Buffer
		err := runSteps(context.Background(), &out, &errOut, []step{
			{dir, "first", []string{"sh", "-c", "exit 1"}},
			{dir, "second", []string{"go", "mod", "tidy"}},
			{dir, "third", []string{"go", "work", "sync"}},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "first failed")
		assert.ErrorContains(t, err, "go mod tidy (in "+dir+")")
		assert.ErrorContains(t, err, "go work sync (in "+dir+")")
	})
}
