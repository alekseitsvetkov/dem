package parser

import (
	"os"
	"strings"
	"testing"
)

func TestParseResults_ParsesFixture(t *testing.T) {
	f, err := os.Open("testdata/results.html")
	if err != nil {
		t.Fatalf("failed to open fixture: %v", err)
	}
	defer f.Close()

	results, err := ParseResults(f, "")
	if err != nil {
		t.Fatalf("ParseResults returned error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// First result
	r1 := results[0]
	if r1.MatchID != "333" {
		t.Errorf("expected MatchID '333', got '%s'", r1.MatchID)
	}
	if r1.Team1 != "Team Alpha" {
		t.Errorf("expected Team1 'Team Alpha', got '%s'", r1.Team1)
	}
	if r1.Team2 != "Team Beta" {
		t.Errorf("expected Team2 'Team Beta', got '%s'", r1.Team2)
	}
	if r1.Score != "2-1" {
		t.Errorf("expected Score '2-1', got '%s'", r1.Score)
	}
	if r1.Event != "Test Event 1" {
		t.Errorf("expected Event 'Test Event 1', got '%s'", r1.Event)
	}
	if r1.Date != "2025-01-20" {
		t.Errorf("expected Date '2025-01-20', got '%s'", r1.Date)
	}
	if r1.Format != "bo3" {
		t.Errorf("expected Format 'bo3', got '%s'", r1.Format)
	}
	if !strings.HasPrefix(r1.SourceURL, "https://www.hltv.org") {
		t.Errorf("expected sourceURL to start with 'https://www.hltv.org', got '%s'", r1.SourceURL)
	}

	// Second result
	r2 := results[1]
	if r2.MatchID != "444" {
		t.Errorf("expected MatchID '444', got '%s'", r2.MatchID)
	}
	if r2.Team1 != "Team Gamma" {
		t.Errorf("expected Team1 'Team Gamma', got '%s'", r2.Team1)
	}
	if r2.Team2 != "Team Delta" {
		t.Errorf("expected Team2 'Team Delta', got '%s'", r2.Team2)
	}
	if r2.Score != "16-10" {
		t.Errorf("expected Score '16-10', got '%s'", r2.Score)
	}
	if r2.Event != "Another Event" {
		t.Errorf("expected Event 'Another Event', got '%s'", r2.Event)
	}
	if r2.Date != "2025-03-05" {
		t.Errorf("expected Date '2025-03-05', got '%s'", r2.Date)
	}
	if r2.Format != "bo1" {
		t.Errorf("expected Format 'bo1', got '%s'", r2.Format)
	}
	if !strings.HasPrefix(r2.SourceURL, "https://www.hltv.org") {
		t.Errorf("expected sourceURL to start with 'https://www.hltv.org', got '%s'", r2.SourceURL)
	}
}

func TestParseResults_MissingTeam1(t *testing.T) {
	input := `<!DOCTYPE html><html><body>
		<div class="result-con" data-match-id="555">
			<span class="team2">Team Two</span>
			<span class="score">1-0</span>
		</div>
	</body></html>`

	_, err := ParseResults(strings.NewReader(input), "")
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
	if pe.Field != "team1" {
		t.Errorf("expected Field 'team1', got '%s'", pe.Field)
	}
}

func TestParseResults_MissingMatchID(t *testing.T) {
	input := `<!DOCTYPE html><html><body>
		<div class="result-con">
			<span class="team1">Team A</span>
			<span class="team2">Team B</span>
			<span class="score">1-0</span>
		</div>
	</body></html>`

	_, err := ParseResults(strings.NewReader(input), "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	pe, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}
	if pe.Field != "match_id" {
		t.Errorf("expected Field 'match_id', got '%s'", pe.Field)
	}
}

func TestParseResults_MissingOptionalFieldsAreEmpty(t *testing.T) {
	input := `<!DOCTYPE html><html><body>
		<div class="result-con" data-match-id="666">
			<span class="team1">Team X</span>
			<span class="team2">Team Y</span>
			<span class="score">0-0</span>
		</div>
	</body></html>`

	results, err := ParseResults(strings.NewReader(input), "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Event != "" {
		t.Errorf("expected empty Event, got '%s'", r.Event)
	}
	if r.Date != "" {
		t.Errorf("expected empty Date, got '%s'", r.Date)
	}
	if r.Format != "" {
		t.Errorf("expected empty Format, got '%s'", r.Format)
	}
	// SourceURL should fall back to the passed-in sourceURL
	if r.SourceURL != "" {
		t.Errorf("expected empty SourceURL (since no match link present), got '%s'", r.SourceURL)
	}
}

func TestParseResults_ReturnsEmptySliceForNoResults(t *testing.T) {
	input := `<!DOCTYPE html><html><body><p>No results here</p></body></html>`

	results, err := ParseResults(strings.NewReader(input), "")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if results == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
