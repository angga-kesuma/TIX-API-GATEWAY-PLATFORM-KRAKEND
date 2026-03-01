package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	redisrate "github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"
	"github.com/throttled/throttled/v2"
)

func RegisterRedisRateLimit(next http.Handler, config map[string]any) (http.Handler, error) {
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		panic("can't connect to redis")
	}

	redisLimitConfig := mustValidRedisLimitConfig(config)
	rateLimiters := newRedisGCRARateLimiters(redisLimitConfig, redisClient)

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		httpRateLimiter := throttled.HTTPRateLimiterCtx{
			DeniedHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
			}),
			Error: func(w http.ResponseWriter, r *http.Request, err error) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			RateLimiter: &finalRateLimiter{
				rateLimiters: rateLimiters,
				request:      request,
			},
		}

		httpRateLimiter.RateLimit(next).ServeHTTP(writer, request)
	}), nil
}

type finalRateLimiter struct {
	rateLimiters []redisGCRARateLimiter
	request      *http.Request
}

func (f *finalRateLimiter) RateLimitCtx(ctx context.Context, _ string, quantity int) (bool, throttled.RateLimitResult, error) {
	if len(f.rateLimiters) == 1 {
		return f.rateLimiters[0].result(ctx, f.request, quantity)
	}

	var wg sync.WaitGroup
	var once sync.Once

	var finalResult throttled.RateLimitResult
	var finalError error
	var finalIsLimited bool

	for _, limiter := range f.rateLimiters {
		wg.Add(1)

		go func(limiter redisGCRARateLimiter) {
			defer wg.Done()

			isLimited, result, err := limiter.result(ctx, f.request, quantity)

			if isLimited || err != nil {
				once.Do(func() {
					finalIsLimited = isLimited
					finalResult = result
					finalError = err
				})
			}
		}(limiter)
	}

	wg.Wait()

	return finalIsLimited, finalResult, finalError
}

func newRedisGCRARateLimiters(config redisRateLimitConfig, client *redis.Client) []redisGCRARateLimiter {
	redisLimiter := redisrate.NewLimiter(client)
	limiters := config.Limiters

	rateLimiters := make([]redisGCRARateLimiter, len(limiters))
	for idx, limiterConfig := range limiters {
		rateLimiters[idx] = newRedisGCRARateLimiter(redisLimiter, config.Endpoint, limiterConfig)
	}

	return rateLimiters
}

func newRedisGCRARateLimiter(limiter *redisrate.Limiter, endpoint string, limiterConfig limiterConfig) redisGCRARateLimiter {
	return redisGCRARateLimiter{
		limiter:          limiter,
		defaultRateLimit: limiterConfig.redisRateLimit(),
		rateLimitPerKey: lo.MapEntries(limiterConfig.LimitByKey, func(key string, limit limit) (string, redisrate.Limit) {
			return key, limit.redisRateLimit()
		}),
		prefixKey:    fmt.Sprintf("%s:%s", serviceID, endpoint),
		varyByKeyers: limiterConfig.VaryBy.keyers(),
	}
}

type redisGCRARateLimiter struct {
	limiter          *redisrate.Limiter
	defaultRateLimit redisrate.Limit
	rateLimitPerKey  map[string]redisrate.Limit
	prefixKey        string
	varyByKeyers     []varyByKeyer
}

func (r *redisGCRARateLimiter) result(ctx context.Context, request *http.Request, quantity int) (bool, throttled.RateLimitResult, error) {
	rateLimitKey := r.rateLimitKey(request)
	limit := lo.ValueOr(r.rateLimitPerKey, rateLimitKey, r.defaultRateLimit)

	rlc := throttled.RateLimitResult{
		Limit:      limit.Burst,
		RetryAfter: -1,
	}

	result, err := r.limiter.AllowN(ctx, rateLimitKey, limit, quantity)
	if err != nil {
		return false, rlc, err
	}
	if result.Allowed <= 0 {
		rlc.RetryAfter = result.RetryAfter
	}
	rlc.Remaining = result.Remaining
	rlc.ResetAfter = result.ResetAfter

	return result.Allowed <= 0, rlc, err
}

