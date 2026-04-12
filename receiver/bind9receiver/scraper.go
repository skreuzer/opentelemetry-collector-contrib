package bind9receiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/bind9receiver"

import (
	"context"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/bind9receiver/internal/metadata"
)

type bind9Scraper struct {
	client   client
	settings component.TelemetrySettings
	cfg      *Config
	mb       *metadata.MetricsBuilder
}

func newBind9Scraper(settings receiver.Settings, cfg *Config) *bind9Scraper {
	mb := metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings)
	return &bind9Scraper{
		settings: settings.TelemetrySettings,
		cfg:      cfg,
		mb:       mb,
	}
}

func (s *bind9Scraper) start(ctx context.Context, host component.Host) error {
	c, err := newBind9Client(ctx, s.cfg, host, s.settings)
	if err != nil {
		return err
	}
	s.client = c
	return nil
}

func (s *bind9Scraper) scrape(context.Context) (pmetric.Metrics, error) {
	if s.client == nil {
		return pmetric.NewMetrics(), nil
	}

	now := pcommon.NewTimestampFromTime(time.Now())

	serverStats, err := s.client.GetServerStats()
	if err != nil {
		s.settings.Logger.Error("Failed to fetch BIND9 server stats", zap.Error(err))
		return pmetric.NewMetrics(), err
	}

	for opcode, count := range serverStats.OpCodes {
		if attrVal, ok := metadata.MapAttributeOpcode[opcode]; ok {
			s.mb.RecordBind9ServerQueriesDataPoint(now, int64(count), attrVal)
		}
	}

	for rcode, count := range serverStats.RCodes {
		if attrVal, ok := metadata.MapAttributeRcode[rcode]; ok {
			s.mb.RecordBind9ServerResponsesDataPoint(now, int64(count), attrVal)
		}
	}

	for viewName, viewData := range serverStats.Views {
		for qtype, count := range viewData.Resolver.QTypes {
			s.mb.RecordBind9ResolverQueriesDataPoint(now, int64(count), viewName, qtype)
		}

		if queryV6, ok := viewData.Resolver.Stats["Queryv6"]; ok {
			s.mb.RecordBind9ResolverResponsesDataPoint(now, int64(queryV6), viewName)
		} else if queryV4, ok := viewData.Resolver.Stats["Queryv4"]; ok {
			s.mb.RecordBind9ResolverResponsesDataPoint(now, int64(queryV4), viewName)
		}

		s.mb.RecordBind9ResolverCacheHitsDataPoint(now, int64(viewData.Resolver.CacheStats.CacheHits), viewName)
		s.mb.RecordBind9ResolverCacheMissesDataPoint(now, int64(viewData.Resolver.CacheStats.CacheMisses), viewName)
		s.mb.RecordBind9ResolverCacheEntriesDataPoint(now, int64(viewData.Resolver.CacheStats.CacheNodes), viewName)

		if valOk, ok := viewData.Resolver.Stats["ValOk"]; ok {
			s.mb.RecordBind9ResolverValidationSuccessDataPoint(now, int64(valOk), viewName)
		}
		if valAttempt, ok := viewData.Resolver.Stats["ValAttempt"]; ok {
			s.mb.RecordBind9ResolverValidationAttemptDataPoint(now, int64(valAttempt), viewName)
		}
	}

	memStats, err := s.client.GetMemoryStats()
	if err != nil {
		s.settings.Logger.Error("Failed to fetch BIND9 memory stats", zap.Error(err))
	} else {
		for _, ctx := range memStats.Contexts {
			s.mb.RecordBind9MemoryUsageDataPoint(now, ctx.InUse, ctx.Name)
			s.mb.RecordBind9MemoryTotalDataPoint(now, ctx.Total, ctx.Name)
		}
	}

	netStats, err := s.client.GetNetworkStats()
	if err != nil {
		s.settings.Logger.Error("Failed to fetch BIND9 network stats", zap.Error(err))
	} else {
		for stat, count := range netStats.SockStats {
			switch {
			case strings.HasSuffix(stat, "OpenFail"):
				s.mb.RecordBind9NetworkSocketOpenFailuresDataPoint(now, count, stat)
			case strings.HasSuffix(stat, "Open"):
				s.mb.RecordBind9NetworkSocketOpenedDataPoint(now, count, stat)
			case strings.HasSuffix(stat, "Close"):
				s.mb.RecordBind9NetworkSocketClosedDataPoint(now, count, stat)
			case strings.HasSuffix(stat, "Active"):
				s.mb.RecordBind9NetworkSocketActiveDataPoint(now, count, stat)
			case strings.HasSuffix(stat, "ConnFail"):
				s.mb.RecordBind9NetworkSocketConnectionFailuresDataPoint(now, count, stat)
			case strings.HasSuffix(stat, "Conn"):
				s.mb.RecordBind9NetworkSocketConnectionDataPoint(now, count, stat)
			case strings.HasSuffix(stat, "BindFail"):
				s.mb.RecordBind9NetworkSocketBindFailuresDataPoint(now, count, stat)
			case strings.HasSuffix(stat, "AcceptFail"):
				s.mb.RecordBind9NetworkSocketAcceptFailuresDataPoint(now, count, stat)
			case strings.HasSuffix(stat, "Accept"):
				s.mb.RecordBind9NetworkSocketAcceptDataPoint(now, count, stat)
			case strings.HasSuffix(stat, "SendErr"):
				s.mb.RecordBind9NetworkSocketSendErrorsDataPoint(now, count, stat)
			case strings.HasSuffix(stat, "RecvErr"):
				s.mb.RecordBind9NetworkSocketReceiveErrorsDataPoint(now, count, stat)
			}
		}
	}

	trafficStats, err := s.client.GetTrafficStats()
	if err != nil {
		s.settings.Logger.Error("Failed to fetch BIND9 traffic stats", zap.Error(err))
	} else {
		type trafficEntry struct {
			transport metadata.AttributeNetworkTransport
			ipVersion metadata.AttributeNetworkType
			direction metadata.AttributeDNSMessageDirection
			buckets   map[string]int64
		}
		entries := []trafficEntry{
			{metadata.AttributeNetworkTransportUdp, metadata.AttributeNetworkTypeIpv4, metadata.AttributeDNSMessageDirectionReceived, trafficStats.DNSUDPRequestsSizesReceivedIPv4},
			{metadata.AttributeNetworkTransportUdp, metadata.AttributeNetworkTypeIpv4, metadata.AttributeDNSMessageDirectionSent, trafficStats.DNSUDPResponsesSizesSentIPv4},
			{metadata.AttributeNetworkTransportTcp, metadata.AttributeNetworkTypeIpv4, metadata.AttributeDNSMessageDirectionReceived, trafficStats.DNSTCPRequestsSizesReceivedIPv4},
			{metadata.AttributeNetworkTransportTcp, metadata.AttributeNetworkTypeIpv4, metadata.AttributeDNSMessageDirectionSent, trafficStats.DNSTCPResponsesSizesSentIPv4},
			{metadata.AttributeNetworkTransportUdp, metadata.AttributeNetworkTypeIpv6, metadata.AttributeDNSMessageDirectionReceived, trafficStats.DNSUDPRequestsSizesReceivedIPv6},
			{metadata.AttributeNetworkTransportUdp, metadata.AttributeNetworkTypeIpv6, metadata.AttributeDNSMessageDirectionSent, trafficStats.DNSUDPResponsesSizesSentIPv6},
			{metadata.AttributeNetworkTransportTcp, metadata.AttributeNetworkTypeIpv6, metadata.AttributeDNSMessageDirectionReceived, trafficStats.DNSTCPRequestsSizesReceivedIPv6},
			{metadata.AttributeNetworkTransportTcp, metadata.AttributeNetworkTypeIpv6, metadata.AttributeDNSMessageDirectionSent, trafficStats.DNSTCPResponsesSizesSentIPv6},
		}
		for _, entry := range entries {
			for bucket, count := range entry.buckets {
				s.mb.RecordBind9NetworkTrafficDataPoint(now, count, entry.transport, entry.ipVersion, entry.direction, bucket)
			}
		}
	}

	metrics := s.mb.Emit()
	metrics.ResourceMetrics().At(0).Resource().Attributes().PutStr("service.version", serverStats.Version)
	return metrics, nil
}
