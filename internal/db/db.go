package db

import (
    "database/sql"
    "aidalinfo/ansible-lite/internal/logger"
    _ "github.com/mattn/go-sqlite3"
)

// Fonction pour initialiser la base de données SQLite
func InitDB(dbPath string) error {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        logger.Log("ERROR", "Impossible d'ouvrir la base de données : %v", err)
        return err
    }
    defer db.Close()

    sqlStmt := `
    CREATE TABLE IF NOT EXISTS repos (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT,  -- Ajout du nom du dépôt
        repo_url TEXT,
        last_commit TEXT,
        watch_interval TEXT,
        branch TEXT
    );`
    _, err = db.Exec(sqlStmt)
    if err != nil {
        logger.Log("ERROR", "Impossible de créer la table : %v", err)
        return err
    }

    return nil
}

// Récupérer le dernier commit pour un dépôt
func GetLastCommit(dbPath, repoURL string) (string, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        logger.Log("ERROR", "Impossible d'ouvrir la base de données : %v", err)
        return "", err
    }
    defer db.Close()

    var lastCommit string
    err = db.QueryRow("SELECT last_commit FROM repos WHERE repo_url = ?", repoURL).Scan(&lastCommit)
    if err != nil {
        logger.Log("ERROR", "Erreur lors de la récupération du dernier commit pour le dépôt %s : %v", repoURL, err)
        return "", err
    }

    return lastCommit, nil
}

// Mettre à jour le dernier commit pour un dépôt
func UpdateLastCommit(dbPath, name, repoURL, lastCommit, watchInterval, branch string) error {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        logger.Log("ERROR", "Impossible d'ouvrir la base de données : %v", err)
        return err
    }
    defer db.Close()

    var exists bool
    err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM repos WHERE repo_url = ?)", repoURL).Scan(&exists)
    if err != nil {
        logger.Log("ERROR", "Erreur lors de la vérification de l'existence du dépôt %s : %v", repoURL, err)
        return err
    }

    if exists {
        _, err = db.Exec("UPDATE repos SET name = ?, last_commit = ?, watch_interval = ?, branch = ? WHERE repo_url = ?", name, lastCommit, watchInterval, branch, repoURL)
    } else {
        _, err = db.Exec("INSERT INTO repos (name, repo_url, last_commit, watch_interval, branch) VALUES (?, ?, ?, ?, ?)", name, repoURL, lastCommit, watchInterval, branch)
    }

    if err != nil {
        logger.Log("ERROR", "Erreur lors de la mise à jour du dépôt %s : %v", repoURL, err)
        return err
    }

    return nil
}
