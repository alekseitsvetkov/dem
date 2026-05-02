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

func TestGetDemo_Success(t *testing.T) {
	fixtureData, err := os.ReadFile("../hltv/parser/testdata/match-with-demo.html")
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
	p := NewDemoProvider(WithDemoClient(client))

	link, err := p.GetDemo(context.Background(), 107224)
	if err != nil {
		t.Fatalf("GetDemo returned error: %v", err)
	}
	if link.MatchID != "107224" {
		t.Fatalf("MatchID = %q, want %q", link.MatchID, "107224")
	}
	if !strings.Contains(link.DemoURL, "https://www.hltv.org/download/demo/107224") {
		t.Fatalf("DemoURL = %q, want to contain %q", link.DemoURL, "https://www.hltv.org/download/demo/107224")
	}
	if link.MatchURL != "https://www.hltv.org/matches/107224/-" {
		t.Fatalf("MatchURL = %q, want %q", link.MatchURL, "https://www.hltv.org/matches/107224/-")
	}
}

func TestGetDemo_Unavailable(t *testing.T) {
	fixtureData, err := os.ReadFile("../hltv/parser/testdata/match-without-demo.html")
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
	p := NewDemoProvider(WithDemoClient(client))

	link, err := p.GetDemo(context.Background(), 99999)
	if err != nil {
		t.Fatalf("GetDemo returned error for unavailable demo: %v", err)
	}
	if link.DemoURL != "" {
		t.Fatalf("DemoURL = %q, want empty", link.DemoURL)
	}
	if link.MatchID == "" {
		t.Fatalf("MatchID is empty, want non-empty partial DemoLink context")
	}
	if !strings.Contains(link.MatchURL, "https://www.hltv.org/matches/99999/-") {
		t.Fatalf("MatchURL = %q, want to contain %q", link.MatchURL, "https://www.hltv.org/matches/99999/-")
	}
}

func TestGetDemo_NetworkError(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("network failure")
	})
	client := hltv.NewClient(hltv.WithHTTPClient(&http.Client{Transport: transport}))
	p := NewDemoProvider(WithDemoClient(client))

	_, err := p.GetDemo(context.Background(), 107224)
	if err == nil {
		t.Fatalf("GetDemo returned nil error")
	}

	var providerErr *hltv.ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %T, want *hltv.ProviderError", err)
	}
	if providerErr.Code != hltv.ErrorCodeNetwork {
		t.Fatalf("code = %q, want %q", providerErr.Code, hltv.ErrorCodeNetwork)
	}
}
