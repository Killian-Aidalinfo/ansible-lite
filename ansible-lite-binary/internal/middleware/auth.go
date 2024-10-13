package middleware

import (
	"net/http"
	"aidalinfo/ansible-lite/internal/config"
)

// Middleware pour vérifier le token d'API
func ValidateToken(next http.Handler, cfg *config.GlobalConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Récupérer le token de l'en-tête Authorization
		token := r.Header.Get("Authorization")

		// Vérifier si le token est présent et correspond au token attendu
		if token == "" || token != cfg.Global.Credentials {
			http.Error(w, "Token invalide ou manquant", http.StatusUnauthorized)
			return
		}

		// Si le token est valide, continuer vers le prochain handler
		next.ServeHTTP(w, r)
	})
}
