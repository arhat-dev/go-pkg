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
	"io"
	"net"
	"os"
	"time"

	"go.uber.org/multierr"
)

var _ net.Addr = (*PipeAddr)(nil)

type PipeAddr struct {
	Path string
}

func (a *PipeAddr) Network() string {
	return "pipe"
}

func (a *PipeAddr) String() string {
	return a.Path
}

var _ net.Conn = (*PipeConn)(nil)

type PipeConn struct {
	r *os.File
	w *os.File

	hold *os.File
}

func (c *PipeConn) Read(b []byte) (n int, err error) {
	var (
		count int
		size  = len(b)
	)

	for n < size {
		// check fd so we won't read on a closed file
		if c.r.Fd() == 0 {
			return n, io.EOF
		}

		count, err = c.r.Read(b[n:])
		n += count
		if err != nil {
			return
		}
	}

	return
}

func (c *PipeConn) Write(b []byte) (n int, err error) {
	return c.w.Write(b)
}

func (c *PipeConn) Close() error {
	err := multierr.Combine(c.r.Close(), c.w.Close(), c.hold.Close())
	_ = os.Remove(c.w.Name())
	_ = os.Remove(c.r.Name())
	_ = os.Remove(c.hold.Name())
	return err
}

func (c *PipeConn) LocalAddr() net.Addr {
	return &PipeAddr{Path: c.w.Name()}
}

func (c *PipeConn) RemoteAddr() net.Addr {
	return &PipeAddr{Path: c.r.Name()}
}

func (c *PipeConn) SetDeadline(t time.Time) error {
	return multierr.Append(c.r.SetDeadline(t), c.w.SetDeadline(t))
}

func (c *PipeConn) SetReadDeadline(t time.Time) error {
	return c.r.SetReadDeadline(t)
}

func (c *PipeConn) SetWriteDeadline(t time.Time) error {
	return c.w.SetWriteDeadline(t)
}
