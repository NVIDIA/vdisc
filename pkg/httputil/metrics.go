// Copyright © 2019 NVIDIA Corporation
package httputil

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func WithMetrics(trans http.RoundTripper, prefix string) http.RoundTripper {
	promInFlight := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: prefix + "_in_flight_requests",
		Help: "A gauge of in-flight requests for the wrapped client.",
	})
	promApiRequests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_api_requests_total",
			Help: "A counter for requests from the wrapped client.",
		},
		[]string{"code", "method"},
	)
	promDnsLatencyVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    prefix + "_dns_duration_seconds",
			Help:    "Trace dns latency histogram.",
			Buckets: []float64{.005, .01, .025, .05},
		},
		[]string{"event"},
	)
	promTlsLatencyVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    prefix + "_tls_duration_seconds",
			Help:    "Trace tls latency histogram.",
			Buckets: []float64{.05, .1, .25, .5},
		},
		[]string{"event"},
	)
	promLatencyVec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    prefix + "_request_duration_seconds",
			Help:    "A histogram of request latencies.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{},
	)

	// Register all of the metrics in the standard registry.
	prometheus.MustRegister(
		promApiRequests,
		promTlsLatencyVec,
		promDnsLatencyVec,
		promLatencyVec,
		promInFlight,
	)

	trace := &promhttp.InstrumentTrace{
		DNSStart: func(t float64) {
			promDnsLatencyVec.WithLabelValues("dns_start")
		},
		DNSDone: func(t float64) {
			promDnsLatencyVec.WithLabelValues("dns_done")
		},
		TLSHandshakeStart: func(t float64) {
			promTlsLatencyVec.WithLabelValues("tls_handshake_start")
		},
		TLSHandshakeDone: func(t float64) {
			promTlsLatencyVec.WithLabelValues("tls_handshake_done")
		},
	}

	// Wrap the default RoundTripper with middleware.
	return promhttp.InstrumentRoundTripperInFlight(promInFlight,
		promhttp.InstrumentRoundTripperCounter(promApiRequests,
			promhttp.InstrumentRoundTripperTrace(trace,
				promhttp.InstrumentRoundTripperDuration(promLatencyVec, trans),
			),
		),
	)
}
