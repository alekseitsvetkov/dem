package parser

import (
	"io"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/alekseitsvetkov/dem/internal/domain"
)

// ParseDemoLink parses a demo download URL from an HLTV match page HTML body.
//
// When a demo link is available, it returns a complete DemoLink with the resolved
// DemoURL and a nil error.
//
// When no demo is available, it returns a PARTIAL DemoLink (MatchID and MatchURL
// populated, DemoURL empty) alongside a non-nil *ParseError with code
// "unavailable_data". Callers should check the error AND can use the partial
// DemoLink for match context.
func ParseDemoLink(r io.Reader, matchURL string) (domain.DemoLink, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return domain.DemoLink{}, &ParseError{Code: ErrorCodeParse, Area: "demo", Err: err}
	}

	matchID := extractMatchID(matchURL)

	// Primary selector: [data-demo-link] attribute (live HLTV markup).
	if sel := doc.Find("[data-demo-link]"); sel.Length() > 0 {
		if href, ok := sel.First().Attr("data-demo-link"); ok && strings.TrimSpace(href) != "" {
			return domain.DemoLink{
				MatchID:  matchID,
				MatchURL: matchURL,
				DemoURL:  resolveURL(href),
			}, nil
		}
	}

	// Fallback selector: [data-manuel-download] attribute.
	if sel := doc.Find("[data-manuel-download]"); sel.Length() > 0 {
		if href, ok := sel.First().Attr("href"); ok && strings.TrimSpace(href) != "" {
			return domain.DemoLink{
				MatchID:  matchID,
				MatchURL: matchURL,
				DemoURL:  resolveURL(href),
			}, nil
		}
	}

	// No demo available — return partial result with context.
	return domain.DemoLink{
		MatchID:  matchID,
		MatchURL: matchURL,
	}, &ParseError{
		Code:    ErrorCodeUnavailableData,
		Area:    "demo",
		Message: "no demo available for this match",
	}
}

// extractMatchID extracts the numeric match ID from an HLTV match URL.
//
// Example: "https://www.hltv.org/matches/107224/esl-pro-league-season-21" → "107224"
//
// It parses the URL path, expecting /matches/<id>/... structure. If the path
// does not match the expected pattern, it falls back to the last path segment.
// On URL parse failure, it returns an empty string.
func extractMatchID(matchURL string) string {
	u, err := url.Parse(matchURL)
	if err != nil {
		return ""
	}
	// Path format: /matches/<id>/<slug>
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "matches" {
		if _, err := strconv.Atoi(parts[1]); err == nil {
			return parts[1]
		}
	}
	// Fallback: last path segment as ID
	return path.Base(u.Path)
}
