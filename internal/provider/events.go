package provider

import (
	"bytes"
	"context"
	"strings"

	"github.com/alekseitsvetkov/dem/internal/domain"
	"github.com/alekseitsvetkov/dem/internal/hltv"
	"github.com/alekseitsvetkov/dem/internal/hltv/parser"
)

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
func filterByTier(events []domain.Event, tier string) []domain.Event {
	tier = strings.TrimSpace(tier)
	if tier == "" {
		return events
	}
	var filtered []domain.Event
	for _, e := range events {
		if strings.EqualFold(strings.TrimSpace(e.Tier), tier) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
