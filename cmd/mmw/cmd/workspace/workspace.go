package workspace

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "workspace",
		Short:         "Manage Go workspace modules",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(NewTidyCmd())
	cmd.AddCommand(NewStatusCmd())
	cmd.AddCommand(NewSyncCmd())

	return cmd
}

func workModules(root string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.work"))
	if err != nil {
		return nil, fmt.Errorf("read go.work: %w", err)
	}

	var modules []string

	inUse := false

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "use (":
			inUse = true
		case inUse && line == ")":
			inUse = false
		case inUse && line != "":
			modules = append(modules, filepath.Clean(line))
		case strings.HasPrefix(line, "use ") && !strings.HasSuffix(line, "("):
			modules = append(modules, filepath.Clean(strings.TrimSpace(strings.TrimPrefix(line, "use "))))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parse go.work: %w", err)
	}

	if len(modules) == 0 {
		return nil, errors.New("workspace: no modules found in go.work")
	}

	return modules, nil
}

type ioStreams struct {
	out    io.Writer
	errOut io.Writer
}

func runGoCmd(ctx context.Context, w ioStreams, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Stdout = w.out
	cmd.Stderr = w.errOut

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run go: %w", err)
	}

	return nil
}
