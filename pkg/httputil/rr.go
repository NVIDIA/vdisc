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
	"crypto/tls"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/golang-lru/simplelru"
	"go.uber.org/zap"
)

type LookupHostFunc func(host string) (addrs []string, err error)
type DialFunc func(ctx context.Context, network string, addr string) (conn net.Conn, err error)

type RoundRobinTransportConfig struct {
	// TLSClientConfig specifies the TLS configuration to use with
	// tls.Client.
	// If nil, the default configuration is used.
	// If non-nil, HTTP/2 support may not be enabled by default.
	TLSClientConfig *tls.Config

	// TLSHandshakeTimeout specifies the maximum amount of time waiting to
	// wait for a TLS handshake. Zero means no timeout.
	TLSHandshakeTimeout time.Duration // Go 1.3

	// MaxHosts controls the number of hosts for which we maintain
	// RoundTrippers. Zero means no limit.
	MaxHosts int

	// MaxIdleConnsPerHost, if non-zero, controls the maximum idle
	// (keep-alive) connections to keep per-host. If zero,
	// DefaultMaxIdleConnsPerHost is used.
	MaxIdleConnsPerHost int

	// MaxConnsPerHost optionally limits the total number of
	// connections per host, including connections in the dialing,
	// active, and idle states. On limit violation, dials will block.
	//
	// Zero means no limit.
	MaxConnsPerHost int // Go 1.11

	// IdleConnTimeout is the maximum amount of time an idle
	// (keep-alive) connection will remain idle before closing
	// itself.
	// Zero means no limit.
	IdleConnTimeout time.Duration // Go 1.7

	// ResponseHeaderTimeout, if non-zero, specifies the amount of
	// time to wait for a server's response headers after fully
	// writing the request (including its body, if any). This
	// time does not include the time to read the response body.
	ResponseHeaderTimeout time.Duration // Go 1.1

	// ExpectContinueTimeout, if non-zero, specifies the amount of
	// time to wait for a server's first response headers after fully
	// writing the request headers if the request has an
	// "Expect: 100-continue" header. Zero means no timeout and
	// causes the body to be sent immediately, without
	// waiting for the server to approve.
	// This time does not include the time to send the request header.
	ExpectContinueTimeout time.Duration // Go 1.6

	// ProxyConnectHeader optionally specifies headers to send to
	// proxies during CONNECT requests.
	ProxyConnectHeader http.Header // Go 1.8

	// MaxResponseHeaderBytes specifies a limit on how many
	// response bytes are allowed in the server's response
	// header.
	//
	// Zero means to use a default limit.
	MaxResponseHeaderBytes int64 // Go 1.7

	// WriteBufferSize specifies the size of the write buffer used
	// when writing to the transport.
	// If zero, a default (currently 4KB) is used.
	WriteBufferSize int // Go 1.13

	// ReadBufferSize specifies the size of the read buffer used
	// when reading from the transport.
	// If zero, a default (currently 4KB) is used.
	ReadBufferSize int // Go 1.13

	// ForceAttemptHTTP2 controls whether HTTP/2 is enabled when a non-zero
	// Dial, DialTLS, or DialContext func or TLSClientConfig is provided.
	// By default, use of any those fields conservatively disables HTTP/2.
	// To use a custom dialer or TLS config and still attempt HTTP/2
	// upgrades, set this to true.
	ForceAttemptHTTP2 bool // Go 1.13

	// DialerTimeout is the maximum amount of time a dial will wait
	// for a connect to complete.
	DialerTimeout time.Duration

	// LookupHost looks up the given host using the local resolver. It
	// returns a slice of that host's addresses. Used for testing.
	LookupHost LookupHostFunc

	// Dial allows injection of a DialFunc for testing.
	Dial DialFunc
}

func NewRoundRobinTransport(cfg RoundRobinTransportConfig) http.RoundTripper {
	maxHosts := cfg.MaxHosts
	if maxHosts <= 0 {
		maxHosts = math.MaxInt64
	}
	lru, _ := simplelru.NewLRU(maxHosts, func(key interface{}, ent interface{}) {
		t := ent.(*hostRoundRobinTransport)
		t.StopResolving()
	})
	return &roundRobinTransport{
		cfg: cfg,
		lru: lru,
	}
}

type roundRobinTransport struct {
	cfg RoundRobinTransportConfig

	mu  sync.Mutex
	lru *simplelru.LRU
}

func (rrt *roundRobinTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var t *hostRoundRobinTransport

	host := req.URL.Hostname()
	rrt.mu.Lock()
	ent, hit := rrt.lru.Get(host)
	if hit {
		t = ent.(*hostRoundRobinTransport)
	} else {
		t = newHostRoundRobinTransport(host, rrt.cfg)
		rrt.lru.Add(host, t)
	}
	rrt.mu.Unlock()

	return t.RoundTrip(req)
}

