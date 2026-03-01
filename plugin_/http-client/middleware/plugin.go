package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
)

const (
	configKeyNames      = "names"
	configKeyNoRedirect = "no-redirect"
)

type (
	registerer string

	registerMiddlewareFunc func(http.Handler, map[string]any) (http.Handler, error)

	availableMiddleware struct {
		name string
		f    registerMiddlewareFunc
	}
)

var middlewares = map[string]registerMiddlewareFunc{}
var redisClient *redis.Client

var ClientRegisterer = registerer("middleware")

var middlewareLogPrefix = fmt.Sprintf("[PLUGIN %s]", ClientRegisterer)

func (r registerer) RegisterClients(f func(
	name string,
	handler func(context.Context, map[string]interface{}) (http.Handler, error),
)) {
	f(string(r), r.registerClients)
}

func (r registerer) registerClients(_ context.Context, extra map[string]interface{}) (http.Handler, error) {
	config, ok := extra[string(r)].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("config for %s not found", r)
	}

	var names []any
	if namesConfig, ok := config[configKeyNames]; ok {
		names, ok = namesConfig.([]any)
		if !ok {
			return nil, errors.New("failed to type cast namesConfig (middleware pipeline)")
		}
	}

	return buildHandler(names, config)
}

var noRedirectHTTPClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		// Prevent following redirects
		return http.ErrUseLastResponse
	},
}

func buildHandler(names []any, config map[string]any) (http.Handler, error) {
	last := len(names) - 1

	var next http.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var urlErr *url.Error
		httpClient := http.DefaultClient

		if isNoRedirect, _ := config[configKeyNoRedirect].(bool); isNoRedirect {
			httpClient = noRedirectHTTPClient
		}

		request.Header = setMandatoryHeaders(request)

		logger.Info("Request ID: " + request.Header.Get(headerKeyRequestID))

		resp, err := httpClient.Do(request)
		if errors.As(err, &urlErr) {
			logger.Error(middlewareLogPrefix, fmt.Sprintf("Failed to httpClient.Do, path: %s, error: %s", request.URL.Path, err.Error()))
			if urlErr.Timeout() {
				writer.WriteHeader(http.StatusGatewayTimeout)
				return
			}
			if urlErr.Temporary() {
				writer.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}
		if err != nil {
			logger.Error(middlewareLogPrefix, fmt.Sprintf("Failed to httpClient.Do, path: %s, error: %s", request.URL.Path, err.Error()))
			writer.WriteHeader(http.StatusBadGateway)
			return
		}

		// Set Header from Response
		for k, hs := range resp.Header {
			for _, h := range hs {
				writer.Header().Add(k, h)
			}
		}

		// Set StatusCode from Response
		writer.WriteHeader(resp.StatusCode)
		if resp.Body == nil {
			return
		}

		// Set final response to writer
		_, _ = io.Copy(writer, resp.Body)
		resp.Body.Close()
	})

	for i := last; i >= 0; i-- {
		name := names[i].(string)
		c, _ := config[name].(map[string]any)
		handler, err := middlewares[name](next, c)
		if err != nil {
			return nil, err
		}
		next = handler
	}

	return next, nil
}

func registerAvailableMiddlewares(availableMiddlewares []availableMiddleware) {
	for _, m := range availableMiddlewares {
		logger.Info(middlewareLogPrefix, fmt.Sprintf("Registering middleware: %s", m.name))
		if _, ok := middlewares[m.name]; ok {
			logger.Warning(middlewareLogPrefix, fmt.Sprintf("Middleware %s already registered", m.name))
		}
		middlewares[m.name] = m.f
	}
}

var logger Logger = NoOpLogger{}

func (registerer) RegisterLogger(v interface{}) {
	l, ok := v.(Logger)
	if !ok {
		return
	}
	logger = l
	logger.Debug(middlewareLogPrefix, "Logger loaded")
	initPlugin()
}

func newRedisClient() *redis.Client {
	redisHost := os.Getenv("REDIS_HOST")
	redisPass := os.Getenv("REDIS_PASSWORD")
	redisDB := os.Getenv("REDIS_DB_INDEX")
	redisDBIdx, err := strconv.Atoi(redisDB)
	if err != nil {
		redisDBIdx = 0
	}

	return redis.NewClient(&redis.Options{
		Addr:     redisHost,
		Password: redisPass,
		DB:       redisDBIdx,
	})
}

func initPlugin() {
	logger.Info(middlewareLogPrefix, "Registering middleware plugin")
	registerAvailableMiddlewares([]availableMiddleware{
		{
			name: "error-handler",
			f:    RegisterErrorHandler,
		},
		{
			name: "redis-rate-limit",
			f:    RegisterRedisRateLimit,
		},
		{
			name: "member-auth",
			f:    RegisterMemberAuth,
		},
	})
	redisClient = newRedisClient()
}

type Logger interface {
	Debug(v ...interface{})
	Info(v ...interface{})
	Warning(v ...interface{})
	Error(v ...interface{})
	Critical(v ...interface{})
	Fatal(v ...interface{})
}

type NoOpLogger struct{}

func (n NoOpLogger) Debug(_ ...interface{})    {}
func (n NoOpLogger) Info(_ ...interface{})     {}
func (n NoOpLogger) Warning(_ ...interface{})  {}
func (n NoOpLogger) Error(_ ...interface{})    {}
func (n NoOpLogger) Critical(_ ...interface{}) {}
func (n NoOpLogger) Fatal(_ ...interface{})    {}
