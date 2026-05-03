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

func newEventsCommand(out io.Writer, errOut io.Writer, p provider.EventsProvider) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "List HLTV events as JSON",
		Long: "Fetch events from HLTV, optionally filter by tier, and return them as a JSON array. " +
			"When --tier is omitted, all events are returned unfiltered. " +
			"When --limit is 0 (default), all events from the page are returned.",
		RunE: func(cmd *cobra.Command, args []string) error {
			tier, _ := cmd.Flags().GetString("tier")
			limit, _ := cmd.Flags().GetInt("limit")

			// D-09: Validation before any network access
			if limit < 0 {
				_ = output.WriteErrorJSON(errOut, "validation_error",
					"--limit must be greater than or equal to 0",
					map[string]any{"flag": "limit", "value": limit})
				return fmt.Errorf("--limit must be >= 0")
			}

			events, err := p.GetEvents(cmd.Context(), tier, limit)
			if err != nil {
				_ = mapEventsError(errOut, err)
				return err
			}

			if events == nil {
				events = []domain.Event{} // ensure JSON array, not null
			}
			return output.WriteJSON(out, events, nil)
		},
	}

	cmd.Flags().String("tier", "", `Filter events by tier. Use "1" for tier-1 heuristic (prize pool > $250K or known organizer keywords). Also supports exact match like "Intl. LAN", "Online", "Major"`)
	cmd.Flags().Int("limit", 0, "Maximum number of events to return (0 = no limit)")
	return cmd
}

// mapEventsError maps provider and parser errors to JSON error envelopes on stderr.
func mapEventsError(w io.Writer, err error) error {
	switch e := err.(type) {
	case *hltv.ProviderError:
		return output.WriteErrorJSON(w, e.Code, e.Message, e.Details())
	case *parser.ParseError:
		return output.WriteErrorJSON(w, e.Code, e.Message, e.Details())
	default:
		return output.WriteErrorJSON(w, "internal_error", err.Error(), nil)
	}
}
