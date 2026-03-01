package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	redisrate "github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestRegisterRedisRateLimit(t *testing.T) {
	t.Run("when redisClient not exist", func(t *testing.T) {
		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		})

		config := map[string]any{}

		assert.Panics(t, func() { _, _ = RegisterRedisRateLimit(nextFn, config) }, "The code did not panic")
	})

	t.Run("when missing endpoint config", func(t *testing.T) {
		server, _ := miniredis.Run()
		redisClient = redis.NewClient(&redis.Options{
			Addr: server.Addr(),
		})
		defer server.Close()

		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		})

		config := map[string]any{}

		assert.Panics(t, func() { _, _ = RegisterRedisRateLimit(nextFn, config) }, "The code did not panic")
	})

	t.Run("when missing limiters config", func(t *testing.T) {
		server, _ := miniredis.Run()
		redisClient = redis.NewClient(&redis.Options{
			Addr: server.Addr(),
		})
		defer server.Close()

		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		})

		config := map[string]any{
			"endpoint": "test-endpoint",
		}

		assert.Panics(t, func() { _, _ = RegisterRedisRateLimit(nextFn, config) }, "The code did not panic")
	})

	t.Run("when invalid limiter config", func(t *testing.T) {
		server, _ := miniredis.Run()
		redisClient = redis.NewClient(&redis.Options{
			Addr: server.Addr(),
		})
		defer server.Close()

		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		})

		config := map[string]any{
			"endpoint": "test-endpoint",
			"limiters": []map[string]any{
				{
					"maxBurst":      0, // Invalid: zero burst
					"maxRatePerMin": 10,
					"varyBy": map[string]any{
						"method": true,
					},
				},
			},
		}

		assert.Panics(t, func() { _, _ = RegisterRedisRateLimit(nextFn, config) }, "The code did not panic")
	})

	t.Run("when success with basic config", func(t *testing.T) {
		server, _ := miniredis.Run()
		redisClient = redis.NewClient(&redis.Options{
			Addr: server.Addr(),
		})
		defer server.Close()

		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		})

		config := map[string]any{
			"endpoint": "test-endpoint",
			"limiters": []map[string]any{
				{
					"maxBurst":      5,
					"maxRatePerMin": 10,
					"varyBy": map[string]any{
						"method": true,
					},
				},
			},
		}

		rateLimitFn, err := RegisterRedisRateLimit(nextFn, config)
		assert.NoError(t, err)

		writer := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)

		rateLimitFn.ServeHTTP(writer, req)

		assert.Equal(t, http.StatusOK, writer.Code)
	})

	t.Run("when success with complex config like ttd_test.json", func(t *testing.T) {
		server, _ := miniredis.Run()
		redisClient = redis.NewClient(&redis.Options{
			Addr: server.Addr(),
		})
		defer server.Close()

		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		})

		config := map[string]any{
			"endpoint": "ttd-gateway/tix-events-v2-inventory/partners/v2/products/count",
			"limiters": []map[string]any{
				{
					"maxBurst":      1,
					"maxRatePerMin": 10,
					"varyBy": map[string]any{
						"method": true,
					},
				},
				{
					"maxBurst":      1,
					"maxRatePerMin": 10,
					"varyBy": map[string]any{
						"method":  true,
						"headers": []string{"x-product-id"},
						"params":  []string{"test"},
					},
				},
				{
					"maxBurst":      1,
					"maxRatePerMin": 10,
					"varyBy": map[string]any{
						"user": true,
					},
					"override": map[string]any{
						"TTD_GATEWAY:ttd-gateway/tix-events-v2-inventory/partners/v2/products/count:sitti.eldrin@tiket.com": map[string]any{
							"maxBurst":      1,
							"maxRatePerMin": 10,
						},
					},
				},
			},
		}

		rateLimitFn, err := RegisterRedisRateLimit(nextFn, config)
		assert.NoError(t, err)

		writer := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/tix-events-v2-inventory/partners/v2/products/count", nil)
		req.Header.Set("x-product-id", "12345")

		rateLimitFn.ServeHTTP(writer, req)

		assert.Equal(t, http.StatusOK, writer.Code)
	})

	t.Run("when rate limit exceeded", func(t *testing.T) {
		server, _ := miniredis.Run()
		redisClient = redis.NewClient(&redis.Options{
			Addr: server.Addr(),
		})
		defer server.Close()

		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		})

		config := map[string]any{
			"endpoint": "test-endpoint",
			"limiters": []map[string]any{
				{
					"maxBurst":      1,
					"maxRatePerMin": 1,
					"varyBy": map[string]any{
						"headers": []string{"TTD-Partner-Id"},
					},
				},
			},
		}

		rateLimitFn, err := RegisterRedisRateLimit(nextFn, config)
		assert.NoError(t, err)

		// First request should succeed
		firstWriter := httptest.NewRecorder()
		firstReq := httptest.NewRequest("GET", "/test", nil)
		firstReq.Header.Set("TTD-Partner-Id", "35010227")
		rateLimitFn.ServeHTTP(firstWriter, firstReq)
		assert.Equal(t, http.StatusOK, firstWriter.Code)

		// Second request should be rate limited
		secondWriter := httptest.NewRecorder()
		secondReq := httptest.NewRequest("GET", "/test", nil)
		secondReq.Header.Set("TTD-Partner-Id", "35010227")
		rateLimitFn.ServeHTTP(secondWriter, secondReq)
		assert.Equal(t, http.StatusTooManyRequests, secondWriter.Code)
	})

	t.Run("when rate limit resets after time window", func(t *testing.T) {
		server, _ := miniredis.Run()
		redisClient = redis.NewClient(&redis.Options{
			Addr: server.Addr(),
		})
		defer server.Close()

		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		})

		config := map[string]any{
			"endpoint": "test-endpoint",
			"limiters": []map[string]any{
				{
					"maxBurst":      1,
					"maxRatePerMin": 1,
					"varyBy": map[string]any{
						"headers": []string{"TTD-Partner-Id"},
					},
				},
			},
		}

		rateLimitFn, err := RegisterRedisRateLimit(nextFn, config)
		assert.NoError(t, err)

		// First request should succeed
		firstWriter := httptest.NewRecorder()
		firstReq := httptest.NewRequest("GET", "/test", nil)
		firstReq.Header.Set("TTD-Partner-Id", "35010227")
		rateLimitFn.ServeHTTP(firstWriter, firstReq)
		assert.Equal(t, http.StatusOK, firstWriter.Code)

		// Second request should be rate limited
		secondWriter := httptest.NewRecorder()
		secondReq := httptest.NewRequest("GET", "/test", nil)
		secondReq.Header.Set("TTD-Partner-Id", "35010227")
		rateLimitFn.ServeHTTP(secondWriter, secondReq)
		assert.Equal(t, http.StatusTooManyRequests, secondWriter.Code)

		// Wait for rate limit to reset (simulate time passage)
		server.FastForward(time.Minute + time.Second)

		// Third request should succeed after reset
		thirdWriter := httptest.NewRecorder()
		thirdReq := httptest.NewRequest("GET", "/test", nil)
		thirdReq.Header.Set("TTD-Partner-Id", "35010227")
		rateLimitFn.ServeHTTP(thirdWriter, thirdReq)
		assert.Equal(t, http.StatusOK, thirdWriter.Code)
	})

	t.Run("when different partners have separate rate limits", func(t *testing.T) {
		server, _ := miniredis.Run()
		redisClient = redis.NewClient(&redis.Options{
			Addr: server.Addr(),
		})
		defer server.Close()

		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		})

		config := map[string]any{
			"endpoint": "test-endpoint",
			"limiters": []map[string]any{
				{
					"maxBurst":      1,
					"maxRatePerMin": 1,
					"varyBy": map[string]any{
						"headers": []string{"TTD-Partner-Id"},
					},
				},
			},
		}

		rateLimitFn, err := RegisterRedisRateLimit(nextFn, config)
		assert.NoError(t, err)

		// Partner 1 first request
		writer1 := httptest.NewRecorder()
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.Header.Set("TTD-Partner-Id", "35010227")
		rateLimitFn.ServeHTTP(writer1, req1)
		assert.Equal(t, http.StatusOK, writer1.Code)

		// Partner 2 first request (should succeed even though Partner 1 is rate limited)
		writer2 := httptest.NewRecorder()
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.Header.Set("TTD-Partner-Id", "35010228")
		rateLimitFn.ServeHTTP(writer2, req2)
		assert.Equal(t, http.StatusOK, writer2.Code)

		// Partner 1 second request (should be rate limited)
		writer1Second := httptest.NewRecorder()
		req1Second := httptest.NewRequest("GET", "/test", nil)
		req1Second.Header.Set("TTD-Partner-Id", "35010227")
		rateLimitFn.ServeHTTP(writer1Second, req1Second)
		assert.Equal(t, http.StatusTooManyRequests, writer1Second.Code)
	})

	t.Run("when concurrent requests with rate limiting", func(t *testing.T) {
		server, _ := miniredis.Run()
		redisClient = redis.NewClient(&redis.Options{
			Addr: server.Addr(),
		})
		defer server.Close()

		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Millisecond) // Simulate processing time
		})

		config := map[string]any{
			"endpoint": "test-endpoint",
			"limiters": []map[string]any{
				{
					"maxBurst":      2,
					"maxRatePerMin": 2,
					"varyBy": map[string]any{
						"headers": []string{"TTD-Partner-Id"},
					},
				},
			},
		}

		rateLimitFn, err := RegisterRedisRateLimit(nextFn, config)
		assert.NoError(t, err)

		// Make concurrent requests
		results := make(chan int, 5)
		for i := 0; i < 5; i++ {
			go func() {
				writer := httptest.NewRecorder()
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("TTD-Partner-Id", "35010227")
				rateLimitFn.ServeHTTP(writer, req)
				results <- writer.Code
			}()
		}

		// Collect results
		successCount := 0
		rateLimitedCount := 0
		for i := 0; i < 5; i++ {
			code := <-results
			if code == http.StatusOK {
				successCount++
			} else if code == http.StatusTooManyRequests {
				rateLimitedCount++
			}
		}

		// - all 5 requests produced either OK or 429
		// - at least one was rate limited
		// - not all requests succeeded
		assert.Equal(t, 5, successCount+rateLimitedCount)
		assert.GreaterOrEqual(t, rateLimitedCount, 1)
		assert.Less(t, successCount, 5)
	})
}

