package repos

import (
    "fmt"
    "sync"
    "net/http"
    "encoding/json"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "github.com/robfig/cron/v3"
    "aidalinfo/ansible-lite/internal/db"
    "aidalinfo/ansible-lite/internal/logger"
    "database/sql"
    "regexp"
)

// Structure pour lire la réponse de l'API GitHub
type GithubCommit struct {
    SHA string `json:"sha"`
}

// Structure pour un dépôt individuel
type Repo struct {
    Name     string 
    URL      string `yaml:"url"`
    Watcher  string `yaml:"watcher"`
    Init     string `yaml:"init"`
    Branch   string `yaml:"branch"`
    Path     string `yaml:"path"`
    Auth     bool   `yaml:"auth"`
}

// Nouvelle structure pour la liste des dépôts avec une map pour les noms des dépôts
type ReposConfig struct {
    Repos map[string]Repo `yaml:"repos"`
    Flux  map[string]Flux `yaml:"flux"`
}

func planRepoCron(c *cron.Cron, wg *sync.WaitGroup, repo Repo, dbPath string, ghToken string) {
    _, err := c.AddFunc(repo.Watcher, func() {
        logger.Log("INFO", "Tâche planifiée exécutée pour le dépôt %s (%s)", repo.Name, repo.URL)
        wg.Add(1)
        go func() {
            defer wg.Done()
            err := processRepo(dbPath, repo, ghToken)
            if err != nil {
                logger.Log("ERROR", "Erreur lors du traitement du dépôt %s (%s) : %v", repo.Name, repo.URL, err)
            }
        }()
    })
    if err != nil {
        logger.Log("ERROR", "Erreur lors de l'ajout du cron pour le dépôt %s : %v", repo.Name, err)
    }
}

func processRepo(dbPath string, repo Repo, ghToken string) error {
    logger.Log("INFO", "Démarrage du traitement pour le dépôt %s (%s)", repo.Name, repo.URL)
    
    repoPath := filepath.Join(repo.Path, repoNameFromURL(repo.URL))
    if _, err := os.Stat(repo.Path); os.IsNotExist(err) {
        logger.Log("INFO", "Création du répertoire pour le dépôt %s : %s", repo.Name, repo.Path)
        err = os.MkdirAll(repo.Path, 0750)
        if err != nil {
            logger.Log("ERROR", "Impossible de créer le répertoire %s : %v", repo.Path, err)
            return nil // Ne pas arrêter l'application, continuez
        }
    }

    // Récupérer le dernier commit de la base de données
    lastCommit, err := db.GetLastCommit(dbPath, repo.URL)
    if err != nil && err != sql.ErrNoRows {
        logger.Log("ERROR", "Erreur lors de la récupération du dernier commit pour le dépôt %s : %v", repo.Name, err)
        return nil // Continuer même en cas d'erreur
    }

    // Récupérer le dernier commit depuis GitHub
    latestCommit, err := getLatestCommit(repo.URL, repo.Branch, ghToken, repo.Auth)
    if err != nil {
        logger.Log("ERROR", "Erreur lors de la récupération du dernier commit depuis GitHub pour le dépôt %s : %v", repo.URL, err)
        return nil // Continuer même en cas d'erreur
    }

    // Comparer les commits
    if lastCommit == latestCommit {
        logger.Log("INFO", "Aucun nouveau commit pour le dépôt %s, rien à faire", repo.Name)
        return nil
    }

    // Clonage du dépôt
    err = cloneRepo(repo.URL, repo.Branch, repoPath, ghToken, repo.Auth)
    if err != nil {
        logger.Log("ERROR", "Erreur lors du clonage du dépôt %s : %v", repo.URL, err)
        return nil // Continuer même en cas d'erreur
    }

    // Exécution du script d'init
    err = runInitScript(repo.Init, repoPath)
    if err != nil {
        logger.Log("ERROR", "Erreur lors de l'exécution du script init pour le dépôt %s : %v", repo.URL, err)
        return nil // Continuer même en cas d'erreur
    }

    // Mettre à jour le dernier commit
    err = db.UpdateLastCommit(dbPath, repo.Name, repo.URL, latestCommit, repo.Watcher, repo.Branch)
    if err != nil {
        logger.Log("ERROR", "Erreur lors de la mise à jour du dernier commit dans la base de données pour le dépôt %s : %v", repo.URL, err)
        return nil // Continuer même en cas d'erreur
    }

    logger.Log("INFO", "Traitement du dépôt %s terminé avec succès", repo.Name)
    return nil
}

// Fonction pour extraire le nom du dépôt à partir de l'URL
func repoNameFromURL(url string) string {
    parts := strings.Split(url, "/")
    name := parts[len(parts)-1]
    return strings.TrimSuffix(name, ".git")
}

