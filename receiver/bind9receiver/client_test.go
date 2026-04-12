package bind9receiver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/bind9receiver/internal/metadata"
)

func TestGetServerStats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		if req.URL.Path == "/json/v1/server" {
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(`{
  "json-stats-version":"1.7",
  "version":"9.18.39",
  "opcodes":{"QUERY":10},
  "rcodes":{"NOERROR":8},
  "views":{}
}`))
			return
		}
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfg := &Config{
		ClientConfig: confighttp.ClientConfig{
			Endpoint: srv.URL,
		},
	}

	c, err := newBind9Client(t.Context(), cfg, componenttest.NewNopHost(), receivertest.NewNopSettings(metadata.Type).TelemetrySettings)
	require.NoError(t, err)

	stats, err := c.GetServerStats()
	require.NoError(t, err)
	require.Equal(t, "9.18.39", stats.Version)
	require.Equal(t, 10, stats.OpCodes["QUERY"])
	require.Equal(t, 8, stats.RCodes["NOERROR"])
}

func TestGetMemoryStats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		if req.URL.Path == "/json/v1/mem" {
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(`{
  "json-stats-version":"1.7",
  "version":"9.18.39",
  "memory":{
    "TotalUse":1000,
    "InUse":500,
    "contexts":[
      {"name":"main","total":800,"inuse":400}
    ]
  }
}`))
			return
		}
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfg := &Config{
		ClientConfig: confighttp.ClientConfig{
			Endpoint: srv.URL,
		},
	}

	c, err := newBind9Client(t.Context(), cfg, componenttest.NewNopHost(), receivertest.NewNopSettings(metadata.Type).TelemetrySettings)
	require.NoError(t, err)

	stats, err := c.GetMemoryStats()
	require.NoError(t, err)
	require.Equal(t, int64(1000), stats.TotalUse)
	require.Equal(t, int64(500), stats.InUse)
	require.Len(t, stats.Contexts, 1)
	require.Equal(t, "main", stats.Contexts[0].Name)
}

func TestGetNetworkStats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		if req.URL.Path == "/json/v1/net" {
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(`{
  "json-stats-version":"1.7",
  "version":"9.18.39",
  "sockstats":{
    "UDP4Open":42,
    "TCP4Active":40,
    "UDP4ConnFail":26
  }
}`))
			return
		}
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfg := &Config{
		ClientConfig: confighttp.ClientConfig{
			Endpoint: srv.URL,
		},
	}

	c, err := newBind9Client(t.Context(), cfg, componenttest.NewNopHost(), receivertest.NewNopSettings(metadata.Type).TelemetrySettings)
	require.NoError(t, err)

	stats, err := c.GetNetworkStats()
	require.NoError(t, err)
	require.Equal(t, int64(42), stats.SockStats["UDP4Open"])
	require.Equal(t, int64(40), stats.SockStats["TCP4Active"])
	require.Equal(t, int64(26), stats.SockStats["UDP4ConnFail"])
}

func TestGetTrafficStats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		if req.URL.Path == "/json/v1/traffic" {
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write([]byte(`{
  "json-stats-version":"1.7",
  "version":"9.18.39",
  "traffic":{
    "dns-udp-requests-sizes-received-ipv4":{
      "32-47":10
    },
    "dns-udp-responses-sizes-sent-ipv4":{
      "32-47":1,
      "80-95":6
    },
    "dns-tcp-requests-sizes-received-ipv4":{},
    "dns-tcp-responses-sizes-sent-ipv4":{},
    "dns-udp-requests-sizes-received-ipv6":{},
    "dns-udp-responses-sizes-sent-ipv6":{},
    "dns-tcp-requests-sizes-received-ipv6":{},
    "dns-tcp-responses-sizes-sent-ipv6":{}
  }
}`))
			return
		}
		rw.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfg := &Config{
		ClientConfig: confighttp.ClientConfig{
			Endpoint: srv.URL,
		},
	}

	c, err := newBind9Client(t.Context(), cfg, componenttest.NewNopHost(), receivertest.NewNopSettings(metadata.Type).TelemetrySettings)
	require.NoError(t, err)

	stats, err := c.GetTrafficStats()
	require.NoError(t, err)
	require.Equal(t, int64(10), stats.DNSUDPRequestsSizesReceivedIPv4["32-47"])
	require.Equal(t, int64(1), stats.DNSUDPResponsesSizesSentIPv4["32-47"])
	require.Equal(t, int64(6), stats.DNSUDPResponsesSizesSentIPv4["80-95"])
	require.Empty(t, stats.DNSTCPRequestsSizesReceivedIPv4)
}

func TestClientError(t *testing.T) {
	cfg := &Config{
		ClientConfig: confighttp.ClientConfig{
			Endpoint: "http://localhost:0",
		},
	}

	c, err := newBind9Client(t.Context(), cfg, componenttest.NewNopHost(), receivertest.NewNopSettings(metadata.Type).TelemetrySettings)
	require.NoError(t, err)

	_, err = c.GetServerStats()
	require.Error(t, err)
}

func TestClientNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := &Config{
		ClientConfig: confighttp.ClientConfig{
			Endpoint: srv.URL,
		},
	}

	c, err := newBind9Client(t.Context(), cfg, componenttest.NewNopHost(), receivertest.NewNopSettings(metadata.Type).TelemetrySettings)
	require.NoError(t, err)

	_, err = c.GetServerStats()
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected 200 response, got 500")
}
