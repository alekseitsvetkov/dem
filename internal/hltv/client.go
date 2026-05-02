package hltv

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	DefaultUserAgent = "dem/dev"
	DefaultTimeout   = 15 * time.Second
)

type Client struct {
	httpClient *http.Client
	userAgent  string
}

type ClientOption func(*Client)

func NewClient(opts ...ClientOption) *Client {
	client := &Client{
		httpClient: &http.Client{Timeout: DefaultTimeout},
		userAgent:  DefaultUserAgent,
	}

	for _, opt := range opts {
		opt(client)
	}

	if client.httpClient == nil {
		client.httpClient = &http.Client{Timeout: DefaultTimeout}
	}
	if client.userAgent == "" {
		client.userAgent = DefaultUserAgent
	}

	return client
}

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(client *Client) {
		client.httpClient = httpClient
	}
}

func WithUserAgent(userAgent string) ClientOption {
	return func(client *Client) {
		client.userAgent = userAgent
	}
}

func (c *Client) Fetch(ctx context.Context, pageURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, &ProviderError{
			Code:    ErrorCodeNetwork,
			Message: "create hltv request",
			URL:     pageURL,
			Err:     err,
		}
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &ProviderError{
			Code:    ErrorCodeNetwork,
			Message: "fetch hltv page",
			URL:     pageURL,
			Err:     err,
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ProviderError{
			Code:    ErrorCodeNetwork,
			Message: "read hltv response",
			URL:     pageURL,
			Err:     err,
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, &ProviderError{
			Code:       ErrorCodeHTTP,
			Message:    fmt.Sprintf("hltv returned status %d", resp.StatusCode),
			URL:        pageURL,
			StatusCode: resp.StatusCode,
		}
	}

	return body, nil
}
