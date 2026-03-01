package main

import (
	"context"
	"fmt"
	"net/http"
)

const (
	configKeyNames = "names"
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

var HandlerRegisterer = registerer("middleware")

var middlewareLogPrefix = fmt.Sprintf("[SERVER PLUGIN %s]", HandlerRegisterer)

func (r registerer) RegisterHandlers(f func(
	name string,
	handler func(context.Context, map[string]any, http.Handler) (http.Handler, error),
)) {
	f(string(r), r.registerHandlers)
}

func (r registerer) registerHandlers(_ context.Context, extra map[string]any, next http.Handler) (http.Handler, error) {
	config, ok := extra[string(r)].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("config for %s not found", r)
	}

	var names []any
	if namesConfig, ok := config[configKeyNames]; ok {
		names, ok = namesConfig.([]any)
		if !ok {
			return nil, fmt.Errorf("failed to type cast namesConfig (middleware pipeline)")
		}
	}

	return buildHandler(names, config, next)
}

func buildHandler(names []any, config map[string]any, next http.Handler) (http.Handler, error) {
	last := len(names) - 1
	handler := next

	for i := last; i >= 0; i-- {
		name := names[i].(string)
		cfg, _ := config[name].(map[string]any)

		mw, ok := middlewares[name]
		if !ok {
			return nil, fmt.Errorf("middleware %s not registered", name)
		}

		h, err := mw(handler, cfg)
		if err != nil {
			return nil, err
		}

		handler = h
	}

	return handler, nil
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
}

func init() {
	logger.Info(middlewareLogPrefix, "Registering middleware plugin")
	registerAvailableMiddlewares([]availableMiddleware{
		{
			name: "static-response",
			f:    RegisterStaticResponse,
		},
	})
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
