package hltv

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	client := NewClient()

	if client.userAgent != DefaultUserAgent {
		t.Fatalf("userAgent = %q, want %q", client.userAgent, DefaultUserAgent)
	}
	if client.httpClient.Timeout != DefaultTimeout {
		t.Fatalf("timeout = %s, want %s", client.httpClient.Timeout, DefaultTimeout)
	}
}

func TestFetchSendsUserAgent(t *testing.T) {
	var gotUserAgent string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotUserAgent = r.Header.Get("User-Agent")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})

	client := NewClient(
		WithUserAgent("dem/test"),
		WithHTTPClient(&http.Client{Transport: transport}),
	)
	body, err := client.Fetch(context.Background(), "https://example.test/events")
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if string(body) != "ok" {
		t.Fatalf("body = %q, want ok", string(body))
	}
	if gotUserAgent != "dem/test" {
		t.Fatalf("User-Agent = %q, want dem/test", gotUserAgent)
	}
}

func TestFetchUsesInjectedHTTPClient(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("from fake transport")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})
	client := NewClient(WithHTTPClient(&http.Client{Transport: transport}))

	body, err := client.Fetch(context.Background(), "https://example.test/events")
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if string(body) != "from fake transport" {
		t.Fatalf("body = %q, want fake transport body", string(body))
	}
}

func TestFetchMapsHTTPStatusError(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Body:       io.NopCloser(strings.NewReader("unavailable")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	})

	client := NewClient(WithHTTPClient(&http.Client{Transport: transport}))
	_, err := client.Fetch(context.Background(), "https://example.test/results")
	if err == nil {
		t.Fatalf("Fetch returned nil error")
	}

	var providerErr *ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %T, want *ProviderError", err)
	}
	if providerErr.Code != ErrorCodeHTTP {
		t.Fatalf("code = %q, want %q", providerErr.Code, ErrorCodeHTTP)
	}
	if providerErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", providerErr.StatusCode)
	}
}

func TestFetchMapsInvalidURLToNetworkError(t *testing.T) {
	client := NewClient()
	_, err := client.Fetch(context.Background(), "\n")
	if err == nil {
		t.Fatalf("Fetch returned nil error")
	}

	var providerErr *ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %T, want *ProviderError", err)
	}
	if providerErr.Code != ErrorCodeNetwork {
		t.Fatalf("code = %q, want %q", providerErr.Code, ErrorCodeNetwork)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
