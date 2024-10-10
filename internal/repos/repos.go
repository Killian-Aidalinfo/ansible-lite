package repos

import (
    "fmt"
    "io/ioutil"
    "sync"
    "gopkg.in/yaml.v2"
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
)

// Structure pour lire la réponse de l'API GitHub
type GitHubCommit struct {
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
}

// Nouvelle structure pour la liste des dépôts avec une map pour les noms des dépôts
type ReposConfig struct {
    Repos map[string]Repo `yaml:"repos"`
}

// Charger le fichier repos.yaml
func LoadReposConfig(path string) (*ReposConfig, error) {
    var reposConfig ReposConfig
    data, err := ioutil.ReadFile(path)
    if err != nil {
        logger.Log("ERROR", "Impossible de lire le fichier repos.yaml : %v", err)
        return nil, err
    }

    err = yaml.Unmarshal(data, &reposConfig)
    if err != nil {
        logger.Log("ERROR", "Erreur lors du parsing du fichier repos.yaml : %v", err)
        return nil, err
    }

    // Assigner les noms de dépôt dans chaque Repo
    for name, repo := range reposConfig.Repos {
        repo.Name = name
        reposConfig.Repos[name] = repo
    }

    return &reposConfig, nil
}

// Planifier les tâches pour chaque dépôt
func ScheduleRepos(reposConfig *ReposConfig, dbPath string) {
    c := cron.New()

    var wg sync.WaitGroup // Créer un WaitGroup pour synchroniser les goroutines

    for name, repo := range reposConfig.Repos {
        repo.Name = name // Stocker le nom du dépôt
        // Ajouter la tâche cron
        _, err := c.AddFunc(repo.Watcher, func() {
            logger.Log("INFO", "Tâche planifiée exécutée pour le dépôt %s (%s)", repo.Name, repo.URL)
            wg.Add(1) // Ajouter une tâche au WaitGroup
            go func(r Repo) {
                defer wg.Done() // Indiquer que la goroutine est terminée
                err := processRepo(dbPath, r)
                if err != nil {
                    logger.Log("ERROR", "Erreur lors du traitement du dépôt %s (%s) : %v", r.Name, r.URL, err)
                }
            }(repo) // Passer "repo" en paramètre à la goroutine
        })
        if err != nil {
            logger.Log("ERROR", "Erreur lors de l'ajout du cron pour le dépôt %s (%s) : %v", repo.Name, repo.URL, err)
        } else {
            logger.Log("INFO", "Cron ajouté pour le dépôt %s (%s) avec l'expression '%s'", repo.Name, repo.URL, repo.Watcher)
        }

        // Exécuter la tâche immédiatement au démarrage
        logger.Log("INFO", "Exécution immédiate de la tâche pour le dépôt %s (%s)", repo.Name, repo.URL)
        wg.Add(1) // Ajouter une tâche pour l'exécution initiale
        go func(r Repo) {
            defer wg.Done() // Indiquer que la goroutine est terminée
            err := processRepo(dbPath, r)
            if err != nil {
                logger.Log("ERROR", "Erreur lors de l'exécution initiale pour le dépôt %s (%s) : %v", r.Name, r.URL, err)
            }
        }(repo)
    }

    // Démarrer le cron pour les exécutions futures
    c.Start()

    // Attendre que toutes les goroutines se terminent
    wg.Wait()
}

