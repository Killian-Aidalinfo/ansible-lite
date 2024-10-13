package api

import (
    "fmt"
    "log"
    "net/http"
    "aidalinfo/ansible-lite/internal/endpoints"
)

// Démarrer le serveur HTTP avec le port passé en paramètre
func StartServer(port int) {
    // Initialiser les routes depuis le package endpoint
    endpoints.InitRoutes()

    // Démarrer le serveur sur le port spécifié
    log.Printf("Serveur API démarré sur le port %d", port)
    if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
        log.Fatalf("Erreur lors du démarrage du serveur HTTP : %v", err)
    }
}
