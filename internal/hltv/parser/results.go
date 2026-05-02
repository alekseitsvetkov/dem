package parser

import (
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/alekseitsvetkov/dem/internal/domain"
)

// ParseResults parses an HLTV results page HTML document into typed domain.Result values.
// Required fields: match ID (data-match-id attribute on .result-con), team1, team2, score.
// Event, Date, and Format are optional and may be empty strings.
func ParseResults(r io.Reader, sourceURL string) ([]domain.Result, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, &ParseError{Code: ErrorCodeParse, Area: "results", Err: err}
	}

	var results []domain.Result
	var parseErr error

	doc.Find(".result-con").EachWithBreak(func(i int, s *goquery.Selection) bool {
		matchID, exists := s.Attr("data-match-id")
		if !exists || strings.TrimSpace(matchID) == "" {
			parseErr = &ParseError{
				Code:    ErrorCodeParse,
				Area:    "results",
				Field:   "match_id",
				Message: "missing match ID",
			}
			return false
		}

		team1Text := strings.TrimSpace(s.Find(".team1").Text())
		if team1Text == "" {
			parseErr = &ParseError{
				Code:    ErrorCodeParse,
				Area:    "results",
				Field:   "team1",
				Message: "missing team1",
			}
			return false
		}

		team2Text := strings.TrimSpace(s.Find(".team2").Text())
		if team2Text == "" {
			parseErr = &ParseError{
				Code:    ErrorCodeParse,
				Area:    "results",
				Field:   "team2",
				Message: "missing team2",
			}
			return false
		}

		scoreText := strings.TrimSpace(s.Find(".score").Text())
		if scoreText == "" {
			parseErr = &ParseError{
				Code:    ErrorCodeParse,
				Area:    "results",
				Field:   "score",
				Message: "missing score",
			}
			return false
		}

		eventName := strings.TrimSpace(s.Find(".event-name").Text())
		date := strings.TrimSpace(s.Find(".date").Text())
		format := strings.TrimSpace(s.Find(".format").Text())

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
			MatchID:  matchID,
			Team1:    team1Text,
			Team2:    team2Text,
			Score:    scoreText,
			Event:    eventName,
			Date:     date,
			Format:   format,
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
