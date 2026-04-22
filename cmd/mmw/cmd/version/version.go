package version

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Version is the only value set via -ldflags at build time:
//
//	go build -ldflags "-X github.com/piprim/mmw/cmd/mmw/cmd/version.Version=v1.2.3"
//
// Commit and BuildTime are read from the VCS info embedded by go build (Go 1.18+).
var Version = "dev"

const hashLen = 7

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the mmw version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(buildInfo())
		},
	}
}

func buildInfo() string {
	commit, commitTime, dirty := vcsInfo()
	if commit == "" {
		return Version
	}
	if dirty {
		return fmt.Sprintf("%s commit=%s* built=%s", Version, commit, commitTime)
	}

	return fmt.Sprintf("%s commit=%s built=%s", Version, commit, commitTime)
}

func vcsInfo() (commit, commitTime string, dirty bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", "", false
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) > hashLen {
				commit = s.Value[:hashLen]
			} else {
				commit = s.Value
			}
		case "vcs.time":
			commitTime = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}

	return commit, commitTime, dirty
}
