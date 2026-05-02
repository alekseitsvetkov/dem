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

func TestGetResultsSuccess(t *testing.T) {
	fixtureData, err := os.ReadFile("../hltv/parser/testdata/results.html")
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
	p := NewResultsProvider(WithResultsClient(client))

	results, err := p.GetResults(context.Background(), 0)
	if err != nil {
		t.Fatalf("GetResults returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].MatchID != "333" {
		t.Fatalf("results[0].MatchID = %q, want %q", results[0].MatchID, "333")
	}
	if results[0].Team1 != "Team Alpha" {
		t.Fatalf("results[0].Team1 = %q, want %q", results[0].Team1, "Team Alpha")
	}
	if results[0].Score != "2-1" {
		t.Fatalf("results[0].Score = %q, want %q", results[0].Score, "2-1")
	}
}

func TestGetResultsWithLimit(t *testing.T) {
	fixtureData, err := os.ReadFile("../hltv/parser/testdata/results.html")
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
	p := NewResultsProvider(WithResultsClient(client))

	results, err := p.GetResults(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetResults returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].MatchID != "333" {
		t.Fatalf("results[0].MatchID = %q, want %q", results[0].MatchID, "333")
	}
}

func TestGetResultsNetworkError(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("network failure")
	})
	client := hltv.NewClient(hltv.WithHTTPClient(&http.Client{Transport: transport}))
	p := NewResultsProvider(WithResultsClient(client))

	_, err := p.GetResults(context.Background(), 0)
	if err == nil {
		t.Fatalf("GetResults returned nil error")
	}

	var providerErr *hltv.ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %T, want *hltv.ProviderError", err)
	}
	if providerErr.Code != hltv.ErrorCodeNetwork {
		t.Fatalf("code = %q, want %q", providerErr.Code, hltv.ErrorCodeNetwork)
	}
}
