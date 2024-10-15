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
    "log"
    "regexp"
)

// Structure pour lire la réponse de l'API GitHub
type GithubCommit struct {
    SHA string `json:"sha"`
}
type GithubTag struct {
    Name string `json:"name"`
}
type Flux struct {
    Name     string   `yaml:"name"`   // Nom du flux
    URLs     []string `yaml:"urls"`   // Liste d'URLs
    Watcher  string   `yaml:"watcher"`
    Regex    string   `yaml:"regex"`
    InitRepo string   `yaml:"init_repo"`
    Init     string   `yaml:"init"`
    Branch   string   `yaml:"branch"`
    Path     string   `yaml:"path"`
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
    Flux  map[string]Flux `yaml:"flux"`
}

// Charger le fichier repos.yaml
func LoadReposConfig(path string, dbPath string, ghToken string) (*ReposConfig, error) {
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

    // Charger et insérer les flux dans la base de données
    err = loadFluxs(dbPath, path, ghToken)
    if err != nil {
        // On loggue l'erreur sans la retourner, afin de ne pas stopper l'application
        logger.Log("ERROR", "Erreur lors du chargement des flux : %v", err)
    }

    return &reposConfig, nil
}


// Charger les flux depuis le fichier repos.yaml et les insérer dans la base de données
func loadFluxs(dbPath, reposConfigPath, ghToken string) error {
    var reposConfig ReposConfig
    data, err := ioutil.ReadFile(reposConfigPath)
    if err != nil {
        log.Printf("Impossible de lire le fichier repos.yaml : %v", err)
        return err
    }

    err = yaml.Unmarshal(data, &reposConfig)
    if err != nil {
        log.Printf("Erreur lors du parsing du fichier repos.yaml : %v", err)
        return err
    }

    for fluxName, flux := range reposConfig.Flux {
        for _, url := range flux.URLs {
            exists, err := db.FluxExists(dbPath, fluxName, url)
            if err != nil {
                log.Printf("Erreur lors de la vérification de l'existence du flux %s : %v", fluxName, err)
                continue // Continue même en cas d'erreur
            }

            if !exists {
                latestTag, err := getLatestTagFromAPI(url, flux.Regex, ghToken)
                if err != nil {
                    log.Printf("Erreur lors de la récupération des tags pour %s : %v", url, err)
                    continue // Continue même en cas d'erreur
                }

                err = db.InsertFlux(dbPath, fluxName, url, flux.Regex)
                if err != nil {
                    log.Printf("Erreur lors de l'insertion du flux %s dans la base de données : %v", fluxName, err)
                    continue // Continue même en cas d'erreur
                }

                err = db.UpdateFluxLastTag(dbPath, fluxName, url, latestTag)
                if err != nil {
                    log.Printf("Erreur lors de la mise à jour du dernier tag du flux %s : %v", fluxName, err)
                    continue // Continue même en cas d'erreur
                }

                log.Printf("Flux %s avec l'URL %s et le dernier tag %s inséré dans la base de données", fluxName, url, latestTag)
            } else {
                log.Printf("Flux %s avec l'URL %s existe déjà dans la base de données", fluxName, url)
            }
        }
    }

    return nil // Retourne nil pour ne pas stopper l'application même s'il y a eu des erreurs
}



// Planifier les tâches pour chaque dépôt
func ScheduleRepos(reposConfig *ReposConfig, dbPath string, ghToken string) {
    c := cron.New()
    var wg sync.WaitGroup

    // Planifier les dépôts
    for name, repo := range reposConfig.Repos {
        repo.Name = name
        planRepoCron(c, &wg, repo, dbPath)
    }

    // Planifier les flux
    for fluxName, flux := range reposConfig.Flux {
        planFluxCron(c, &wg, fluxName, flux, dbPath, ghToken)
    }

    c.Start()
    wg.Wait()
}


func planRepoCron(c *cron.Cron, wg *sync.WaitGroup, repo Repo, dbPath string) {
    _, err := c.AddFunc(repo.Watcher, func() {
        logger.Log("INFO", "Tâche planifiée exécutée pour le dépôt %s (%s)", repo.Name, repo.URL)
        wg.Add(1)
        go func() {
            defer wg.Done()
            err := processRepo(dbPath, repo)
            if err != nil {
                logger.Log("ERROR", "Erreur lors du traitement du dépôt %s (%s) : %v", repo.Name, repo.URL, err)
            }
        }()
    })
    if err != nil {
        logger.Log("ERROR", "Erreur lors de l'ajout du cron pour le dépôt %s : %v", repo.Name, err)
    }
}

func planFluxCron(c *cron.Cron, wg *sync.WaitGroup, fluxName string, flux Flux, dbPath string, ghToken string) {
    _, err := c.AddFunc(flux.Watcher, func() {
        logger.Log("INFO", "Tâche planifiée exécutée pour le flux %s", fluxName)
        wg.Add(1)
        go func() {
            defer wg.Done()
            err := processFlux(dbPath, fluxName, flux, ghToken)
            if err != nil {
                logger.Log("ERROR", "Erreur lors du traitement du flux %s : %v", fluxName, err)
            }
        }()
    })
    if err != nil {
        logger.Log("ERROR", "Erreur lors de l'ajout du cron pour le flux %s : %v", fluxName, err)
    }
}

