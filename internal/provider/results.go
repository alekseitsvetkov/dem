package provider

import (
	"bytes"
	"context"

	"github.com/alekseitsvetkov/dem/internal/domain"
	"github.com/alekseitsvetkov/dem/internal/hltv"
	"github.com/alekseitsvetkov/dem/internal/hltv/parser"
)

// ResultsProvider is the interface for fetching and limiting results.
type ResultsProvider interface {
	GetResults(ctx context.Context, limit int) ([]domain.Result, error)
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

// GetResults fetches results from HLTV and truncates to the given limit.
// A limit of 0 or negative means no truncation (return all).
func (p *resultsProvider) GetResults(ctx context.Context, limit int) ([]domain.Result, error) {
	body, err := p.client.Fetch(ctx, p.urls.ResultsURL())
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
