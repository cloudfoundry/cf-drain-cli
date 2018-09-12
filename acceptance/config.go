package acceptance

import (
	"log"
	"time"

	envstruct "code.cloudfoundry.org/go-envstruct"
)

type TestConfig struct {
	CFAdminUser     string `env:"CF_ADMIN_USER,     required"`
	CFAdminPassword string `env:"CF_ADMIN_PASSWORD, required"`
	CFDomain        string `env:"CF_DOMAIN,         required"`

	SkipCertVerify bool `env:"SKIP_CERT_VERIFY"`

	DefaultTimeout time.Duration `env:"DEFAULT_TIMEOUT"`
	AppPushTimeout time.Duration `env:"APP_PUSH_TIMEOUT"`
}

var config *TestConfig

func LoadConfig() (*TestConfig, error) {
	config := &TestConfig{
		DefaultTimeout: 45 * time.Second,
		AppPushTimeout: 90 * time.Second,
	}
	err := envstruct.Load(config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func Config() *TestConfig {
	if config != nil {
		return config
	}

	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("failed to load drain test config: %s", err)
	}
	config = cfg
	return config
}
