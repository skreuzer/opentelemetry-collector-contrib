// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package dnslookupreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dnslookupreceiver"

import (
	"context"
	"fmt"
	"time"

	"github.com/miekg/dns"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dnslookupreceiver/internal/metadata"
)

type dnsLookupScraper struct {
	settings component.TelemetrySettings
	cfg      *Config
	mb       *metadata.MetricsBuilder
	client   *dns.Client
}

func newDnsLookupScraper(settings receiver.Settings, cfg *Config) *dnsLookupScraper {
	mb := metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings)
	return &dnsLookupScraper{
		settings: settings.TelemetrySettings,
		cfg:      cfg,
		mb:       mb,
		client: &dns.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func dnsTypeFromString(recordType string) (uint16, error) {
	switch recordType {
	case "A":
		return dns.TypeA, nil
	case "AAAA":
		return dns.TypeAAAA, nil
	case "CNAME":
		return dns.TypeCNAME, nil
	case "MX":
		return dns.TypeMX, nil
	case "TXT":
		return dns.TypeTXT, nil
	case "NS":
		return dns.TypeNS, nil
	case "SOA":
		return dns.TypeSOA, nil
	case "SRV":
		return dns.TypeSRV, nil
	case "PTR":
		return dns.TypePTR, nil
	default:
		return 0, fmt.Errorf("unsupported DNS record type: %s", recordType)
	}
}

func (s *dnsLookupScraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	now := pcommon.NewTimestampFromTime(time.Now())

	for _, query := range s.cfg.Queries {
		qtype, err := dnsTypeFromString(query.RecordType)
		if err != nil {
			s.settings.Logger.Warn("Skipping query with invalid record type",
				zap.String("hostname", query.Hostname),
				zap.String("record_type", query.RecordType),
				zap.Error(err))
			continue
		}

		for _, endpoint := range query.Endpoints {
			s.performLookup(now, query.Hostname, query.RecordType, qtype, endpoint)
		}
	}

	return s.mb.Emit(), nil
}

func (s *dnsLookupScraper) performLookup(now pcommon.Timestamp, hostname, recordType string, qtype uint16, endpoint string) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(hostname), qtype)
	msg.RecursionDesired = true

	start := time.Now()
	r, _, err := s.client.Exchange(msg, endpoint)
	elapsed := time.Since(start)

	responseCode := "ERROR"
	var answersCount int64

	if err != nil {
		s.settings.Logger.Warn("DNS lookup failed",
			zap.String("hostname", hostname),
			zap.String("endpoint", endpoint),
			zap.Error(err))
	} else {
		responseCode = dns.RcodeToString[r.Rcode]
		answersCount = int64(len(r.Answer))
	}

	durationSeconds := float64(elapsed) / float64(time.Second)

	s.mb.RecordDnsLookupDurationDataPoint(now, durationSeconds, hostname, recordType, endpoint, responseCode)
	s.mb.RecordDnsLookupAnswersCountDataPoint(now, answersCount, hostname, recordType, endpoint, responseCode)
}
