package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	envstruct "code.cloudfoundry.org/go-envstruct"
)

type Config struct {
	SourceID  string `env:"SOURCE_ID,        required"`
	GroupName string `env:"GROUP_NAME,       required"`

	UpdateInterval time.Duration `env:"UPDATE_INTERVAL"`

	LogCacheHost string
}

func loadConfig() Config {
	cfg := Config{
		UpdateInterval: 30 * time.Second,
	}

	if err := envstruct.Load(&cfg); err != nil {
		log.Fatalf("failed to load config")
	}

	cfg.LogCacheHost = getLogCacheHost()
	return cfg
}

func getLogCacheHost() string {
	vcapEnv := os.Getenv("VCAP_APPLICATION")
	if vcapEnv == "" {
		log.Fatalf("failed to load VCAP_APPLICATION from envrionment")
	}

	var vcap struct {
		API string `json:"cf_api"`
	}

	err := json.Unmarshal([]byte(vcapEnv), &vcap)
	if err != nil {
		log.Fatalf("failed to unmarshal VCAP_APPLICATION")
	}

	return strings.Replace(vcap.API, "https://api", "http://log-cache", 1)
}
