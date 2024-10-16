package repos

import (
    "fmt"
    "io/ioutil"
    "sync"
    "gopkg.in/yaml.v2"
    "net/http"
    "encoding/json"
    "strings"
    "github.com/robfig/cron/v3"
    "aidalinfo/ansible-lite/internal/db"
    "aidalinfo/ansible-lite/internal/logger"
    "log"
    "regexp"
)

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

type GithubTag struct {
	Name string `json:"name"`
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
			logger.Log("ERROR", "Erreur lors du parsing du fichier repos.yaml : %v", err)
			return err
	}

	for fluxName, flux := range reposConfig.Flux {
			for _, url := range flux.URLs {
					exists, err := db.FluxExists(dbPath, fluxName, url)
					if err != nil {
							logger.Log("ERROR", "Erreur lors de la vérification de l'existence du flux %s : %v", fluxName, err)
							continue 
					}

					if !exists {
							latestTag, err := getLatestTagFromAPI(url, flux.Regex, ghToken)
							if err != nil {
									logger.Log("ERROR", "Erreur lors de la récupération des tags pour %s : %v", url, err)
									continue 
							}

							err = db.InsertFlux(dbPath, fluxName, url, flux.Regex)
							if err != nil {
									logger.Log("ERROR", "Erreur lors de l'insertion du flux %s dans la base de données : %v", fluxName, err)
									continue 
							}

							err = db.UpdateFluxLastTag(dbPath, fluxName, url, latestTag)
							if err != nil {
									logger.Log("ERROR", "Erreur lors de la mise à jour du dernier tag du flux %s : %v", fluxName, err)
									continue
							}

							logger.Log("INFO", "Flux %s avec l'URL %s et le dernier tag %s inséré dans la base de données", fluxName, url, latestTag)
					} else {
							logger.Log("INFO", "Flux %s avec l'URL %s existe déjà dans la base de données", fluxName, url)
					}
			}
	}

	return nil
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
					continue 
			}

			// Récupérer le dernier tag correspondant à la regex en utilisant l'API GitHub et le token
			newTag, err := getLatestTagFromAPI(url, flux.Regex, ghToken)
			if err != nil {
					logger.Log("ERROR", "Erreur lors de la récupération du tag GitHub pour l'URL %s : %v", url, err)
					continue 
			}

			// Si un nouveau tag est détecté
			if newTag != "" && newTag != lastTag {
					logger.Log("INFO", "Nouveau tag détecté pour %s (flux: %s) : %s", url, fluxName, newTag)

					// Cloner le dépôt d'initialisation (si nécessaire)
					err := cloneRepo(flux.InitRepo, flux.Branch, flux.Path)
					if err != nil {
							logger.Log("ERROR", "Erreur lors du clonage du dépôt %s : %v", flux.InitRepo, err)
							continue 
					}

					// Exécuter le script init (si nécessaire)
					err = runInitScript(flux.Init, flux.Path)
					if err != nil {
							logger.Log("ERROR", "Erreur lors de l'exécution du script init pour le flux %s : %v", fluxName, err)
							continue
					}

					// Mettre à jour le dernier tag dans la base de données
					err = db.UpdateFluxLastTag(dbPath, fluxName, url, newTag)
					if err != nil {
							logger.Log("ERROR", "Erreur lors de la mise à jour du dernier tag pour %s : %v", url, err)
							continue
					}
			} else {
					logger.Log("INFO", "Aucun nouveau tag détecté pour %s dans le flux %s", url, fluxName)
			}
	}
	return nil 
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
					logger.Log("ERROR", "erreur lors de la création de la requête HTTP : %v", err)
					return "", err 
			}
			req.Header.Set("Authorization", "token "+ghToken)

			// Effectuer la requête
			resp, err := client.Do(req)
			if err != nil {
					logger.Log("ERROR", "erreur lors de la requête HTTP pour %s : %v", repoURL, err)
					return "", err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
					logger.Log("ERROR","réponse HTTP inattendue pour %s : %s", repoURL, resp.Status)
					return "", err
			}

			// Décoder la réponse en une liste de tags
			var tags []GithubTag
			err = json.NewDecoder(resp.Body).Decode(&tags)
			if err != nil {
					logger.Log("ERROR", "erreur lors du décodage de la réponse JSON pour %s : %v", repoURL, err)
					return "", err
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
	// logger.Log("INFO", "Tags récupérés pour %s :", repoURL)
	// for _, tag := range allTags {
	// 		logger.Log("INFO", "Tag: %s", tag.Name)
	// }

	// Vérifier chaque tag avec la regex
	re, err := regexp.Compile(regexPattern)
	if err != nil {
    logger.Log("ERROR", fmt.Sprintf("erreur lors de la compilation de la regex : %v", err))
    return "", err
}

	for _, tag := range allTags {
			if re.MatchString(tag.Name) {
					return tag.Name, nil
			}
	}
	logger.Log("ERROR", "aucun tag trouvé correspondant à la regex : %s", regexPattern)
	return "", err
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