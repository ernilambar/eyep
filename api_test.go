package main

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testBackend(baseURL, family string) *httpBackend {
	b := newHTTPBackend(family)
	b.ipifyURL = baseURL
	b.lookupURL = func(ip string) string {
		return baseURL + "/json/" + ip + "?fields=" + lookupFields
	}
	return b
}

func TestRequestShape(t *testing.T) {
	var method, path, query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path, query = r.Method, r.URL.Path, r.URL.RawQuery
		_, _ = io.WriteString(w, "198.51.100.7")
	}))
	defer srv.Close()

	b := testBackend(srv.URL, "tcp4")
	body, err := b.discover()
	if err != nil {
		t.Fatalf("discover error = %v", err)
	}
	if body != "198.51.100.7" {
		t.Errorf("discover body = %q", body)
	}
	if method != http.MethodGet || path != "/" {
		t.Errorf("discovery request = %s %s, want GET /", method, path)
	}

	if _, err := b.lookup("8.8.8.8"); err != nil {
		t.Fatalf("lookup error = %v", err)
	}
	if path != "/json/8.8.8.8" {
		t.Errorf("lookup path = %q, want /json/8.8.8.8", path)
	}
	if query != "fields="+lookupFields {
		t.Errorf("lookup query = %q, want fields=%s", query, lookupFields)
	}
}

func TestLookupIPv6Path(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = io.WriteString(w, "{}")
	}))
	defer srv.Close()

	if _, err := testBackend(srv.URL, "tcp4").lookup("2001:db8::1"); err != nil {
		t.Fatalf("lookup error = %v", err)
	}
	if path != "/json/2001:db8::1" {
		t.Errorf("lookup path = %q, want /json/2001:db8::1", path)
	}
}

func TestHTTPErrorStatusFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "nope")
	}))
	defer srv.Close()

	b := testBackend(srv.URL, "tcp4")
	if _, err := b.discover(); err == nil {
		t.Error("discover error = nil, want HTTP status error")
	}
	if _, err := b.lookup("8.8.8.8"); err == nil {
		t.Error("lookup error = nil, want HTTP status error")
	}
}

func TestRedirectsAreFollowed(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/final", http.StatusFound)
	})
	mux.HandleFunc("/final", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "198.51.100.7")
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	body, err := testBackend(srv.URL, "tcp4").discover()
	if err != nil {
		t.Fatalf("discover error = %v", err)
	}
	if body != "198.51.100.7" {
		t.Errorf("discover body = %q", body)
	}
}

func TestTimeouts(t *testing.T) {
	if connectTimeout != 5*time.Second {
		t.Errorf("connectTimeout = %v, want 5s", connectTimeout)
	}
	if requestTimeout != 10*time.Second {
		t.Errorf("requestTimeout = %v, want 10s", requestTimeout)
	}
	if got := newDialer().Timeout; got != connectTimeout {
		t.Errorf("dialer timeout = %v, want %v", got, connectTimeout)
	}
	if got := newHTTPBackend("tcp4").client.Timeout; got != requestTimeout {
		t.Errorf("client timeout = %v, want %v", got, requestTimeout)
	}
}

func TestIPv6FamilyUsesIPv6Only(t *testing.T) {
	ln, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "198.51.100.7")
	}))
	srv.Listener.Close()
	srv.Listener = ln
	srv.Start()
	defer srv.Close()

	body, err := testBackend(srv.URL, "tcp6").discover()
	if err != nil {
		t.Fatalf("tcp6 discover error = %v", err)
	}
	if body != "198.51.100.7" {
		t.Errorf("discover body = %q", body)
	}
	if _, err := testBackend(srv.URL, "tcp4").discover(); err == nil {
		t.Error("tcp4 backend reached an IPv6-only listener")
	}
}
