package xui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInventoryAndRealityScan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		var obj any
		switch r.URL.Path {
		case "/panel/api/server/status":
			obj = map[string]any{"xray": map[string]any{"state": "running"}}
		case "/panel/api/inbounds/list":
			obj = []any{
				map[string]any{"protocol": "vless", "settings": `{"clients":[{"id":"a"},{"id":"b"}]}`},
				map[string]any{"protocol": "socks", "settings": `{}`},
			}
		case "/panel/api/clients/list":
			obj = []any{map[string]any{"email": "a"}, map[string]any{"email": "b"}, map[string]any{"email": "c"}}
		case "/panel/api/server/scanRealityTargets":
			obj = []any{
				map[string]any{"target": "slow.example:443", "feasible": true, "tls13": true, "h2": true, "x25519": true, "certValid": true, "serverNames": []string{"slow.example"}, "latencyMs": 80},
				map[string]any{"target": "fast.example:443", "feasible": true, "tls13": true, "h2": true, "x25519": true, "certValid": true, "serverNames": []string{"fast.example"}, "latencyMs": 20},
				map[string]any{"target": "bad.example:443", "feasible": false},
			}
		default:
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "obj": obj})
	}))
	defer server.Close()

	client, err := New(server.URL, "test-token")
	if err != nil {
		t.Fatal(err)
	}
	inv, err := client.Inventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if inv.Nodes != 2 || inv.Clients != 3 || inv.ConfiguredClients != 2 || inv.Socks5 != 1 {
		t.Fatalf("unexpected inventory: %+v", inv)
	}
	targets, err := client.ScanRealityTargets(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 || targets[0].SNI != "fast.example" {
		t.Fatalf("unexpected targets: %+v", targets)
	}
}
