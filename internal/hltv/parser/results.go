package parser

import (
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/alekseitsvetkov/dem/internal/domain"
)

// ParseResults parses an HLTV results page HTML document into typed domain.Result values.
// Handles the current HLTV structure where .result-con wraps an <a> with href containing the match URL.
func ParseResults(r io.Reader, sourceURL string) ([]domain.Result, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, &ParseError{Code: ErrorCodeParse, Area: "results", Err: err}
	}

	var results []domain.Result
	var parseErr error

	doc.Find(".result-con").EachWithBreak(func(i int, s *goquery.Selection) bool {
		// Extract match ID from the <a href="/matches/{id}/...">
		matchID := extractMatchIDFromSelection(s)
		if matchID == "" {
			parseErr = &ParseError{
				Code:    ErrorCodeParse,
				Area:    "results",
				Field:   "match_id",
				Message: "missing match ID",
			}
			return false
		}

		team1Text := strings.TrimSpace(s.Find(".team1 .team").First().Text())
		if team1Text == "" {
			parseErr = &ParseError{
				Code:    ErrorCodeParse,
				Area:    "results",
				Field:   "team1",
				Message: "missing team1",
			}
			return false
		}

		team2Text := strings.TrimSpace(s.Find(".team2 .team").First().Text())
		if team2Text == "" {
			parseErr = &ParseError{
				Code:    ErrorCodeParse,
				Area:    "results",
				Field:   "team2",
				Message: "missing team2",
			}
			return false
		}

		scoreText := strings.TrimSpace(s.Find(".result-score").Text())
		if scoreText == "" {
			parseErr = &ParseError{
				Code:    ErrorCodeParse,
				Area:    "results",
				Field:   "score",
				Message: "missing score",
			}
			return false
		}

		eventName := strings.TrimSpace(s.Find(".event-name").First().Text())
		date := strings.TrimSpace(s.Find(".date").First().Text())
		format := strings.TrimSpace(s.Find(".map-text").First().Text())

		resolvedURL := sourceURL
		s.Find("a").EachWithBreak(func(j int, a *goquery.Selection) bool {
			href, hrefExists := a.Attr("href")
			if hrefExists && strings.Contains(href, "/matches/") {
				if resolved := resolveURL(href); resolved != "" {
					resolvedURL = resolved
				}
				return false
			}
			return true
		})

		results = append(results, domain.Result{
			MatchID:   matchID,
			Team1:     team1Text,
			Team2:     team2Text,
			Score:     scoreText,
			Event:     eventName,
			Date:      date,
			Format:    format,
			SourceURL: resolvedURL,
		})
		return true
	})

	if parseErr != nil {
		return nil, parseErr
	}
	if results == nil {
		results = []domain.Result{}
	}
	return results, nil
}

// extractMatchIDFromSelection finds the first <a> with a /matches/ URL in the selection
// and extracts the numeric match ID from the path.
func extractMatchIDFromSelection(s *goquery.Selection) string {
	var matchID string
	s.Find("a").EachWithBreak(func(_ int, a *goquery.Selection) bool {
		href, exists := a.Attr("href")
		if !exists {
			return true
		}
		if id := extractMatchIDFromURL(href); id != "" {
			matchID = id
			return false
		}
		return true
	})
	return matchID
}

// extractMatchIDFromURL extracts the numeric match ID from an HLTV match URL path.
// Example: "/matches/2393245/spirit-vs-vitality" → "2393245"
func extractMatchIDFromURL(href string) string {
	href = strings.TrimSpace(href)
	parts := strings.Split(strings.Trim(href, "/"), "/")
	if len(parts) >= 2 && parts[0] == "matches" {
		if parts[1] != "" {
			return parts[1]
		}
	}
	return ""
}
