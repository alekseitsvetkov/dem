package provider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/alekseitsvetkov/dem/internal/hltv"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestGetEventsSuccess(t *testing.T) {
	fixtureData, err := os.ReadFile("../hltv/parser/testdata/events.html")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(fixtureData))),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})
	client := hltv.NewClient(hltv.WithHTTPClient(&http.Client{Transport: transport}))
	p := NewEventsProvider(WithEventsClient(client))

	events, err := p.GetEvents(context.Background(), "", 0)
	if err != nil {
		t.Fatalf("GetEvents returned error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	if events[0].ID != "8242" {
		t.Fatalf("events[0].ID = %q, want %q", events[0].ID, "8242")
	}
	if events[0].Name != "IEM Rio 2026" {
		t.Fatalf("events[0].Name = %q, want %q", events[0].Name, "IEM Rio 2026")
	}
	if events[0].Tier != "Intl. LAN" {
		t.Fatalf("events[0].Tier = %q, want %q", events[0].Tier, "Intl. LAN")
	}
}

func TestGetEventsWithTierFilter(t *testing.T) {
	fixtureData, err := os.ReadFile("../hltv/parser/testdata/events.html")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(fixtureData))),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})
	client := hltv.NewClient(hltv.WithHTTPClient(&http.Client{Transport: transport}))
	p := NewEventsProvider(WithEventsClient(client))

	events, err := p.GetEvents(context.Background(), "Intl. LAN", 0)
	if err != nil {
		t.Fatalf("GetEvents returned error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].ID != "8242" {
		t.Fatalf("events[0].ID = %q, want %q", events[0].ID, "8242")
	}
	if events[0].Tier != "Intl. LAN" {
		t.Fatalf("events[0].Tier = %q, want %q", events[0].Tier, "Intl. LAN")
	}
}

func TestGetEventsWithLimit(t *testing.T) {
	fixtureData, err := os.ReadFile("../hltv/parser/testdata/events.html")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(fixtureData))),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})
	client := hltv.NewClient(hltv.WithHTTPClient(&http.Client{Transport: transport}))
	p := NewEventsProvider(WithEventsClient(client))

	events, err := p.GetEvents(context.Background(), "", 1)
	if err != nil {
		t.Fatalf("GetEvents returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].ID != "8242" {
		t.Fatalf("events[0].ID = %q, want %q", events[0].ID, "8242")
	}
}

func TestGetEventsTier1(t *testing.T) {
	fixtureData, err := os.ReadFile("../hltv/parser/testdata/events.html")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(string(fixtureData))),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})
	client := hltv.NewClient(hltv.WithHTTPClient(&http.Client{Transport: transport}))
	p := NewEventsProvider(WithEventsClient(client))

	events, err := p.GetEvents(context.Background(), "1", 0)
	if err != nil {
		t.Fatalf("GetEvents returned error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3 (all fixtures are tier-1 by heuristic)", len(events))
	}
}

func TestGetEventsNetworkError(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("network failure")
	})
	client := hltv.NewClient(hltv.WithHTTPClient(&http.Client{Transport: transport}))
	p := NewEventsProvider(WithEventsClient(client))

	_, err := p.GetEvents(context.Background(), "", 0)
	if err == nil {
		t.Fatalf("GetEvents returned nil error")
	}

	var providerErr *hltv.ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %T, want *hltv.ProviderError", err)
	}
	if providerErr.Code != hltv.ErrorCodeNetwork {
		t.Fatalf("code = %q, want %q", providerErr.Code, hltv.ErrorCodeNetwork)
	}
}
