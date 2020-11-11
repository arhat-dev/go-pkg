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
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// TimeoutReader is a reader with read timeout
//
// It is designed for those want to read some data from a stream, and the size
// of the data is unknown, but still want to pipe data to some destination.
type TimeoutReader struct {
	buf []byte

	timer *time.Timer

	rs readerWithDeadlineCaps

	checkHasBufferedData func() (bool, error)
	cannotSetDeadline    chan struct{}

	r io.Reader

	// signal to notify user can do ReadUntilTimeout operation
	hasData chan struct{}

	started uint32
	err     *atomic.Value
	mu      *sync.RWMutex
}

type readerWithDeadlineCaps interface {
	SetReadDeadline(t time.Time) error
}

type readerWithSyscallConn interface {
	SyscallConn() (syscall.RawConn, error)
}

type readerWithSyscallControl interface {
	Control(f func(fd uintptr)) error
}

// NewTimeoutReader creates a new idle timeout reader
func NewTimeoutReader(r io.Reader) *TimeoutReader {
	var (
		timer *time.Timer
		rs    readerWithDeadlineCaps

		cannotSetDeadline = make(chan struct{})
	)

	// check if can set read deadline
	switch t := r.(type) {
	case readerWithDeadlineCaps:
		// clear read deadline in advance to check set deadline capability
		if t.SetReadDeadline(time.Time{}) == nil {
			rs = t
		}
	default:
		// mark set read deadline not working
		close(cannotSetDeadline)
		timer = time.NewTimer(0)
		if !timer.Stop() {
			<-timer.C
		}
	}

	var (
		control func(f func(fd uintptr)) error
	)

	// check if can check buffered data
	switch t := r.(type) {
	case readerWithSyscallConn:
		rawConn, err := t.SyscallConn()
		if err == nil {
			control = rawConn.Control
		}
	case readerWithSyscallControl:
		control = t.Control
	}

	checkHasBufferedData := func() (bool, error) {
		return true, nil
	}

	if timer == nil && control != nil {
		// support both set read deadline and syscall control
		checkHasBufferedData = func() (hasData bool, err error) {
			n := 0
			errno := syscall.Errno(0)
			err2 := control(func(fd uintptr) {
				for i := 0; i < 1024; i++ {
					n, errno = CheckBytesToRead(fd)
					if errno != 0 {
						err = errno
						return
					}
					if n != 0 {
						return
					}

					runtime.Gosched()
				}
			})
			if err == nil && n == 0 {
				err = err2
			}

			return n != 0, err
		}
	}

	return &TimeoutReader{
		timer: timer,

		rs: rs,

		checkHasBufferedData: checkHasBufferedData,
		cannotSetDeadline:    cannotSetDeadline,

		r: r,

		hasData: make(chan struct{}),

		started: 0,
		err:     new(atomic.Value),
		mu:      new(sync.RWMutex),
	}
}

// Error returns the error happened during reading in background
func (t *TimeoutReader) Error() error {
	if e := t.err.Load(); e != nil {
		return e.(error)
	}

	return nil
}

// FallbackReading is a helper routine for data reading from readers has no SetReadDeadline
// or SetReadDeadline failed when being called
//
// this function will block until EOF or error, so must be called in a goroutine other than
// the one you are reading data
func (t *TimeoutReader) FallbackReading() {
	if !atomic.CompareAndSwapUint32(&t.started, 0, 1) {
		// background reading already started
		return
	}

	var (
		n   int
		err error
	)

	<-t.cannotSetDeadline

	// not able to set read deadline any more
	// case 1: SetReadDeadline not supported
	// case 2: SetReadDeadline function call failed
	// case 3: or no more data wanted

	// check if reader is ok
	_, err = t.r.Read(make([]byte, 0))
	if err != nil {
		// reader got some error, usually closed
		return
	}

	oneByte := make([]byte, 1)
	for {
		// read one byte a time to avoid being blocked
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

			// rely on the default slice grow
			t.buf = append(t.buf, oneByte[0])

			t.mu.Unlock()
		}

		if err != nil {
			t.err.Store(err)
			break
		}
	}
}

