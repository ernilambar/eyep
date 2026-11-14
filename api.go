package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const (
	// Network timeouts so an unresponsive endpoint cannot hang the CLI.
	connectTimeout = 5 * time.Second
	requestTimeout = 10 * time.Second
)

const (
	ipifyEndpoint  = "https://api64.ipify.org"
	lookupEndpoint = "http://ip-api.com/json/"
	lookupFields   = "status,message,country,countryCode,regionName,city,zip,lat,lon,timezone,isp,org,as,query"
)

// backend abstracts the two network calls so CLI behaviour can be tested
// without a network.
type backend interface {
	discover() (string, error)
	lookup(ip string) (string, error)
}

type httpBackend struct {
	ipifyURL  string
	lookupURL func(ip string) string
	client    *http.Client
}

// newHTTPBackend builds a backend pinned to the given dial network ("tcp4" or
// "tcp6"), mirroring curl's -4/-6.
func newHTTPBackend(family string) *httpBackend {
	if family == "" {
		family = "tcp4"
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	dialer := newDialer()
	transport.DialContext = func(ctx context.Context, _, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, family, addr)
	}
	return &httpBackend{
		ipifyURL: ipifyEndpoint,
		lookupURL: func(ip string) string {
			return lookupEndpoint + ip + "?fields=" + lookupFields
		},
		client: &http.Client{Timeout: requestTimeout, Transport: transport},
	}
}

func newDialer() *net.Dialer {
	return &net.Dialer{Timeout: connectTimeout, KeepAlive: 30 * time.Second}
}

func (b *httpBackend) discover() (string, error) {
	return b.get(b.ipifyURL)
}

func (b *httpBackend) lookup(ip string) (string, error) {
	return b.get(b.lookupURL(ip))
}

func (b *httpBackend) get(url string) (string, error) {
	resp, err := b.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// curl -f fails on HTTP errors.
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
