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

package confhelper

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

func TestKubeClientConfig_NewKubeClient(t *testing.T) {
	kubeConfigPath := os.Getenv("TEST_KUBECONFIG")
	_, sampleKubeConfig, err := (&KubeClientConfig{
		KubeconfigPath: kubeConfigPath,
		RateLimit: KubeClientRateLimitConfig{
			Enabled: true,
			QPS:     10,
			Burst:   100,
		},
	}).NewKubeClient(nil, true)

	assert.NoError(t, err)

	tests := []struct {
		name   string
		config *KubeClientConfig

		applyRateLimitConfig bool
		existingKubeconfig   *rest.Config
		wantErr              bool
	}{
		{
			name: "Fake Client",
			config: &KubeClientConfig{
				Fake: true,
			},
			applyRateLimitConfig: false,
			existingKubeconfig:   nil,
			wantErr:              false,
		},
		{
			name: "Config Client With Default RateLimit",
			config: &KubeClientConfig{
				KubeconfigPath: kubeConfigPath,
			},
			applyRateLimitConfig: false,
			existingKubeconfig:   nil,
			wantErr:              false,
		},
		{
			name: "Config Client With AlwaysAllow RateLimit",
			config: &KubeClientConfig{
				KubeconfigPath: kubeConfigPath,
				RateLimit: KubeClientRateLimitConfig{
					Enabled: false,
					QPS:     0,
					Burst:   0,
				},
			},
			applyRateLimitConfig: true,
		},
		{
			name: "Config Client With RateLimit",
			config: &KubeClientConfig{
				KubeconfigPath: kubeConfigPath,
				RateLimit: KubeClientRateLimitConfig{
					Enabled: true,
					QPS:     10,
					Burst:   100,
				},
			},
			applyRateLimitConfig: true,
		},
		{
			name:   "InCluster Client",
			config: &KubeClientConfig{},
		},
		{
			name:               "Client With Existing Config",
			config:             &KubeClientConfig{},
			existingKubeconfig: sampleKubeConfig,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotClient, gotConfig, err := test.config.NewKubeClient(test.existingKubeconfig, test.applyRateLimitConfig)
			if test.wantErr {
				assert.Error(t, err)
				assert.Nil(t, gotClient)
				assert.Nil(t, gotConfig)
				return
			}

			if !assert.NoError(t, err) {
				assert.FailNow(t, "failed to create kube client")
			}
			assert.NotNil(t, gotClient)
			assert.NotNil(t, gotConfig)

			// nolint:gosimple
			var expectedType kubernetes.Interface
			expectedType = &kubernetes.Clientset{}
			if test.config.Fake {
				expectedType = &fake.Clientset{}
			}

			if !assert.IsType(t, expectedType, gotClient) {
				t.Log("type of created client", reflect.TypeOf(gotClient).String(), "not expected", reflect.TypeOf(expectedType).String())
			}

			if test.config.Fake {
				// no more tests for fake client
				return
			}

			if test.existingKubeconfig != nil {
				assert.Equal(t, test.existingKubeconfig.Host, gotConfig.Host)
			} else {
				if test.applyRateLimitConfig {
					if test.config.RateLimit.Enabled {
						assert.True(t, reflect.DeepEqual(
							flowcontrol.NewTokenBucketRateLimiter(
								test.config.RateLimit.QPS, test.config.RateLimit.Burst), gotConfig.RateLimiter))
					} else {
						assert.True(t, reflect.DeepEqual(flowcontrol.NewFakeAlwaysRateLimiter(), gotConfig.RateLimiter))
					}
				} else {
					assert.Nil(t, gotConfig.RateLimiter)
				}
			}
		})
	}
}
