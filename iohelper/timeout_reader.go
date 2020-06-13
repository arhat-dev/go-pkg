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

package iohelper

import (
	"bytes"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// TimeoutReader is a reader with read timeout
//
// It is designed for those want to read some data from a stream, and the size
// of the data is unknown, but still want to pipe data to some destination.
type TimeoutReader struct {
	buf         *bytes.Buffer
	maxDataSize int
	r           io.Reader

	// signal to notify user can do ReadWithTimeout operation
	hasData chan struct{}
	// signal to notify the size of buffered data has reached maxDataSize
	dataFull chan struct{}

	started uint32
	err     *atomic.Value
	mu      *sync.RWMutex
}

// NewTimeoutReader creates a new idle timeout reader
func NewTimeoutReader(r io.Reader, maxDataSize int) *TimeoutReader {
	return &TimeoutReader{
		// internal buffer starts from 4k bytes and will grow to maxDataSize
		buf:         new(bytes.Buffer),
		maxDataSize: maxDataSize,
		hasData:     make(chan struct{}),
		dataFull:    make(chan struct{}),
		err:         new(atomic.Value),
		r:           r,
		mu:          new(sync.RWMutex),
	}
}

// Error returns the error happened during reading in background
func (t *TimeoutReader) Error() error {
	if e := t.err.Load(); e != nil {
		return e.(error)
	}

	return nil
}

// StartBackgroundReading until EOF or error returned, should be called in a goroutine
// other than the one you are reading
func (t *TimeoutReader) StartBackgroundReading() {
	if !atomic.CompareAndSwapUint32(&t.started, 0, 1) {
		return
	}

	var (
		n   int
		err error
	)

	oneByte := make([]byte, 1)
	for {
		// read one byte a time to avoid blocking
		n, err = t.r.Read(oneByte)
		switch n {
		case 0:
			// no bytes read or error happened
			if err == nil {
				err = io.EOF
			}

			t.mu.RLock()

			select {
			case <-t.hasData:
			default:
				close(t.hasData)
			}

			t.mu.RUnlock()
		case 1:
			t.mu.Lock()

			select {
			case <-t.hasData:
			default:
				close(t.hasData)
			}

			t.buf.WriteByte(oneByte[0])
			if t.buf.Len() >= t.maxDataSize {
				select {
				case <-t.dataFull:
				default:
					close(t.dataFull)
				}
			}

			t.mu.Unlock()
		}

		if err != nil {
			t.err.Store(err)
			break
		}
	}

}

// WaitUntilHasData is used to minimize call of ReadWithTimeout when the timeout is
// a small duration
func (t *TimeoutReader) WaitUntilHasData(stopSig <-chan struct{}) bool {
	if t.Error() != nil {
		t.mu.RLock()
		defer t.mu.RUnlock()

		if t.buf.Len() == 0 {
			// no data unread
			return false
		}
		// has data
		return true
	}

	t.mu.RLock()
	hasData := t.hasData
	t.mu.RUnlock()

	select {
	case <-stopSig:
		return false
	case <-hasData:
		t.mu.RLock()
		defer t.mu.RUnlock()

		if t.buf.Len() == 0 {
			// no data unread
			return false
		}
		// has data
		return true
	}
}

// ReadWithTimeout perform a read operation on buffered data, return a chunk
// of data if
//
// the size of the buffered data has reached or maxed out the `maxDataSize`,
// then the size of returned data chunk will be `maxDataSize`
//
// or
//
// the operation timed out, will return all of the buffered data
func (t *TimeoutReader) ReadWithTimeout(timeout time.Duration) []byte {
	if timeout < 0 {
		return nil
	}

	var timedOut bool

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		// take a snapshot of current buffer size
		t.mu.RLock()
		n := t.buf.Len()
		t.mu.RUnlock()

		if n >= t.maxDataSize {
			n = t.maxDataSize
		}

		if n == t.maxDataSize || timedOut {
			if n == 0 {
				return nil
			}

			t.mu.Lock()

			size := t.buf.Len()
			if size < n {
				n = size
			}

			data := make([]byte, n)
			_, _ = t.buf.Read(data)

			size = t.buf.Len()
			if size < t.maxDataSize {
				t.dataFull = make(chan struct{})
			}

			if size == 0 {
				t.hasData = make(chan struct{})
			}

			t.mu.Unlock()
			return data
		}

		select {
		case <-timer.C:
			timedOut = true
		case <-t.dataFull:
		}
	}
}
