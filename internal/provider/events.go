package provider

import (
	"bytes"
	"context"
	"strings"

	"github.com/alekseitsvetkov/dem/internal/domain"
	"github.com/alekseitsvetkov/dem/internal/hltv"
	"github.com/alekseitsvetkov/dem/internal/hltv/parser"
)

// tier1Keywords are event name keywords that identify tier 1 tournaments.
var tier1Keywords = []string{
	"IEM",
	"PGL",
	"Blast",
	"StarLadder",
	"FISSURE",
	"Esports World Cup",
	"Major",
	"BetBoom",
}

// EventsProvider is the interface for fetching and filtering events.
type EventsProvider interface {
	GetEvents(ctx context.Context, tier string, limit int) ([]domain.Event, error)
}

type eventsProvider struct {
	client *hltv.Client
	urls   hltv.URLs
}

// EventsProviderOption is a functional option for configuring EventsProvider.
type EventsProviderOption func(*eventsProvider)

// NewEventsProvider creates a new EventsProvider with the given options.
func NewEventsProvider(opts ...EventsProviderOption) EventsProvider {
	p := &eventsProvider{
		client: hltv.NewClient(),
		urls:   hltv.NewURLs(""),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithEventsClient sets the HTTP client used by the provider.
func WithEventsClient(c *hltv.Client) EventsProviderOption {
	return func(p *eventsProvider) { p.client = c }
}

// WithEventsURLs sets the URL helper used by the provider.
func WithEventsURLs(u hltv.URLs) EventsProviderOption {
	return func(p *eventsProvider) { p.urls = u }
}

// GetEvents fetches events from HLTV, optionally filters by tier,
// and truncates to the given limit.
// A limit of 0 or negative means no truncation (return all).
// An empty tier string means no filtering.
func (p *eventsProvider) GetEvents(ctx context.Context, tier string, limit int) ([]domain.Event, error) {
	body, err := p.client.Fetch(ctx, p.urls.EventsURL())
	if err != nil {
		return nil, err // Pass through ProviderError without remapping (D-03, D-08)
	}

	events, err := parser.ParseEvents(bytes.NewReader(body), p.urls.EventsURL())
	if err != nil {
		return nil, err // Pass through ParseError without remapping
	}

	if tier != "" {
		events = filterByTier(events, tier)
	}
	if limit > 0 && limit < len(events) {
		events = events[:limit]
	}

	return events, nil
}

// filterByTier filters events by tier using case-insensitive comparison.
// When tier is "1", uses a heuristic: prize pool > $250K or name matches
// known tier-1 organizer keywords.
func filterByTier(events []domain.Event, tier string) []domain.Event {
	tier = strings.TrimSpace(tier)
	if tier == "" {
		return events
	}

	// Special case: --tier 1 uses heuristic instead of exact match
	if tier == "1" {
		return filterTier1(events)
	}

	var filtered []domain.Event
	for _, e := range events {
		if strings.EqualFold(strings.TrimSpace(e.Tier), tier) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// filterTier1 filters events by the tier-1 heuristic:
// prize pool > $250,000 or name matches known organizer keywords.
func filterTier1(events []domain.Event) []domain.Event {
	var filtered []domain.Event
	for _, e := range events {
		if e.PrizePool > 250_000 || matchesKeyword(e.Name) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// matchesKeyword checks if the event name contains any known tier-1 keyword
// (case-insensitive).
func matchesKeyword(name string) bool {
	upper := strings.ToUpper(name)
	for _, kw := range tier1Keywords {
		if strings.Contains(upper, strings.ToUpper(kw)) {
			return true
		}
	}
	return false
}