func TestRateLimitKeyGeneration(t *testing.T) {
	t.Run("test method vary by keyer", func(t *testing.T) {
		methodKeyer := &methodVaryByKeyer{}
		req := httptest.NewRequest("POST", "/test", nil)
		assert.Equal(t, "POST", methodKeyer.key(req))
	})

	t.Run("test header vary by keyer", func(t *testing.T) {
		headerKeyer := &headerVaryByKeyer{headerKeys: []string{"TTD-Partner-Id", "User-Agent"}}
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("TTD-Partner-Id", "35010227")
		req.Header.Set("User-Agent", "test-agent")
		assert.Equal(t, "35010227:test-agent", headerKeyer.key(req))
	})

	t.Run("test param vary by keyer", func(t *testing.T) {
		paramKeyer := &paramVaryByKeyer{paramKeys: []string{"partnerId", "userId"}}
		req := httptest.NewRequest("GET", "/test?partnerId=123&userId=456", nil)
		assert.Equal(t, "123:456", paramKeyer.key(req))
	})

	t.Run("test username vary by keyer", func(t *testing.T) {
		usernameKeyer := &usernameVaryByKeyer{}
		req := httptest.NewRequest("GET", "/test", nil)

		// Test with default username
		assert.Equal(t, "guest", usernameKeyer.key(req))

		// Test with session data in context
		ctx := context.WithValue(req.Context(), memberAuthSessionDataKey, memberAuthSessionData{
			Username: "testuser@example.com",
		})
		req = req.WithContext(ctx)
		assert.Equal(t, "testuser@example.com", usernameKeyer.key(req))
	})
}

