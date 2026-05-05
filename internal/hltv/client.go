package hltv

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
	"golang.org/x/net/publicsuffix"
)

const (
	DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	DefaultTimeout   = 15 * time.Second
)

type Client struct {
	httpClient *http.Client
	userAgent  string
}

type ClientOption func(*Client)

func NewClient(opts ...ClientOption) *Client {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})

	// http2.Transport with Chrome-uTLS for HLTV main pages (poller).
	transport := &http2.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string, cfg *tls.Config) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: 10 * time.Second}
			conn, err := dialer.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			serverName, _, _ := net.SplitHostPort(addr)
			uconn := utls.UClient(conn, &utls.Config{ServerName: serverName}, utls.HelloChrome_131)
			if err := uconn.HandshakeContext(ctx); err != nil {
				conn.Close()
				return nil, err
			}
			return uconn, nil
		},
	}

	client := &Client{
		httpClient: &http.Client{
			Timeout:   DefaultTimeout,
			Transport: transport,
			Jar:       jar,
		},
		userAgent: DefaultUserAgent,
	}

	for _, opt := range opts {
		opt(client)
	}

	if client.httpClient == nil {
		jar2, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		client.httpClient = &http.Client{Timeout: DefaultTimeout, Jar: jar2}
	}
	if client.userAgent == "" {
		client.userAgent = DefaultUserAgent
	}

	return client
}

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(client *Client) { client.httpClient = httpClient }
}

func WithUserAgent(userAgent string) ClientOption {
	return func(client *Client) { client.userAgent = userAgent }
}

func (c *Client) Fetch(ctx context.Context, pageURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, &ProviderError{Code: ErrorCodeNetwork, Message: "create request", URL: pageURL, Err: err}
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &ProviderError{Code: ErrorCodeNetwork, Message: "fetch hltv page", URL: pageURL, Err: err}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ProviderError{Code: ErrorCodeNetwork, Message: "read hltv response", URL: pageURL, Err: err}
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
