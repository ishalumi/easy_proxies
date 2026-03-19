package pool

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"easy_proxies/internal/monitor"
	M "github.com/sagernet/sing/common/metadata"
)

func TestHTTPProbeSupportsPlainHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/generate_204" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("parse http server port: %v", err)
	}

	conn, err := net.Dial("tcp", u.Host)
	if err != nil {
		t.Fatalf("dial http server: %v", err)
	}
	defer conn.Close()

	latency, err := httpProbe(context.Background(), conn, monitor.ProbeRequest{
		Destination: M.ParseSocksaddrHostPort(u.Hostname(), uint16(port)),
		Scheme:      "http",
		HostHeader:  u.Host,
		ServerName:  u.Hostname(),
		Path:        "/generate_204",
	})
	if err != nil {
		t.Fatalf("http probe failed: %v", err)
	}
	if latency <= 0 {
		t.Fatalf("expected positive latency, got %v", latency)
	}
}

func TestHTTPProbeSupportsHTTPS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse tls server url: %v", err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("parse tls server port: %v", err)
	}

	conn, err := net.Dial("tcp", u.Host)
	if err != nil {
		t.Fatalf("dial https server: %v", err)
	}
	defer conn.Close()

	latency, err := httpProbe(context.Background(), conn, monitor.ProbeRequest{
		Destination:    M.ParseSocksaddrHostPort(u.Hostname(), uint16(port)),
		Scheme:         "https",
		HostHeader:     u.Host,
		ServerName:     u.Hostname(),
		Path:           "/",
		SkipCertVerify: true,
	})
	if err != nil {
		t.Fatalf("https probe failed: %v", err)
	}
	if latency <= 0 {
		t.Fatalf("expected positive latency, got %v", latency)
	}
}
