// +build !windows

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
	"bufio"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/multierr"

	"arhat.dev/pkg/hashhelper"
	"arhat.dev/pkg/iohelper"
)

var _ net.Conn = (*pipeConn)(nil)

type pipeConn struct {
	r *os.File
	w *os.File

	hold *os.File
}

func (c *pipeConn) Read(b []byte) (n int, err error) {
	if c.r.Fd() == 0 {
		return 0, io.EOF
	}

	return c.r.Read(b)
}

func (c *pipeConn) Write(b []byte) (n int, err error) {
	return c.w.Write(b)
}

func (c *pipeConn) Close() error {
	_ = syscall.SetNonblock(int(c.r.Fd()), true)
	_ = c.SetDeadline(time.Now().Add(time.Second))

	err := multierr.Combine(c.r.Close(), c.w.Close(), c.hold.Close())
	_ = os.Remove(c.w.Name())
	_ = os.Remove(c.r.Name())
	_ = os.Remove(c.hold.Name())
	return err
}

func (c *pipeConn) LocalAddr() net.Addr {
	return &PipeAddr{Path: c.w.Name()}
}

func (c *pipeConn) RemoteAddr() net.Addr {
	return &PipeAddr{Path: c.r.Name()}
}

func (c *pipeConn) SetDeadline(t time.Time) error {
	return multierr.Append(c.r.SetDeadline(t), c.w.SetDeadline(t))
}

func (c *pipeConn) SetReadDeadline(t time.Time) error {
	return c.r.SetReadDeadline(t)
}

func (c *pipeConn) SetWriteDeadline(t time.Time) error {
	return c.w.SetWriteDeadline(t)
}

var _ net.Listener = (*PipeListener)(nil)

type PipeListener struct {
	connDir string

	localW *os.File
	r      *bufio.Reader

	localR *os.File
	pcs    map[string]*pipeConn
	mu     *sync.RWMutex
	closed bool
}

// Accept new pipe connections, once we have received incoming coming message
// from `path`, the message content is expected to be the path of a named pipe
// listened by the client
//
// If the provided path is a valid named pipe, the listener will create another
// named pipe for client and write to the pipe provided by the client
func (c *PipeListener) Accept() (net.Conn, error) {
accept:
	path, err := c.r.ReadString('\n')
	if err != nil {
		return nil, err
	}

	path = strings.TrimRight(path, "\n")
	if path == "" {
		// new line prefix
		goto accept
	}

	if !filepath.IsAbs(path) {
		goto accept
	}

	// incoming path should be a pipe for server to write, so we can write
	// response to the client
	serverW, err := os.OpenFile(path, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		goto accept
	}

	conn, err := func() (*pipeConn, error) {
		defer func() {
			if err != nil {
				_, _ = serverW.WriteString(fmt.Sprintf("error:%v\n", err))
				_ = serverW.Close()
			}

			c.mu.Unlock()
		}()
		c.mu.Lock()

		if c.closed {
			return nil, io.EOF
		}

		// close old session if any
		oldConn, exists := c.pcs[path]
		if exists {
			_ = oldConn.Close()
		}

		// open a new pipe writer for this client
		serverReadFile := filepath.Join(c.connDir, hashhelper.MD5SumHex([]byte(path)))

		var serverR, clientW *os.File
		serverR, clientW, err = createPipe(serverReadFile, 0666)
		if err != nil {
			return nil, err
		}

		// notify client new read pipe
		_, err = serverW.Write(append([]byte(serverReadFile), '\n'))
		if err != nil {
			return nil, err
		}

		conn := &pipeConn{
			r: serverR,
			w: serverW,

			hold: clientW,
		}
		c.pcs[path] = conn
		return conn, nil
	}()
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (c *PipeListener) Close() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, conn := range c.pcs {
		_ = conn.Close()
	}

	_ = c.localW.Close()
	return c.localR.Close()
}

func (c *PipeListener) Addr() net.Addr {
	return &PipeAddr{
		Path: c.localR.Name(),
	}
}

// ListenPipe will create a named pipe at path and listen incomming message
func ListenPipe(path, connDir string, perm os.FileMode) (net.Listener, error) {
	if path == "" {
		path = os.TempDir()
	}

	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	p, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// listen path not found, treat as pipe file to be created
	} else if p.IsDir() {
		path, err = iohelper.TempFilename(path, "pipe-listener-*")
		if err != nil {
			return nil, fmt.Errorf("failed to generate random listen addr")
		}
	}

	if connDir == "" {
		connDir, err = ioutil.TempDir(os.TempDir(), "pipe-server-*")
	} else {
		connDir, err = filepath.Abs(connDir)
	}
	if err != nil {
		return nil, err
	}

	// max path length is 4096
	// we use hex(md5sum(remote path)) as local write path
	if len(connDir) > 4096-md5.Size*2 {
		return nil, fmt.Errorf("connDir name too long")
	}

	dirInfo, err := os.Stat(connDir)
	if err != nil {
		return nil, err
	}
	if !dirInfo.IsDir() {
		return nil, fmt.Errorf("connDir is not a directory")
	}

	localR, localW, err := createPipe(path, uint32(perm))
	if err != nil {
		return nil, err
	}

	return &PipeListener{
		connDir: connDir,
		// linux path size limit is 4096
		r:      bufio.NewReaderSize(localR, 4096),
		localW: localW,
		localR: localR,
		pcs:    make(map[string]*pipeConn),
		mu:     new(sync.RWMutex),
	}, nil
}

