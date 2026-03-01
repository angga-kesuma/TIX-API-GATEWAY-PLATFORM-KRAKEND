package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultErrorHandler(t *testing.T) {
	config := map[string]any{}

	t.Run("when bad request error with 400 http status", func(t *testing.T) {
		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(headerKeyContentType, "text/plain")
			w.WriteHeader(http.StatusBadRequest)
		})

		errorHandlerFn, _ := RegisterErrorHandler(nextFn, config)

		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/search", nil)

		errorHandlerFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusBadRequest, respWriter.Code)
		assert.Equal(t, "bad request\n", respWriter.Body.String())
	})

	t.Run("when unauthorized error with 401 http status", func(t *testing.T) {
		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(headerKeyContentType, "text/plain")
			w.WriteHeader(http.StatusUnauthorized)
		})

		errorHandlerFn, _ := RegisterErrorHandler(nextFn, config)

		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/search", nil)

		errorHandlerFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusUnauthorized, respWriter.Code)
		assert.Equal(t, "unauthorized\n", respWriter.Body.String())
	})

	t.Run("when gateway timeout error with 504 http status", func(t *testing.T) {
		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(headerKeyContentType, "text/plain")
			w.WriteHeader(http.StatusGatewayTimeout)
		})

		errorHandlerFn, _ := RegisterErrorHandler(nextFn, config)

		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/search", nil)

		errorHandlerFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusGatewayTimeout, respWriter.Code)
		assert.Equal(t, "gateway timeout\n", respWriter.Body.String())
	})

	t.Run("when service unavailable error with 503 http status", func(t *testing.T) {
		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(headerKeyContentType, "text/plain")
			w.WriteHeader(http.StatusServiceUnavailable)
		})

		errorHandlerFn, _ := RegisterErrorHandler(nextFn, config)

		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/search", nil)

		errorHandlerFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusServiceUnavailable, respWriter.Code)
		assert.Equal(t, "service unavailable\n", respWriter.Body.String())
	})

	t.Run("when bad gateway error with 502 http status", func(t *testing.T) {
		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(headerKeyContentType, "text/plain")
			w.WriteHeader(http.StatusBadGateway)
		})

		errorHandlerFn, _ := RegisterErrorHandler(nextFn, config)

		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/search", nil)

		errorHandlerFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusBadGateway, respWriter.Code)
		assert.Equal(t, "bad gateway\n", respWriter.Body.String())
	})

	t.Run("when too many request error with 429 http status", func(t *testing.T) {
		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(headerKeyContentType, "text/plain")
			w.WriteHeader(http.StatusTooManyRequests)
		})

		errorHandlerFn, _ := RegisterErrorHandler(nextFn, config)

		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/search", nil)

		errorHandlerFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusTooManyRequests, respWriter.Code)
		assert.Equal(t, "too many request\n", respWriter.Body.String())
	})

	t.Run("when unidentified error occurs", func(t *testing.T) {
		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(headerKeyContentType, "text/plain")
			w.WriteHeader(http.StatusInternalServerError)
		})

		errorHandlerFn, _ := RegisterErrorHandler(nextFn, config)

		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/search", nil)

		errorHandlerFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusInternalServerError, respWriter.Code)
		assert.Equal(t, "internal server error\n", respWriter.Body.String())
	})
}

