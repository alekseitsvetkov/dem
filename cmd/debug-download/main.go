package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	utls "github.com/refraction-networking/utls"
)

const testURL = "https://www.hltv.org/download/demo/107729"

func main() {
	matchURL := os.Args[1]
	fmt.Printf("Testing download URL: %s\n", matchURL)
	fmt.Println("---")

	// Test 1: Standard Go TLS, HTTP/1.1 only
	fmt.Println("Test 1: crypto/tls with NextProtos: [http/1.1]")
	testStandardHTTP1(matchURL)

	// Test 2: uTLS Chrome, HTTP/1.1 only (NextProtos override)
	fmt.Println("Test 2: uTLS HelloChrome_131 with NextProtos: [http/1.1]")
	testUTLS(matchURL, utls.HelloChrome_131, []string{"http/1.1"})

	// Test 3: uTLS Firefox, HTTP/1.1 only
	fmt.Println("Test 3: uTLS HelloFirefox_Auto with NextProtos: [http/1.1]")
	testUTLS(matchURL, utls.HelloFirefox_Auto, []string{"http/1.1"})

	// Test 4: uTLS Chrome, no NextProtos (default ALPN from fingerprint)
	fmt.Println("Test 4: uTLS HelloChrome_131, default ALPN (includes h2)")
	testUTLS(matchURL, utls.HelloChrome_131, nil)
}

func testStandardHTTP1(url string) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			NextProtos: []string{"http/1.1"},
		},
		TLSNextProto: make(map[string]func(string, *tls.Conn) http.RoundTripper),
	}
	client := &http.Client{Transport: transport, Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	printResult("crypto/tls http1.1", resp, err)
}

func testUTLS(url string, hello utls.ClientHelloID, nextProtos []string) {
	transport := &http.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: 10 * time.Second}
			conn, err := dialer.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			serverName, _, _ := net.SplitHostPort(addr)
			cfg := &utls.Config{ServerName: serverName}
			if nextProtos != nil {
				cfg.NextProtos = nextProtos
			}
			uconn := utls.UClient(conn, cfg, hello)
			if err := uconn.HandshakeContext(ctx); err != nil {
				conn.Close()
				return nil, err
			}
			return uconn, nil
		},
		TLSNextProto: make(map[string]func(string, *tls.Conn) http.RoundTripper),
	}
	client := &http.Client{Transport: transport, Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	printResult(hello.Str()+" nextprotos="+fmt.Sprint(nextProtos), resp, err)
}

func printResult(label string, resp *http.Response, err error) {
	if err != nil {
		fmt.Printf("  %s: ERROR: %v\n", label, err)
		return
	}
	if resp != nil {
		defer resp.Body.Close()
		fmt.Printf("  %s: HTTP %d, Proto: %s\n", label, resp.StatusCode, resp.Proto)
		if loc, _ := resp.Location(); loc != nil {
			fmt.Printf("    Location: %s\n", loc.String())
		}
		// Print cloudflare challenge indicator
		if cfChallenge := resp.Header.Get("cf-mitigated"); cfChallenge != "" {
			fmt.Printf("    Cloudflare: %s\n", cfChallenge)
		}
	}
}
