package checks_test

import (
	"context"
	"testing"

	"github.com/piprim/mmw/internal/pkg/checks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockChecker is a test double for Checker.
type mockChecker struct {
	name   string
	result checks.Result
	err    error
}

func (m *mockChecker) Name() string { return m.name }

func (m *mockChecker) Check(_ context.Context, _ []string) (checks.Result, error) {
	if m.err != nil {
		return checks.Result{}, m.err
	}

	return m.result, nil
}

func TestRunPreCommit_EmptyCheckers(t *testing.T) {
	results, err := checks.RunPreCommit(t.Context(), []checks.Checker{}, []string{}, false)

	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestRunPreCommit_CollectsAllResults(t *testing.T) {
	c1 := &mockChecker{
		name: "checker-a",
		result: checks.Result{
			CheckerName: "checker-a",
			Violations:  []checks.Violation{{File: "a.go", Message: "issue"}},
		},
	}
	c2 := &mockChecker{
		name: "checker-b",
		result: checks.Result{
			CheckerName: "checker-b",
			Violations:  []checks.Violation{},
		},
	}

	results, err := checks.RunPreCommit(t.Context(), []checks.Checker{c1, c2}, []string{"a.go"}, false)

	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.True(t, results[0].HasViolations())
	assert.False(t, results[1].HasViolations())
}

func TestRunPreCommit_ContinuesAfterViolation(t *testing.T) {
	// Both checkers have violations; without fail-fast both must run.
	c1 := &mockChecker{
		name: "first",
		result: checks.Result{
			CheckerName: "first",
			Violations:  []checks.Violation{{Message: "violation in first"}},
		},
	}
	c2 := &mockChecker{
		name: "second",
		result: checks.Result{
			CheckerName: "second",
			Violations:  []checks.Violation{{Message: "violation in second"}},
		},
	}

	results, err := checks.RunPreCommit(t.Context(), []checks.Checker{c1, c2}, []string{"x.go"}, false)

	require.NoError(t, err)
	require.Len(t, results, 2, "both checkers must run even when first has violations")
	assert.True(t, results[0].HasViolations())
	assert.True(t, results[1].HasViolations())
}

func TestRunPreCommit_FailFast_StopsAfterFirstViolation(t *testing.T) {
	c1 := &mockChecker{
		name: "first",
		result: checks.Result{
			CheckerName: "first",
			Violations:  []checks.Violation{{Message: "violation"}},
		},
	}
	c2 := &mockChecker{
		name: "second",
		result: checks.Result{
			CheckerName: "second",
			Violations:  []checks.Violation{},
		},
	}

	results, err := checks.RunPreCommit(t.Context(), []checks.Checker{c1, c2}, []string{"x.go"}, true)

	require.NoError(t, err)
	assert.Len(t, results, 1, "fail-fast must stop after the first checker with violations")
}

func TestRunPreCommit_PropagatesInternalError(t *testing.T) {
	c := &mockChecker{
		name: "broken",
		err:  assert.AnError,
	}

	results, err := checks.RunPreCommit(t.Context(), []checks.Checker{c}, []string{"x.go"}, false)

	require.Error(t, err)
	assert.Empty(t, results, "no results should precede the first checker that errors")
}

func TestRunPreCommit_CancelledContext_PassedThrough(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // pre-cancel

	// RunPreCommit does not short-circuit on ctx.Err(); cancellation is delegated
	// to each Checker implementation. A checker that ignores context will still run.
	c := &mockChecker{
		name:   "ctx-unaware",
		result: checks.Result{CheckerName: "ctx-unaware", Violations: []checks.Violation{}},
	}

	results, err := checks.RunPreCommit(ctx, []checks.Checker{c}, []string{}, false)

	require.NoError(t, err)
	assert.Len(t, results, 1)
}
