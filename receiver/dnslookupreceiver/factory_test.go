// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package dnslookupreceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dnslookupreceiver/internal/metadata"
)

func TestType(t *testing.T) {
	factory := NewFactory()
	ft := factory.Type()
	require.Equal(t, metadata.Type, ft)
}

func TestValidConfig(t *testing.T) {
	factory := NewFactory()
	err := componenttest.CheckConfigStruct(factory.CreateDefaultConfig())
	require.NoError(t, err)
}

func TestCreateMetrics(t *testing.T) {
	factory := NewFactory()
	metricsReceiver, err := factory.CreateMetrics(
		t.Context(),
		receivertest.NewNopSettings(metadata.Type),
		&Config{
			ControllerConfig: scraperhelper.ControllerConfig{
				CollectionInterval: 10 * time.Second,
				InitialDelay:       time.Second,
			},
			Timeout: 5 * time.Second,
			Queries: []QueryConfig{
				{
					Hostname:   "example.com",
					RecordType: "A",
					Endpoints:  []string{"8.8.8.8:53"},
				},
			},
		},
		consumertest.NewNop(),
	)
	require.NoError(t, err)
	require.NotNil(t, metricsReceiver)
}
