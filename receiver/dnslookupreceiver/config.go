// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package dnslookupreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dnslookupreceiver"

import (
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/scraper/scraperhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dnslookupreceiver/internal/metadata"
)

type QueryConfig struct {
	Hostname   string   `mapstructure:"hostname"`
	RecordType string   `mapstructure:"record_type"`
	Endpoints  []string `mapstructure:"endpoints"`
}

func (qc *QueryConfig) Validate() error {
	if qc.Hostname == "" {
		return errors.New("hostname must be specified")
	}
	if qc.RecordType == "" {
		return errors.New("record_type must be specified")
	}
	if len(qc.Endpoints) == 0 {
		return errors.New("at least one endpoint must be specified")
	}
	return nil
}

type Config struct {
	scraperhelper.ControllerConfig `mapstructure:",squash"`
	Timeout                        time.Duration                 `mapstructure:"timeout"`
	Queries                        []QueryConfig                 `mapstructure:"queries"`
	MetricsBuilderConfig           metadata.MetricsBuilderConfig `mapstructure:",squash"`

	_ struct{}
}

func (c *Config) Validate() error {
	if c.Timeout <= 0 {
		return errors.New("timeout must be positive")
	}
	if len(c.Queries) == 0 {
		return errors.New("at least one query must be specified")
	}
	for i, q := range c.Queries {
		if err := q.Validate(); err != nil {
			return fmt.Errorf("query[%d]: %w", i, err)
		}
	}
	return nil
}
