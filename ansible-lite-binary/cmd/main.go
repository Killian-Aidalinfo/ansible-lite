package main

import (
    "flag"
    "aidalinfo/ansible-lite/internal/initapp"
    "aidalinfo/ansible-lite/internal/repos"
    "aidalinfo/ansible-lite/internal/logger"
)

func main() {
    // Définir l'argument --config pour spécifier le chemin du fichier de configuration
    configPath := flag.String("config", "config.yaml", "Chemin vers le fichier de configuration")
    flag.Parse()

    // Initialiser l'application avec le fichier de configuration spécifié
    cfg, err := initapp.InitApp(*configPath)
    if err != nil {
        logger.Log("ERROR", "Erreur lors de l'initialisation de l'application : %v", err)
        return
    }

    // Charger la configuration des dépôts (repos.yaml)
    reposConfig, err := repos.LoadReposConfig(cfg.Global.ReposConfig, cfg.Global.DBPath, cfg.Global.GithubToken)
    if err != nil {
        logger.Log("ERROR", "Erreur lors du chargement des dépôts : %v", err)
        return
    }

    // Démarrer la surveillance des dépôts (scheduling) avec le chemin de la base de données
    repos.ScheduleRepos(reposConfig, cfg.Global.DBPath, cfg.Global.GithubToken)

    // Garder l'application active
    select {}
}
