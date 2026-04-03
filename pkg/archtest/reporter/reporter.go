package reporter

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

type Reporter struct {
	w           io.Writer
	failedCount int
	passedCount int
}

func NewReporter(w io.Writer) *Reporter {
	return &Reporter{w: w}
}

func (r *Reporter) PrintHeader(title string) {
	fmt.Fprint(r.w, "\n")
	r.printSeparator()
	c := color.New(color.FgCyan).Add(color.Underline)
	c.Fprint(r.w, title+"\n\n")
}

func (r *Reporter) PrintCheck(name, description string, err error) {
	fmt.Fprintf(r.w, "- Checking service %s\n", name)
	fmt.Fprintf(r.w, "  %s\n", description)

	if err != nil {
		c := color.New(color.FgRed)
		c.Fprintf(r.w, "  ✗ FAILED:\n%v\n\n", err)
		r.failedCount++
	} else {
		c := color.New(color.FgGreen)
		c.Fprintf(r.w, "  ✓ PASSED\n\n")
		r.passedCount++
	}
}

func (r *Reporter) Summary() int {
	fmt.Fprintf(r.w, "Passed: %d\n", r.passedCount)
	fmt.Fprintf(r.w, "Failed: %d\n", r.failedCount)
	fmt.Fprintf(r.w, " Total: %d\n\n", r.failedCount+r.passedCount)

	r.printSeparator()
	if r.failedCount > 0 {
		c := color.New(color.FgRed)
		c.Fprintf(r.w, "✗ Architecture validation failed\n")

		return 1
	}

	c := color.New(color.FgGreen)
	c.Fprintf(r.w, "✓ All architecture checks passed\n")

	return 0
}

func (r *Reporter) printSeparator() {
	separator := "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	fmt.Fprintf(r.w, "%s\n\n", separator)
}
