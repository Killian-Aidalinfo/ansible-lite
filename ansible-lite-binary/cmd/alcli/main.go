package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"io/ioutil"
	"aidalinfo/ansible-lite/internal/config"
	"encoding/json"
	"github.com/olekukonko/tablewriter"
	"os"
)

// Structure pour stocker les détails des exécutions récupérées depuis l'API
type ExecutionDetail struct {
	RepoName   string `json:"RepoName"`
	RepoURL    string `json:"RepoURL"`
	CommitID   string `json:"CommitID"`
	ExecutedAt string `json:"ExecutedAt"`
}

// Fonction pour exécuter la commande "repos list"
func reposListCommand(cfg *config.GlobalConfig) {
	// Construire l'URL avec le port provenant de la configuration
	apiURL := fmt.Sprintf("http://localhost:%d/executions", cfg.Global.Port)

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

	// Vérifier si l'API a renvoyé une erreur
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Erreur de l'API : %s", string(body))
	}

	// Parser le JSON de la réponse en une slice de ExecutionDetail
	var executionDetails []ExecutionDetail
	if err := json.Unmarshal(body, &executionDetails); err != nil {
		log.Fatalf("Erreur lors du parsing du JSON : %v", err)
	}

	// Afficher les données dans un tableau formaté
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Repo Name", "Repo URL", "Commit ID", "Executed At"})

	for _, exec := range executionDetails {
		table.Append([]string{exec.RepoName, exec.RepoURL, exec.CommitID, exec.ExecutedAt})
	}

	table.Render() // Afficher le tableau dans le terminal
}

// Fonction pour exécuter la commande "status"
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
	fmt.Println(string(body))
}

func main() {
	// Définir l'argument --config pour spécifier le chemin du fichier de configuration
	configPath := flag.String("config", "config.yaml", "Chemin vers le fichier de configuration")
	flag.Parse() // Analyser les flags avant de récupérer les arguments

	// Charger la configuration depuis le fichier spécifié
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Erreur lors du chargement de la configuration : %v", err)
	}

	// Récupérer la sous-commande (par exemple "repos list")
	args := flag.Args()
	if len(args) < 1 {
		log.Fatal("Aucune commande spécifiée. Exemple : alcli status ou alcli repos list")
	}

	// Vérifier la sous-commande et l'exécuter
	switch args[0] {
	case "status":
		statusCommand(cfg)
	case "executions":
		if len(args) > 1 && args[1] == "list" {
			reposListCommand(cfg)
		} else {
			fmt.Println("Sous-commande inconnue pour 'repos'. Utilisez 'list' après 'repos'.")
		}
	default:
		fmt.Println("Commande inconnue. Utilisez 'status' ou 'repos list'.")
	}
}
