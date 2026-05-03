package cli

import (
	"fmt"
	"io"
	"strconv"

	"github.com/alekseitsvetkov/dem/internal/domain"
	"github.com/alekseitsvetkov/dem/internal/hltv"
	"github.com/alekseitsvetkov/dem/internal/hltv/parser"
	"github.com/alekseitsvetkov/dem/internal/output"
	"github.com/alekseitsvetkov/dem/internal/provider"
	"github.com/spf13/cobra"
)

func newResultsCommand(out io.Writer, errOut io.Writer, p provider.ResultsProvider) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "results [event_id]",
		Short: "List completed HLTV match results as JSON",
		Long: "Fetch completed match results from HLTV and return them as a JSON array. " +
			"Pass an event ID to filter results by event. " +
			"When --limit is 0 (default), all results from the page are returned.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, _ := cmd.Flags().GetInt("limit")

			// D-09: Validation before any network access
			if limit < 0 {
				_ = output.WriteErrorJSON(errOut, "validation_error",
					"--limit must be greater than or equal to 0",
					map[string]any{"flag": "limit", "value": limit})
				return fmt.Errorf("--limit must be >= 0")
			}

			eventID := 0
			if len(args) > 0 {
				var err error
				eventID, err = strconv.Atoi(args[0])
				if err != nil {
					_ = output.WriteErrorJSON(errOut, "validation_error",
						"event ID must be a number",
						map[string]any{"arg": args[0]})
					return fmt.Errorf("invalid event ID: %q", args[0])
				}
			}

			results, err := p.GetResults(cmd.Context(), eventID, limit)
			if err != nil {
				_ = mapResultsError(errOut, err)
				return err
			}

			if results == nil {
				results = []domain.Result{} // ensure JSON array, not null
			}
			return output.WriteJSON(out, results, nil)
		},
	}

	cmd.Flags().Int("limit", 0, "Maximum number of results to return (0 = no limit)")
	return cmd
}

// mapResultsError maps provider and parser errors to JSON error envelopes on stderr.
func mapResultsError(w io.Writer, err error) error {
	switch e := err.(type) {
	case *hltv.ProviderError:
		return output.WriteErrorJSON(w, e.Code, e.Message, e.Details())
	case *parser.ParseError:
		return output.WriteErrorJSON(w, e.Code, e.Message, e.Details())
	default:
		return output.WriteErrorJSON(w, "internal_error", err.Error(), nil)
	}
}
