// Copyright © 2019 NVIDIA Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httputil

import (
	"context"
	"errors"
	"net"
	"sync"
)

func newTestNetwork() *testNetwork {
	return &testNetwork{make(map[string]map[string]*testListener)}
}

type testNetwork struct {
	listeners map[string]map[string]*testListener
}

func (tnet *testNetwork) Listen(network, address string) (net.Listener, error) {
	listeners, ok := tnet.listeners[network]
	if !ok {
		listeners = make(map[string]*testListener)
		tnet.listeners[network] = listeners
	}
	l, ok := listeners[address]
	if !ok {
		l = &testListener{
			addr:   &testAddr{network, address},
			accept: make(chan net.Conn),
			done:   make(chan struct{}),
		}
		listeners[address] = l
	}
	return l, nil
}

func (tnet *testNetwork) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	listeners, ok := tnet.listeners[network]
	if ok {
		l, ok := listeners[address]
		if ok {
			return l.connect()
		}
	}
	return nil, errors.New("address unknown to test listener")
}

func (tnet *testNetwork) Shutdown() {
	for _, listeners := range tnet.listeners {
		for _, l := range listeners {
			l.Close()
		}
	}
}

type testAddr struct {
	network string
	addr    string
}

func (ta *testAddr) Network() string {
	return ta.network
}
func (ta *testAddr) String() string {
	return ta.addr
}

type testListener struct {
	addr   net.Addr
	accept chan net.Conn
	once   sync.Once
	done   chan struct{}
}

func (tl *testListener) connect() (net.Conn, error) {
	a, b := net.Pipe()

	select {
	case <-tl.done:
		return nil, errors.New("test listener closed")
	case tl.accept <- b:
		return a, nil
	}
}

func (tl *testListener) Accept() (net.Conn, error) {
	select {
	case <-tl.done:
		return nil, errors.New("test listener closed")
	case conn := <-tl.accept:
		return conn, nil
	}
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (tl *testListener) Close() error {
	tl.once.Do(func() { close(tl.done) })
	return nil
}

// Addr returns the listener's network address.
func (tl *testListener) Addr() net.Addr {
	return tl.addr
}
