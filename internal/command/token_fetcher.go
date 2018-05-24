package command

import (
	"encoding/json"
	"os"
)

type TokenFetcher struct {
	configPath string
}

func NewTokenFetcher(configPath string) *TokenFetcher {
	return &TokenFetcher{
		configPath: configPath,
	}
}

func (tf *TokenFetcher) RefreshToken() (string, error) {
	f, err := os.Open(tf.configPath)
	if err != nil {
		return "", err
	}

	var config struct {
		RefreshToken string `json:"RefreshToken"`
	}

	err = json.NewDecoder(f).Decode(&config)
	return config.RefreshToken, err
}