func TestJSONRequestErrorHandler(t *testing.T) {
	config := map[string]any{}

	t.Run("when bad request error with 400 http status on JSON request", func(t *testing.T) {
		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		})

		errorHandlerFn, _ := RegisterErrorHandler(nextFn, config)

		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/order", nil)
		req.Header.Add(headerKeyContentType, applicationJSONContentType)

		errorHandlerFn.ServeHTTP(respWriter, req)

		var jsonResp jsonResponse[any]
		_ = json.Unmarshal(respWriter.Body.Bytes(), &jsonResp)

		assert.Equal(t, http.StatusBadRequest, respWriter.Code)
		assert.Equal(t, "BAD_REQUEST", jsonResp.Code)
		assert.Equal(t, "Bad request", jsonResp.Message)
	})

	t.Run("when unauthorized error with 401 http status on JSON request", func(t *testing.T) {
		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})

		errorHandlerFn, _ := RegisterErrorHandler(nextFn, config)

		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/order", nil)
		req.Header.Add(headerKeyContentType, applicationJSONContentType)

		errorHandlerFn.ServeHTTP(respWriter, req)

		var jsonResp jsonResponse[any]
		_ = json.Unmarshal(respWriter.Body.Bytes(), &jsonResp)

		assert.Equal(t, http.StatusUnauthorized, respWriter.Code)
		assert.Equal(t, "UNAUTHORIZED", jsonResp.Code)
		assert.Equal(t, "Unauthorized", jsonResp.Message)
	})

	t.Run("when gateway timeout error with 504 http status on JSON request", func(t *testing.T) {
		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusGatewayTimeout)
		})

		errorHandlerFn, _ := RegisterErrorHandler(nextFn, config)

		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/order", nil)
		req.Header.Add(headerKeyContentType, applicationJSONContentType)

		errorHandlerFn.ServeHTTP(respWriter, req)

		var jsonResp jsonResponse[any]
		_ = json.Unmarshal(respWriter.Body.Bytes(), &jsonResp)

		assert.Equal(t, http.StatusGatewayTimeout, respWriter.Code)
		assert.Equal(t, "GATEWAY_TIMEOUT", jsonResp.Code)
		assert.Equal(t, "Service timeout", jsonResp.Message)
	})

	t.Run("when service unavailable error with 503 http status on JSON request", func(t *testing.T) {
		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		})

		errorHandlerFn, _ := RegisterErrorHandler(nextFn, config)

		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/order", nil)
		req.Header.Add(headerKeyContentType, applicationJSONContentType)

		errorHandlerFn.ServeHTTP(respWriter, req)

		var jsonResp jsonResponse[any]
		_ = json.Unmarshal(respWriter.Body.Bytes(), &jsonResp)

		assert.Equal(t, http.StatusServiceUnavailable, respWriter.Code)
		assert.Equal(t, "SERVICE_UNAVAILABLE", jsonResp.Code)
		assert.Equal(t, "Service is currently unavailable", jsonResp.Message)
	})

	t.Run("when bad gateway error with 502 http status on JSON request", func(t *testing.T) {
		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		})

		errorHandlerFn, _ := RegisterErrorHandler(nextFn, config)

		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/order", nil)
		req.Header.Add(headerKeyContentType, applicationJSONContentType)

		errorHandlerFn.ServeHTTP(respWriter, req)

		var jsonResp jsonResponse[any]
		_ = json.Unmarshal(respWriter.Body.Bytes(), &jsonResp)

		assert.Equal(t, http.StatusBadGateway, respWriter.Code)
		assert.Equal(t, "BAD_GATEWAY", jsonResp.Code)
		assert.Equal(t, "Bad gateway", jsonResp.Message)
	})

	t.Run("when too many request error with 429 http status on JSON request", func(t *testing.T) {
		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		})

		errorHandlerFn, _ := RegisterErrorHandler(nextFn, config)

		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/order", nil)
		req.Header.Add(headerKeyContentType, applicationJSONContentType)

		errorHandlerFn.ServeHTTP(respWriter, req)

		var jsonResp jsonResponse[any]
		_ = json.Unmarshal(respWriter.Body.Bytes(), &jsonResp)

		assert.Equal(t, http.StatusTooManyRequests, respWriter.Code)
		assert.Equal(t, "TOO_MANY_REQUEST", jsonResp.Code)
		assert.Equal(t, "Too many request", jsonResp.Message)
	})

	t.Run("when unidentified error occurs on JSON request", func(t *testing.T) {
		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		errorHandlerFn, _ := RegisterErrorHandler(nextFn, config)

		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/order", nil)
		req.Header.Add(headerKeyContentType, applicationJSONContentType)

		errorHandlerFn.ServeHTTP(respWriter, req)

		var jsonResp jsonResponse[any]
		_ = json.Unmarshal(respWriter.Body.Bytes(), &jsonResp)

		assert.Equal(t, http.StatusInternalServerError, respWriter.Code)
		assert.Equal(t, "SYSTEM_ERROR", jsonResp.Code)
		assert.Equal(t, "Internal system error", jsonResp.Message)
	})

	t.Run("when success", func(t *testing.T) {
		nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(headerKeyContentType, applicationJSONContentType)
			w.WriteHeader(http.StatusInternalServerError)

			jsonResp := jsonResponse[any]{
				Code:       "SUCCESS",
				Message:    "Success",
				ServerTime: getServerTime(),
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(jsonResp)
		})

		errorHandlerFn, _ := RegisterErrorHandler(nextFn, config)

		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/order", nil)
		req.Header.Add(headerKeyContentType, applicationJSONContentType)

		errorHandlerFn.ServeHTTP(respWriter, req)

		var jsonResp jsonResponse[any]
		_ = json.Unmarshal(respWriter.Body.Bytes(), &jsonResp)

		assert.Equal(t, http.StatusInternalServerError, respWriter.Code)
		assert.Equal(t, "SUCCESS", jsonResp.Code)
		assert.Equal(t, "Success", jsonResp.Message)
	})
}
