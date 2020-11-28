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
	otapitrace "go.opentelemetry.io/otel/trace"
)

func TestTracingConfig_CreateIfEnabled(t *testing.T) {
	tests := []*TracingConfig{
		{
			Enabled: true,
			Format:  "otlp",
		},
		{
			Enabled:  true,
			Format:   "zipkin",
			Endpoint: "http://test",
		},
		{
			Enabled:      true,
			Format:       "jaeger",
			EndpointType: "agent",
			Endpoint:     "test:1234",
		},
		{
			Enabled:      true,
			Format:       "jaeger",
			EndpointType: "collector",
			Endpoint:     "test:1234",
		},
	}

	for _, test := range tests {
		t.Run(test.Format+"/"+test.EndpointType, func(t *testing.T) {
			p, err := test.CreateIfEnabled(true, nil)
			if !assert.NoError(t, err) {
				return
			}

			assert.NotNil(t, p)

			m := p.Tracer("test")
			ctx, span := m.Start(context.Background(), "test", otapitrace.WithRecord())
			_ = ctx
			span.End(otapitrace.WithLinks())
		})
	}
}
