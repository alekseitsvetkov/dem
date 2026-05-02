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
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].ID != "111" {
		t.Fatalf("events[0].ID = %q, want %q", events[0].ID, "111")
	}
	if events[0].Name != "Test Event 1" {
		t.Fatalf("events[0].Name = %q, want %q", events[0].Name, "Test Event 1")
	}
	if events[0].Tier != "S-Tier" {
		t.Fatalf("events[0].Tier = %q, want %q", events[0].Tier, "S-Tier")
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

	events, err := p.GetEvents(context.Background(), "S-Tier", 0)
	if err != nil {
		t.Fatalf("GetEvents returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].ID != "111" {
		t.Fatalf("events[0].ID = %q, want %q", events[0].ID, "111")
	}
	if events[0].Tier != "S-Tier" {
		t.Fatalf("events[0].Tier = %q, want %q", events[0].Tier, "S-Tier")
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
	if events[0].ID != "111" {
		t.Fatalf("events[0].ID = %q, want %q", events[0].ID, "111")
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