func TestLimitValidation(t *testing.T) {
	t.Run("valid limit", func(t *testing.T) {
		limit := limit{MaxBurst: 5, MaxRatePerMin: 10}
		assert.True(t, limit.isValid())
	})

	t.Run("invalid limit - zero values", func(t *testing.T) {
		l := limit{MaxBurst: 0, MaxRatePerMin: 10}
		assert.False(t, l.isValid())

		l = limit{MaxBurst: 5, MaxRatePerMin: 0}
		assert.False(t, l.isValid())
	})

	t.Run("invalid limit - negative values", func(t *testing.T) {
		l := limit{MaxBurst: -1, MaxRatePerMin: 10}
		assert.False(t, l.isValid())

		l = limit{MaxBurst: 5, MaxRatePerMin: -1}
		assert.False(t, l.isValid())
	})
}

func TestLimiterConfigValidation(t *testing.T) {
	t.Run("valid limiter config", func(t *testing.T) {
		config := limiterConfig{
			limit:  limit{MaxBurst: 5, MaxRatePerMin: 10},
			VaryBy: varyBy{Method: true},
			LimitByKey: map[string]limit{
				"key1": {MaxBurst: 3, MaxRatePerMin: 5},
			},
		}
		assert.True(t, config.isValid())
	})

	t.Run("invalid limiter config - invalid default limit", func(t *testing.T) {
		config := limiterConfig{
			limit:  limit{MaxBurst: 0, MaxRatePerMin: 10},
			VaryBy: varyBy{Method: true},
		}
		assert.False(t, config.isValid())
	})

	t.Run("invalid limiter config - invalid key limit", func(t *testing.T) {
		config := limiterConfig{
			limit:  limit{MaxBurst: 5, MaxRatePerMin: 10},
			VaryBy: varyBy{Method: true},
			LimitByKey: map[string]limit{
				"key1": {MaxBurst: 0, MaxRatePerMin: 5},
			},
		}
		assert.False(t, config.isValid())
	})
}