func processFlux(dbPath, fluxName string, flux Flux, ghToken string) error {
    logger.Log("INFO", "Démarrage du traitement pour le flux %s", fluxName)

    for _, url := range flux.URLs {
        lastTag, err := db.GetLastTag(dbPath, fluxName, url)
        if err != nil {
            logger.Log("ERROR", "Erreur lors de la récupération du dernier tag pour l'URL %s dans le flux %s : %v", url, fluxName, err)
            continue // Ne pas arrêter l'exécution pour ce flux, continuez avec le suivant
        }

        // Récupérer le dernier tag correspondant à la regex en utilisant l'API GitHub et le token
        newTag, err := getLatestTagFromAPI(url, flux.Regex, ghToken)
        if err != nil {
            logger.Log("ERROR", "Erreur lors de la récupération du tag GitHub pour l'URL %s : %v", url, err)
            continue // Ne pas arrêter, continuez avec le flux suivant
        }

        // Si un nouveau tag est détecté
        if newTag != "" && newTag != lastTag {
            logger.Log("INFO", "Nouveau tag détecté pour %s (flux: %s) : %s", url, fluxName, newTag)

            // Cloner le dépôt d'initialisation (si nécessaire)
            err := cloneRepo(flux.InitRepo, flux.Branch, flux.Path)
            if err != nil {
                logger.Log("ERROR", "Erreur lors du clonage du dépôt %s : %v", flux.InitRepo, err)
                continue // Ne pas arrêter, continuez avec le flux suivant
            }

            // Exécuter le script init (si nécessaire)
            err = runInitScript(flux.Init, flux.Path)
            if err != nil {
                logger.Log("ERROR", "Erreur lors de l'exécution du script init pour le flux %s : %v", fluxName, err)
                continue // Ne pas arrêter, continuez avec le flux suivant
            }

            // Mettre à jour le dernier tag dans la base de données
            err = db.UpdateFluxLastTag(dbPath, fluxName, url, newTag)
            if err != nil {
                logger.Log("ERROR", "Erreur lors de la mise à jour du dernier tag pour %s : %v", url, err)
                continue // Ne pas arrêter, continuez avec le flux suivant
            }
        } else {
            logger.Log("INFO", "Aucun nouveau tag détecté pour %s dans le flux %s", url, fluxName)
        }
    }
    return nil // Ne pas retourner d'erreur pour laisser l'application continuer
}

func processRepo(dbPath string, repo Repo) error {
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
    latestCommit, err := getLatestCommit(repo.URL, repo.Branch)
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
    err = cloneRepo(repo.URL, repo.Branch, repoPath)
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

// Fonction pour obtenir le dernier tag depuis l'API GitHub en utilisant un token
func getLatestTagFromAPI(repoURL, regexPattern, ghToken string) (string, error) {
    apiURL := convertRepoURLToAPITags(repoURL)
    client := &http.Client{}
    page := 1
    var allTags []GithubTag

    // Boucle pour paginer et récupérer tous les tags
    for {
        // Ajouter le paramètre de pagination `page`
        paginatedURL := fmt.Sprintf("%s?page=%d", apiURL, page)
        req, err := http.NewRequest("GET", paginatedURL, nil)
        if err != nil {
            return "", fmt.Errorf("erreur lors de la création de la requête HTTP : %v", err)
        }
        req.Header.Set("Authorization", "token "+ghToken)

        // Effectuer la requête
        resp, err := client.Do(req)
        if err != nil {
            return "", fmt.Errorf("erreur lors de la requête HTTP pour %s : %v", repoURL, err)
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            return "", fmt.Errorf("réponse HTTP inattendue pour %s : %s", repoURL, resp.Status)
        }

        // Décoder la réponse en une liste de tags
        var tags []GithubTag
        err = json.NewDecoder(resp.Body).Decode(&tags)
        if err != nil {
            return "", fmt.Errorf("erreur lors du décodage de la réponse JSON pour %s : %v", repoURL, err)
        }

        // Si aucun tag n'est retourné, c'est la fin de la pagination
        if len(tags) == 0 {
            break
        }

        // Ajouter les tags récupérés à la liste des tags
        allTags = append(allTags, tags...)
        page++
    }

    // Afficher tous les tags récupérés
    log.Printf("Tags récupérés pour %s :", repoURL)
    for _, tag := range allTags {
        log.Printf("Tag: %s", tag.Name)
    }

    // Vérifier chaque tag avec la regex
    re, err := regexp.Compile(regexPattern)
    if err != nil {
        return "", fmt.Errorf("erreur lors de la compilation de la regex : %v", err)
    }

    for _, tag := range allTags {
        if re.MatchString(tag.Name) {
            return tag.Name, nil
        }
    }

    return "", fmt.Errorf("aucun tag trouvé correspondant à la regex : %s", regexPattern)
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
// Compiler la regex pour vérifier les tags
func compileRegex(pattern string) (*regexp.Regexp, error) {
    re, err := regexp.Compile(pattern)
    if err != nil {
        return nil, fmt.Errorf("erreur lors de la compilation de la regex : %v", err)
    }
    return re, nil
}
// Convertir l'URL du dépôt en URL de l'API GitHub pour récupérer les tags
func convertRepoURLToAPITags(repoURL string) string {
    // Remplacer la partie de l'URL GitHub par celle de l'API
    repoAPIURL := strings.Replace(repoURL, "https://github.com/", "https://api.github.com/repos/", 1)
    // Supprimer l'éventuel suffixe .git dans l'URL
    repoAPIURL = strings.TrimSuffix(repoAPIURL, ".git")
    // Ajouter le chemin pour accéder aux tags
    return fmt.Sprintf("%s/tags", repoAPIURL)
}
