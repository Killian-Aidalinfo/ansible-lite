package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"io/ioutil"
	"aidalinfo/ansible-lite/internal/config"
)

func statusCommand(cfg *config.GlobalConfig) {
	// Construire l'URL avec le port provenant de la configuration
	apiURL := fmt.Sprintf("http://localhost:%d/status", cfg.Global.Port)

	// Préparer la requête avec le token depuis la configuration
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Fatalf("Erreur lors de la création de la requête : %v", err)
	}

	// Ajouter l'en-tête Authorization avec le token d'API
	req.Header.Add("Authorization", cfg.Global.Credentials)

	// Envoyer la requête
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Erreur lors de l'envoi de la requête : %v", err)
	}
	defer resp.Body.Close()

	// Lire la réponse
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Erreur lors de la lecture de la réponse : %v", err)
	}

	// Afficher la réponse
	fmt.Println("Réponse de l'API:")
	fmt.Println(string(body))
}

func main() {
	// Définir l'argument --config pour spécifier le chemin du fichier de configuration
	configPath := flag.String("config", "config.yaml", "Chemin vers le fichier de configuration")
	command := flag.String("command", "", "Commande à exécuter (ex: status)")
	flag.Parse()

	// Charger la configuration depuis le fichier spécifié
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Erreur lors du chargement de la configuration : %v", err)
	}

	// Exécuter la commande spécifiée
	switch *command {
	case "status":
		statusCommand(cfg)
	default:
		fmt.Println("Commande inconnue. Utilisez --command=status pour vérifier l'état de l'API.")
	}
}
