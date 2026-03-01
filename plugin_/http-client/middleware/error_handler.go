package main

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	"github.com/beevik/etree"
	"github.com/samber/lo"
)

type bodyWriter func(http.ResponseWriter, *http.Request, *etree.Document)

func RegisterErrorHandler(next http.Handler, config map[string]any) (http.Handler, error) {
	cfg := mustValidErrorHandlerConfig(config)
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var doc *etree.Document

		bufferedWriter := httptest.NewRecorder()
		next.ServeHTTP(bufferedWriter, request)

		httpCode := bufferedWriter.Code

		for k, vv := range bufferedWriter.Header() {
			for _, v := range vv {
				writer.Header().Add(k, v)
			}
		}

		if httpCode == http.StatusNotFound && cfg.DownstreamNotFound {
			writeBody := newClientNotFoundJsonBodyWriter()
			writer.Header().Del("Content-Length")
			writer.WriteHeader(httpCode)
			writeBody(writer, request, doc)

			return
		}

		if bufferedWriter.Body.Len() > 0 {
			writer.WriteHeader(httpCode)

			_, _ = io.Copy(writer, bufferedWriter.Body)
			return
		}

		mediaType, _, _ := mime.ParseMediaType(bufferedWriter.Header().Get(headerKeyContentType))
		if mediaType == "" {
			mediaType = applicationJSONContentType
		}

		// create response based on content type & http code
		writeBody := lo.Switch[string, bodyWriter](mediaType).
			Case(applicationJSONContentType, newJsonBodyWriter(httpCode)).
			Default(newPlainTextBodyWriter(httpCode))

		writer.WriteHeader(httpCode)
		writeBody(writer, request, doc)
	}), nil
}

type jsonResponse[T any] struct {
	Code       string      `json:"code"`
	Message    string      `json:"message"`
	Data       T           `json:"data,omitempty"`
	ServerTime json.Number `json:"serverTime"`
}

func getServerTime() json.Number {
	return json.Number(strconv.FormatInt(time.Now().Unix(), 10))
}

func newPlainTextBodyWriter(code int) bodyWriter {
	return func(w http.ResponseWriter, _ *http.Request, _ *etree.Document) {
		fmt.Fprintln(w, lo.Switch[int, string](code).
			Case(http.StatusBadRequest, "bad request").
			Case(http.StatusUnauthorized, "unauthorized").
			Case(http.StatusGatewayTimeout, "gateway timeout").
			Case(http.StatusServiceUnavailable, "service unavailable").
			Case(http.StatusBadGateway, "bad gateway").
			Case(http.StatusTooManyRequests, "too many request").
			Default("internal server error"))
	}
}

func newJsonBodyWriter(code int) bodyWriter {
	return func(w http.ResponseWriter, _ *http.Request, _ *etree.Document) {
		data := lo.Switch[int, jsonResponse[any]](code).
			Case(http.StatusBadRequest, jsonResponse[any]{
				Code:       "BAD_REQUEST",
				Message:    "Bad request",
				ServerTime: getServerTime(),
			}).
			Case(http.StatusUnauthorized, jsonResponse[any]{
				Code:       "UNAUTHORIZED",
				Message:    "Unauthorized",
				ServerTime: getServerTime(),
			}).
			Case(http.StatusGatewayTimeout, jsonResponse[any]{
				Code:       "GATEWAY_TIMEOUT",
				Message:    "Service timeout",
				ServerTime: getServerTime(),
			}).
			Case(http.StatusServiceUnavailable, jsonResponse[any]{
				Code:       "SERVICE_UNAVAILABLE",
				Message:    "Service is currently unavailable",
				ServerTime: getServerTime(),
			}).
			Case(http.StatusBadGateway, jsonResponse[any]{
				Code:       "BAD_GATEWAY",
				Message:    "Bad gateway",
				ServerTime: getServerTime(),
			}).
			Case(http.StatusTooManyRequests, jsonResponse[any]{
				Code:       "TOO_MANY_REQUEST",
				Message:    "Too many request",
				ServerTime: getServerTime(),
			}).
			Default(jsonResponse[any]{
				Code:       "SYSTEM_ERROR",
				Message:    "Internal system error",
				ServerTime: getServerTime(),
			})
		_ = json.NewEncoder(w).Encode(data)
	}
}

func newClientNotFoundJsonBodyWriter() bodyWriter {
	return func(w http.ResponseWriter, _ *http.Request, _ *etree.Document) {
		response := jsonResponse[any]{
			Code:       "CLIENT_NOT_FOUND",
			Message:    "Client not found",
			ServerTime: getServerTime(),
		}
		_ = json.NewEncoder(w).Encode(response)
	}
}

type errorHandlerConfig struct {
	DownstreamNotFound bool `json:"downstreamNotFound"`
}

func mustValidErrorHandlerConfig(config map[string]any) errorHandlerConfig {
	var cfg errorHandlerConfig
	cfgBytes, err := json.Marshal(config)
	if err != nil {
		panic("marshalling error handler config error")
	}
	if err := json.Unmarshal(cfgBytes, &cfg); err != nil {
		panic("unmarshalling error handler config error")
	}

	return cfg
}
