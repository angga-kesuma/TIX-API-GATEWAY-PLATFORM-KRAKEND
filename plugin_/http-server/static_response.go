package main

import (
	"fmt"
	"net/http"
)

type staticResponseConfig struct {
	Status  int
	Body    string
	Headers map[string]string
}

func RegisterStaticResponse(next http.Handler, cfg map[string]any) (http.Handler, error) {
	if cfg == nil {
		return next, nil
	}

	mapping, err := newStaticResponseConfig(cfg)
	if err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, ok := mapping[r.URL.Path]
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		logger.Info(
			middlewareLogPrefix,
			fmt.Sprintf("static-response: path=%s status=%d", r.URL.Path, resp.Status),
		)

		for k, v := range resp.Headers {
			w.Header().Set(k, v)
		}

		w.WriteHeader(resp.Status)
		w.Write([]byte(resp.Body))
	}), nil
}

func newStaticResponseConfig(cfg map[string]any) (map[string]staticResponseConfig, error) {
	result := make(map[string]staticResponseConfig)

	for path, raw := range cfg {
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid cfg for %s", path)
		}

		status := int(m["status"].(float64))
		body := m["body"].(string)
		headersRaw := m["headers"].(map[string]any)

		hdr := map[string]string{}
		for k, v := range headersRaw {
			hdr[k] = v.(string)
		}

		result[path] = staticResponseConfig{
			Status:  status,
			Body:    body,
			Headers: hdr,
		}
	}

	return result, nil
}