func newHostRoundRobinTransport(host string, cfg RoundRobinTransportConfig) *hostRoundRobinTransport {
	t := &hostRoundRobinTransport{
		host: host,
		cfg:  cfg,
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	// ensure Resolve() has been run at least once
	t.Resolve()
	go t.resolveLoop()
	return t
}

type hostRoundRobinTransport struct {
	// the host name this transport resolves
	host string
	cfg  RoundRobinTransportConfig

	reqnum uint64
	rand   *rand.Rand

	mu         sync.RWMutex
	ips        []string
	transports map[string]*http.Transport

	done chan interface{}
}

func (hrrt *hostRoundRobinTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqid := atomic.AddUint64(&hrrt.reqnum, 1)

	hrrt.mu.RLock()
	if len(hrrt.ips) < 1 {
		hrrt.mu.RUnlock()
		return nil, fmt.Errorf("no round robin IPs available for %q", hrrt.host)
	}
	t := hrrt.transports[hrrt.ips[reqid%uint64(len(hrrt.ips))]]
	hrrt.mu.RUnlock()

	return t.RoundTrip(req)
}

func (hrrt *hostRoundRobinTransport) Resolve() {
	lookupHost := hrrt.cfg.LookupHost
	if lookupHost == nil {
		lookupHost = net.LookupHost
	}
	ips, err := lookupHost(hrrt.host)
	if err != nil {
		zap.L().Error("round robin lookuphost error", zap.String("host", hrrt.host), zap.Error(err))
		return
	}

	hrrt.rand.Shuffle(len(ips), func(i, j int) {
		ips[i], ips[j] = ips[j], ips[i]
	})

	hrrt.mu.RLock()
	// populate a new transport set
	primeTransports := make(map[string]*http.Transport)
	for _, ip := range ips {
		var t *http.Transport
		var ok bool

		if hrrt.transports != nil {
			t, ok = hrrt.transports[ip]
		}
		if !ok {
			df := hrrt.cfg.Dial
			if df == nil {
				var nd net.Dialer
				nd.Timeout = hrrt.cfg.DialerTimeout
				df = nd.DialContext
			}

			dialer := &staticIPDialer{ip, df}
			t = &http.Transport{
				Proxy:                  http.ProxyFromEnvironment,
				DialContext:            dialer.DialContext,
				TLSClientConfig:        hrrt.cfg.TLSClientConfig,
				TLSHandshakeTimeout:    hrrt.cfg.TLSHandshakeTimeout,
				MaxIdleConnsPerHost:    hrrt.cfg.MaxIdleConnsPerHost,
				MaxConnsPerHost:        hrrt.cfg.MaxConnsPerHost,
				IdleConnTimeout:        hrrt.cfg.IdleConnTimeout,
				ResponseHeaderTimeout:  hrrt.cfg.ResponseHeaderTimeout,
				ExpectContinueTimeout:  hrrt.cfg.ExpectContinueTimeout,
				ProxyConnectHeader:     hrrt.cfg.ProxyConnectHeader,
				MaxResponseHeaderBytes: hrrt.cfg.MaxResponseHeaderBytes,
				WriteBufferSize:        hrrt.cfg.WriteBufferSize,
				ReadBufferSize:         hrrt.cfg.ReadBufferSize,
				ForceAttemptHTTP2:      hrrt.cfg.ForceAttemptHTTP2,
			}
		}
		primeTransports[ip] = t
	}
	hrrt.mu.RUnlock()
	hrrt.mu.Lock()
	defer hrrt.mu.Unlock()

	hrrt.ips = ips
	hrrt.transports = primeTransports
}

func (hrrt *hostRoundRobinTransport) StopResolving() {
	close(hrrt.done)
}

func (hrrt *hostRoundRobinTransport) resolveLoop() {
	for {
		select {
		case <-hrrt.done:
			return
		case <-time.After(5*time.Minute + time.Duration(hrrt.rand.Int31n(5))*time.Minute):
			hrrt.Resolve()
		}
	}
}

type staticIPDialer struct {
	ipaddr string
	dial   DialFunc
}

func (d *staticIPDialer) Dial(network string, addr string) (conn net.Conn, err error) {
	return d.DialContext(context.Background(), network, addr)
}

func (d *staticIPDialer) DialContext(ctx context.Context, network string, addr string) (conn net.Conn, err error) {
	separator := strings.LastIndex(addr, ":")
	return d.dial(ctx, network, d.ipaddr+addr[separator:])
}
