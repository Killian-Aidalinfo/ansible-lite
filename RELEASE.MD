# Release v0.0.4 - Ansible Lite

## 🇬🇧 English Version

### 🚀 New Features
- **Private repository support**: You can now add `auth: true` in the `repos.yaml` configuration file to support private repositories.
- **Token-based authentication**: Added the ability to use a token for cloning private repositories via GitHub. In the `config.yaml` configuration file, the `gh_token` parameter allows specifying the token to use.
- **GitHub Flux management**: The `flux` mode has been introduced, allowing you to monitor GitHub tags and automatically clone an init repository for each new tag.
- **Internal API creation**: An internal API has been added, which will later allow managing the tool via CLI.

### 🐛 Bug Fixes
- The timestamp in the logs no longer appears twice.

### 📦 Installation
Run an apt upgrade to update your installation:
`sudo apt upgrade ansible-lite`

Download and install the `.deb` package:
- `ansible-lite_amd64.deb`

---

## 🇫🇷 Version Française

### 🚀 Nouvelles fonctionnalités
- **Support de repository privée** : Il est possible d'ajouter `auth: true` dans le fichier de configuration `repos.yaml` pour prendre en charge les dépôts privés.
- **Authentification via token** : Ajout de la possibilité d'utiliser un token pour cloner des dépôts privés via GitHub. Dans le fichier de configuration `config.yaml`, le paramètre `gh_token` permet de spécifier le token à utiliser.
- **Gestion des flux GitHub** : Le mode `flux` fait son apparition, il permet de surveiller des tags GitHub et de cloner automatiquement un dépôt d'init à chaque nouveau tag.
- **Création d'une API interne** : Une API interne a été ajoutée pour plus tard permettre la gestion de l'outil en CLI.

### 🐛 Corrections de bugs
- L'horodatage dans les logs n'apparaît plus en double.

### 📦 Installation
Faites un apt upgrade pour mettre à jour votre installation :

`sudo apt upgrade ansible-lite`

Téléchargez et installez le package `.deb` :
- `ansible-lite_amd64.deb`
