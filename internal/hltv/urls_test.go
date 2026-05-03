package hltv

import "testing"

func TestURLsUseDefaultBaseURL(t *testing.T) {
	urls := NewURLs("")

	if got := urls.EventsURL(); got != "https://www.hltv.org/events/archive" {
		t.Fatalf("EventsURL() = %q, want default events URL", got)
	}
	if got := urls.ResultsURL(); got != "https://www.hltv.org/results" {
		t.Fatalf("ResultsURL() = %q, want default results URL", got)
	}
	if got := urls.MatchURL(123); got != "https://www.hltv.org/matches/123/-" {
		t.Fatalf("MatchURL() = %q, want default match URL", got)
	}
}

func TestURLsUseCustomBaseURL(t *testing.T) {
	urls := NewURLs("http://127.0.0.1:8080")

	if got := urls.EventsURL(); got != "http://127.0.0.1:8080/events/archive" {
		t.Fatalf("EventsURL() = %q, want custom events URL", got)
	}
}

func TestURLsTrimTrailingSlash(t *testing.T) {
	urls := NewURLs("http://127.0.0.1:8080/")

	if got := urls.MatchURL(123); got != "http://127.0.0.1:8080/matches/123/-" {
		t.Fatalf("MatchURL() = %q, want trailing slash trimmed", got)
	}
}
