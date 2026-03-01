package httpclient

import (
	"crypto/tls"
	"net/http"
	"time"
)

type HTTPConfig struct {
	MaxIdleConns          int           `yaml:"max_idle_conns"`
	MaxIdleConnsPerHost   int           `yaml:"max_idle_conns_per_host"`
	MaxConnsPerHost       int           `yaml:"max_conns_per_host"`
	IdleConnTimeout       time.Duration `yaml:"idle_conn_timeout"`
	TimeOut               time.Duration `yaml:"time_out"`
	DialTimeout           time.Duration `yaml:"dial_timeout"`
	DialKeepAlive         time.Duration `yaml:"dial_keep_alive"`
	TLSHandshakeTimeout   time.Duration `yaml:"tls_handshake_timeout"`
	ExpectContinueTimeout time.Duration `yaml:"expect_continue_timeout"`
	ResponseHeaderTimeout time.Duration `yaml:"response_header_timeout"`
	ForceTLS              bool          `yaml:"force_tls"`
	DisableKeepAlives     bool          `yaml:"disable_keep_alives"`
	DisableCompression    bool          `yaml:"disable_compression"`
	InsecureSkipVerify    bool          `yaml:"insecure_skip_verify"`
}

// NewHTTPClient creates an HTTP client with recommended transport configuration
// for optimal performance and reliability in API gateway scenarios.
//
// Recommended Configuration:
//   - MaxIdleConns: 100 (reuse connections across hosts)
//   - MaxIdleConnsPerHost: 10-50 (connections per host)
//   - MaxConnsPerHost: 50-100 (limit concurrent connections)
//   - IdleConnTimeout: 90 seconds (clean up idle connections)
//   - DialTimeout: 30 seconds (connection establishment timeout)
//   - DialKeepAlive: 30 seconds (TCP keep-alive probes)
//   - TLSHandshakeTimeout: 10 seconds (TLS negotiation timeout)
//   - ResponseHeaderTimeout: 30 seconds (wait for response headers)
//   - TimeOut: 60-120 seconds (total request timeout)
func NewHTTPClient(config *HTTPConfig) *http.Client {

	// Create transport with recommended settings
	transport := &http.Transport{
		// Connection pooling settings
		MaxIdleConns:        getOrDefaultInt(config.MaxIdleConns, 100),
		MaxIdleConnsPerHost: getOrDefaultInt(config.MaxIdleConnsPerHost, 0),
		MaxConnsPerHost:     getOrDefaultInt(config.MaxConnsPerHost, 0),

		// Timeouts
		IdleConnTimeout:       getOrDefault(config.IdleConnTimeout, 90*time.Second),
		TLSHandshakeTimeout:   getOrDefault(config.TLSHandshakeTimeout, 10*time.Second),
		ExpectContinueTimeout: getOrDefault(config.ExpectContinueTimeout, 1*time.Second),
		ResponseHeaderTimeout: getOrDefault(config.ResponseHeaderTimeout, 30*time.Second),

		// Connection behavior
		DisableKeepAlives:  config.DisableKeepAlives,
		DisableCompression: config.DisableCompression,

		// HTTP/2 support (enabled by default)
		ForceAttemptHTTP2: true,
	}

	// Configure TLS if required
	if config.ForceTLS || !config.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			CurvePreferences:   []tls.CurveID{tls.CurveP256, tls.X25519},
			InsecureSkipVerify: config.InsecureSkipVerify,
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   getOrDefault(config.TimeOut, 120*time.Second),
	}
}

// getOrDefault returns the provided duration if non-zero, otherwise returns the default
func getOrDefault(d, defaultD time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return defaultD
}

// getOrDefaultInt returns the provided int if non-zero, otherwise returns the default
func getOrDefaultInt(v, defaultV int) int {
	if v > 0 {
		return v
	}
	return defaultV
}