func (t *TimeoutReader) hasDataInBuf() bool {
	// take a snapshot of the channel pointer in case it got closed and recreated
	t.mu.RLock()
	defer t.mu.RUnlock()

	return len(t.buf) != 0
}

// WaitForData is a helper function used to check if there is data available in reader
// so we can reduce actual call of Read when the timeout is a short duration
//
// when return value is true, you can call Read to read data, otherwise, false means
// the stopSig has signaled, and we have no idea whether you should continue Read
// from the reader
func (t *TimeoutReader) WaitForData(stopSig <-chan struct{}) bool {
	select {
	case <-t.cannotSetDeadline:
		// in one byte read mode
	default:
		hasData, err := t.checkHasBufferedData()
		if err != nil {
			return true
		}

		for !hasData {
			select {
			case <-stopSig:
				return false
			default:
				runtime.Gosched()
			}

			hasData, err = t.checkHasBufferedData()
			if err != nil {
				// error happened when checking buffered data
				return true
			}
		}

		return hasData
	}

	// in one byte read mode

	if t.Error() != nil {
		// error happened, no more data will be read from background
		return t.hasDataInBuf()
	}

	// take a snapshot of the channel pointer in case it got recreated
	t.mu.RLock()
	hasData := t.hasData
	t.mu.RUnlock()

	select {
	case <-stopSig:
		return false
	case <-hasData:
		// if no data buffered but signal released
		// error happened when reading, should not
		// wait on reader anymore
		return t.hasDataInBuf()
	}
}

// Read performs a read operation with timeout option, function will return when
// maxWait exceeded or p is full
func (t *TimeoutReader) Read(maxWait time.Duration, p []byte) (n int, err error) {
loop:
	for {
		select {
		case <-t.cannotSetDeadline:
			// not able to set read deadline any more, read from buffered data
			break loop
		default:
			// try to set read deadline first
			if t.rs.SetReadDeadline(time.Now().Add(maxWait)) != nil {
				// signal not able to set read deadline
				close(t.cannotSetDeadline)

				// clear read deadline, best effort
				_ = t.rs.SetReadDeadline(time.Time{})

				// check if reader is alive
				_, err = t.r.Read(make([]byte, 0))
				if err != nil {
					t.err.Store(err)
					return 0, err
				}

				// reader is alive, create timer and fallback
				t.timer = time.NewTimer(0)
				if !t.timer.Stop() {
					<-t.timer.C
				}

				break
			}

			n, err = t.r.Read(p)
			if err != nil {
				t.err.Store(err)

				// read failed, signal not able to set read deadline
				close(t.cannotSetDeadline)
				return
			}

			// clear read deadline, best effort
			_ = t.rs.SetReadDeadline(time.Time{})

			return
		}
	}

	maxReadSize := len(p)

	// take a snapshot for the size of buffered data
	t.mu.RLock()
	size := len(t.buf)
	t.mu.RUnlock()

	n = size
	if n > maxReadSize {
		n = maxReadSize
	}

	if !t.timer.Reset(maxWait) {
		// failed to rest, timer fired, try to fix
		if !t.timer.Stop() {
			select {
			case <-t.timer.C:
			default:
				// should not happened, just in case
			}
		}

		// best effort
		_ = t.timer.Reset(maxWait)
	}

	defer func() {
		// clean up timer no matter fired or not
		if !t.timer.Stop() {
			select {
			case <-t.timer.C:
			default:
				// should not happened, just in case
			}
		}
	}()

	<-t.timer.C

	// take a snapshot again for the size of buffered data
	t.mu.RLock()
	size = len(t.buf)
	t.mu.RUnlock()

	if size == 0 {
		return 0, t.Error()
	}

	n = size
	if n > maxReadSize {
		// do not overflow
		n = maxReadSize
	}

	// copy buffered data
	t.mu.Lock()

	err = t.Error()
	n = copy(p, t.buf[:n])
	t.buf = t.buf[n:]

	// handle has data check
	size = len(t.buf)
	if size == 0 && err == nil {
		// no data buffered, need to wait for it next time
		t.hasData = make(chan struct{})
	}

	t.mu.Unlock()
	return
}
