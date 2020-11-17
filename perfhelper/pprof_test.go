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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPProfConfig_CreateHTTPHandlerIfEnabled(t *testing.T) {
	const testHTTPPathPrefix = "/foo"
	tests := []struct {
		name      string
		target    string
		expectErr bool
	}{
		{
			name:   "index",
			target: "",
		},
		{
			name:   "cmdline page",
			target: "cmdline",
		},
		{
			name:   "symbol page",
			target: "symbol",
		},
		{
			name:   "trace page",
			target: "trace",
		},
		{
			name:   "cpu profile",
			target: "profile",
		},
		{
			name:   "allocs profile",
			target: "allocs",
		},
		{
			name:   "block profile",
			target: "block",
		},
		{
			name:   "goroutine profile",
			target: "goroutine",
		},
		{
			name:   "heap profile",
			target: "heap",
		},
		{
			name:   "mutex profile",
			target: "mutex",
		},
		{
			name:   "threadcreate profile",
			target: "threadcreate",
		},
	}

	config := &PProfConfig{
		Enabled: true,
	}

	h := config.CreateHTTPHandlersIfEnabled(true)
	if !assert.NotNil(t, h) {
		return
	}

	mux := http.NewServeMux()
	for p, handler := range h {
		mux.Handle(filepath.Join(testHTTPPathPrefix, p), handler)
	}

	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// req := httptest.NewRequest(http.MethodGet, testHTTPPathPrefix+test.target, nil)

			resp, err := srv.Client().Get(srv.URL + filepath.Join(testHTTPPathPrefix, test.target))
			if test.expectErr {
				assert.Error(t, err)
				return
			}

			if !assert.NoError(t, err) {
				return
			}

			defer func() {
				_ = resp.Body.Close()
			}()

			assert.EqualValues(t, 200, resp.StatusCode)
			data, err := ioutil.ReadAll(resp.Body)
			assert.NoError(t, err)
			assert.Greater(t, len(data), 0)
		})
	}
}
