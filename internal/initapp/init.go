package initapp

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"aidalinfo/ansible-lite/internal/config"
	"aidalinfo/ansible-lite/internal/db"
	"aidalinfo/ansible-lite/internal/logger" // Utiliser le package logger pour la gestion des logs
)

// InitApp est la fonction d'initialisation principale qui gère les logs et la base de données
func InitApp(configPath string) (*config.GlobalConfig, error) {
	// Charger la configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Log("ERROR", "Erreur lors du chargement de la configuration : %v", err)
		return nil, fmt.Errorf("Erreur lors du chargement de la configuration : %v", err)
	}

	// Vérifier si le répertoire parent du fichier de log existe, sinon le créer
	logDir := filepath.Dir(cfg.Global.LogPath)
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		err = os.MkdirAll(logDir, 0750)
		if err != nil {
			logger.Log("ERROR", "Impossible de créer le répertoire de log : %v", err)
			return nil, fmt.Errorf("Impossible de créer le répertoire de log : %v", err)
		}
	}

	// Ouvrir le fichier de log
	logFile, err := os.OpenFile(cfg.Global.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Log("ERROR", "Impossible d'ouvrir le fichier de log : %v", err)
		return nil, fmt.Errorf("Impossible d'ouvrir le fichier de log : %v", err)
	}
	// Rediriger le logger vers le fichier de log avec log.SetOutput
	log.SetOutput(logFile)

	// Exemple de message de log avec le niveau INFO
	logger.Log("INFO", "Démarrage de l'application")

	// Initialiser la base de données
	err = db.InitDB(cfg.Global.DBPath)
	if err != nil {
		logger.Log("ERROR", "Erreur lors de l'initialisation de la base de données : %v", err)
		return nil, err
	}

	logger.Log("INFO", "Application démarrée avec succès.")
	return cfg, nil
}
