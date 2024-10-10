package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// Structure pour stocker la configuration globale
type GlobalConfig struct {
	Global struct {
		Type        string `yaml:"type"`
		DBPath      string `yaml:"db_path"`
		LogPath     string `yaml:"log_path"`
		LogLevel    string `yaml:"log_level"`
		ReposConfig string `yaml:"repos_config"`
		Port        int    `yaml:"port"`
	} `yaml:"GLOBAL"`
}

// Fonction pour charger le fichier de configuration YAML
func LoadConfig(path string) (*GlobalConfig, error) {
	var config GlobalConfig
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("impossible de lire le fichier de configuration : %v", err)
	}

	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return nil, fmt.Errorf("erreur lors du parsing YAML : %v", err)
	}

	return &config, nil
}
