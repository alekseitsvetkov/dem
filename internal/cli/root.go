package cli

import (
	"io"
	"os"

	"github.com/alekseitsvetkov/dem/internal/output"
	"github.com/alekseitsvetkov/dem/internal/provider"
	"github.com/spf13/cobra"
)

func Execute() int {
	root := NewRootCommand(os.Stdout, os.Stderr)
	if err := root.Execute(); err != nil {
		_ = output.WriteErrorJSON(os.Stderr, "command_error", err.Error(), nil)
		return 1
	}

	return 0
}

func NewRootCommand(out io.Writer, errOut io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:           "dem",
		Short:         "Fetch HLTV events, results, and match demo links as JSON",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.SetOut(out)
	root.SetErr(errOut)
	root.AddCommand(newVersionCommand(out))
	root.AddCommand(newEventsCommand(out, errOut, provider.NewEventsProvider()))
	root.AddCommand(newResultsCommand(out, errOut, provider.NewResultsProvider()))
	root.AddCommand(newDemoCommand(out, errOut, provider.NewDemoProvider()))

	return root
}