// Fonction pour traiter un dépôt individuel
func processRepo(dbPath string, repo Repo) error {
    logger.Log("INFO", "Démarrage du traitement pour le dépôt %s (%s)", repo.Name, repo.URL)
    
    repoPath := filepath.Join(repo.Path, repoNameFromURL(repo.URL))

    // Créer le répertoire si nécessaire
    if _, err := os.Stat(repo.Path); os.IsNotExist(err) {
        logger.Log("INFO", "Création du répertoire pour le dépôt %s : %s", repo.Name, repo.Path)
        err = os.MkdirAll(repo.Path, 0750)
        if err != nil {
            logger.Log("ERROR", "Impossible de créer le répertoire %s : %v", repo.Path, err)
            return err
        }
    }

    // Assurer la suppression du répertoire cloné après l'exécution du script, peu importe le résultat
    defer func() {
        logger.Log("INFO", "Suppression du répertoire cloné %s", repoPath)
        err := os.RemoveAll(repoPath)
        if err != nil {
            logger.Log("ERROR", "Erreur lors de la suppression du répertoire cloné %s : %v", repoPath, err)
        }
    }()

    // Récupérer le dernier commit de la base de données
    lastCommit, err := db.GetLastCommit(dbPath, repo.URL)
    if err == sql.ErrNoRows {
        logger.Log("INFO", "Aucun commit trouvé pour le dépôt %s (%s), récupération du premier commit depuis GitHub", repo.Name, repo.URL)

        // Récupérer le dernier commit depuis GitHub
        latestCommit, err := getLatestCommit(repo.URL, repo.Branch)
        if err != nil {
            logger.Log("ERROR", "Erreur lors de la récupération du dernier commit depuis GitHub pour le dépôt %s : %v", repo.URL, err)
            return err
        }

        // Clonage du dépôt
        logger.Log("INFO", "Clonage du dépôt %s", repo.URL)
        err = cloneRepo(repo.URL, repo.Branch, repoPath)
        if err != nil {
            logger.Log("ERROR", "Erreur lors du clonage du dépôt %s : %v", repo.URL, err)
            return err
        }

        // Assurez-vous que le clonage est terminé avant de lancer le script
        logger.Log("INFO", "Clonage du dépôt %s terminé. Lancement du script d'initialisation %s", repo.Name, repo.Init)

        // Exécution du script init.sh si disponible
        err = runInitScript(repo.Init, repoPath)
        if err != nil {
            logger.Log("ERROR", "Erreur lors de l'exécution du script init pour le dépôt %s : %v", repo.URL, err)
            return err
        }

        // Mettre à jour le dernier commit dans la base de données
        logger.Log("INFO", "Mise à jour du dernier commit pour le dépôt %s", repo.Name)
        err = db.UpdateLastCommit(dbPath, repo.Name, repo.URL, latestCommit, repo.Watcher, repo.Branch)
        if err != nil {
            logger.Log("ERROR", "Erreur lors de la mise à jour du dernier commit dans la base de données pour le dépôt %s : %v", repo.URL, err)
            return err
        }

        logger.Log("INFO", "Premier commit pour le dépôt %s enregistré : %s", repo.Name, latestCommit)
    } else if err != nil {
        logger.Log("ERROR", "Erreur lors de la récupération du dernier commit pour le dépôt %s : %v", repo.Name, err)
        return err
    } else {
        logger.Log("INFO", "Dernier commit existant pour le dépôt %s : %s", repo.Name, lastCommit)
    }

    return nil
}


// Fonction pour extraire le nom du dépôt à partir de l'URL
func repoNameFromURL(url string) string {
    parts := strings.Split(url, "/")
    name := parts[len(parts)-1]
    return strings.TrimSuffix(name, ".git")
}

// Fonction pour obtenir le dernier commit depuis GitHub
func getLatestCommit(repoURL, branch string) (string, error) {
    apiURL := convertRepoURLToAPI(repoURL, branch)

    // Ajoute un log pour voir que l'on tente d'obtenir le dernier commit
    logger.Log("INFO", "Tentative de récupération du dernier commit pour %s sur la branche %s via %s", repoURL, branch, apiURL)

    resp, err := http.Get(apiURL)
    if err != nil {
        logger.Log("ERROR", "Erreur lors de la requête HTTP pour %s : %v", repoURL, err)
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        logger.Log("ERROR", "Réponse HTTP inattendue pour %s : %s", repoURL, resp.Status)
        return "", fmt.Errorf("réponse HTTP inattendue : %s", resp.Status)
    }

    var commit GitHubCommit
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
func cloneRepo(url, branch, path string) error {
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

    // Ajout de l'option --depth=1 pour ne récupérer que le dernier commit
    cmd := exec.Command("git", "clone", "--branch", branch, "--depth", "1", url, path)
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