func (r *redisGCRARateLimiter) rateLimitKey(request *http.Request) string {
	keys := []string{r.prefixKey}

	for _, keyer := range r.varyByKeyers {
		keys = append(keys, keyer.key(request))
	}

	return strings.Join(keys, ":")
}

type limit struct {
	MaxBurst      int `json:"maxBurst"`
	MaxRatePerMin int `json:"maxRatePerMin"`
}

func (l *limit) isValid() bool {
	if l.MaxBurst <= 0 || l.MaxRatePerMin <= 0 {
		return false
	}

	return true
}

func (l *limit) redisRateLimit() redisrate.Limit {
	return redisrate.Limit{
		Rate:   l.MaxRatePerMin,
		Burst:  l.MaxBurst,
		Period: time.Minute,
	}
}

type varyByKeyer interface {
	key(request *http.Request) string
}

type methodVaryByKeyer struct{}

func (m *methodVaryByKeyer) key(request *http.Request) string {
	return request.Method
}

type usernameVaryByKeyer struct{}

func (u *usernameVaryByKeyer) key(request *http.Request) string {
	username := defaultUsername
	if sessionData, ok := request.Context().Value(memberAuthSessionDataKey).(memberAuthSessionData); ok {
		username = sessionData.Username
	}

	return strings.ToLower(username)
}

type headerVaryByKeyer struct {
	headerKeys []string
}

func (h *headerVaryByKeyer) key(request *http.Request) string {
	reqHeader := request.Header
	values := make([]string, len(h.headerKeys))

	for idx, headerKey := range h.headerKeys {
		values[idx] = strings.ToLower(reqHeader.Get(headerKey))
	}

	return strings.Join(values, ":")
}

type paramVaryByKeyer struct {
	paramKeys []string
}

func (p *paramVaryByKeyer) key(request *http.Request) string {
	reqParams := request.URL.Query()
	values := make([]string, len(p.paramKeys))

	for idx, paramKey := range p.paramKeys {
		values[idx] = strings.ToLower(reqParams.Get(paramKey))
	}

	return strings.Join(values, ":")
}

type varyBy struct {
	Method  bool     `json:"method"`
	User    bool     `json:"user"`
	Headers []string `json:"headers"`
	Params  []string `json:"params"`
}

func (v *varyBy) keyers() []varyByKeyer {
	keyers := make([]varyByKeyer, 0)

	if v.Method {
		keyers = append(keyers, &methodVaryByKeyer{})
	}

	if v.User {
		keyers = append(keyers, &usernameVaryByKeyer{})
	}

	if len(v.Headers) > 0 {
		keyers = append(keyers, &headerVaryByKeyer{
			headerKeys: v.Headers,
		})
	}

	if len(v.Params) > 0 {
		keyers = append(keyers, &paramVaryByKeyer{
			paramKeys: v.Params,
		})
	}

	return keyers
}

type limiterConfig struct {
	limit
	VaryBy     varyBy           `json:"varyBy"`
	LimitByKey map[string]limit `json:"override"`
}

func (l *limiterConfig) isValid() bool {
	if !l.limit.isValid() {
		return false
	}

	for _, keyLimit := range l.LimitByKey {
		if !keyLimit.isValid() {
			return false
		}
	}

	return true
}

type redisRateLimitConfig struct {
	Endpoint string          `json:"endpoint"`
	Limiters []limiterConfig `json:"limiters"`
}

func mustValidRedisLimitConfig(config map[string]any) redisRateLimitConfig {
	endpoint, ok := config["endpoint"].(string)
	if !ok {
		panic("missing base endpoint config")
	}

	var cfg redisRateLimitConfig
	cfgBytes, err := json.Marshal(config)
	if err != nil {
		panic("marshalling config error " + endpoint)
	}
	if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
		panic("unmarshalling config error " + endpoint)
	}

	if len(cfg.Limiters) == 0 {
		panic("missing limiters config " + endpoint)
	}

	for _, limiter := range cfg.Limiters {
		if !limiter.isValid() {
			panic("invalid limiters config " + endpoint)
		}
	}

	return redisRateLimitConfig{
		Endpoint: cfg.Endpoint,
		Limiters: cfg.Limiters,
	}
}
