package httpcaller

import (
	"net/http"
	"sync"

	"github.com/tiket/angga-kesuma/pkg/helper/httpclient"
)

/*
This is HTTP Client we use for calling the backend service
**/

var (
	once        sync.Once
	pools       map[string]*http.Client
	defaultName = "default"
)

func Register(configs map[string]*httpclient.HTTPConfig) map[string]*http.Client {
	once.Do(func() {
		pools = make(map[string]*http.Client)
		hasDefault := false
		for name, cfg := range configs {
			pools[name] = httpclient.NewHTTPClient(cfg)
			if name == defaultName {
				hasDefault = true
			}
		}

		// if no 'default' config
		if !hasDefault {
			// create default config
			pools[defaultName] = httpclient.NewHTTPClient(&httpclient.HTTPConfig{})
		}

	})
	return pools
}
func Get(poolName string) *http.Client {
	if poolName == "" {
		return pools[defaultName]
	}

	pool, ok := pools[poolName]
	if !ok {
		return pools[defaultName]
	}

	return pool
}
