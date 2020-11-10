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

package nethelper

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
)

const (
	defaultPacketReadBufSize = 65537
)

// PortForward network traffic
// the parameters:
// 	ctx is used to cancel dial operation
// 	dialer is optional for custom network dial options
// 	address is the endpoint address
// 	protocol is the protocol name, e.g. tcp, udp, tcp4
// 	upstream is the data channel to the endpoint
// 	packetReadBuf is the buffer used for udp/ip/unix connection
// the return values:
// 	downstream is used to read data sent from the forwarded port and close connection
// 	closeWrite is intended to close write in stream oriented connection
// 	readErrCh is used to check read error and whether donwstream reading finished
//	err if not nil the port forward failed
func PortForward(
	ctx context.Context,
	dialer *net.Dialer,
	address string,
	protocol string,
	port int32,
	upstream io.Reader,
	packetReadBuf []byte,
) (
	downstream io.ReadCloser,
	closeWrite func(),
	readErrCh <-chan error,
	err error,
) {
	if dialer == nil {
		dialer = new(net.Dialer)
	}

	conn, err := dialer.DialContext(
		ctx, protocol,
		net.JoinHostPort(address, strconv.FormatInt(int64(port), 10)),
	)
	if err != nil {
		return nil, nil, nil, err
	}

	downstream = conn
	closeWrite = func() {}

	switch c := conn.(type) {
	case *net.UnixConn:
		// unix connection can be packet or stream base, check proto
		errCh := make(chan error)
		readErrCh = errCh
		closeWrite = func() {
			_ = c.CloseWrite()
		}

		go handleCopyConn(ctx, c, upstream, errCh, packetReadBuf)
	case *net.TCPConn:
		errCh := make(chan error)
		readErrCh = errCh
		closeWrite = func() {
			_ = c.CloseWrite()
		}

		go func() {
			defer close(errCh)
			// take advantage of splice syscall if possible
			_, err2 := c.ReadFrom(upstream)
			if err2 != nil {
				select {
				case errCh <- err2:
				case <-ctx.Done():
				}
			}
		}()
	case net.PacketConn:
		// udp, ip connection
		errCh := make(chan error)
		readErrCh = errCh
		go handleCopyConn(ctx, conn, upstream, errCh, packetReadBuf)
	default:
		// unknown connection, how could it succeeded?
		if conn != nil {
			_ = conn.Close()
		}
		return nil, nil, nil, fmt.Errorf("unexpected %q connection", protocol)
	}

	return
}

func handleCopyConn(
	ctx context.Context,
	downstream io.Writer,
	upstream io.Reader,
	errCh chan<- error,
	buf []byte,
) {
	defer close(errCh)

	if len(buf) == 0 {
		buf = make([]byte, defaultPacketReadBufSize)
	}

	_, err := io.CopyBuffer(downstream, upstream, buf)
	if err != nil {
		select {
		case <-ctx.Done():
		case errCh <- err:
		}
	}
}
