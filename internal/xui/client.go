package xui

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

type response struct {
	Success *bool           `json:"success"`
	Message string          `json:"msg"`
	Object  json.RawMessage `json:"obj"`
}

type Inventory struct {
	Nodes             int            `json:"nodes"`
	Clients           int            `json:"clients"`
	ConfiguredClients int            `json:"configuredClients"`
	Socks5            int            `json:"socks5"`
	Protocols         map[string]int `json:"protocols"`
	LatencyMS         int            `json:"latencyMs"`
}

type RealityTarget struct {
	Target     string `json:"target"`
	SNI        string `json:"sni"`
	LatencyMS  int    `json:"latencyMs"`
	TLSVersion string `json:"tlsVersion"`
	ALPN       string `json:"alpn"`
}

func New(rawURL, token string) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid management URL")
	}
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true}, MaxIdleConns: 8, IdleConnTimeout: 30 * time.Second}
	return &Client{baseURL: strings.TrimRight(parsed.String(), "/"), token: token, http: &http.Client{Transport: transport, Timeout: 25 * time.Second}}, nil
}

func (c *Client) request(ctx context.Context, method, path string, body any, timeout time.Duration) (json.RawMessage, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+"/"+strings.TrimLeft(path, "/"), reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	httpClient := *c.http
	httpClient.Timeout = timeout
	started := time.Now()
	resp, err := httpClient.Do(req)
	_ = started
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var envelope response
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("HTTP %d non-JSON response", resp.StatusCode)
	}
	if resp.StatusCode >= 400 || (envelope.Success != nil && !*envelope.Success) {
		if envelope.Message == "" {
			envelope.Message = resp.Status
		}
		return nil, fmt.Errorf("%s", envelope.Message)
	}
	return envelope.Object, nil
}

func (c *Client) Inventory(ctx context.Context) (Inventory, error) {
	started := time.Now()
	if _, err := c.request(ctx, http.MethodGet, "/panel/api/server/status", nil, 15*time.Second); err != nil {
		return Inventory{}, err
	}
	raw, err := c.request(ctx, http.MethodGet, "/panel/api/inbounds/list", nil, 20*time.Second)
	if err != nil {
		return Inventory{}, err
	}
	var inbounds []struct {
		Protocol string `json:"protocol"`
		Settings any    `json:"settings"`
	}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &inbounds); err != nil {
			return Inventory{}, fmt.Errorf("decode inbounds: %w", err)
		}
	}
	inv := Inventory{Nodes: len(inbounds), Protocols: map[string]int{}, LatencyMS: int(time.Since(started).Milliseconds())}
	for _, inbound := range inbounds {
		protocol := strings.ToLower(inbound.Protocol)
		if protocol == "" {
			protocol = "unknown"
		}
		inv.Protocols[protocol]++
		if protocol == "mixed" || protocol == "socks" || protocol == "socks5" {
			inv.Socks5++
		}
		var settings map[string]any
		switch value := inbound.Settings.(type) {
		case string:
			_ = json.Unmarshal([]byte(value), &settings)
		case map[string]any:
			settings = value
		}
		if clients, ok := settings["clients"].([]any); ok {
			inv.ConfiguredClients += len(clients)
		}
	}
	inv.Clients = inv.ConfiguredClients
	if clientsRaw, err := c.request(ctx, http.MethodGet, "/panel/api/clients/list", nil, 20*time.Second); err == nil {
		var clients []any
		if json.Unmarshal(clientsRaw, &clients) == nil {
			inv.Clients = len(clients)
		}
	}
	return inv, nil
}

func (c *Client) ScanRealityTargets(ctx context.Context, targets string) ([]RealityTarget, error) {
	raw, err := c.request(ctx, http.MethodPost, "/panel/api/server/scanRealityTargets", map[string]string{"targets": strings.TrimSpace(targets)}, 90*time.Second)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Target      string   `json:"target"`
		Feasible    bool     `json:"feasible"`
		TLS13       bool     `json:"tls13"`
		H2          bool     `json:"h2"`
		X25519      bool     `json:"x25519"`
		CertValid   bool     `json:"certValid"`
		ServerNames []string `json:"serverNames"`
		LatencyMS   int      `json:"latencyMs"`
		TLSVersion  string   `json:"tlsVersion"`
		ALPN        string   `json:"alpn"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("decode scan results: %w", err)
	}
	result := make([]RealityTarget, 0, len(rows))
	for _, row := range rows {
		if !row.Feasible || !row.TLS13 || !row.H2 || !row.X25519 || !row.CertValid {
			continue
		}
		sni := ""
		for _, name := range row.ServerNames {
			if name != "" && !strings.Contains(name, "*") {
				sni = name
				break
			}
		}
		if sni == "" {
			sni = strings.Split(row.Target, ":")[0]
		}
		if sni == "" || row.Target == "" {
			continue
		}
		result = append(result, RealityTarget{Target: row.Target, SNI: sni, LatencyMS: row.LatencyMS, TLSVersion: row.TLSVersion, ALPN: row.ALPN})
	}
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].LatencyMS < result[i].LatencyMS {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result, nil
}
