package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPluginMiddleware(t *testing.T) {
	r := registerer("middleware")

	r.RegisterLogger(NoOpLogger{})

	t.Run("when middleware config not found", func(t *testing.T) {
		extra := map[string]interface{}{
			"not-middleware": "",
		}

		_, err := r.registerClients(context.Background(), extra)

		assert.Equal(t, "config for middleware not found", err.Error())
	})

	t.Run("when middleware names config not found", func(t *testing.T) {
		extra := map[string]interface{}{
			"middleware": map[string]interface{}{},
		}

		_, err := r.registerClients(context.Background(), extra)

		assert.Nil(t, err)
	})

	t.Run("when failed to type cast middleware names", func(t *testing.T) {
		extra := map[string]interface{}{
			"middleware": map[string]interface{}{
				"names": 1,
			},
		}

		_, err := r.registerClients(context.Background(), extra)

		assert.Error(t, err)
		assert.Equal(t, errors.New("failed to type cast namesConfig (middleware pipeline)"), err)
	})

	t.Run("when register with multiple middlewares", func(t *testing.T) {
		extra := map[string]interface{}{
			"middleware": map[string]interface{}{
				"names": []any{
					"error-handler",
					"member-auth",
				},
				"member-auth": map[string]any{
					"authServer": "http://localhost:1234/auth",
					"timeout":    "5s",
				},
			},
		}

		_, err := r.registerClients(context.Background(), extra)

		assert.Nil(t, err)
	})

	t.Run("when bad gateway error", func(t *testing.T) {
		// create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		}))
		defer server.Close()

		// config
		extra := map[string]interface{}{
			"middleware": map[string]interface{}{
				"names": []any{},
			},
		}

		handlerFn, _ := r.registerClients(context.Background(), extra)

		// construct request
		writer := httptest.NewRecorder()
		req := httptest.NewRequest("POST", server.URL+"/search", nil)

		handlerFn.ServeHTTP(writer, req)

		assert.Equal(t, http.StatusBadGateway, writer.Code)
	})

	t.Run("when success", func(t *testing.T) {
		// create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		}))
		defer server.Close()

		// config
		extra := map[string]interface{}{
			"middleware": map[string]interface{}{
				"names": []any{},
			},
		}

		handlerFn, _ := r.registerClients(context.Background(), extra)

		// construct request
		writer := httptest.NewRecorder()
		req := httptest.NewRequest("POST", server.URL+"/search", nil)
		req.RequestURI = ""

		handlerFn.ServeHTTP(writer, req)

		assert.Equal(t, http.StatusOK, writer.Code)
	})
}
