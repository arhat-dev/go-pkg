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
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"arhat.dev/pkg/iohelper"
)

func TestTimeoutReader_Read(t *testing.T) {
	const input = "test"

	// startTime := time.Now()
	r := iohelper.NewTimeoutReader(strings.NewReader(input))
	go r.FallbackReading()

	buf := make([]byte, len(input)+1)

	n, _ := r.Read(time.Second, buf[0:])
	assert.Equal(t, io.EOF, r.Error())
	assert.Equal(t, input, string(buf[:n]))
	// if elp := time.Since(startTime); elp < time.Second {
	// 	assert.FailNow(t, "timout failed", "elp", elp)
	// }

	r = iohelper.NewTimeoutReader(strings.NewReader(input))
	go r.FallbackReading()

	// startTime = time.Now()
	n, _ = r.Read(time.Second, buf[0:len(input)-1])
	// assert.Equal(t, io.EOF, r.Error())
	assert.Equal(t, input[:len(input)-1], string(buf[:n]))
	// if elp := time.Since(startTime); elp > time.Millisecond {
	// 	assert.FailNow(t, "non-timout failed", "elp", elp)
	// }

	pr, pw := io.Pipe()
	r = iohelper.NewTimeoutReader(pr)
	go r.FallbackReading()

	// startTime = time.Now()
	stopSig := make(chan struct{})
	go func() {
		for i := 0; i < 10; i++ {
			time.Sleep(time.Millisecond * 100)
			_, _ = pw.Write([]byte(input))
		}
		_ = pw.Close()
	}()

	var count int
	for r.WaitForData(stopSig) {
		n, _ := r.Read(time.Millisecond, buf[0:])
		assert.Greater(t, n, 0)
		count++
	}
	assert.GreaterOrEqual(t, count, 10)
	assert.Equal(t, io.EOF, r.Error())

	// if elp := time.Since(startTime); elp < time.Second {
	// 	assert.FailNow(t, "close failed", "elp", elp)
	// }
}

func TestTimeoutReader_ReadNet(t *testing.T) {
	const testData = "test"

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
	go tr.FallbackReading()

	buf := make([]byte, len(testData)+1)
	count := 0
	for tr.WaitForData(context.TODO().Done()) {
		count++
		n, err := tr.Read(time.Second, buf[0:])
		if count == 6 {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err, "failed to read test data")
			assert.EqualValues(t, testData, string(buf[:n]))
			assert.Greater(t, n, 0)
		}
	}

	assert.EqualValues(t, 6, count)
}
