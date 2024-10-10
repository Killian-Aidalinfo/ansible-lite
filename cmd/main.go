package main

import (
    "aidalinfo/ansible-lite/internal/initapp"
    "aidalinfo/ansible-lite/internal/repos"
    "aidalinfo/ansible-lite/internal/logger"
)

func main() {
    // Initialiser l'application en passant le chemin du fichier de configuration
    cfg, err := initapp.InitApp("config.yaml")
    if err != nil {
        logger.Log("ERROR", "Erreur lors de l'initialisation de l'application : %v", err)
        return
    }

    // Charger la configuration des dépôts (repos.yaml)
    reposConfig, err := repos.LoadReposConfig(cfg.Global.ReposConfig)
    if err != nil {
        logger.Log("ERROR", "Erreur lors du chargement des dépôts : %v", err)
        return
    }

    // Démarrer la surveillance des dépôts (scheduling) avec le chemin de la base de données
    repos.ScheduleRepos(reposConfig, cfg.Global.DBPath)

    // Garder l'application active
    select {}
}
