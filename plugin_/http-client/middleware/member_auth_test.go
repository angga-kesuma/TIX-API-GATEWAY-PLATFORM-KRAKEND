package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterMemberAuth(t *testing.T) {
	nextFn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	c := &http.Cookie{}
	c.Name = "aat"
	c.Value = "123"

	t.Run("when privilegeIds config not exist", func(t *testing.T) {
		config := map[string]any{}

		assert.Panics(t, func() { _, _ = RegisterMemberAuth(nextFn, config) }, "The code did not panic")
	})

	t.Run("when privilegeIds config not exist", func(t *testing.T) {
		config := map[string]any{
			"privilegeIds": []any{"123"},
		}

		assert.Panics(t, func() { _, _ = RegisterMemberAuth(nextFn, config) }, "The code did not panic")
	})

	t.Run("when timeout config not exist", func(t *testing.T) {
		config := map[string]any{
			"privilegeIds": []any{"123"},
			"authServer":   "http://localhost:1234/auth",
		}

		assert.Panics(t, func() { _, _ = RegisterMemberAuth(nextFn, config) }, "The code did not panic")
	})

	t.Run("when timeout config cant be parsed", func(t *testing.T) {
		config := map[string]any{
			"privilegeIds": []any{"123"},
			"authServer":   "http://localhost:1234/auth",
			"timeout":      "x",
		}

		assert.Panics(t, func() { _, _ = RegisterMemberAuth(nextFn, config) }, "The code did not panic")
	})

	t.Run("when request has no cookie", func(t *testing.T) {
		memberAuthFn, _ := RegisterMemberAuth(nextFn, getMemberAuthConfig())
		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)

		memberAuthFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusUnauthorized, respWriter.Code)
	})

	t.Run("failed when NewRequestWithContext", func(t *testing.T) {
		config := getMemberAuthConfig()
		config[configKeyAuthServer] = "://"
		memberAuthFn, _ := RegisterMemberAuth(nextFn, config)
		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(c)

		memberAuthFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusInternalServerError, respWriter.Code)
	})

	t.Run("failed when timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))

		defer server.Close()
		config := getMemberAuthConfig()
		config[configKeyAuthServer] = server.URL
		config[configKeyTimeout] = "1ns"
		memberAuthFn, _ := RegisterMemberAuth(nextFn, config)
		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(c)

		memberAuthFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusInternalServerError, respWriter.Code)
	})

	t.Run("failed when failed io.ReadAll", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "1")
			w.WriteHeader(http.StatusAccepted)
		}))

		defer server.Close()
		config := getMemberAuthConfig()
		config[configKeyAuthServer] = server.URL
		memberAuthFn, _ := RegisterMemberAuth(nextFn, config)
		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(c)

		memberAuthFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusInternalServerError, respWriter.Code)
	})

	t.Run("failed when status is not OK", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))

		defer server.Close()
		config := getMemberAuthConfig()
		config[configKeyAuthServer] = server.URL
		memberAuthFn, _ := RegisterMemberAuth(nextFn, config)
		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(c)

		memberAuthFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusUnauthorized, respWriter.Code)
	})

	t.Run("failed when empty response body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		defer server.Close()
		config := getMemberAuthConfig()
		config[configKeyAuthServer] = server.URL
		memberAuthFn, _ := RegisterMemberAuth(nextFn, config)
		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(c)

		memberAuthFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusUnauthorized, respWriter.Code)
	})

	t.Run("failed when unmarshall", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "some-string")
		}))

		defer server.Close()
		config := getMemberAuthConfig()
		config[configKeyAuthServer] = server.URL
		memberAuthFn, _ := RegisterMemberAuth(nextFn, config)
		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(c)

		memberAuthFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusInternalServerError, respWriter.Code)
	})

	t.Run("failed when status is not SUCCESS", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			jsonResp := memberAuthSessionResponse{
				Code:    "FAILED",
				Message: "Failed",
			}

			_ = json.NewEncoder(w).Encode(jsonResp)

			w.WriteHeader(http.StatusOK)
		}))

		defer server.Close()
		config := getMemberAuthConfig()
		config[configKeyAuthServer] = server.URL
		memberAuthFn, _ := RegisterMemberAuth(nextFn, config)
		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(c)

		memberAuthFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusUnauthorized, respWriter.Code)
	})

	t.Run("failed when priv is not sufficient", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			jsonResp := memberAuthSessionResponse{
				Code: "SUCCESS",
				Data: memberAuthSessionData{
					Priv: "000",
				},
			}

			_ = json.NewEncoder(w).Encode(jsonResp)

			w.WriteHeader(http.StatusOK)
		}))

		defer server.Close()
		config := getMemberAuthConfig()
		config[configKeyAuthServer] = server.URL
		memberAuthFn, _ := RegisterMemberAuth(nextFn, config)
		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(c)

		memberAuthFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusUnauthorized, respWriter.Code)
	})

	t.Run("happy path", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			jsonResp := memberAuthSessionResponse{
				Code: "SUCCESS",
				Data: memberAuthSessionData{
					Priv: "123,456,789",
				},
			}

			_ = json.NewEncoder(w).Encode(jsonResp)

			w.WriteHeader(http.StatusOK)
		}))

		defer server.Close()
		config := getMemberAuthConfig()
		config[configKeyAuthServer] = server.URL
		memberAuthFn, _ := RegisterMemberAuth(nextFn, config)
		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(c)

		memberAuthFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusOK, respWriter.Code)
	})

	t.Run("success but cookieType empty", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			jsonResp := memberAuthSessionResponse{
				Code: "SUCCESS",
				Data: memberAuthSessionData{
					Priv: "123,456,789",
				},
			}

			_ = json.NewEncoder(w).Encode(jsonResp)

			w.WriteHeader(http.StatusOK)
		}))

		defer server.Close()
		config := getMemberAuthWithEmptyCookieTypeConfig()
		config[configKeyAuthServer] = server.URL
		memberAuthFn, _ := RegisterMemberAuth(nextFn, config)
		respWriter := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)

		c := &http.Cookie{}
		c.Name = "session_access_token"
		c.Value = "123"

		req.AddCookie(c)

		memberAuthFn.ServeHTTP(respWriter, req)

		assert.Equal(t, http.StatusOK, respWriter.Code)
	})
}

