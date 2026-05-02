package parser

import (
	"os"
	"strings"
	"testing"
)

func TestParseDemoLink_WithDemo(t *testing.T) {
	f, err := os.Open("testdata/match-with-demo.html")
	if err != nil {
		t.Fatalf("failed to open fixture: %v", err)
	}
	defer f.Close()

	link, err := ParseDemoLink(f, "https://www.hltv.org/matches/107224/-")
	if err != nil {
		t.Fatalf("ParseDemoLink returned unexpected error: %v", err)
	}

	if link.DemoURL == "" {
		t.Errorf("expected non-empty DemoURL, got empty")
	}
	if !strings.Contains(link.DemoURL, "https://www.hltv.org/download/demo/107224") {
		t.Errorf("expected DemoURL to contain 'https://www.hltv.org/download/demo/107224', got '%s'", link.DemoURL)
	}
	if link.MatchID != "107224" {
		t.Errorf("expected MatchID '107224', got '%s'", link.MatchID)
	}
	if link.MatchURL != "https://www.hltv.org/matches/107224/-" {
		t.Errorf("expected MatchURL 'https://www.hltv.org/matches/107224/-', got '%s'", link.MatchURL)
	}
}

func TestParseDemoLink_WithoutDemo(t *testing.T) {
	f, err := os.Open("testdata/match-without-demo.html")
	if err != nil {
		t.Fatalf("failed to open fixture: %v", err)
	}
	defer f.Close()

	link, err := ParseDemoLink(f, "https://www.hltv.org/matches/99999/-")
	if err == nil {
		t.Fatal("expected error for match without demo, got nil")
	}

	if link.MatchID != "99999" {
		t.Errorf("expected MatchID '99999' in partial DemoLink, got '%s'", link.MatchID)
	}
	if link.MatchURL != "https://www.hltv.org/matches/99999/-" {
		t.Errorf("expected MatchURL 'https://www.hltv.org/matches/99999/-', got '%s'", link.MatchURL)
	}
	if link.DemoURL != "" {
		t.Errorf("expected empty DemoURL, got '%s'", link.DemoURL)
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}
	if parseErr.Code != ErrorCodeUnavailableData {
		t.Errorf("expected error code '%s', got '%s'", ErrorCodeUnavailableData, parseErr.Code)
	}
	if parseErr.Area != "demo" {
		t.Errorf("expected error area 'demo', got '%s'", parseErr.Area)
	}
}

func TestParseDemoLink_InvalidHTML(t *testing.T) {
	link, err := ParseDemoLink(strings.NewReader("not valid html <<>> {{{"), "https://www.hltv.org/matches/107224/-")

	// goquery handles malformed HTML gracefully, so this may return unavailable_data
	// instead of a parse error. Check both possible outcomes.
	if err == nil {
		// goquery parsed it; should not find a demo link in junk HTML
		if link.DemoURL != "" {
			t.Errorf("expected empty DemoURL from junk HTML, got '%s'", link.DemoURL)
		}
		return
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}
	if parseErr.Code != ErrorCodeParse && parseErr.Code != ErrorCodeUnavailableData {
		t.Errorf("expected error code '%s' or '%s', got '%s'", ErrorCodeParse, ErrorCodeUnavailableData, parseErr.Code)
	}
	if parseErr.Code == ErrorCodeParse && parseErr.Area != "demo" {
		t.Errorf("expected error area 'demo' for parse error, got '%s'", parseErr.Area)
	}
}

func TestParseDemoLink_EmptyBody(t *testing.T) {
	link, err := ParseDemoLink(strings.NewReader(""), "https://www.hltv.org/matches/107224/-")

	if err == nil {
		t.Fatal("expected error for empty body, got nil")
	}

	if link.DemoURL != "" {
		t.Errorf("expected empty DemoURL for empty body, got '%s'", link.DemoURL)
	}

	parseErr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T", err)
	}
	// goquery may handle empty input gracefully and fall through to
	// unavailable_data, or it may return a parse error. Accept both.
	if parseErr.Code != ErrorCodeParse && parseErr.Code != ErrorCodeUnavailableData {
		t.Errorf("expected error code '%s' or '%s', got '%s'", ErrorCodeParse, ErrorCodeUnavailableData, parseErr.Code)
	}
	if parseErr.Area != "demo" {
		t.Errorf("expected error area 'demo', got '%s'", parseErr.Area)
	}
}

func TestExtractMatchID(t *testing.T) {
	tests := []struct {
		href     string
		expected string
	}{
		{"https://www.hltv.org/matches/107224/esl-pro-league", "107224"},
		{"https://www.hltv.org/matches/12345/-", "12345"},
		{"https://www.hltv.org/matches/5", "5"},
		// url.Parse treats "not-a-url" as a valid path-only relative URL,
		// so extractMatchID returns the base path "not-a-url".
		{"not-a-url", "not-a-url"},
		// url.Parse("") yields Path="." — path.Base reflects that.
		{"", "."},
	}

	for _, tt := range tests {
		got := extractMatchID(tt.href)
		if got != tt.expected {
			t.Errorf("extractMatchID(%q) = %q, want %q", tt.href, got, tt.expected)
		}
	}
}
