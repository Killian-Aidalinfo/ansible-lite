package api

import (
	"fmt"
	"log"
	"net/http"
	"aidalinfo/ansible-lite/internal/endpoints"
	"aidalinfo/ansible-lite/internal/middleware"
	"aidalinfo/ansible-lite/internal/config"
)

// Démarrer le serveur HTTP avec le port passé en paramètre et la configuration pour le token
func StartServer(port int, cfg *config.GlobalConfig) {
	// Initialiser les routes depuis le package endpoint
	mux := http.NewServeMux()
	endpoints.InitRoutes(mux, cfg)

	// Appliquer le middleware pour valider le token
	handlerWithMiddleware := middleware.ValidateToken(mux, cfg)

	// Démarrer le serveur sur le port spécifié
	log.Printf("Serveur API démarré sur le port %d", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), handlerWithMiddleware); err != nil {
		log.Fatalf("Erreur lors du démarrage du serveur HTTP : %v", err)
	}
}
