# HTTP Transport Configuration Recommendations

## Overview
This document provides best practices for configuring `http.Transport` in the KrakenD API Gateway platform.

## Key Transport Settings

### Connection Pooling

| Setting | Recommended | Purpose |
|---------|-------------|---------|
| **MaxIdleConns** | 100 | Global idle connections pool across all hosts |
| **MaxIdleConnsPerHost** | 50 | Idle connections per host (reuse persistent connections) |
| **MaxConnsPerHost** | 100 | Limit concurrent connections per host |

**Rationale:**
- Connection reuse reduces overhead of establishing new connections
- `MaxConnsPerHost` prevents overwhelming downstream services
- Balances resource usage with throughput

### Timeout Settings

| Setting | Recommended | Purpose |
|---------|-------------|---------|
| **DialTimeout** | 30s | Time to establish TCP connection |
| **DialKeepAlive** | 30s | TCP keep-alive probe interval |
| **TLSHandshakeTimeout** | 10s | TLS negotiation timeout |
| **ExpectContinueTimeout** | 1s | Wait for "100 Continue" response |
| **ResponseHeaderTimeout** | 30s | Wait for response headers from server |
| **Client Timeout** | 60-120s | Total request timeout |
| **IdleConnTimeout** | 90s | Close idle connections after this duration |

**Rationale:**
- Prevents hanging connections that consume resources
- `DialTimeout` ensures quick failure detection
- `ResponseHeaderTimeout` catches slow/unresponsive servers early
- Keep-alive probes detect broken connections

### TLS Configuration

```go
TLSClientConfig: &tls.Config{
    MinVersion:               tls.VersionTLS12,    // Enforce TLS 1.2+
    CurvePreferences:         []tls.CurveID{       // Modern curves
        tls.CurveP256,
        tls.X25519,
    },
    PreferServerCipherSuites: true,                // Use server's preference
    InsecureSkipVerify:       false,               // Verify certificates
}
```

**Rationale:**
- TLS 1.2+ provides modern encryption standards
- Curve preferences balance security and performance
- Server cipher preference enables forward secrecy
- Certificate verification prevents MITM attacks

## Configuration File Example

```yaml
# config/settings/http-transport.yaml
http_client:
  max_idle_conns: 100
  max_idle_conns_per_host: 50
  max_conns_per_host: 100
  idle_conn_timeout: 90s
  time_out: 120s
  dial_timeout: 30s
  dial_keep_alive: 30s
  tls_handshake_timeout: 10s
  expect_continue_timeout: 1s
  response_header_timeout: 30s
  force_tls: true
  disable_keep_alives: false
  disable_compression: false
  insecure_skip_verify: false
```

## Performance Tuning Guidelines

### For High-Traffic Scenarios
- **Increase MaxIdleConns**: 200-300 (more connection reuse)
- **Increase MaxIdleConnsPerHost**: 100-150 (more parallel requests per host)
- **Increase MaxConnsPerHost**: 200-300 (higher concurrency limit)
- **Decrease IdleConnTimeout**: 30-60s (aggressive cleanup)

### For Latency-Sensitive Services
- **Decrease DialTimeout**: 10-15s (faster failure detection)
- **Decrease ResponseHeaderTimeout**: 10-20s (quicker timeout)
- **Decrease TLSHandshakeTimeout**: 5s (fail faster on TLS issues)

### For Reliability/Stability
- **Lower MaxConnsPerHost**: 50 (protect downstream services)
- **Increase all timeouts**: 60s+ (allow slow responses)
- **Enable keep-alives**: true (maintain connection state)

## HTTP/2 Support

The configuration enables `ForceAttemptHTTP2: true` which:
- Automatically uses HTTP/2 when available
- Falls back to HTTP/1.1 if server doesn't support it
- Improves multiplexing and header compression
- **Note**: Requires TLS (HTTPS)

## Monitoring Recommendations

Monitor these metrics to tune configuration:

1. **Connection Pool Stats**
   - Idle connections count
   - Active connections count
   - Connection reuse ratio

2. **Timeout Metrics**
   - Dial timeout errors
   - Response header timeout errors
   - TLS handshake failures

3. **Performance Metrics**
   - Request latency (p50, p95, p99)
   - Throughput (requests/sec)
   - Error rate by error type

## Security Best Practices

1. **Always use TLS in production**
   ```yaml
   force_tls: true
   insecure_skip_verify: false
   ```

2. **Keep TLS version updated**
   - Minimum: TLS 1.2
   - Recommended: TLS 1.3 capable

3. **Validate certificates**
   ```yaml
   insecure_skip_verify: false  # Always enabled in prod
   ```

4. **Use strong cipher suites**
   - Modern curves: P256, X25519
   - Server cipher preference

## Default Values

If not specified in config, defaults are:

```go
MaxIdleConns:          100
MaxIdleConnsPerHost:   50
MaxConnsPerHost:       100
IdleConnTimeout:       90s
DialTimeout:           30s
DialKeepAlive:         30s
TLSHandshakeTimeout:   10s
ExpectContinueTimeout: 1s
ResponseHeaderTimeout: 30s
Client Timeout:        120s
ForceTLS:              true (if configured)
```

## Common Issues and Solutions

### Problem: "connection reset by peer"
- Increase `IdleConnTimeout` (server closing connections too aggressively)
- Reduce `DialKeepAlive` interval (probes sent more frequently)

### Problem: Slow upstream responses
- Increase `ResponseHeaderTimeout`
- Check if `ResponseHeaderTimeout` is too aggressive

### Problem: High memory usage
- Decrease `MaxIdleConns` and `MaxIdleConnsPerHost`
- Increase `IdleConnTimeout` to close idle connections faster

### Problem: TLS errors
- Increase `TLSHandshakeTimeout`
- Check certificate validity and chain
- Ensure `MinVersion` matches server capabilities

## References

- [Go http.Transport Documentation](https://pkg.go.dev/net/http#Transport)
- [Go net.Dialer Documentation](https://pkg.go.dev/net#Dialer)
- [OWASP: Transport Layer Protection](https://owasp.org/www-project-top-ten/)
