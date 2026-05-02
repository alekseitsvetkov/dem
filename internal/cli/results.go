package cli

import (
	"fmt"
	"io"

	"github.com/alekseitsvetkov/dem/internal/domain"
	"github.com/alekseitsvetkov/dem/internal/hltv"
	"github.com/alekseitsvetkov/dem/internal/hltv/parser"
	"github.com/alekseitsvetkov/dem/internal/output"
	"github.com/alekseitsvetkov/dem/internal/provider"
	"github.com/spf13/cobra"
)

func newResultsCommand(out io.Writer, errOut io.Writer, p provider.ResultsProvider) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "results",
		Short: "List completed HLTV match results as JSON",
		Long: "Fetch completed match results from HLTV and return them as a JSON array. " +
			"When --limit is 0 (default), all results from the page are returned.",
		RunE: func(cmd *cobra.Command, args []string) error {
			limit, _ := cmd.Flags().GetInt("limit")

			// D-09: Validation before any network access
			if limit < 0 {
				_ = output.WriteErrorJSON(errOut, "validation_error",
					"--limit must be greater than or equal to 0",
					map[string]any{"flag": "limit", "value": limit})
				return fmt.Errorf("--limit must be >= 0")
			}

			results, err := p.GetResults(cmd.Context(), limit)
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
