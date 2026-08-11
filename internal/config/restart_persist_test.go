package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a tiny helper for the restart tests.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestLoad_PersistsAndRestoresPortsAcrossRestart proves that ports survive a
// process restart even when nodes.txt is reordered and a node is renamed: the
// same node (by stable identity) keeps its proxy port, and the sidecar is
// written.
func TestLoad_PersistsAndRestoresPortsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	nodesPath := filepath.Join(dir, "nodes.txt")

	writeFile(t, cfgPath, `mode: multi-port
multi_port:
  address: 127.0.0.1
  base_port: 24000
nodes_file: nodes.txt
management:
  enabled: false
`)

	const (
		uriA = "vless://uuid-a@a.example.com:443?type=ws&security=tls#NodeA"
		uriB = "vless://uuid-b@b.example.com:443?type=ws&security=tls#NodeB"
		uriC = "vless://uuid-c@c.example.com:443?type=ws&security=tls#NodeC"
	)

	// First boot: A, B, C in order.
	writeFile(t, nodesPath, uriA+"\n"+uriB+"\n"+uriC+"\n")
	cfg1, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	want := map[string]uint16{} // stableKey -> port from first boot
	for _, n := range cfg1.Nodes {
		want[n.NodeKey()] = n.Port
	}
	if len(want) != 3 {
		t.Fatalf("expected 3 nodes on first boot, got %d", len(want))
	}

	// Sidecar must have been written.
	if _, err := os.Stat(filepath.Join(dir, nodePortMapFile)); err != nil {
		t.Fatalf("expected %s to be created: %v", nodePortMapFile, err)
	}

	// Restart: nodes.txt reordered (C, B, A) and A renamed (fragment changed).
	uriARenamed := "vless://uuid-a@a.example.com:443?type=ws&security=tls#NodeA-Renamed-100GB"
	writeFile(t, nodesPath, uriC+"\n"+uriB+"\n"+uriARenamed+"\n")
	cfg2, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}

	for _, n := range cfg2.Nodes {
		got := n.Port
		if exp, ok := want[n.NodeKey()]; ok {
			if got != exp {
				t.Errorf("node %q: port changed across restart %d -> %d", n.Name, exp, got)
			}
		} else {
			t.Errorf("node %q has unexpected stable key (port preservation would miss it)", n.Name)
		}
	}
}

// TestLoad_NewNodeGetsFreshPortAfterRestart verifies a node added before a
// restart gets a fresh, non-colliding port while existing nodes keep theirs.
func TestLoad_NewNodeGetsFreshPortAfterRestart(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	nodesPath := filepath.Join(dir, "nodes.txt")

	writeFile(t, cfgPath, `mode: multi-port
multi_port:
  address: 127.0.0.1
  base_port: 24500
nodes_file: nodes.txt
management:
  enabled: false
`)

	uriA := "vless://uuid-a@a.example.com:443#A"
	uriB := "vless://uuid-b@b.example.com:443#B"
	writeFile(t, nodesPath, uriA+"\n"+uriB+"\n")
	cfg1, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	portA := cfg1.Nodes[0].Port

	// Restart with a brand-new node D added.
	uriD := "vless://uuid-d@d.example.com:443#D"
	writeFile(t, nodesPath, uriA+"\n"+uriB+"\n"+uriD+"\n")
	cfg2, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}

	seen := map[uint16]bool{}
	var portANew, portD uint16
	for _, n := range cfg2.Nodes {
		if n.Port == 0 {
			t.Errorf("node %q has no port", n.Name)
		}
		if seen[n.Port] {
			t.Fatalf("port collision after restart: %d", n.Port)
		}
		seen[n.Port] = true
		switch n.URI {
		case uriA:
			portANew = n.Port
		case uriD:
			portD = n.Port
		}
	}
	if portANew != portA {
		t.Errorf("existing node A: port changed across restart %d -> %d", portA, portANew)
	}
	if portD == 0 {
		t.Errorf("new node D did not receive a port")
	}
}

// TestSaveNodePortMap_RoundTrip checks the sidecar encodes the stable-key map
// and reads back identically.
func TestSaveNodePortMap_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		Mode:      "multi-port",
		MultiPort: MultiPortConfig{Address: "127.0.0.1", BasePort: 24000},
		Nodes: []NodeConfig{
			{URI: "vless://uuid-a@a.example.com:443#A", Port: 24000},
			{URI: "vless://uuid-b@b.example.com:443#B", Port: 24001},
		},
	}
	cfg.SetFilePath(cfgPath)

	if err := cfg.SaveNodePortMap(); err != nil {
		t.Fatalf("save: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, nodePortMapFile))
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	var got map[string]uint16
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("decode sidecar: %v", err)
	}
	if got[stableNodeKey("vless://uuid-a@a.example.com:443#A")] != 24000 {
		t.Errorf("node A port not persisted correctly: %v", got)
	}
	if got[stableNodeKey("vless://uuid-b@b.example.com:443#B")] != 24001 {
		t.Errorf("node B port not persisted correctly: %v", got)
	}
}