func TestVaryByKeyers(t *testing.T) {
	t.Run("test varyBy keyers generation", func(t *testing.T) {
		varyBy := varyBy{
			Method:  true,
			User:    true,
			Headers: []string{"TTD-Partner-Id", "User-Agent"},
			Params:  []string{"partnerId", "userId"},
		}

		keyers := varyBy.keyers()
		assert.Len(t, keyers, 4) // method + user + headers + params

		// Test that we get the right types
		assert.IsType(t, &methodVaryByKeyer{}, keyers[0])
		assert.IsType(t, &usernameVaryByKeyer{}, keyers[1])
		assert.IsType(t, &headerVaryByKeyer{}, keyers[2])
		assert.IsType(t, &paramVaryByKeyer{}, keyers[3])
	})

	t.Run("test varyBy with only method", func(t *testing.T) {
		varyBy := varyBy{Method: true}
		keyers := varyBy.keyers()
		assert.Len(t, keyers, 1)
		assert.IsType(t, &methodVaryByKeyer{}, keyers[0])
	})

	t.Run("test varyBy with only headers", func(t *testing.T) {
		varyBy := varyBy{Headers: []string{"TTD-Partner-Id"}}
		keyers := varyBy.keyers()
		assert.Len(t, keyers, 1)
		assert.IsType(t, &headerVaryByKeyer{}, keyers[0])
	})
}

func TestRedisGCRARateLimiter(t *testing.T) {
	server, _ := miniredis.Run()
	defer server.Close()

	redisClient = redis.NewClient(&redis.Options{
		Addr: server.Addr(),
	})

	t.Run("test rate limit key generation", func(t *testing.T) {
		limiter := redisGCRARateLimiter{
			prefixKey: "TTD_GATEWAY:test-endpoint",
			varyByKeyers: []varyByKeyer{
				&methodVaryByKeyer{},
				&headerVaryByKeyer{headerKeys: []string{"TTD-Partner-Id"}},
			},
		}

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("TTD-Partner-Id", "35010227")

		key := limiter.rateLimitKey(req)
		assert.Equal(t, "TTD_GATEWAY:test-endpoint:GET:35010227", key)
	})

	t.Run("test rate limit with override", func(t *testing.T) {
		limiter := redisGCRARateLimiter{
			prefixKey: "TTD_GATEWAY:test-endpoint",
			varyByKeyers: []varyByKeyer{
				&headerVaryByKeyer{headerKeys: []string{"TTD-Partner-Id"}},
			},
			defaultRateLimit: redisrate.Limit{
				Rate:   10,
				Burst:  5,
				Period: time.Minute,
			},
			rateLimitPerKey: map[string]redisrate.Limit{
				"TTD_GATEWAY:test-endpoint:35010227": {
					Rate:   5,
					Burst:  2,
					Period: time.Minute,
				},
			},
		}

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("TTD-Partner-Id", "35010227")

		key := limiter.rateLimitKey(req)
		limit := limiter.rateLimitPerKey[key]
		assert.Equal(t, 5, limit.Rate)
		assert.Equal(t, 2, limit.Burst)
	})
}
