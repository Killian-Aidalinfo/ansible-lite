package db

import (
    "database/sql"
    "aidalinfo/ansible-lite/internal/logger"
    _ "github.com/mattn/go-sqlite3"
)

type ExecutionDetail struct {
    RepoName   string
    RepoURL    string
    CommitID   string
    ExecutedAt string
}

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

    CREATE TABLE IF NOT EXISTS flux (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        flux_name TEXT,  -- Nom du flux (ex: satelease)
        url TEXT,   -- URL du dépôt surveillé
        last_tag TEXT,  -- Dernier tag récupéré
        regex TEXT
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

// Récupérer les informations du dépôt à partir de son ID
func GetRepoByID(dbPath string, repoID int) (string, string, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        logger.Log("ERROR", "Impossible d'ouvrir la base de données : %v", err)
        return "", "", err
    }
    defer db.Close()

    var name, repoURL string
    err = db.QueryRow("SELECT name, repo_url FROM repos WHERE id = ?", repoID).Scan(&name, &repoURL)
    if err != nil {
        if err == sql.ErrNoRows {
            logger.Log("ERROR", "Aucun dépôt trouvé pour l'ID : %d", repoID)
            return "", "", nil
        }
        logger.Log("ERROR", "Erreur lors de la récupération des informations pour le dépôt avec l'ID %d : %v", repoID, err)
        return "", "", err
    }
    return name, repoURL, nil
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


// Récupérer les exécutions avec le nom et l'URL du dépôt
func GetExecutionDetails(dbPath string) ([]ExecutionDetail, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        logger.Log("ERROR", "Impossible d'ouvrir la base de données : %v", err)
        return nil, err
    }
    defer db.Close()

    query := `
        SELECT repos.name, repos.repo_url, executions.commit_id, executions.execution_time
        FROM executions
        JOIN repos ON executions.repo_id = repos.id
        ORDER BY executions.execution_time DESC
    `

    rows, err := db.Query(query)
    if err != nil {
        logger.Log("ERROR", "Erreur lors de la récupération des détails des exécutions : %v", err)
        return nil, err
    }
    defer rows.Close()

    var details []ExecutionDetail
    for rows.Next() {
        var detail ExecutionDetail
        if err := rows.Scan(&detail.RepoName, &detail.RepoURL, &detail.CommitID, &detail.ExecutedAt); err != nil {
            logger.Log("ERROR", "Erreur lors du scan des lignes : %v", err)
            return nil, err
        }
        details = append(details, detail)
    }

    return details, nil
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
// Vérifier si un flux existe déjà dans la base de données
func FluxExists(dbPath, fluxName, url string) (bool, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        logger.Log("ERROR", "Impossible d'ouvrir la base de données : %v", err)
        return false, err
    }
    defer db.Close()

    var exists bool
    err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM flux WHERE flux_name = ? AND url = ?)", fluxName, url).Scan(&exists)
    if err != nil {
        logger.Log("ERROR", "Erreur lors de la vérification de l'existence du flux %s pour l'URL %s : %v", fluxName, url, err)
        return false, err
    }

    return exists, nil
}


func InsertFlux(dbPath, fluxName, url, regex string) error {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        logger.Log("ERROR", "Impossible d'ouvrir la base de données : %v", err)
        return err
    }
    defer db.Close()

    _, err = db.Exec("INSERT INTO flux (flux_name, url, regex) VALUES (?, ?, ?)", fluxName, url, regex)
    if err != nil {
        logger.Log("ERROR", "Erreur lors de l'insertion du flux %s : %v", fluxName, err)
        return err
    }

    logger.Log("INFO", "Le flux %s a été ajouté avec succès à la base de données pour l'URL %s", fluxName, url)
    return nil
}

// Mettre à jour le dernier tag pour une URL dans un flux
func UpdateFluxLastTag(dbPath, fluxName, url, lastTag string) error {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        logger.Log("ERROR", "Impossible d'ouvrir la base de données : %v", err)
        return err
    }
    defer db.Close()

    _, err = db.Exec("UPDATE flux SET last_tag = ? WHERE flux_name = ? AND url = ?", lastTag, fluxName, url)
    if err != nil {
        logger.Log("ERROR", "Erreur lors de la mise à jour du dernier tag pour le flux %s (%s) : %v", fluxName, url, err)
        return err
    }

    logger.Log("INFO", "Le dernier tag %s pour le flux %s (URL: %s) a été mis à jour", lastTag, fluxName, url)
    return nil
}

func GetLastTag(dbPath, fluxName, url string) (string, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        logger.Log("ERROR", "Impossible d'ouvrir la base de données : %v", err)
        return "", err
    }
    defer db.Close()

    var lastTag sql.NullString
    err = db.QueryRow("SELECT last_tag FROM flux WHERE flux_name = ? AND url = ?", fluxName, url).Scan(&lastTag)
    if err != nil {
        if err == sql.ErrNoRows {
            logger.Log("INFO", "Aucun tag trouvé pour le flux %s et l'URL %s, il sera ajouté", fluxName, url)
            return "", nil // Aucun tag trouvé
        }
        logger.Log("ERROR", "Erreur lors de la récupération du dernier tag pour le flux %s et l'URL %s : %v", fluxName, url, err)
        return "", err
    }

    if lastTag.Valid {
        return lastTag.String, nil
    } else {
        return "", nil // Si le tag est NULL, on retourne une chaîne vide
    }
}
