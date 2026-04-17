package checks

import "context"

// RunPreCommit runs all checkers sequentially against the provided files.
//
// targets is the pre-selected list of files (from git); callers are responsible
// for obtaining this list via SelectFiles or StagedFiles.  Each checker receives
// the same targets slice — checkers are expected to filter by extension internally.
//
// When failFast is true, RunPreCommit stops after the first checker that reports
// at least one violation and returns the partial results collected so far.
//
// An internal error (tool not found, file unreadable) from any checker stops the
// run immediately and the error is returned to the caller.
func RunPreCommit(ctx context.Context, checkers []Checker, targets []string, failFast bool) ([]Result, error) {
	results := []Result{}

	for _, checker := range checkers {
		result, err := checker.Check(ctx, targets)
		if err != nil {
			return results, err
		}

		results = append(results, result)

		if failFast && result.HasViolations() {
			return results, nil
		}
	}

	return results, nil
}