// Fonction pour obtenir le dernier commit depuis GitHub
func getLatestCommit(repoURL, branch, ghToken string, auth bool) (string, error) {
    apiURL := convertRepoURLToAPI(repoURL, branch)

    // Ajoute un log pour voir que l'on tente d'obtenir le dernier commit
    logger.Log("INFO", "Tentative de récupération du dernier commit pour %s sur la branche %s via %s", repoURL, branch, apiURL)

    client := &http.Client{}
    req, err := http.NewRequest("GET", apiURL, nil)
    if err != nil {
        logger.Log("ERROR", "Erreur lors de la création de la requête HTTP : %v", err)
        return "", err
    }

    if auth {
        // Ajouter l'authentification par token si auth est vrai
        req.Header.Set("Authorization", "token "+ghToken)
    }

    resp, err := client.Do(req)
    if err != nil {
        logger.Log("ERROR", "Erreur lors de la requête HTTP pour %s : %v", repoURL, err)
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        logger.Log("ERROR", "Réponse HTTP inattendue pour %s : %s", repoURL, resp.Status)
        return "", fmt.Errorf("réponse HTTP inattendue : %s", resp.Status)
    }

    var commit GithubCommit
    err = json.NewDecoder(resp.Body).Decode(&commit)
    if err != nil {
        logger.Log("ERROR", "Erreur lors du décodage de la réponse JSON pour %s : %v", repoURL, err)
        return "", err
    }

    logger.Log("INFO", "Dernier commit pour %s : %s", repoURL, commit.SHA)
    return commit.SHA, nil
}

// Convertir l'URL du dépôt en URL de l'API GitHub
func convertRepoURLToAPI(repoURL, branch string) string {
    repoAPIURL := strings.Replace(repoURL, "https://github.com/", "https://api.github.com/repos/", 1)
    repoAPIURL = strings.TrimSuffix(repoAPIURL, ".git")
    return fmt.Sprintf("%s/commits/%s", repoAPIURL, branch)
}

// Cloner un dépôt depuis GitHub en ne récupérant que le dernier commit
func cloneRepo(url, branch, path, ghToken string, auth bool) error {
    // Vérifier si le répertoire existe déjà
    if _, err := os.Stat(path); !os.IsNotExist(err) {
        // Si le dossier existe déjà, le supprimer
        logger.Log("INFO", "Le répertoire %s existe déjà. Suppression avant le clonage.", path)
        err := os.RemoveAll(path)
        if err != nil {
            logger.Log("ERROR", "Impossible de supprimer le répertoire %s : %v", path, err)
            return err
        }
    }

    logger.Log("INFO", "Clonage du dépôt %s (branche : %s) dans le répertoire %s", url, branch, path)

    var cmd *exec.Cmd
    if auth {
        // Utiliser le token pour l'authentification
        urlWithAuth := strings.Replace(url, "https://", "https://"+ghToken+"@", 1)
        cmd = exec.Command("git", "clone", "--branch", branch, "--depth", "1", urlWithAuth, path)
    } else {
        // Clonage normal sans token
        cmd = exec.Command("git", "clone", "--branch", branch, "--depth", "1", url, path)
    }
    
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    // Exécuter la commande de clonage et attendre qu'elle soit terminée
    err := cmd.Run()
    if err != nil {
        logger.Log("ERROR", "Erreur lors du clonage du dépôt %s : %v", url, err)
        return err
    }

    logger.Log("INFO", "Clonage du dépôt %s terminé avec succès", url)
    return nil
}

// Exécuter le script init.sh dans le dépôt cloné
func runInitScript(scriptName, repoPath string) error {
    scriptPath := filepath.Join(repoPath, scriptName)

    // Vérifier si le script existe
    if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
        logger.Log("INFO", "Le script %s n'existe pas dans le dépôt %s", scriptName, repoPath)
        return nil // Pas d'erreur, on continue normalement
    }

    // Rendre le script exécutable
    err := os.Chmod(scriptPath, 0750)
    if err != nil {
        logger.Log("ERROR", "Impossible de rendre le script %s exécutable : %v", scriptName, err)
        return err
    }

    logger.Log("INFO", "Exécution du script %s dans le dépôt %s", scriptName, repoPath)

    cmd := exec.Command("./" + scriptName)
    cmd.Dir = repoPath
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    // Attendre que le script soit complètement exécuté
    err = cmd.Run()
    if err != nil {
        logger.Log("ERROR", "Erreur lors de l'exécution du script %s : %v", scriptName, err)
        return err
    }

    logger.Log("INFO", "Script %s exécuté avec succès", scriptName)
    return nil
}

// Fonction pour obtenir le dernier tag correspondant à une regex depuis GitHub
func getLatestTag(repoPath, regexPattern string) (string, error) {
    // Effectuer un fetch des tags sans modifier le dépôt local
    cmd := exec.Command("git", "-C", repoPath, "fetch", "--tags", "--quiet")
    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("erreur lors du fetch des tags : %v", err)
    }

    // Lister et filtrer les tags en fonction de la regex
    cmd = exec.Command("git", "-C", repoPath, "tag", "--sort=-creatordate")
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("erreur lors de la récupération des tags : %v", err)
    }

    tags := strings.Split(string(output), "\n")

    // Compiler la regex
    re, err := regexp.Compile(regexPattern)
    if err != nil {
        return "", fmt.Errorf("erreur lors de la compilation de la regex : %v", err)
    }

    // Chercher le dernier tag correspondant à la regex
    for _, tag := range tags {
        tag = strings.TrimSpace(tag)
        if re.MatchString(tag) {
            return tag, nil
        }
    }

    return "", fmt.Errorf("aucun tag trouvé correspondant à la regex : %s", regexPattern)
}
