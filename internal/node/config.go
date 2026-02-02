package node

import (
	"github.com/cgalvisleon/et/file"
)

type Config struct {
	Nodes []string `json:"nodes"`
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
* getNodes: Returns the nodes
* @return []string, error
**/
func getNodes() ([]string, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}

	return config.Nodes, nil
}
