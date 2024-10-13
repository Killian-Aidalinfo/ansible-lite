package endpoints

import (
    "fmt"
    "net/http"
    "aidalinfo/ansible-lite/internal/config"
    "aidalinfo/ansible-lite/internal/middleware"
)

// Handler pour l'endpoint /status
func StatusHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Ready!")
}

// Initialiser les routes
func InitRoutes(mux *http.ServeMux, cfg *config.GlobalConfig) {
    mux.Handle("/status", middleware.ValidateToken(http.HandlerFunc(StatusHandler), cfg))
    mux.Handle("/executions", middleware.ValidateToken(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ExecutionDetailsHandler(w, r, cfg)
    }), cfg))
}
