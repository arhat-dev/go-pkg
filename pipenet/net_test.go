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

package pipenet

import (
	"context"
	"math"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"arhat.dev/pkg/iohelper"
)

func TestPipeNet(t *testing.T) {
	tests := []struct {
		name          string
		randLocalPath bool
		localPath     string
	}{
		{
			name:      "No local path",
			localPath: "",
		},
		{
			name:      "Local path dir",
			localPath: t.TempDir(),
		},
		{
			name:          "Local path file",
			randLocalPath: true,
		},
	}

	testData := []byte(strings.Repeat("hello world", math.MaxUint16))

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			listenPipePath, err := iohelper.TempFilename(t.TempDir(), "*")
			if !assert.NoError(t, err, "failed to create temp file") {
				panic(err)
			}

			l1, err := ListenPipe("", t.TempDir(), 0600)
			if !assert.NoError(t, err, "failed to listen random pipe addr") {
				return
			}

			l2, err := ListenPipe(listenPipePath, "", 0600)
			if !assert.NoError(t, err, "failed to listen pipe addr %q", listenPipePath) {
				return
			}

			serverRead := make(chan struct{})
			for _, ln := range []net.Listener{l1, l2} {
				l := ln
				go func() {
					ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
					defer func() {
						cancel()
						_ = l.Close()
					}()

					path := l.Addr().String()
					{
						conn, err2 := DialContext(ctx, path)
						if !assert.NoError(t, err2) {
							return
						}
						_, err2 = conn.Write(testData)
						assert.NoError(t, err2)
						_ = conn.Close()
					}
					<-serverRead

					{
						conn, err2 := Dial(path)
						if !assert.NoError(t, err2) {
							return
						}
						_, err2 = conn.Write(testData)
						assert.NoError(t, err2)
						_ = conn.Close()
					}
					<-serverRead

					{
						localPath := test.localPath
						if test.randLocalPath {
							var err2 error
							localPath, err2 = iohelper.TempFilename(t.TempDir(), "*")
							if !assert.NoError(t, err2, "failed to create temp file") {
								panic(err2)
							}
						}

						conn, err2 := DialPipeContext(context.TODO(), &PipeAddr{Path: localPath}, &PipeAddr{Path: path})
						if !assert.NoError(t, err2) {
							return
						}
						_, err2 = conn.Write(testData)
						assert.NoError(t, err2)
						_ = conn.Close()
					}
					<-serverRead
				}()

				for i := 0; i < 3; i++ {
					conn, err := l.Accept()
					if !assert.NoError(t, err, "failed to accept required pipe connection %d", i+1) {
						return
					}

					buf := make([]byte, len(testData))
					n, err := conn.Read(buf)
					_ = n
					assert.NoError(t, err)

					assert.EqualValues(t, testData, buf[:n])

					_ = conn.Close()

					serverRead <- struct{}{}
				}
			}
		})
	}
}

func BenchmarkPipeNet(b *testing.B) {
	ListenPipe("", b.TempDir(), 0600)
	for i := 0; i < b.N; i++ {

	}
}
