package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/angga-kesuma/TIX-API-GATEWAY-PLATFORM-KRAKEND/pkg/helper/httpcaller"
	"github.com/angga-kesuma/TIX-API-GATEWAY-PLATFORM-KRAKEND/pkg/plugin/config"
	"github.com/angga-kesuma/TIX-API-GATEWAY-PLATFORM-KRAKEND/pkg/plugin/middleware"
)

/**

This is compatible with http-client krakend plugins
you should have in mind this will be run every time the endpoints call

*/

/*
*
NewPlugins, will be called when initialization of plugins,so this only run once
krakend using RegisterLogger(v interface{}) as initialization method
*/
func NewPlugins() {

	// call deps
	cfg := config.LoadConfig()

	// init httpcaller
	httpcaller.Register(cfg.CallerPool)

	// init middleware
	middleware.RegisterMiddleware(cfg)

}

func Run(_ context.Context, extra map[string]interface{}) (http.Handler, error) {
	fmt.Sprintf("%v", extra)
	return serveRequest(&config.EndpointConfig{})
}

// serveRequest creates an HTTP handler function that proxies requests to backend services.
// this is last middleware to call
// Returns an http.HandlerFunc that processes incoming HTTP requests and proxies them to the configured backend
func serveRequest(cfg *config.EndpointConfig) (http.Handler, error) {

	var last http.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var (
			client = httpcaller.Get(cfg.PoolName)
			//requestId = req.Header.Get(header.XRequestId)
			//language  = req.Header.Get(header.AcceptLanguage)
		)

		if cfg.IsLog {
			//httpLogger.LogRequest(req)
			// log here
		}

		// Override Content-Type to application/json
		for name, values := range req.Header {
			if name == "Content-Type" {
				req.Header.Set("Content-Type", "application/json")
				continue
			}
			req.Header[name] = values
		}
		// Clear RequestURI to avoid "http: Request.RequestURI can't be set in client requests" error
		req.RequestURI = ""

		// Send an HTTP request and returns an HTTP response object.
		var urlErr *url.Error
		resp, err := client.Do(req)
		if errors.As(err, &urlErr) {
			logger.Error(fmt.Sprintf("Failed to httpClient.Do, path: %s, error: %s", req.URL.Path, err.Error()))
			if urlErr.Timeout() {
				w.WriteHeader(http.StatusGatewayTimeout)
				return
			}
			if urlErr.Temporary() {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to httpClient.Do, path: %s, error: %s", req.URL.Path, err.Error()))
			w.WriteHeader(http.StatusBadGateway)
			return
		}

		defer resp.Body.Close()
		// headers
		for name, values := range resp.Header {
			w.Header()[name] = values
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if cfg.IsLog {
			// log here
			//httpLogger.LogResponse(req.URL.Path, resp.StatusCode, body, resp.Header, requestId, isBodyDecompressed)
			//httpLogger.Logger.Debug("========> Response Status:", resp.StatusCode)
		}

		w.WriteHeader(resp.StatusCode)
		w.Write(body)
	})

	return middleware.RunMiddleware(last, cfg)

}
