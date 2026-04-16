// Package archtest provides architectural validation for the mmw modular monolith.
package archtest

import (
	"errors"
	"fmt"
	"os"

	"github.com/piprim/mmw/pkg/archtest/custom"
	"github.com/piprim/mmw/pkg/archtest/orchestrator"
	"github.com/piprim/mmw/pkg/archtest/reporter"
)

// Validator is the interface all custom arch validators implement.
type Validator interface {
	// Name returns the name of the validator
	Name() string
	// Description returns the description of the validator
	Description() string
	// Check checks ;)
	Check() error
}

// RunAll runs all architectural validators from repoRoot and returns exit code (0=pass, 1=fail).
func RunAll(repoRoot string) int {
	rep := reporter.NewReporter(os.Stdout)
	rep.PrintHeader("Architecture Validation")

	// Per-module arch:check tasks via mise
	services, err := orchestrator.DiscoverServices(
		repoRoot+"/modules", "arch:check",
	)
	if err != nil {
		rep.PrintCheck("discovery", "Discover modules with arch:check task", err)

		return rep.Summary()
	}
	for _, svc := range services {
		var checkErr error
		if svc.HasArchCheck {
			result := orchestrator.RunServiceCheck(svc.Path, svc.Name)
			if result.ExitCode != 0 {
				checkErr = fmt.Errorf("%s (exit %d)", result.Output, result.ExitCode)
			}
		} else {
			checkErr = errors.New("no mise task 'arch:check'")
		}

		rep.PrintCheck(svc.Name, "Validating service architecture boundaries", checkErr)
	}

	// Custom cross-cutting validators
	validators := []Validator{
		&custom.ContractPurityValidator{
			ContractsDir: repoRoot + "/contracts/definitions",
			RepoRoot:     repoRoot,
		},
		&custom.LibDependencyValidator{
			LibsDir:  repoRoot + "/libs",
			MmwDir:   repoRoot + "/mmw",
			RepoRoot: repoRoot,
		},
		&custom.DomainPurityValidator{ModulesDir: repoRoot + "/modules"},
		&custom.ApplicationPurityValidator{ModulesDir: repoRoot + "/modules"},
	}
	for _, v := range validators {
		rep.PrintCheck(v.Name(), v.Description(), v.Check())
	}

	return rep.Summary()
}
