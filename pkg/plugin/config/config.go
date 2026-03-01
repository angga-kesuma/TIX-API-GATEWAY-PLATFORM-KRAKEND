package config

import (
	"fmt"
	"os"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/tiket/angga-kesuma/pkg/helper/httpclient"
	"github.com/tiket/angga-kesuma/pkg/helper/yamlparser"
	"gopkg.in/yaml.v3"
)

/**
we have two kind of config
Application config
and
Endpoints config
*/

type EndpointConfig struct {
	PoolName string
	IsLog    bool
}

// AppConfig represents the structure of your YAML config
// Update fields as needed
type AppConfig struct {
	ServiceName   string                            `yaml:"service_name"`
	MemberSession *OutboundConfig                   `yaml:"member_session"`
	RedisConfig   *RedisConfig                      `yaml:"redis_config"`
	CallerPool    map[string]*httpclient.HTTPConfig `yaml:"caller_pool"`
	Middlewares   []string                          `yaml:"middlewares"`
}

type RedisConfig struct {
	IsCluster   bool     `yaml:"is_cluster"`
	ClusterHost []string `yaml:"cluster_host"`
	Host        string   `yaml:"host"`
	Password    string   `yaml:"password"`
	DBIndex     int      `yaml:"db_index"`
}

type OutboundConfig struct {
	BaseUrl    string                 `yaml:"base_url"`
	HTTPConfig *httpclient.HTTPConfig `yaml:"http_config"`
}

func LoadConfig() *AppConfig {
	configPath := os.Getenv("GLOBAL_CONFIG_PATH")
	if configPath == "" {
		configPath = "/etc/krakend/global_config.yaml"
	}

	configSecret := os.Getenv("GLOBAL_SECRET_PATH")
	if configSecret == "" {
		configSecret = "/etc/krakend/global_secret.yaml"
	}

	logger.Info(fmt.Sprintf("Loaded config and host path : %s %s", configPath, configSecret))
	GlobalConfig, err := readYaml(configPath, configSecret)
	if err != nil {
		fmt.Printf("Error load config : %v", err)
	}

	return GlobalConfig
}

// loadConfig loads two YAML files and merges them into a single AppConfig struct dynamically
func readYaml(pathConfig, pathSecret string) (*AppConfig, error) {
	confByte, err := yamlparser.ReadYamlConfig(pathConfig, pathSecret)
	if err != nil {
		return nil, err
	}

	c := &AppConfig{}
	if err = yaml.Unmarshal(confByte, c); err != nil {
		return nil, err
	}

	return c, nil
}
