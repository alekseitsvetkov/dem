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

// ParseEvents parses an HLTV events page into typed domain.Event values.
// Matches the live HLTV HTML structure using .a-reset.small-event selectors.
func ParseEvents(r io.Reader, sourceURL string) ([]domain.Event, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, &ParseError{Code: ErrorCodeParse, Area: "events", Err: err}
	}

	var events []domain.Event

	doc.Find("a.a-reset.small-event").Each(func(_ int, s *goquery.Selection) {
		href, hrefExists := s.Attr("href")
		if !hrefExists || strings.TrimSpace(href) == "" {
			return // skip entries without href
		}

		eventID := extractEventID(href)
		name := strings.TrimSpace(s.Find(".text-ellipsis").First().Text())
		if name == "" {
			return // skip entries without a name
		}

		// Tier/type is in the .gtSmartphone-only cell of the first row
		tier := strings.TrimSpace(s.Find(".gtSmartphone-only").First().Text())

		// Prize pool is in the .prizePoolEllipsis cell, text like "$1,000,000"
		prizePool := 0
		if prizeText := strings.TrimSpace(s.Find(".prizePoolEllipsis").First().Text()); prizeText != "" {
			cleaned := strings.NewReplacer("$", "", ",", "").Replace(prizeText)
			if n, err := strconv.Atoi(cleaned); err == nil {
				prizePool = n
			}
		}

		// Location is in .smallCountry .col-desc (first col-desc in eventDetails)
		location := ""
		locSpan := s.Find(".smallCountry .col-desc").First()
		if locSpan.Length() > 0 {
			location = strings.TrimSuffix(strings.TrimSpace(locSpan.Text()), "|")
			location = strings.TrimSpace(location)
		}

		// Dates from span[data-unix] inside eventDetails row
		dateSpans := s.Find("span[data-unix]")
		startDate := ""
		endDate := ""
		if dateSpans.Length() >= 1 {
			startDate = strings.TrimSpace(dateSpans.First().Text())
		}
		if dateSpans.Length() >= 2 {
			endDate = strings.TrimSpace(dateSpans.Eq(1).Text())
		}

		resolvedURL := "https://www.hltv.org" + href
		if sourceURL != "" {
			resolvedURL = sourceURL
		}

		events = append(events, domain.Event{
			ID:        eventID,
			Name:      name,
			StartDate: startDate,
			EndDate:   endDate,
			Location:  location,
			Tier:      tier,
			PrizePool: prizePool,
			SourceURL: resolvedURL,
		})
	})

	if events == nil {
		events = []domain.Event{}
	}
	return events, nil
}

// extractEventID extracts the numeric event ID from an HLTV event URL path.
// Example: "/events/8242/iem-rio-2026" → "8242"
func extractEventID(href string) string {
	// href looks like "/events/8242/iem-rio-2026"
	parts := strings.Split(strings.Trim(href, "/"), "/")
	if len(parts) >= 2 && parts[0] == "events" {
		if _, err := strconv.Atoi(parts[1]); err == nil {
			return parts[1]
		}
	}
	// Fallback: use last path segment as ID
	return path.Base(href)
}

// resolveURL resolves a relative href against the HLTV base URL.
func resolveURL(href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	base, err := url.Parse("https://www.hltv.org")
	if err != nil {
		return ""
	}
	rel, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return base.ResolveReference(rel).String()
}
