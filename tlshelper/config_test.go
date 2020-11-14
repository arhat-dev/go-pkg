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

package tlshelper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTLSConfig_GetTLSConfig(t *testing.T) {
	{
		serverConfig := &TLSConfig{
			Enabled: true,
			CaCert:  "../test/testdata/ca.crt",
			Cert:    "../test/testdata/tls-cert.pem",
			Key:     "../test/testdata/tls-key.pem",
		}
		c, err := serverConfig.GetTLSConfig(false)
		if !assert.NoError(t, err) {
			return
		}
		assert.Len(t, c.Certificates, 1)
		assert.NotNil(t, c.ClientAuth)
	}

	{
		clientConfig := &TLSConfig{
			Enabled: true,
			CaCert:  "../test/testdata/ca.crt",
			Cert:    "../test/testdata/client-tls-cert.pem",
			Key:     "../test/testdata/client-tls-key.pem",
		}

		c, err := clientConfig.GetTLSConfig(false)
		if !assert.NoError(t, err) {
			return
		}
		assert.Len(t, c.Certificates, 1)
		assert.NotNil(t, c.RootCAs)
	}
}
