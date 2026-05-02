package provider

import (
	"bytes"
	"context"
	"errors"

	"github.com/alekseitsvetkov/dem/internal/domain"
	"github.com/alekseitsvetkov/dem/internal/hltv"
	"github.com/alekseitsvetkov/dem/internal/hltv/parser"
)

// DemoProvider is the interface for fetching a demo download link for an HLTV match.
type DemoProvider interface {
	GetDemo(ctx context.Context, matchID int) (domain.DemoLink, error)
}

type demoProvider struct {
	client *hltv.Client
	urls   hltv.URLs
}

// DemoProviderOption is a functional option for configuring DemoProvider.
type DemoProviderOption func(*demoProvider)

// NewDemoProvider creates a new DemoProvider with the given options.
func NewDemoProvider(opts ...DemoProviderOption) DemoProvider {
	p := &demoProvider{
		client: hltv.NewClient(),
		urls:   hltv.NewURLs(""),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// WithDemoClient sets the HTTP client used by the provider.
func WithDemoClient(c *hltv.Client) DemoProviderOption {
	return func(p *demoProvider) { p.client = c }
}

// WithDemoURLs sets the URL helper used by the provider.
func WithDemoURLs(u hltv.URLs) DemoProviderOption {
	return func(p *demoProvider) { p.urls = u }
}

// GetDemo fetches the HLTV match page for the given match ID and extracts
// the demo download link. When a demo is available, the returned DemoLink
// includes demo_url with the full absolute download URL. When no demo is
// available, the method returns a partial DemoLink (with DemoURL empty)
// and nil error. The caller can check link.DemoURL == "" to detect
// unavailable demos.
func (p *demoProvider) GetDemo(ctx context.Context, matchID int) (domain.DemoLink, error) {
	matchURL := p.urls.MatchURL(matchID)

	body, err := p.client.Fetch(ctx, matchURL)
	if err != nil {
		return domain.DemoLink{}, err // Pass through ProviderError without remapping
	}

	link, err := parser.ParseDemoLink(bytes.NewReader(body), matchURL)
	if err != nil {
		var parseErr *parser.ParseError
		if errors.As(err, &parseErr) && parseErr.Code == parser.ErrorCodeUnavailableData {
			// D-03: Unavailable data → return success with partial DemoLink
			return link, nil
		}
		return domain.DemoLink{}, err // Pass through other ParseErrors
	}

	return link, nil
}
