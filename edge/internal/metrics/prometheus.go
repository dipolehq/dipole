/*
Package metrics defines Prometheus metrics for the Agent Proxy and starts
an HTTP server on the configured metrics port to serve /metrics.
*/
package metrics

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ConnectionsActive = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "dipole_proxy_connections_active",
		Help: "Number of active connections by protocol",
	}, []string{"protocol"})

	ConnectionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dipole_proxy_connections_total",
		Help: "Total connections by protocol and outcome",
	}, []string{"protocol", "outcome"})

	CredentialInjectionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dipole_proxy_credential_injections_total",
		Help: "Total credential injections by protocol and type",
	}, []string{"protocol", "injection_type"})

	PolicyEvaluationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "dipole_proxy_policy_evaluations_total",
		Help: "Total policy evaluations by result",
	}, []string{"result"})

	PolicyEvaluationDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "dipole_proxy_policy_evaluation_duration_seconds",
		Help:    "Duration of policy evaluation requests to Cloud",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 5},
	})

	TLSCertsGenerated = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dipole_proxy_tls_certificates_generated_total",
		Help: "Total TLS leaf certificates generated",
	})

	TLSCertCacheSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dipole_proxy_tls_cert_cache_size",
		Help: "Current number of cached TLS certificates",
	})

	GRPCConnectionStatus = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dipole_proxy_grpc_connection_status",
		Help: "gRPC connection to Cloud (1=connected, 0=disconnected)",
	})

	CredentialCacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dipole_proxy_credential_cache_hits_total",
		Help: "Total credential cache hits (served from cache, skipped Cloud call)",
	})

	CredentialCacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dipole_proxy_credential_cache_misses_total",
		Help: "Total credential cache misses (expired or not found, called Cloud)",
	})

	CredentialCacheSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dipole_proxy_credential_cache_size",
		Help: "Current number of entries in the credential cache",
	})

	TLSCATrustInjectionFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dipole_proxy_tls_ca_trust_injection_failures_total",
		Help: "Total failures writing the tenant CA certificate to the shared trust volume. A nonzero value means TLS Decrypt is signing leaf certs server-side but clients on this host have not been given the CA to trust them, so connections will fail certificate validation.",
	})
)

func init() {
	prometheus.MustRegister(
		ConnectionsActive,
		ConnectionsTotal,
		CredentialInjectionsTotal,
		PolicyEvaluationsTotal,
		PolicyEvaluationDuration,
		TLSCertsGenerated,
		TLSCertCacheSize,
		GRPCConnectionStatus,
		CredentialCacheHits,
		CredentialCacheMisses,
		CredentialCacheSize,
		TLSCATrustInjectionFailures,
	)
}

func StartMetricsServer(port int) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	addr := fmt.Sprintf(":%d", port)
	return http.ListenAndServe(addr, mux)
}
