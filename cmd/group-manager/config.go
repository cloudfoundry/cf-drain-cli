package main

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	envstruct "code.cloudfoundry.org/go-envstruct"
)

type VCap struct {
	API       string `json:"cf_api"`
	SpaceGUID string `json:"space_id"`
}

func (v *VCap) UnmarshalEnv(data string) error {
	err := json.Unmarshal([]byte(data), &v)

	return err
}

type Config struct {
	SourceID  string `env:"SOURCE_ID"`
	GroupName string `env:"GROUP_NAME,       required"`
	VCap      VCap   `env:"VCAP_APPLICATION, required"`

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

	cfg.LogCacheHost = strings.Replace(cfg.VCap.API, "https://api", "http://log-cache", 1)
	cfg.VCap.API = strings.Replace(cfg.VCap.API, "https://api", "http://api", 1)

	return cfg
}
