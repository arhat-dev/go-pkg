/*
Copyright 2019 The arhat.dev Authors.

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

package iohelper_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"arhat.dev/pkg/iohelper"
)

func TestTimeoutReader_Read(t *testing.T) {
	const testData = "test"

	t.Run("one byte read", func(t *testing.T) {
		// startTime := time.Now()
		r := iohelper.NewTimeoutReader(strings.NewReader(testData))
		go r.FallbackReading(context.Background().Done())

		buf := make([]byte, len(testData)+1)

		data, _, _ := r.Read(time.Second, buf[0:])
		assert.Equal(t, io.EOF, r.Error())
		assert.Equal(t, testData, string(data))
		// if elp := time.Since(startTime); elp < time.Second {
		// 	assert.FailNow(t, "timout failed", "elp", elp)
		// }

		r = iohelper.NewTimeoutReader(strings.NewReader(testData))
		go r.FallbackReading(context.Background().Done())

		// startTime = time.Now()
		data, _, _ = r.Read(time.Second, buf[0:len(testData)-1])
		// assert.Equal(t, io.EOF, r.Error())
		assert.Equal(t, testData[:len(testData)-1], string(data))
		// if elp := time.Since(startTime); elp > time.Millisecond {
		// 	assert.FailNow(t, "non-timout failed", "elp", elp)
		// }

		pr, pw := io.Pipe()
		r = iohelper.NewTimeoutReader(pr)
		go r.FallbackReading(context.Background().Done())

		// startTime = time.Now()
		stopSig := make(chan struct{})
		go func() {
			for i := 0; i < 10; i++ {
				time.Sleep(time.Millisecond * 100)
				_, _ = pw.Write([]byte(testData))
			}
			_ = pw.Close()
		}()

		var count int
		for r.WaitForData(stopSig) {
			data, _, _ := r.Read(time.Millisecond, buf[0:])
			assert.Greater(t, len(data), 0)
			count++
		}
		assert.GreaterOrEqual(t, count, 10)
		assert.Equal(t, io.EOF, r.Error())

		// if elp := time.Since(startTime); elp < time.Second {
		// 	assert.FailNow(t, "close failed", "elp", elp)
		// }
	})

	t.Run("setReadDeadline full support", func(t *testing.T) {
		l, err := net.Listen("tcp", "localhost:0")
		if !assert.NoError(t, err, "failed to listen tcp addr for test") {
			return
		}

		srvClosed := make(chan struct{})
		go func() {
			srvConn, err2 := l.Accept()
			if !assert.NoError(t, err2, "failed to accept conn") {
				return
			}
			for i := 0; i < 5; i++ {
				_, err2 = srvConn.Write([]byte(testData))
				assert.NoError(t, err2, "failed to write test data")
				time.Sleep(5 * time.Second)
			}

			_ = srvConn.Close()
			close(srvClosed)
		}()

		clientConn, err := net.Dial("tcp", l.Addr().String())
		if !assert.NoError(t, err, "failed to create test connection") {
			return
		}

		go func() {
			<-srvClosed
			_ = clientConn.Close()
		}()

		tr := iohelper.NewTimeoutReader(clientConn)
		go tr.FallbackReading(context.Background().Done())

		buf := make([]byte, len(testData)+1)
		count := 0
		for tr.WaitForData(context.TODO().Done()) {
			data, _, err := tr.Read(time.Second, buf[0:])
			if err == iohelper.ErrDeadlineExceeded && len(data) == 0 {
				continue
			}

			count++
			if count == 6 {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err, "failed to read test data")
				assert.EqualValues(t, testData, string(data))
				assert.Greater(t, len(data), 0)
			}
		}

		assert.EqualValues(t, 5, count)
	})

	t.Run("setReadDeadline partial support", func(t *testing.T) {
		pr, pw, err := os.Pipe()
		if !assert.NoError(t, err, "failed to create os pipe") {
			return
		}

		go func() {
			for i := 0; i < 5; i++ {
				_, err2 := pw.Write([]byte(testData))
				assert.NoError(t, err2, "failed to write test data")
				time.Sleep(5 * time.Second)
			}

			_ = pw.Close()
			_ = pr.Close()
		}()

		tr := iohelper.NewTimeoutReader(pr)
		go tr.FallbackReading(context.Background().Done())

		buf := make([]byte, len(testData)+1)
		count := 0
		for tr.WaitForData(context.TODO().Done()) {
			data, _, err := tr.Read(time.Second, buf[0:])
			if err != nil && err != iohelper.ErrDeadlineExceeded {
				continue
			}

			count++
			if runtime.GOOS == "windows" {
				assert.Equal(t, err, iohelper.ErrDeadlineExceeded)
			} else {
				assert.NoError(t, err, "failed to read test data")
			}

			assert.EqualValues(t, testData, string(data))
			assert.Greater(t, len(data), 0)
		}

		assert.EqualValues(t, 5, count)
	})
}

func BenchmarkTimeoutReader_Read(b *testing.B) {
	type testCase struct {
		chunkSize int
		timeout   time.Duration
	}

	var testCases []testCase
	for _, size := range []int{
		64,
		512,
		4096,
		32768,
		65536,
	} {
		for _, timeout := range []time.Duration{
			// time.Microsecond,
			100 * time.Millisecond,
		} {
			testCases = append(testCases, testCase{
				chunkSize: size,
				timeout:   timeout,
			})
		}
	}

	for _, c := range testCases {
		b.Run(fmt.Sprintf("%d-%s", c.chunkSize, c.timeout.String()), func(b *testing.B) {
			b.Run("setReadDeadline full support", func(b *testing.B) {
				l, err := net.Listen("tcp", "localhost:0")
				if !assert.NoError(b, err, "failed to listen tcp addr for test") {
					return
				}

				expectedSize := b.N * (c.chunkSize / 8)
				go func() {
					srvConn, err2 := l.Accept()
					if !assert.NoError(b, err2, "failed to accept conn") {
						return
					}

					testData := make([]byte, c.chunkSize/8)
					for i := 0; i < b.N; i++ {
						_, _ = srvConn.Write(testData)
					}

					_ = srvConn.Close()
				}()

				clientConn, err := net.Dial("tcp", l.Addr().String())
				if !assert.NoError(b, err, "failed to create test connection") {
					return
				}

				tr := iohelper.NewTimeoutReader(clientConn)
				go tr.FallbackReading(context.Background().Done())
				buf := make([]byte, c.chunkSize)

				neverStop := context.Background().Done()
				actualSize := 0
				b.ResetTimer()
				for tr.WaitForData(neverStop) {
					data, _, _ := tr.Read(c.timeout, buf[0:])
					actualSize += len(data)
					if actualSize == expectedSize {
						b.StopTimer()
						_ = clientConn.Close()
					}
				}

				assert.EqualValues(b, expectedSize, actualSize)
			})

			b.Run("setReadDeadline partial support", func(b *testing.B) {
				pr, pw, err := os.Pipe()
				if !assert.NoError(b, err, "failed to create os pipe") {
					return
				}

				expectedSize := b.N * (c.chunkSize / 8)
				go func() {
					testData := make([]byte, c.chunkSize/8)
					for i := 0; i < b.N; i++ {
						_, _ = pw.Write(testData)
					}

					_ = pw.Close()
				}()

				tr := iohelper.NewTimeoutReader(pr)
				go tr.FallbackReading(context.Background().Done())
				buf := make([]byte, c.chunkSize)

				neverStop := context.Background().Done()
				actualSize := 0
				b.ResetTimer()
				for tr.WaitForData(neverStop) {
					data, _, _ := tr.Read(c.timeout, buf[0:])
					actualSize += len(data)
					if actualSize == expectedSize {
						b.StopTimer()
						_ = pr.Close()
					}
				}

				assert.EqualValues(b, expectedSize, actualSize)
			})

			b.Run("one byte read", func(b *testing.B) {
				pr, pw := io.Pipe()

				expectedSize := b.N * (c.chunkSize / 8)
				go func() {
					testData := make([]byte, c.chunkSize/8)
					for i := 0; i < b.N; i++ {
						_, _ = pw.Write(testData)
					}
				}()

				tr := iohelper.NewTimeoutReader(pr)
				go tr.FallbackReading(context.Background().Done())
				buf := make([]byte, c.chunkSize)

				neverStop := context.Background().Done()
				actualSize := 0
				b.ResetTimer()
				for tr.WaitForData(neverStop) {
					data, _, _ := tr.Read(c.timeout, buf[0:])
					actualSize += len(data)
					if actualSize == expectedSize {
						b.StopTimer()
						_ = pr.Close()
						_ = pw.Close()
					}
				}

				assert.EqualValues(b, expectedSize, actualSize)
			})
		})
	}
}
