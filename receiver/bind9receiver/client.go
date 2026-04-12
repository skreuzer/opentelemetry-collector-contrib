package bind9receiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/bind9receiver"

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type client interface {
	GetServerStats() (*serverStats, error)
	GetMemoryStats() (*memoryStats, error)
	GetNetworkStats() (*networkStats, error)
	GetTrafficStats() (*trafficStats, error)
}

type bind9Client struct {
	httpClient *http.Client
	endpoint   string
	logger     *zap.Logger
}

func newBind9Client(ctx context.Context, cfg *Config, host component.Host, settings component.TelemetrySettings) (client, error) {
	httpClient, err := cfg.ToClient(ctx, host.GetExtensions(), settings)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &bind9Client{
		httpClient: httpClient,
		endpoint:   cfg.Endpoint,
		logger:     settings.Logger,
	}, nil
}

func (c *bind9Client) GetServerStats() (*serverStats, error) {
	body, err := c.get("/json/v1/server")
	if err != nil {
		return nil, err
	}

	var stats serverStats
	if err := json.Unmarshal(body, &stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal server stats: %w", err)
	}

	return &stats, nil
}

func (c *bind9Client) GetMemoryStats() (*memoryStats, error) {
	body, err := c.get("/json/v1/mem")
	if err != nil {
		return nil, err
	}

	var resp memoryStatsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal memory stats: %w", err)
	}

	return &resp.Memory, nil
}

func (c *bind9Client) GetNetworkStats() (*networkStats, error) {
	body, err := c.get("/json/v1/net")
	if err != nil {
		return nil, err
	}

	var stats networkStats
	if err := json.Unmarshal(body, &stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal network stats: %w", err)
	}

	return &stats, nil
}

func (c *bind9Client) GetTrafficStats() (*trafficStats, error) {
	body, err := c.get("/json/v1/traffic")
	if err != nil {
		return nil, err
	}

	var resp trafficStatsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal traffic stats: %w", err)
	}

	return &resp.Traffic, nil
}

func (c *bind9Client) get(path string) ([]byte, error) {
	url := c.endpoint + path
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to GET %s: %w", url, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Warn("failed to close response body", zap.Error(closeErr))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected 200 response, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

type serverStats struct {
	JSONStatsVersion string          `json:"json-stats-version"`
	BootTime         string          `json:"boot-time"`
	ConfigTime       string          `json:"config-time"`
	CurrentTime      string          `json:"current-time"`
	Version          string          `json:"version"`
	OpCodes          map[string]int  `json:"opcodes"`
	RCodes           map[string]int  `json:"rcodes"`
	Views            map[string]view `json:"views"`
}

type view struct {
	Resolver resolver `json:"resolver"`
}

type resolver struct {
	Stats      map[string]int `json:"stats"`
	QTypes     map[string]int `json:"qtypes"`
	Cache      map[string]int `json:"cache"`
	CacheStats cacheStats     `json:"cachestats"`
}

type cacheStats struct {
	CacheHits   int `json:"CacheHits"`
	CacheMisses int `json:"CacheMisses"`
	CacheNodes  int `json:"CacheNodes"`
	QueryHits   int `json:"QueryHits"`
	QueryMisses int `json:"QueryMisses"`
}

type memoryStatsResponse struct {
	Memory memoryStats `json:"memory"`
}

type memoryStats struct {
	TotalUse int64           `json:"TotalUse"`
	InUse    int64           `json:"InUse"`
	Malloced int64           `json:"Malloced"`
	Contexts []memoryContext `json:"contexts"`
}

type memoryContext struct {
	Name  string `json:"name"`
	Total int64  `json:"total"`
	InUse int64  `json:"inuse"`
}

type networkStats struct {
	SockStats map[string]int64 `json:"sockstats"`
}

type trafficStatsResponse struct {
	Traffic trafficStats `json:"traffic"`
}

type trafficStats struct {
	DNSUDPRequestsSizesReceivedIPv4 map[string]int64 `json:"dns-udp-requests-sizes-received-ipv4"`
	DNSUDPResponsesSizesSentIPv4    map[string]int64 `json:"dns-udp-responses-sizes-sent-ipv4"`
	DNSTCPRequestsSizesReceivedIPv4 map[string]int64 `json:"dns-tcp-requests-sizes-received-ipv4"`
	DNSTCPResponsesSizesSentIPv4    map[string]int64 `json:"dns-tcp-responses-sizes-sent-ipv4"`
	DNSUDPRequestsSizesReceivedIPv6 map[string]int64 `json:"dns-udp-requests-sizes-received-ipv6"`
	DNSUDPResponsesSizesSentIPv6    map[string]int64 `json:"dns-udp-responses-sizes-sent-ipv6"`
	DNSTCPRequestsSizesReceivedIPv6 map[string]int64 `json:"dns-tcp-requests-sizes-received-ipv6"`
	DNSTCPResponsesSizesSentIPv6    map[string]int64 `json:"dns-tcp-responses-sizes-sent-ipv6"`
}
