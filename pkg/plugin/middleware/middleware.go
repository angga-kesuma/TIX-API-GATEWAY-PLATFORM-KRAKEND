package middleware

import (
	"net/http"

	"github.com/tiket/TIX-API-GATEWAY-PLATFORM-KRAKEND/pkg/plugin/config"
)

type Middleware interface {
	Run(next http.Handler, pluginsConfig *config.EndpointConfig) (http.Handler, error)
}

var middlewares []Middleware

func RegisterMiddleware(cfg *config.AppConfig) {

	// order of middleware matters here
	for _, name := range cfg.Middlewares {

		switch name {
		case MemberAuthName:

			// init middleware member_session
			m := NewMemberAuth(cfg)
			middlewares = append(middlewares, m)
			break

		case RedisRateLimitName:
			// init middleware member_session
			m := NewRedisRateLimit(cfg)
			middlewares = append(middlewares, m)
			break
		}
	}

}

func RunMiddleware(last http.Handler, pluginsConfig *config.EndpointConfig) (http.Handler, error) {

	next := last
	for _, middleware := range middlewares {
		n, err := middleware.Run(next, pluginsConfig)
		if err != nil {
			return nil, err
		}
		next = n
	}
	// To reduce unexpected error The sorter should be defined here.
	// Rate limit
	// Member-session Auth
	// Headers
	// reduce cookies
	return next, nil
}
