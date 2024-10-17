package initapp

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"aidalinfo/ansible-lite/internal/config"
	"aidalinfo/ansible-lite/internal/db"
	"aidalinfo/ansible-lite/internal/logger"
	"aidalinfo/ansible-lite/internal/api"
	"aidalinfo/ansible-lite/internal/token" 
	"github.com/natefinch/lumberjack"
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

	//Gestion des logs avec lumberjack
	log.SetOutput(&lumberjack.Logger{
		Filename:   cfg.Global.LogPath,  // Nom du fichier de log
		MaxSize:    50,              // Taille max en mégaoctets avant rotation
		MaxBackups: 5,               // Nombre maximal de fichiers de backup conservés
		MaxAge:     90,              // Durée maximale de conservation des fichiers en jours
		Compress:   true,            // Compresser les fichiers de log archivés
})
	// Vérifier si le token existe dans la configuration
	if cfg.Global.Credentials == "" {  // Utiliser Credentials avec une majuscule
		logger.Log("INFO", "Aucun token trouvé, génération d'un nouveau token")
		newToken, err := token.GenerateToken(32) // Générer un token de 32 octets
		if err != nil {
			logger.Log("ERROR", "Erreur lors de la génération du token : %v", err)
			return nil, err
		}
		cfg.Global.Credentials = newToken

		// Sauvegarder la configuration mise à jour dans le fichier
		err = saveConfig(configPath, cfg)
		if err != nil {
			logger.Log("ERROR", "Erreur lors de la sauvegarde de la configuration mise à jour : %v", err)
			return nil, err
		}
		logger.Log("INFO", "Nouveau token généré et sauvegardé dans la configuration")
	}

	// Exemple de message de log avec le niveau INFO
	logger.Log("INFO", "Démarrage de l'application")

	// Démarrer le serveur API en parallèle
	go api.StartServer(cfg.Global.Port, cfg)

	// Initialiser la base de données
	err = db.InitDB(cfg.Global.DBPath)
	if err != nil {
		logger.Log("ERROR", "Erreur lors de l'initialisation de la base de données : %v", err)
		return nil, err
	}

	logger.Log("INFO", "Application démarrée avec succès.")
	return cfg, nil
}


// Fonction pour sauvegarder la configuration dans le fichier config.yaml
func saveConfig(configPath string, cfg *config.GlobalConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		log.Printf("Erreur lors du marshalling de la configuration : %v", err)
		return err
	}

	err = ioutil.WriteFile(configPath, data, 0644)
	if err != nil {
		log.Printf("Erreur lors de la sauvegarde du fichier de configuration : %v", err)
		return err
	}
	log.Printf("Configuration sauvegardée avec succès")
	return nil
}
