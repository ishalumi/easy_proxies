package monitor

import "testing"

func TestParseProbeRequestWithHTTPSURL(t *testing.T) {
	req, ok := parseProbeRequest("https://harpa.ai", false)
	if !ok {
		t.Fatal("expected probe target to parse")
	}
	if req.Scheme != "https" {
		t.Fatalf("unexpected scheme: %s", req.Scheme)
	}
	if req.HostHeader != "harpa.ai" {
		t.Fatalf("unexpected host header: %s", req.HostHeader)
	}
	if req.ServerName != "harpa.ai" {
		t.Fatalf("unexpected server name: %s", req.ServerName)
	}
	if req.Path != "/" {
		t.Fatalf("unexpected path: %s", req.Path)
	}
	if req.Destination.Port != 443 {
		t.Fatalf("unexpected port: %d", req.Destination.Port)
	}
}

func TestParseProbeRequestWithHTTPPath(t *testing.T) {
	req, ok := parseProbeRequest("http://cp.cloudflare.com/generate_204", true)
	if !ok {
		t.Fatal("expected probe target to parse")
	}
	if req.Scheme != "http" {
		t.Fatalf("unexpected scheme: %s", req.Scheme)
	}
	if req.Path != "/generate_204" {
		t.Fatalf("unexpected path: %s", req.Path)
	}
	if !req.SkipCertVerify {
		t.Fatal("expected skip cert verify flag to be preserved")
	}
	if req.Destination.Port != 80 {
		t.Fatalf("unexpected port: %d", req.Destination.Port)
	}
}

func TestParseProbeRequestWithoutScheme(t *testing.T) {
	req, ok := parseProbeRequest("www.apple.com:80", false)
	if !ok {
		t.Fatal("expected probe target to parse")
	}
	if req.Scheme != "http" {
		t.Fatalf("unexpected scheme: %s", req.Scheme)
	}
	if req.HostHeader != "www.apple.com:80" {
		t.Fatalf("unexpected host header: %s", req.HostHeader)
	}
	if req.Path != "/" {
		t.Fatalf("unexpected path: %s", req.Path)
	}
	if req.Destination.Port != 80 {
		t.Fatalf("unexpected port: %d", req.Destination.Port)
	}
}
