/*
Copyright 2020 The arhat.dev Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package perfhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	otapimetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/unit"
)

func TestMetricsConfig_CreateIfEnabled(t *testing.T) {
	tests := []*MetricsConfig{
		{
			Enabled: true,
			Format:  "otlp",
		},
		{
			Enabled: true,
			Format:  "prometheus",
		},
	}

	for _, test := range tests {
		t.Run(test.Format, func(t *testing.T) {
			p, h, err := test.CreateIfEnabled(true)
			if !assert.NoError(t, err) {
				return
			}

			assert.NotNil(t, p)

			if test.Format == "prometheus" {
				assert.NotNil(t, h)
			}

			m := p.Meter("test")
			counter, err := m.NewInt64Counter(
				"test",
				otapimetric.WithInstrumentationVersion("test"),
				otapimetric.WithDescription("test"),
				otapimetric.WithUnit(unit.Bytes),
			)
			if !assert.NoError(t, err) {
				return
			}

			counter.Add(context.Background(), 1)
		})
	}
}
