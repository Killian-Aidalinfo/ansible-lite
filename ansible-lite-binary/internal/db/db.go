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
    );
    CREATE TABLE IF NOT EXISTS executions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        repo_id INTEGER,
        execution_time DATETIME DEFAULT CURRENT_TIMESTAMP,
        commit_id TEXT,
        FOREIGN KEY (repo_id) REFERENCES repos(id)
    );
    `
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
// Insérer un dépôt dans la table repos
func InsertRepo(dbPath, name, repoURL, lastCommit, watchInterval, branch string) error {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        logger.Log("ERROR", "Impossible d'ouvrir la base de données : %v", err)
        return err
    }
    defer db.Close()

    _, err = db.Exec("INSERT INTO repos (name, repo_url, last_commit, watch_interval, branch) VALUES (?, ?, ?, ?, ?)", name, repoURL, lastCommit, watchInterval, branch)
    if err != nil {
        logger.Log("ERROR", "Erreur lors de l'insertion du dépôt %s : %v", repoURL, err)
        return err
    }

    return nil
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

// Récupérer l'ID du dépôt à partir de son URL
func GetRepoIDByURL(dbPath, repoURL string) (int, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        logger.Log("ERROR", "Impossible d'ouvrir la base de données : %v", err)
        return 0, err
    }
    defer db.Close()

    var repoID int
    err = db.QueryRow("SELECT id FROM repos WHERE repo_url = ?", repoURL).Scan(&repoID)
    if err != nil {
        if err == sql.ErrNoRows {
            logger.Log("ERROR", "Aucun dépôt trouvé pour l'URL : %s", repoURL)
            return 0, nil // Renvoie 0 si aucun dépôt n'est trouvé
        }
        logger.Log("ERROR", "Erreur lors de la récupération de l'ID du dépôt pour l'URL %s : %v", repoURL, err)
        return 0, err
    }
    return repoID, nil
}

// Enregistrer une exécution dans la table executions
func LogExecution(dbPath string, repoID int, commitID string) error {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        logger.Log("ERROR", "Impossible d'ouvrir la base de données : %v", err)
        return err
    }
    defer db.Close()

    logger.Log("INFO", "Insertion de l'exécution dans la base de données (repo_id: %d, commit_id: %s)", repoID, commitID)
    _, err = db.Exec("INSERT INTO executions (repo_id, commit_id) VALUES (?, ?)", repoID, commitID)
    if err != nil {
        logger.Log("ERROR", "Erreur lors de l'insertion de l'exécution pour le dépôt %d : %v", repoID, err)
        return err
    }

    logger.Log("INFO", "Exécution enregistrée avec succès pour le dépôt %d", repoID)
    return nil
}
