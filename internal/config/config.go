package config

import (
	"github.com/cgalvisleon/et/file"
)

type Config struct {
	RPC []string `json:"rpc"`
	TCP []string `json:"tcp"`
}

/**
* getConfig: Returns the config
* @return *Config, error
**/
func getConfig() (*Config, error) {
	filePath := "./config.json"
	var result *Config
	err := file.Read(filePath, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* GetNodes: Returns the nodes
* @return []string, error
**/
func GetNodes() ([]string, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}

	return config.RPC, nil
}
