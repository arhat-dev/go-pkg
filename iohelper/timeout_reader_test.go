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
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"arhat.dev/pkg/iohelper"
)

func TestReadWithTimeout(t *testing.T) {
	const input = "test"

	startTime := time.Now()
	r := iohelper.NewTimeoutReader(strings.NewReader(input), len(input)+1)
	go r.StartBackgroundReading()

	data := r.ReadWithTimeout(time.Second)
	assert.Equal(t, io.EOF, r.Error())
	assert.Equal(t, input, string(data))
	if elp := time.Now().Sub(startTime); elp < time.Second {
		assert.FailNow(t, "timout failed", "elp", elp)
	}

	r = iohelper.NewTimeoutReader(strings.NewReader(input), len(input)-1)
	go r.StartBackgroundReading()

	startTime = time.Now()
	data = r.ReadWithTimeout(time.Second)
	assert.Equal(t, io.EOF, r.Error())
	assert.Equal(t, input[:len(input)-1], string(data))
	if elp := time.Now().Sub(startTime); elp > time.Millisecond {
		assert.FailNow(t, "non-timout failed", "elp", elp)
	}

	pr, pw := io.Pipe()
	r = iohelper.NewTimeoutReader(pr, len(input))
	go r.StartBackgroundReading()

	startTime = time.Now()
	stopSig := make(chan struct{})
	go func() {
		for i := 0; i < 10; i++ {
			time.Sleep(time.Millisecond * 100)
			_, _ = pw.Write([]byte(input))
		}
		_ = pw.Close()
	}()

	var count int
	for r.WaitUntilHasData(stopSig) {
		data := r.ReadWithTimeout(time.Millisecond)
		assert.NotNil(t, data)
		count++
	}
	assert.Equal(t, 10, count)
	assert.Equal(t, io.EOF, r.Error())

	if elp := time.Now().Sub(startTime); elp < time.Second {
		assert.FailNow(t, "close failed", "elp", elp)
	}
}
