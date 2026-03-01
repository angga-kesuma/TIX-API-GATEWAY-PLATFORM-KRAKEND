package middleware

import (
	"fmt"
	"net/http"

	"github.com/redis/go-redis/v9"
	"github.com/tiket/TIX-API-GATEWAY-PLATFORM-KRAKEND/pkg/plugin/config"
)

var RedisRateLimitName = "redis_rate_limit"

func NewRedisRateLimit(cfg *config.AppConfig) *RedisRateLimit {
	o := &RedisRateLimit{}

	o.Cfg = cfg

	if cfg.RedisConfig.IsCluster {
		o.RedisClient = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    cfg.RedisConfig.ClusterHost,
			Password: cfg.RedisConfig.Password,
		})
	} else {
		o.RedisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.RedisConfig.Host,
			Password: cfg.RedisConfig.Password,
			DB:       cfg.RedisConfig.DBIndex,
		})
	}

	return o
}

type RedisRateLimit struct {
	Cfg         *config.AppConfig
	RedisClient redis.Cmdable
}

func (o *RedisRateLimit) Run(next http.Handler, pluginsConfig *config.EndpointConfig) (http.Handler, error) {

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		fmt.Println("Redis rate limit plugin")

		next.ServeHTTP(w, req)
	}), nil
}
