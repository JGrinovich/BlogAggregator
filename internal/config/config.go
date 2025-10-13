package config

import (
	"encoding/json"
	"os"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	DbUrl           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return homeDir + "/" + configFileName, nil
}

func Read() (*Config, error) {
	fp, err := getConfigFilePath()
	if err != nil {
		return nil, err
	}
	config := &Config{}
	fb, err := os.ReadFile(fp)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(fb, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// go
func (c *Config) Write() error {
	fp, err := getConfigFilePath()
	if err != nil {
		return err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(fp, data, 0644)
}

func (cfg *Config) SetUser(username string) error {
	cfg.CurrentUserName = username
	return cfg.Write()
}
