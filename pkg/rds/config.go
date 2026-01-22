package rds

import (
	"encoding/json"
	"os"

	"github.com/cgalvisleon/et/envar"
)

type Config struct {
	Leader string   `json:"leader"`
	Nodes  []string `json:"nodes"`
}

/**
* getConfig: Returns the config
* @return *Config, error
**/
func getConfig() (*Config, error) {
	path := envar.GetStr("PATH_DATA", "./data")
	f, err := os.Open(path + "/config.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var result Config
	err = json.NewDecoder(f).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

/**
* writeConfig: Writes the config
* @param config *Config
* @return error
**/
func writeConfig(config *Config) error {
	path := envar.GetStr("PATH_DATA", "./data")
	f, err := os.Create(path + "/config.json")
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(config)
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

/**
* getLeader: Returns the leader
* @return string, error
**/
func getLeader() (string, error) {
	config, err := getConfig()
	if err != nil {
		return "", err
	}

	return config.Leader, nil
}
