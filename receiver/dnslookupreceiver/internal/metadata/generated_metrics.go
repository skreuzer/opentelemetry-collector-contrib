package metadata

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
)

var Type = component.MustNewType("dnslookup")

const ScopeName = "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dnslookupreceiver"

var Stability = component.StabilityLevelDevelopment

type MetricsStability = component.StabilityLevel

type MetricConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type MetricsConfig struct {
	DnsLookupDuration     MetricConfig `mapstructure:"dns.lookup.duration"`
	DnsLookupAnswersCount MetricConfig `mapstructure:"dns.lookup.answers_count"`
}

type MetricsBuilderConfig struct {
	Metrics MetricsConfig `mapstructure:"metrics"`
}

func DefaultMetricsBuilderConfig() MetricsBuilderConfig {
	return MetricsBuilderConfig{
		Metrics: MetricsConfig{
			DnsLookupDuration:     MetricConfig{Enabled: true},
			DnsLookupAnswersCount: MetricConfig{Enabled: true},
		},
	}
}

type MetricsBuilder struct {
	config  MetricsBuilderConfig
	start   pcommon.Timestamp
	output  pmetric.Metrics
	sm      pmetric.ScopeMetrics
	metrics pmetric.MetricSlice
}

func NewMetricsBuilder(mbc MetricsBuilderConfig, settings receiver.Settings) *MetricsBuilder {
	mb := &MetricsBuilder{
		config: mbc,
		start:  pcommon.Timestamp(0),
		output: pmetric.NewMetrics(),
	}
	mb.sm = mb.output.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
	mb.sm.Scope().SetName(ScopeName)
	mb.metrics = mb.sm.Metrics()
	return mb
}

func (mb *MetricsBuilder) RecordDnsLookupDurationDataPoint(ts pcommon.Timestamp, val float64, dnsQuestionName string, dnsQuestionType string, dnsServerAddress string, dnsResponseCode string, soaSerial string) {
	if !mb.config.Metrics.DnsLookupDuration.Enabled {
		return
	}
	dp := mb.metrics.AppendEmpty()
	dp.SetName("dns.lookup.duration")
	dp.SetDescription("Duration of the DNS lookup in seconds")
	dp.SetUnit("s")
	dp.SetEmptyGauge()
	pt := dp.Gauge().DataPoints().AppendEmpty()
	pt.SetTimestamp(ts)
	pt.SetDoubleValue(val)
	pt.Attributes().PutStr("dns.question.name", dnsQuestionName)
	pt.Attributes().PutStr("dns.question.type", dnsQuestionType)
	pt.Attributes().PutStr("dns.server.address", dnsServerAddress)
	pt.Attributes().PutStr("dns.response.code", dnsResponseCode)
	if soaSerial != "" {
		pt.Attributes().PutStr("dns.soa.serial", soaSerial)
	}
}

func (mb *MetricsBuilder) RecordDnsLookupAnswersCountDataPoint(ts pcommon.Timestamp, val int64, dnsQuestionName string, dnsQuestionType string, dnsServerAddress string, dnsResponseCode string, soaSerial string) {
	if !mb.config.Metrics.DnsLookupAnswersCount.Enabled {
		return
	}
	dp := mb.metrics.AppendEmpty()
	dp.SetName("dns.lookup.answers_count")
	dp.SetDescription("Number of answers returned by the DNS lookup")
	dp.SetUnit("{answers}")
	dp.SetEmptyGauge()
	pt := dp.Gauge().DataPoints().AppendEmpty()
	pt.SetTimestamp(ts)
	pt.SetIntValue(val)
	pt.Attributes().PutStr("dns.question.name", dnsQuestionName)
	pt.Attributes().PutStr("dns.question.type", dnsQuestionType)
	pt.Attributes().PutStr("dns.server.address", dnsServerAddress)
	pt.Attributes().PutStr("dns.response.code", dnsResponseCode)
	if soaSerial != "" {
		pt.Attributes().PutStr("dns.soa.serial", soaSerial)
	}
}

func (mb *MetricsBuilder) Emit() pmetric.Metrics {
	out := mb.output
	mb.output = pmetric.NewMetrics()
	mb.sm = mb.output.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
	mb.sm.Scope().SetName(ScopeName)
	mb.metrics = mb.sm.Metrics()
	return out
}
