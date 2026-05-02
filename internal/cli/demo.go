package cli

import (
	"fmt"
	"io"
	"strconv"

	"github.com/alekseitsvetkov/dem/internal/hltv"
	"github.com/alekseitsvetkov/dem/internal/hltv/parser"
	"github.com/alekseitsvetkov/dem/internal/output"
	"github.com/alekseitsvetkov/dem/internal/provider"
	"github.com/spf13/cobra"
)

func newDemoCommand(out io.Writer, errOut io.Writer, p provider.DemoProvider) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "demo <match-id>",
		Short: "Get demo download link for an HLTV match",
		Long: "Fetch an HLTV match page and return the demo download link as JSON. " +
			"When a demo is available, the response includes demo_url. " +
			"When no demo is available, demo_url is omitted (exit code 0). " +
			"Use 'dem help demo' for help (--help does not work on this command).",
		Args:               cobra.ExactArgs(1),
		DisableFlagParsing: true, // D-07: zero-flag command; prevents cobra from parsing args as flags
		RunE: func(cmd *cobra.Command, args []string) error {
			// D-05, D-06: Validation before any network access
			matchID, err := strconv.Atoi(args[0])
			if err != nil || matchID <= 0 {
				_ = output.WriteErrorJSON(errOut, "validation_error",
					"match-id must be a positive integer",
					map[string]any{"arg": args[0]})
				return fmt.Errorf("invalid match-id: %q", args[0])
			}

			link, err := p.GetDemo(cmd.Context(), matchID)
			if err != nil {
				_ = mapDemoError(errOut, err)
				return err
			}

			return output.WriteJSON(out, link, nil)
		},
	}
	return cmd
}

// mapDemoError maps provider and parser errors to JSON error envelopes on stderr.
func mapDemoError(w io.Writer, err error) error {
	switch e := err.(type) {
	case *hltv.ProviderError:
		return output.WriteErrorJSON(w, e.Code, e.Message, e.Details())
	case *parser.ParseError:
		return output.WriteErrorJSON(w, e.Code, e.Message, e.Details())
	default:
		return output.WriteErrorJSON(w, "internal_error", err.Error(), nil)
	}
}
