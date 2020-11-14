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

package nethelper_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"runtime"
	"strconv"
	"testing"

	"github.com/pion/dtls/v2"
	"github.com/stretchr/testify/assert"

	"arhat.dev/pkg/nethelper"
	"arhat.dev/pkg/pipenet"

	_ "arhat.dev/pkg/nethelper/piondtls"
	_ "arhat.dev/pkg/nethelper/pipenet"
	_ "arhat.dev/pkg/nethelper/stdnet"
)

func TestListenAndDial(t *testing.T) {
	caBytes, err := ioutil.ReadFile("../test/testdata/tls-ca.pem")
	if !assert.NoError(t, err, "failed to load ca cert") {
		return
	}

	serverCert, err := tls.LoadX509KeyPair("../test/testdata/tls-cert.pem", "../test/testdata/tls-key.pem")
	if !assert.NoError(t, err, "failed to load server tls cert pair:") {
		return
	}

	clientCert, err := tls.LoadX509KeyPair("../test/testdata/client-tls-cert.pem", "../test/testdata/client-tls-key.pem")
	if !assert.NoError(t, err, "failed to load client tls cert pair") {
		return
	}

	cp := x509.NewCertPool()
	if !assert.True(t, cp.AppendCertsFromPEM(caBytes)) {
		return
	}

	serverTLS := &tls.Config{
		ClientCAs:          cp,
		Certificates:       []tls.Certificate{serverCert},
		ClientAuth:         tls.RequireAndVerifyClientCert,
		InsecureSkipVerify: false,
	}

	clientTLS := &tls.Config{
		RootCAs:            cp,
		Certificates:       []tls.Certificate{clientCert},
		ServerName:         "localhost",
		InsecureSkipVerify: false,
	}

	tests := []struct {
		network      string
		listenerType interface{}
		connType     interface{}

		listenConfig    interface{}
		tlsConfig       interface{}
		clientTLSConfig interface{}
	}{
		// tcp
		{
			network:      "tcp",
			listenerType: new(net.TCPListener),
			connType:     new(net.TCPConn),
		},
		{
			network:      "tcp4",
			listenerType: new(net.TCPListener),
			connType:     new(net.TCPConn),
		},
		{
			network:      "tcp6",
			listenerType: new(net.TCPListener),
			connType:     new(net.TCPConn),
		},
		// tcp-tls
		{
			network: "tcp",
			// listenerType:    new(tls.listener),
			connType:        new(tls.Conn),
			tlsConfig:       serverTLS,
			clientTLSConfig: clientTLS,
		},
		{
			network: "tcp4",
			// listenerType:    new(tls.listener),
			connType:        new(tls.Conn),
			tlsConfig:       serverTLS,
			clientTLSConfig: clientTLS,
		},
		{
			network: "tcp6",
			// listenerType:    new(tls.listener),
			connType:        new(tls.Conn),
			tlsConfig:       serverTLS,
			clientTLSConfig: clientTLS,
		},
		// pipe
		{
			network:      "pipe",
			listenerType: new(pipenet.PipeListener),
			connType:     new(pipenet.PipeConn),
		},
		// pipe-tls
		{
			network: "pipe",
			// listenerType:    new(tls.listener),
			connType:        new(tls.Conn),
			tlsConfig:       serverTLS,
			clientTLSConfig: clientTLS,
		},
		// udp
		{
			network:      "udp",
			listenerType: new(net.UDPConn),
			connType:     new(net.UDPConn),
		},
		{
			network:      "udp4",
			listenerType: new(net.UDPConn),
			connType:     new(net.UDPConn),
		},
		{
			network:      "udp6",
			listenerType: new(net.UDPConn),
			connType:     new(net.UDPConn),
		},
		// udp-dtls
		{
			network: "udp",
			// listenerType:    new(dtls.listener),
			connType:        new(dtls.Conn),
			tlsConfig:       serverTLS,
			clientTLSConfig: clientTLS,
		},
		{
			network: "udp4",
			// listenerType:    new(dtls.listener),
			connType:        new(dtls.Conn),
			tlsConfig:       serverTLS,
			clientTLSConfig: clientTLS,
		},
		{
			network: "udp6",
			// listenerType:    new(dtls.listener),
			connType:        new(dtls.Conn),
			tlsConfig:       serverTLS,
			clientTLSConfig: clientTLS,
		},
	}

	for _, test := range tests {
		t.Run(test.network+func() string {
			if test.tlsConfig != nil {
				return "-tls"
			}
			return ""
		}(), func(t *testing.T) {
			address := "localhost:0"
			if test.network == "pipe" && runtime.GOOS == "windows" {
				address = `\\.\pipe\test-` + strconv.FormatBool(test.tlsConfig != nil)
			}
			lRaw, err := nethelper.Listen(
				context.TODO(), test.listenConfig, test.network, address, test.tlsConfig,
			)
			if !assert.NoError(t, err) {
				return
			}

			if test.listenerType != nil {
				if !assert.IsType(t, test.listenerType, lRaw) {
					return
				}
			}

			var addr net.Addr
			switch l := lRaw.(type) {
			case net.Listener:
				defer func() {
					_ = l.Close()
				}()
				addr = l.Addr()
				go func() {
					for {
						conn, err2 := l.Accept()
						if err2 != nil {
							return
						}
						_ = conn.Close()
					}
				}()
			case net.PacketConn:
				defer func() {
					_ = l.Close()
				}()
				addr = l.LocalAddr()
			default:
				assert.Fail(t, "invalid non closable listener")
				return
			}

			conn, err := nethelper.Dial(
				context.TODO(), test.listenConfig, test.network, addr.String(), test.clientTLSConfig,
			)
			if !assert.NoError(t, err) {
				return
			}

			defer func() {
				_ = conn.Close()
			}()

			assert.IsType(t, test.connType, conn)
		})
	}
}