func TestGetBearerToken_CookieTypeFromSTT(t *testing.T) {
	t.Run("default to b2c when stt is missing", func(t *testing.T) {
		h := make(http.Header)
		h.Set(headerKeyAuthorization, "Bearer "+mustJWT(map[string]any{
			"aud": "tiket.com",
			// no "stt"
		}))

		_, cType, err := getBearerToken(h)
		assert.NoError(t, err)
		assert.Equal(t, b2cCookieType, cType)
	})

	t.Run("admin when stt matches adminSTTParamValue", func(t *testing.T) {
		adminSTTInt := mustAtoi(t, adminSTTParamValue)

		h := make(http.Header)
		h.Set(headerKeyAuthorization, "Bearer "+mustJWT(map[string]any{
			"aud": "tiket.com",
			"stt": adminSTTInt,
		}))

		_, cType, err := getBearerToken(h)
		assert.NoError(t, err)
		assert.Equal(t, adminCookieType, cType)
	})

	t.Run("b2b when stt matches b2bSTTParamValue", func(t *testing.T) {
		b2bSTTInt := mustAtoi(t, b2bSTTParamValue)

		h := make(http.Header)
		h.Set(headerKeyAuthorization, "Bearer "+mustJWT(map[string]any{
			"aud": "tiket.com",
			"stt": b2bSTTInt,
		}))

		_, cType, err := getBearerToken(h)
		assert.NoError(t, err)
		assert.Equal(t, b2bCookieType, cType)
	})

	t.Run("default to b2c when stt is unknown", func(t *testing.T) {
		h := make(http.Header)
		h.Set(headerKeyAuthorization, "Bearer "+mustJWT(map[string]any{
			"aud": "tiket.com",
			"stt": 999,
		}))

		_, cType, err := getBearerToken(h)
		assert.NoError(t, err)
		assert.Equal(t, b2cCookieType, cType)
	})
}

func mustAtoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	if err != nil {
		t.Fatalf("failed to atoi %q: %v", s, err)
	}
	return n
}

// mustJWT returns a syntactically valid JWT string: header.payload.signature
// We don't need a valid signature because getBearerToken only parses the payload.
func mustJWT(payload map[string]any) string {
	headerJSON := []byte(`{"alg":"none","typ":"JWT"}`)

	payloadJSON, _ := json.Marshal(payload)

	enc := func(b []byte) string {
		return base64.RawURLEncoding.EncodeToString(b)
	}

	return enc(headerJSON) + "." + enc(payloadJSON) + ".sig"
}

func getMemberAuthConfig() map[string]any {
	return map[string]any{
		"authServer":   "http://localhost:1234/auth",
		"privilegeIds": []any{"012", "123", "345", "789"},
		"timeout":      "5s",
		"cookieType":   "B2B",
	}
}

func getMemberAuthWithEmptyCookieTypeConfig() map[string]any {
	return map[string]any{
		"authServer":   "http://localhost:1234/auth",
		"privilegeIds": []any{"123"},
		"Empty":        true,
		"timeout":      "5s",
	}
}
