# Release v0.0.5 - Ansible Lite

## 🇬🇧 English Version

### 🚀 New Features
- **Docker image monitoring**: Added the ability to monitor Docker images in the `repos.yaml` under the `continuous` section. It allows checking for changes in Docker image SHA and running scripts if a new image is detected.
- **Cron Support for continuous part**: You can now define custom `watcher` cron schedules for continuous tasks in the `repos.yaml` file, allowing fine-tuned control over image monitoring intervals.
- **Log rotation**: Implemented log file rotation with an automatic management system for archived files. Logs will be automatically compressed and archived.


### 📦 Installation
Run an apt upgrade to update your installation:
`sudo apt upgrade ansible-lite`

Download and install the `.deb` package:
- `ansible-lite_amd64.deb`

---

## 🇫🇷 Version Française

### 🚀 Nouvelles fonctionnalités
- **Surveillance des images Docker** : Ajout de la possibilité de surveiller des images Docker dans la section `continuous` du fichier `repos.yaml`. Cela permet de vérifier les changements de SHA des images Docker et de lancer des scripts en cas de nouvelle image détectée.
- **Support cron pour la partie `continuous`** : Il est désormais possible de définir des plages de temps personnalisées avec `watcher` pour les tâches continues dans le fichier `repos.yaml`, offrant un contrôle plus précis sur la fréquence de surveillance des images.
- **Rotation des logs** : Implémentation de la rotation des fichiers de logs avec un système de gestion automatique des fichiers archivés. Les logs seront automatiquement compressés et archivés.

### 📦 Installation
Faites un apt upgrade pour mettre à jour votre installation :
`sudo apt upgrade ansible-lite`

Téléchargez et installez le package `.deb` :
- `ansible-lite_amd64.deb`
"""