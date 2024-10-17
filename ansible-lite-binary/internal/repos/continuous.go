package repos

import (
    "fmt"
    "os/exec"
    "strings"
    "sync"
    "github.com/robfig/cron/v3"
    "aidalinfo/ansible-lite/internal/logger"
)

type Continuous struct {
    Name      string   `yaml:"name"`
    Images    []string `yaml:"images"`
    Watcher   string   `yaml:"watcher"`
    InitRepo  string   `yaml:"init_repo"`
    Init      string   `yaml:"init"`
    Branch    string   `yaml:"branch"`
    Path      string   `yaml:"path"`
    Auth      bool     `yaml:"auth"`
}

func planContinuousCron(c *cron.Cron, wg *sync.WaitGroup, continuousName string, continuous Continuous, ghToken string) {
    _, err := c.AddFunc(continuous.Watcher, func() {
        logger.Log("INFO", fmt.Sprintf("Tâche planifiée exécutée pour le dépôt continuous %s", continuousName))
        wg.Add(1)
        go func() {
            defer wg.Done()
            err := processContinuous(continuousName, continuous, ghToken)
            if err != nil {
                logger.Log("ERROR", fmt.Sprintf("Erreur lors du traitement du continuous %s : %v", continuousName, err))
            }
        }()
    })
    if err != nil {
        logger.Log("ERROR", fmt.Sprintf("Erreur lors de l'ajout du cron pour le continuous %s : %v", continuousName, err))
    }
}

func processContinuous(continuousName string, continuous Continuous, ghToken string) error {
	logger.Log("INFO", fmt.Sprintf("Démarrage du traitement pour le continuous %s", continuousName))
	for _, image := range continuous.Images {
			localSHA, err := getLocalDockerImageSHA(image)
			if err != nil {
					logger.Log("ERROR", fmt.Sprintf("Erreur lors de la récupération du SHA local pour l'image Docker %s : %v", image, err))
					continue
			}

			remoteSHA, err := getDockerImageSHA(image)
			if err != nil {
					logger.Log("ERROR", fmt.Sprintf("Erreur lors de la récupération du SHA distant pour l'image Docker %s : %v", image, err))
					continue
			}
			if localSHA != remoteSHA {
					logger.Log("INFO", fmt.Sprintf("Nouveau SHA détecté pour %s (continuous: %s) : %s", image, continuousName, remoteSHA))
					err := cloneRepo(continuous.InitRepo, continuous.Branch, continuous.Path, ghToken, continuous.Auth)
					if err != nil {
							logger.Log("ERROR", fmt.Sprintf("Erreur lors du clonage du dépôt %s : %v", continuous.InitRepo, err))
							return err
					}

					err = runInitScript(continuous.Init, continuous.Path)
					if err != nil {
							logger.Log("ERROR", fmt.Sprintf("Erreur lors de l'exécution du script init pour le continuous %s : %v", continuousName, err))
							return err
					}

					break // Sortir de la boucle dès qu'un SHA change
			} else {
					logger.Log("INFO", fmt.Sprintf("Aucun changement de SHA pour l'image Docker %s", image))
			}
	}

	return nil
}


func getLocalDockerImageSHA(image string) (string, error) {
    cmd := exec.Command("docker", "inspect", "--format={{index .RepoDigests 0}}", image)
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("Erreur lors de la récupération du digest local pour l'image %s : %v", image, err)
    }

    sha := strings.Split(strings.TrimSpace(string(output)), "@")
    if len(sha) < 2 {
        return "", fmt.Errorf("SHA non trouvé pour l'image %s", image)
    }

    return sha[1], nil // Retourner le SHA256
}

func getDockerImageSHA(image string) (string, error) {
    // Pull l'image la plus récente depuis le registre Docker
    pullCmd := exec.Command("docker", "pull", image)
    pullOutput, err := pullCmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("Erreur lors du pull de l'image Docker %s : %v\nSortie : %s", image, err, string(pullOutput))
    }

    // Inspecter l'image pour obtenir le SHA
    inspectCmd := exec.Command("docker", "inspect", "--format={{index .RepoDigests 0}}", image)
    output, err := inspectCmd.Output()
    if err != nil {
        return "", fmt.Errorf("Erreur lors de l'inspection de l'image Docker %s : %v", image, err)
    }

    sha := strings.Split(strings.TrimSpace(string(output)), "@")
    if len(sha) < 2 {
        return "", fmt.Errorf("SHA non trouvé pour l'image %s", image)
    }

    return sha[1], nil // Retourne le SHA256
}
