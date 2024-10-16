package repos

import (
    "io/ioutil"
    "sync"
    "gopkg.in/yaml.v2"
    "github.com/robfig/cron/v3"
    "aidalinfo/ansible-lite/internal/logger"
		"fmt"
)

func LoadReposConfig(path string, dbPath string, ghToken string) (*ReposConfig, error) {
	var reposConfig ReposConfig
	data, err := ioutil.ReadFile(path)
	if err != nil {
			logger.Log("ERROR", fmt.Sprintf("Impossible de lire le fichier repos.yaml : %v", err))
			return nil, err
	}

	err = yaml.Unmarshal(data, &reposConfig)
	if err != nil {
			logger.Log("ERROR", fmt.Sprintf("Erreur lors du parsing du fichier repos.yaml : %v", err))
			return nil, err
	}

	// Utilisation d'un WaitGroup pour synchroniser les goroutines
	var wg sync.WaitGroup

	// Initialisation des dépôts en parallèle
	for name, repo := range reposConfig.Repos {
			wg.Add(1) // On ajoute une goroutine au WaitGroup
			go func(name string, repo Repo) { // On lance chaque tâche dans une goroutine
					defer wg.Done() // Décrémenter le compteur quand la goroutine est terminée
					repo.Name = name
					reposConfig.Repos[name] = repo
					// Charger et initialiser chaque repo ici, si nécessaire
					logger.Log("INFO", fmt.Sprintf("Repo %s initialisé", name))
			}(name, repo)
	}

	// Initialisation des flux en parallèle
	wg.Add(1)
	go func() {
			defer wg.Done() // Décrémenter le compteur pour la tâche de flux
			err := loadFluxs(dbPath, path, ghToken)
			if err != nil {
					logger.Log("ERROR", fmt.Sprintf("Erreur lors du chargement des flux : %v", err))
			} else {
					logger.Log("INFO", "Flux initialisés avec succès")
			}
	}()

	// Attendre que toutes les goroutines se terminent
	wg.Wait()

	return &reposConfig, nil
}

// Planifier les tâches pour chaque dépôt
func ScheduleRepos(reposConfig *ReposConfig, dbPath string, ghToken string) {
	c := cron.New()
	var wg sync.WaitGroup

	// Planifier les dépôts
	for name, repo := range reposConfig.Repos {
			repo.Name = name
			planRepoCron(c, &wg, repo, dbPath, ghToken)
	}

	// Planifier les flux
	for fluxName, flux := range reposConfig.Flux {
			planFluxCron(c, &wg, fluxName, flux, dbPath, ghToken)
	}

	c.Start()
	wg.Wait()
}
