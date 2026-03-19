package pool

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

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

func TestAvailableMembersLockedPrefersHealthyNodes(t *testing.T) {
	mgr, err := monitor.NewManager(monitor.Config{})
	if err != nil {
		t.Fatalf("new monitor manager: %v", err)
	}

	healthy := mgr.Register(monitor.NodeInfo{Tag: "healthy"})
	healthy.MarkInitialCheckDone(true)
	pending := mgr.Register(monitor.NodeInfo{Tag: "pending"})
	failed := mgr.Register(monitor.NodeInfo{Tag: "failed"})
	failed.MarkInitialCheckDone(false)

	p := &poolOutbound{
		members: []*memberState{
			{tag: "healthy", entry: healthy},
			{tag: "pending", entry: pending},
			{tag: "failed", entry: failed},
		},
	}

	got := p.availableMembersLocked(time.Now(), "", make([]*memberState, 0, 3))
	if len(got) != 1 || got[0].tag != "healthy" {
		t.Fatalf("expected only healthy member, got %+v", extractTags(got))
	}
}

func TestAvailableMembersLockedFallsBackToPendingNodes(t *testing.T) {
	mgr, err := monitor.NewManager(monitor.Config{})
	if err != nil {
		t.Fatalf("new monitor manager: %v", err)
	}

	failed := mgr.Register(monitor.NodeInfo{Tag: "failed"})
	failed.MarkInitialCheckDone(false)
	pending := mgr.Register(monitor.NodeInfo{Tag: "pending"})

	p := &poolOutbound{
		members: []*memberState{
			{tag: "failed", entry: failed},
			{tag: "pending", entry: pending},
		},
	}

	got := p.availableMembersLocked(time.Now(), "", make([]*memberState, 0, 2))
	if len(got) != 1 || got[0].tag != "pending" {
		t.Fatalf("expected only pending member, got %+v", extractTags(got))
	}
}

func extractTags(members []*memberState) []string {
	tags := make([]string, 0, len(members))
	for _, member := range members {
		tags = append(tags, member.tag)
	}
	return tags
}
