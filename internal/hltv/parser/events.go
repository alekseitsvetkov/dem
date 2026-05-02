package parser

import (
	"io"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/alekseitsvetkov/dem/internal/domain"
)

const hltvBaseURL = "https://www.hltv.org"

// ParseEvents parses an HLTV events page HTML document into typed domain.Event values.
// Required fields: event ID (data-event-id attribute on .event) and name (text of .event-name link).
// If either is missing, a *ParseError with ErrorCodeParse is returned.
// Tier extraction from .event-tier is best-effort: if the element is absent, Tier is empty string.
func ParseEvents(r io.Reader, sourceURL string) ([]domain.Event, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, &ParseError{Code: ErrorCodeParse, Area: "events", Err: err}
	}

	var events []domain.Event
	var parseErr error

	doc.Find(".event").EachWithBreak(func(i int, s *goquery.Selection) bool {
		id, exists := s.Attr("data-event-id")
		if !exists || strings.TrimSpace(id) == "" {
			parseErr = &ParseError{
				Code:    ErrorCodeParse,
				Area:    "events",
				Field:   "id",
				Message: "missing event ID",
			}
			return false
		}

		nameEl := s.Find("a.event-name")
		nameText := strings.TrimSpace(nameEl.Text())
		if nameText == "" {
			parseErr = &ParseError{
				Code:    ErrorCodeParse,
				Area:    "events",
				Field:   "name",
				Message: "missing event name",
			}
			return false
		}

		tierText := strings.TrimSpace(s.Find(".event-tier").Text())

		dateText := strings.TrimSpace(s.Find(".event-date").Text())
		startDate, endDate := parseDateRange(dateText)

		location := strings.TrimSpace(s.Find(".event-location").Text())

		resolvedURL := sourceURL
		if href, hrefExists := nameEl.Attr("href"); hrefExists && href != "" {
			if resolved := resolveURL(href); resolved != "" {
				resolvedURL = resolved
			}
		}

		events = append(events, domain.Event{
			ID:        id,
			Name:      nameText,
			StartDate: startDate,
			EndDate:   endDate,
			Location:  location,
			Tier:      tierText,
			SourceURL: resolvedURL,
		})
		return true
	})

	if parseErr != nil {
		return nil, parseErr
	}
	if events == nil {
		events = []domain.Event{}
	}
	return events, nil
}

// parseDateRange splits a date range string like "2025-01-15 to 2025-01-20"
// into start and end date strings. If no split is possible, the full text
// is assigned to StartDate and EndDate is empty.
func parseDateRange(dateText string) (string, string) {
	if dateText == "" {
		return "", ""
	}
	parts := strings.SplitN(dateText, " to ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return dateText, ""
}

// resolveURL resolves a relative href against the HLTV base URL.
// If the input is already absolute, it is returned as-is.
// Returns empty string if parsing fails.
func resolveURL(href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	base, err := url.Parse(hltvBaseURL)
	if err != nil {
		return ""
	}
	rel, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return base.ResolveReference(rel).String()
}
