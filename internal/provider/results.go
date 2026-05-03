package provider

import (
	"bytes"
	"context"

	"github.com/alekseitsvetkov/dem/internal/domain"
	"github.com/alekseitsvetkov/dem/internal/hltv"
	"github.com/alekseitsvetkov/dem/internal/hltv/parser"
)

// ResultsProvider is the interface for fetching and limiting results.
// Pass eventID = 0 to fetch from the general results page.
type ResultsProvider interface {
	GetResults(ctx context.Context, eventID int, limit int) ([]domain.Result, error)
}

type resultsProvider struct {
	client *hltv.Client
	urls   hltv.URLs
}

// ResultsProviderOption is a functional option for configuring ResultsProvider.
type ResultsProviderOption func(*resultsProvider)

// NewResultsProvider creates a new ResultsProvider with the given options.
func NewResultsProvider(opts ...ResultsProviderOption) ResultsProvider {
	p := &resultsProvider{
		client: hltv.NewClient(),
		urls:   hltv.NewURLs(""),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithResultsClient sets the HTTP client used by the provider.
func WithResultsClient(c *hltv.Client) ResultsProviderOption {
	return func(p *resultsProvider) { p.client = c }
}

// WithResultsURLs sets the URL helper used by the provider.
func WithResultsURLs(u hltv.URLs) ResultsProviderOption {
	return func(p *resultsProvider) { p.urls = u }
}

// GetResults fetches results from HLTV, optionally for a specific event.
// Pass eventID = 0 to fetch from the general results page.
// A limit of 0 or negative means no truncation (return all).
func (p *resultsProvider) GetResults(ctx context.Context, eventID int, limit int) ([]domain.Result, error) {
	url := p.urls.ResultsURL()
	if eventID > 0 {
		url = p.urls.ResultsURLForEvent(eventID)
	}
	body, err := p.client.Fetch(ctx, url)
	if err != nil {
		return nil, err // Pass through ProviderError without remapping (D-03, D-08)
	}

	results, err := parser.ParseResults(bytes.NewReader(body), p.urls.ResultsURL())
	if err != nil {
		return nil, err // Pass through ParseError without remapping
	}

	if limit > 0 && limit < len(results) {
		results = results[:limit]
	}

	return results, nil
}
