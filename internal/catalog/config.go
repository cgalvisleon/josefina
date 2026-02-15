package catalog

import (
	"github.com/cgalvisleon/et/file"
)

type Config struct {
	Nodes    []string `json:"nodes"`
	IsStrict bool     `json:"is_strict"`
	filePath string   `json:"-"`
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
		result = &Config{
			Nodes:    []string{},
			IsStrict: false,
		}

		file.Write(filePath, result)
	}

	result.filePath = filePath

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
