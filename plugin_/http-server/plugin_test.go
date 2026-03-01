package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerPluginMiddleware(t *testing.T) {
	r := registerer("middleware")

	// use NoOpLogger to avoid nil logger
	r.RegisterLogger(NoOpLogger{})

	t.Run("when middleware config not found", func(t *testing.T) {
		extra := map[string]any{
			"not-middleware": "",
		}

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
		handler, err := r.registerHandlers(context.Background(), extra, next)

		assert.Error(t, err)
		assert.Nil(t, handler)
		assert.Equal(t, "config for middleware not found", err.Error())
	})

	t.Run("when middleware names config not found", func(t *testing.T) {
		extra := map[string]any{
			"middleware": map[string]any{},
		}

		nextCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		})

		handler, err := r.registerHandlers(context.Background(), extra, next)

		assert.NoError(t, err)
		assert.NotNil(t, handler)

		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		handler.ServeHTTP(rr, req)

		assert.True(t, nextCalled, "next handler should be called")
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("when failed to type cast middleware names", func(t *testing.T) {
		extra := map[string]any{
			"middleware": map[string]any{
				"names": 1,
			},
		}

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

		handler, err := r.registerHandlers(context.Background(), extra, next)

		assert.Error(t, err)
		assert.Nil(t, handler)
		assert.Equal(t, "failed to type cast namesConfig (middleware pipeline)", err.Error())
	})

	t.Run("when register with static-response middleware", func(t *testing.T) {
		// config for static-response
		extra := map[string]any{
			"middleware": map[string]any{
				"names": []any{
					"static-response",
				},
				"static-response": map[string]any{
					"/static": map[string]any{
						"status":  201.0,
						"body":    "created",
						"headers": map[string]any{"X-From": "static"},
					},
				},
			},
		}

		nextCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("next"))
		})

		handler, err := r.registerHandlers(context.Background(), extra, next)

		assert.NoError(t, err)
		assert.NotNil(t, handler)

		t.Run("when path matches static-response", func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/static", nil)

			handler.ServeHTTP(rr, req)

			assert.False(t, nextCalled, "next handler should NOT be called for /static")
			assert.Equal(t, http.StatusCreated, rr.Code)
			assert.Equal(t, "created", rr.Body.String())
			assert.Equal(t, "static", rr.Header().Get("X-From"))
		})

		t.Run("when path does not match static-response", func(t *testing.T) {
			nextCalled = false

			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/other", nil)

			handler.ServeHTTP(rr, req)

			assert.True(t, nextCalled, "next handler SHOULD be called for /other")
			assert.Equal(t, http.StatusOK, rr.Code)
			assert.Equal(t, "next", rr.Body.String())
		})
	})
}
