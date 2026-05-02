package hltv

import (
	"errors"
	"testing"
)

func TestProviderErrorDetailsIncludesURL(t *testing.T) {
	err := &ProviderError{Code: ErrorCodeNetwork, URL: "https://example.test"}

	details := err.Details()

	if details["url"] != "https://example.test" {
		t.Fatalf("url detail = %v, want https://example.test", details["url"])
	}
	if _, ok := details["status_code"]; ok {
		t.Fatalf("status_code detail present for network_error: %v", details)
	}
}

func TestProviderErrorDetailsIncludesStatusCode(t *testing.T) {
	err := &ProviderError{Code: ErrorCodeHTTP, StatusCode: 503}

	details := err.Details()

	if details["status_code"] != 503 {
		t.Fatalf("status_code detail = %v, want 503", details["status_code"])
	}
}

func TestProviderErrorUnwrap(t *testing.T) {
	cause := errors.New("transport failed")
	err := &ProviderError{Code: ErrorCodeNetwork, Err: cause}

	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is did not find wrapped cause")
	}
	if errors.Unwrap(err) != cause {
		t.Fatalf("errors.Unwrap = %v, want cause", errors.Unwrap(err))
	}
}
