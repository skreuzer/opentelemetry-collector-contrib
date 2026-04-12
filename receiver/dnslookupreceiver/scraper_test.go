// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package dnslookupreceiver

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver/receivertest"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dnslookupreceiver/internal/metadata"
)

func TestScraper(t *testing.T) {
	addr := startMockDNSServer(t)

	cfg := &Config{
		Timeout: 5 * time.Second,
		Queries: []QueryConfig{
			{
				Hostname:   "example.com",
				RecordType: "A",
				Endpoints:  []string{addr},
			},
		},
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
	}

	s := newDnsLookupScraper(receivertest.NewNopSettings(metadata.Type), cfg)
	metrics, err := s.scrape(t.Context())
	require.NoError(t, err)

	require.Equal(t, 1, metrics.ResourceMetrics().Len())
	rm := metrics.ResourceMetrics().At(0)
	require.Equal(t, 1, rm.ScopeMetrics().Len())
	sm := rm.ScopeMetrics().At(0)
	require.Equal(t, 2, sm.Metrics().Len())

	durationMetric := sm.Metrics().At(0)
	require.Equal(t, "dns.lookup.duration", durationMetric.Name())
	require.Equal(t, pmetric.MetricTypeGauge, durationMetric.Type())
	require.Equal(t, 1, durationMetric.Gauge().DataPoints().Len())

	answersMetric := sm.Metrics().At(1)
	require.Equal(t, "dns.lookup.answers_count", answersMetric.Name())
	require.Equal(t, pmetric.MetricTypeGauge, answersMetric.Type())
	require.Equal(t, 1, answersMetric.Gauge().DataPoints().Len())

	dp := durationMetric.Gauge().DataPoints().At(0)
	val, ok := dp.Attributes().Get("dns.question.name")
	require.True(t, ok)
	require.Equal(t, "example.com", val.AsString())
}

func TestScraperInvalidRecordType(t *testing.T) {
	cfg := &Config{
		Timeout: 5 * time.Second,
		Queries: []QueryConfig{
			{
				Hostname:   "example.com",
				RecordType: "INVALID",
				Endpoints:  []string{"8.8.8.8:53"},
			},
		},
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
	}

	s := newDnsLookupScraper(receivertest.NewNopSettings(metadata.Type), cfg)
	metrics, err := s.scrape(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, metrics.ResourceMetrics().Len())
	require.Equal(t, 0, metrics.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().Len())
}

func TestScraperFailedLookup(t *testing.T) {
	cfg := &Config{
		Timeout: 1 * time.Second,
		Queries: []QueryConfig{
			{
				Hostname:   "example.com",
				RecordType: "A",
				Endpoints:  []string{"127.0.0.1:19999"},
			},
		},
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
	}

	s := newDnsLookupScraper(receivertest.NewNopSettings(metadata.Type), cfg)
	metrics, err := s.scrape(t.Context())
	require.NoError(t, err)

	require.Equal(t, 1, metrics.ResourceMetrics().Len())
	sm := metrics.ResourceMetrics().At(0).ScopeMetrics().At(0)

	durationMetric := sm.Metrics().At(0)
	dp := durationMetric.Gauge().DataPoints().At(0)
	val, ok := dp.Attributes().Get("dns.response.code")
	require.True(t, ok)
	require.Equal(t, "ERROR", val.AsString())
}

func startMockDNSServer(t *testing.T) string {
	t.Helper()

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)

	handler := dns.NewServeMux()
	handler.HandleFunc("example.com.", func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   "example.com.",
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    300,
			},
			A: net.ParseIP("93.184.216.34"),
		})
		w.WriteMsg(m)
	})

	server := &dns.Server{
		PacketConn: pc,
		Handler:    handler,
	}

	go server.ActivateAndServe()

	t.Cleanup(func() {
		server.Shutdown()
	})

	_, port, err := net.SplitHostPort(pc.LocalAddr().String())
	require.NoError(t, err)

	return fmt.Sprintf("127.0.0.1:%s", port)
}
