package parser

import (
	"os"
	"strings"
	"testing"
)

func TestParseEvents_ParsesFixture(t *testing.T) {
	f, err := os.Open("testdata/events.html")
	if err != nil {
		t.Fatalf("failed to open fixture: %v", err)
	}
	defer f.Close()

	events, err := ParseEvents(f, "")
	if err != nil {
		t.Fatalf("ParseEvents returned error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	// First event: IEM Rio 2026
	e1 := events[0]
	if e1.ID != "8242" {
		t.Errorf("expected event ID '8242', got '%s'", e1.ID)
	}
	if e1.Name != "IEM Rio 2026" {
		t.Errorf("expected name 'IEM Rio 2026', got '%s'", e1.Name)
	}
	if e1.Tier != "Intl. LAN" {
		t.Errorf("expected tier 'Intl. LAN', got '%s'", e1.Tier)
	}
	if e1.StartDate != "Apr 13th" {
		t.Errorf("expected start date 'Apr 13th', got '%s'", e1.StartDate)
	}
	if e1.EndDate != "Apr 19th" {
		t.Errorf("expected end date 'Apr 19th', got '%s'", e1.EndDate)
	}
	if e1.Location != "Rio de Janeiro, Brazil" {
		t.Errorf("expected location 'Rio de Janeiro, Brazil', got '%s'", e1.Location)
	}
	if !strings.HasPrefix(e1.SourceURL, "https://www.hltv.org/events/8242") {
		t.Errorf("expected sourceURL to start with 'https://www.hltv.org/events/8242', got '%s'", e1.SourceURL)
	}

	// Second event: PGL Bucharest 2026
	e2 := events[1]
	if e2.ID != "8048" {
		t.Errorf("expected event ID '8048', got '%s'", e2.ID)
	}
	if e2.Name != "PGL Bucharest 2026" {
		t.Errorf("expected name 'PGL Bucharest 2026', got '%s'", e2.Name)
	}
	if e2.Tier != "Intl. LAN" {
		t.Errorf("expected tier 'Intl. LAN', got '%s'", e2.Tier)
	}

	// Third event: BLAST.tv Austin Major 2025
	e3 := events[2]
	if e3.ID != "7902" {
		t.Errorf("expected event ID '7902', got '%s'", e3.ID)
	}
	if e3.Name != "BLAST.tv Austin Major 2025" {
		t.Errorf("expected name 'BLAST.tv Austin Major 2025', got '%s'", e3.Name)
	}
	if e3.Tier != "Major" {
		t.Errorf("expected tier 'Major', got '%s'", e3.Tier)
	}
}

func TestParseEvents_ReturnsEmptySliceForNoEvents(t *testing.T) {
	input := `<html><body><p>No events here</p></body></html>`

	events, err := ParseEvents(strings.NewReader(input), "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if events == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestParseEvents_SkipsEntriesWithoutName(t *testing.T) {
	input := `<a href="/events/999/broken" class="a-reset small-event standard-box">
		<div class="text-ellipsis"></div>
	</a>`

	events, err := ParseEvents(strings.NewReader(input), "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestParseEvents_SkipsEntriesWithoutHref(t *testing.T) {
	input := `<a class="a-reset small-event standard-box">
		<div class="text-ellipsis">No Link Event</div>
	</a>`

	events, err := ParseEvents(strings.NewReader(input), "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestParseEvents_ExtractsEventIDFromURL(t *testing.T) {
	tests := []struct {
		href     string
		expected string
	}{
		{"/events/8242/iem-rio-2026", "8242"},
		{"/events/123/some-event", "123"},
		{"/events/5", "5"},
	}

	for _, tt := range tests {
		got := extractEventID(tt.href)
		if got != tt.expected {
			t.Errorf("extractEventID(%q) = %q, want %q", tt.href, got, tt.expected)
		}
	}
}
