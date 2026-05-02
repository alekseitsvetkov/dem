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

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// First event
	e1 := events[0]
	if e1.ID != "111" {
		t.Errorf("expected event ID '111', got '%s'", e1.ID)
	}
	if e1.Name != "Test Event 1" {
		t.Errorf("expected name 'Test Event 1', got '%s'", e1.Name)
	}
	if e1.Tier != "S-Tier" {
		t.Errorf("expected tier 'S-Tier', got '%s'", e1.Tier)
	}
	if e1.StartDate != "2025-01-15" {
		t.Errorf("expected start date '2025-01-15', got '%s'", e1.StartDate)
	}
	if e1.EndDate != "2025-01-20" {
		t.Errorf("expected end date '2025-01-20', got '%s'", e1.EndDate)
	}
	if e1.Location != "Katowice, Poland" {
		t.Errorf("expected location 'Katowice, Poland', got '%s'", e1.Location)
	}
	if !strings.HasPrefix(e1.SourceURL, "https://www.hltv.org") {
		t.Errorf("expected sourceURL to start with 'https://www.hltv.org', got '%s'", e1.SourceURL)
	}

	// Second event
	e2 := events[1]
	if e2.ID != "222" {
		t.Errorf("expected event ID '222', got '%s'", e2.ID)
	}
	if e2.Name != "Another Event" {
		t.Errorf("expected name 'Another Event', got '%s'", e2.Name)
	}
	if e2.Tier != "A-Tier" {
		t.Errorf("expected tier 'A-Tier', got '%s'", e2.Tier)
	}
	if e2.StartDate != "2025-03-01" {
		t.Errorf("expected start date '2025-03-01', got '%s'", e2.StartDate)
	}
	if e2.EndDate != "2025-03-05" {
		t.Errorf("expected end date '2025-03-05', got '%s'", e2.EndDate)
	}
	if e2.Location != "Cologne, Germany" {
		t.Errorf("expected location 'Cologne, Germany', got '%s'", e2.Location)
	}
	if !strings.HasPrefix(e2.SourceURL, "https://www.hltv.org") {
		t.Errorf("expected sourceURL to start with 'https://www.hltv.org', got '%s'", e2.SourceURL)
	}
}

func TestParseEvents_MissingEventID(t *testing.T) {
	input := `<!DOCTYPE html><html><body>
		<div class="event">
			<a href="/events/999/no-id" class="event-name">No ID Event</a>
		</div>
	</body></html>`

	_, err := ParseEvents(strings.NewReader(input), "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	pe, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}
	if pe.Code != ErrorCodeParse {
		t.Errorf("expected Code 'parse_error', got '%s'", pe.Code)
	}
	if pe.Field != "id" {
		t.Errorf("expected Field 'id', got '%s'", pe.Field)
	}
}

func TestParseEvents_MissingEventName(t *testing.T) {
	input := `<!DOCTYPE html><html><body>
		<div class="event" data-event-id="999">
			<a href="/events/999/no-name" class="event-name"></a>
		</div>
	</body></html>`

	_, err := ParseEvents(strings.NewReader(input), "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	pe, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}
	if pe.Code != ErrorCodeParse {
		t.Errorf("expected Code 'parse_error', got '%s'", pe.Code)
	}
	if pe.Field != "name" {
		t.Errorf("expected Field 'name', got '%s'", pe.Field)
	}
}

func TestParseEvents_MissingTierIsNotError(t *testing.T) {
	input := `<!DOCTYPE html><html><body>
		<div class="event" data-event-id="555">
			<a href="/events/555/no-tier" class="event-name">No Tier Event</a>
		</div>
	</body></html>`

	events, err := ParseEvents(strings.NewReader(input), "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Tier != "" {
		t.Errorf("expected empty tier, got '%s'", events[0].Tier)
	}
}

func TestParseEvents_ReturnsEmptySliceForNoEvents(t *testing.T) {
	input := `<!DOCTYPE html><html><body><p>No events here</p></body></html>`

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
