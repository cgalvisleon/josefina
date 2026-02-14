package config

import (
	"github.com/cgalvisleon/et/file"
)

type Config struct {
	PORT     int      `json:"port"`
	RPC      int      `json:"rpc"`
	Nodes    []string `json:"nodes"`
	IsStrict bool     `json:"is_strict"`
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

	if result == nil {
		return &Config{
			PORT:     1377,
			RPC:      4200,
			Nodes:    []string{},
			IsStrict: false,
		}, nil
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

	return config.Nodes, nil
}
