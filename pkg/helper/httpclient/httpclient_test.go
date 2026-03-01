package httpclient

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"
)

func TestNewHTTPClientWithBasicConfig(t *testing.T) {
	config := &HTTPConfig{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
		TimeOut:             60 * time.Second,
		ForceTLS:            false,
	}

	client := NewHTTPClient(config)

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	if client.Timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", client.Timeout)
	}

	transport := client.Transport.(*http.Transport)
	if transport.MaxIdleConns != 10 {
		t.Errorf("expected MaxIdleConns 10, got %d", transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != 5 {
		t.Errorf("expected MaxIdleConnsPerHost 5, got %d", transport.MaxIdleConnsPerHost)
	}

	if transport.IdleConnTimeout != 30*time.Second {
		t.Errorf("expected IdleConnTimeout 30s, got %v", transport.IdleConnTimeout)
	}
}

func TestNewHTTPClientWithForceTLS(t *testing.T) {
	config := &HTTPConfig{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
		TimeOut:             60 * time.Second,
		ForceTLS:            true,
	}

	client := NewHTTPClient(config)

	transport := client.Transport.(*http.Transport)
	if transport.TLSClientConfig == nil {
		t.Fatal("expected TLSClientConfig to be non-nil when ForceTLS is true")
	}

	if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected MinVersion TLS1.2, got %d", transport.TLSClientConfig.MinVersion)
	}
}

func TestNewHTTPClientWithZeroValues(t *testing.T) {
	config := &HTTPConfig{}

	client := NewHTTPClient(config)

	if client == nil {
		t.Fatal("expected non-nil client with zero values")
	}

	// Should use defaults
	transport := client.Transport.(*http.Transport)
	if transport.MaxIdleConns != 100 {
		t.Errorf("expected default MaxIdleConns 100, got %d", transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != 50 {
		t.Errorf("expected default MaxIdleConnsPerHost 50, got %d", transport.MaxIdleConnsPerHost)
	}

	if client.Timeout != 120*time.Second {
		t.Errorf("expected default timeout 120s, got %v", client.Timeout)
	}
}

func TestNewHTTPClientWithHighValues(t *testing.T) {
	config := &HTTPConfig{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 500,
		MaxConnsPerHost:     500,
		IdleConnTimeout:     5 * time.Minute,
		TimeOut:             10 * time.Minute,
		ForceTLS:            true,
	}

	client := NewHTTPClient(config)

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	transport := client.Transport.(*http.Transport)
	if transport.MaxIdleConns != 1000 {
		t.Errorf("expected MaxIdleConns 1000, got %d", transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != 500 {
		t.Errorf("expected MaxIdleConnsPerHost 500, got %d", transport.MaxIdleConnsPerHost)
	}

	if transport.MaxConnsPerHost != 500 {
		t.Errorf("expected MaxConnsPerHost 500, got %d", transport.MaxConnsPerHost)
	}

	if client.Timeout != 10*time.Minute {
		t.Errorf("expected timeout 10m, got %v", client.Timeout)
	}
}

func TestNewHTTPClientTransportIsHTTP(t *testing.T) {
	config := &HTTPConfig{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
		TimeOut:             60 * time.Second,
	}

	client := NewHTTPClient(config)

	_, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected Transport to be *http.Transport")
	}
}

func TestNewHTTPClientWithAllTimeouts(t *testing.T) {
	config := &HTTPConfig{
		TimeOut:               60 * time.Second,
		DialTimeout:           15 * time.Second,
		DialKeepAlive:         30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 500 * time.Millisecond,
		ResponseHeaderTimeout: 20 * time.Second,
	}

	client := NewHTTPClient(config)

	transport := client.Transport.(*http.Transport)

	if transport.TLSHandshakeTimeout != 5*time.Second {
		t.Errorf("expected TLSHandshakeTimeout 5s, got %v", transport.TLSHandshakeTimeout)
	}

	if transport.ExpectContinueTimeout != 500*time.Millisecond {
		t.Errorf("expected ExpectContinueTimeout 500ms, got %v", transport.ExpectContinueTimeout)
	}

	if transport.ResponseHeaderTimeout != 20*time.Second {
		t.Errorf("expected ResponseHeaderTimeout 20s, got %v", transport.ResponseHeaderTimeout)
	}

	if client.Timeout != 60*time.Second {
		t.Errorf("expected client timeout 60s, got %v", client.Timeout)
	}
}

func TestNewHTTPClientWithKeepAliveDisabled(t *testing.T) {
	config := &HTTPConfig{
		DisableKeepAlives: true,
	}

	client := NewHTTPClient(config)
	transport := client.Transport.(*http.Transport)

	if !transport.DisableKeepAlives {
		t.Error("expected DisableKeepAlives to be true")
	}
}

func TestNewHTTPClientWithCompressionDisabled(t *testing.T) {
	config := &HTTPConfig{
		DisableCompression: true,
	}

	client := NewHTTPClient(config)
	transport := client.Transport.(*http.Transport)

	if !transport.DisableCompression {
		t.Error("expected DisableCompression to be true")
	}
}

func TestNewHTTPClientForceAttemptHTTP2(t *testing.T) {
	config := &HTTPConfig{}

	client := NewHTTPClient(config)
	transport := client.Transport.(*http.Transport)

	if !transport.ForceAttemptHTTP2 {
		t.Error("expected ForceAttemptHTTP2 to be true")
	}
}

func TestNewHTTPClientWithInsecureSkipVerify(t *testing.T) {
	config := &HTTPConfig{
		InsecureSkipVerify: true,
	}

	client := NewHTTPClient(config)
	transport := client.Transport.(*http.Transport)

	if transport.TLSClientConfig == nil {
		t.Fatal("expected TLSClientConfig to be non-nil")
	}

	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify to be true")
	}
}

func TestNewHTTPClientTLSConfiguration(t *testing.T) {
	config := &HTTPConfig{
		ForceTLS: true,
	}

	client := NewHTTPClient(config)
	transport := client.Transport.(*http.Transport)

	if transport.TLSClientConfig == nil {
		t.Fatal("expected TLSClientConfig to be non-nil")
	}

	tlsConfig := transport.TLSClientConfig
	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected MinVersion TLS1.2, got %d", tlsConfig.MinVersion)
	}

	if len(tlsConfig.CurvePreferences) == 0 {
		t.Error("expected CurvePreferences to be configured")
	}

	if !tlsConfig.PreferServerCipherSuites {
		t.Error("expected PreferServerCipherSuites to be true")
	}
}