func Dial(path string) (net.Conn, error) {
	return DialPipe(nil, &PipeAddr{Path: path})
}

func DialContext(ctx context.Context, path string) (net.Conn, error) {
	return DialPipeContext(ctx, nil, &PipeAddr{Path: path})
}

func DialPipe(laddr *PipeAddr, raddr *PipeAddr) (net.Conn, error) {
	return DialPipeContext(context.TODO(), laddr, raddr)
}

func DialPipeContext(ctx context.Context, laddr *PipeAddr, raddr *PipeAddr) (_ net.Conn, err error) {
	if raddr == nil {
		return nil, &net.AddrError{Err: "no remote address provided"}
	}

	if laddr == nil || laddr.Path == "" {
		laddr = &PipeAddr{Path: os.TempDir()}
	} else {
		var localPath string
		localPath, err = filepath.Abs(laddr.Path)
		if err != nil {
			return nil, err
		}

		laddr = &PipeAddr{Path: localPath}
	}

	// check if local path exists
	f, err := os.Stat(laddr.Path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// not exists, treat it as file path to create pipe
	} else if f.IsDir() {
		// use a random temporary file
		laddr.Path, err = iohelper.TempFilename(laddr.Path, "pipe-client-*")
		if err != nil {
			return nil, err
		}
	}

	// create a pipe for server to write response
	clientR, serverW, err := createPipe(laddr.Path, 0666)
	if err != nil {
		return nil, err
	}

	// request a new pipe from server
	clientReq, err := os.OpenFile(raddr.Path, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		return nil, err
	}

	_, err = clientReq.Write(append(append([]byte{'\n'}, []byte(laddr.Path)...), '\n'))
	_ = clientReq.Close()
	if err != nil {
		return nil, &net.OpError{Op: "dial", Net: raddr.Network(), Err: err}
	}

	dialDeadline, ok := ctx.Deadline()
	if !ok {
		// no deadline set
		dialDeadline = time.Now().Add(time.Minute)
	}

	connected := make(chan struct{})
	err = clientR.SetReadDeadline(dialDeadline)
	if err != nil {
		err = nil
		go func() {
			t := time.NewTimer(time.Until(dialDeadline))
			defer func() {
				if !t.Stop() {
					<-t.C
				}
			}()

			select {
			case <-ctx.Done():
				_ = clientR.Close()
				return
			case <-t.C:
				// timeout
				_ = clientR.Close()
				return
			case <-connected:
				_ = clientR.SetReadDeadline(time.Time{})
				return
			}
		}()
	}

	br := bufio.NewReaderSize(clientR, 4096)
	serverReadFile, err := br.ReadString('\n')
	close(connected)
	if err != nil {
		return nil, &net.OpError{Op: "read", Net: raddr.Network(), Err: err}
	}

	serverReadFile = strings.TrimRight(serverReadFile, "\n")
	if strings.HasPrefix(serverReadFile, "error:") {
		return nil, &net.OpError{
			Op: "request", Net: raddr.Network(), Err: fmt.Errorf(strings.TrimPrefix(serverReadFile, "error:")),
		}
	}

	if !filepath.IsAbs(serverReadFile) {
		return nil, &net.AddrError{Addr: serverReadFile, Err: "server did not provide absolute path pipe"}
	}

	clientW, err := os.OpenFile(serverReadFile, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		return nil, err
	}

	return &pipeConn{
		w: clientW,
		r: clientR,

		hold: serverW,
	}, nil
}

func createPipe(path string, perm uint32) (r, w *os.File, err error) {
	defer func() {
		if err == nil {
			return
		}

		if r != nil {
			_ = r.Close()
		}

		if w != nil {
			_ = w.Close()
		}

		_ = os.Remove(path)
	}()

	err = syscall.Mkfifo(path, perm)
	if err != nil {
		return
	}

	errCh := make(chan error, 1)
	go func() {
		defer close(errCh)

		var err2 error
		w, err2 = os.OpenFile(path, os.O_WRONLY, os.ModeNamedPipe)
		if err2 != nil {
			errCh <- err2
			return
		}

		_ = w.SetWriteDeadline(time.Now().Add(time.Second))
		_, err2 = w.Write([]byte{'\n'})
		_ = w.SetWriteDeadline(time.Time{})
		errCh <- err2
	}()

	r, err = os.OpenFile(path, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		return
	}

	err = <-errCh
	if err != nil {
		return
	}

	return r, w, checkInitialByte(r)
}

func checkInitialByte(fR io.Reader) error {
	initialByte := make([]byte, 1)
	_, err := fR.Read(initialByte)
	if err != nil {
		return err
	}

	if initialByte[0] != '\n' {
		return fmt.Errorf("unexpected byte %q", initialByte[0])
	}

	return nil
}
