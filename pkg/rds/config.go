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

func getConfig() (*Config, error) {
	path := envar.GetStr("PATH_DATA", "./data")
	filePath := path + "/config.json"

	// 1) Asegurar directorio
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, err
	}

	// 2) Intentar abrir
	f, err := os.Open(filePath)
	if err != nil {
		// Si no existe, crearlo con valores por defecto
		if os.IsNotExist(err) {
			cfg := Config{} // TODO: pon aquí tus valores por defecto

			b, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return nil, err
			}

			if err := os.WriteFile(filePath, b, 0o644); err != nil {
				return nil, err
			}

			return &cfg, nil
		}
		return nil, err
	}
	defer f.Close()

	// 3) Leer/parsear
	var result Config
	if err := json.NewDecoder(f).Decode(&result); err != nil {
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
