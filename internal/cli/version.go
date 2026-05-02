package cli

import (
	"io"

	"github.com/alekseitsvetkov/dem/internal/output"
	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

type versionInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func newVersionCommand(out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print dem version information as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			return output.WriteJSON(out, versionInfo{
				Name:    "dem",
				Version: Version,
				Commit:  Commit,
				Date:    Date,
			}, nil)
		},
	}
}
