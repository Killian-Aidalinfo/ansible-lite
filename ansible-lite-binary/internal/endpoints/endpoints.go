package endpoints

import (
    "fmt"
    "net/http"
)

// Handler pour l'endpoint /status
func StatusHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Ready!")
}

// Initialiser les routes
func InitRoutes() {
    http.HandleFunc("/status", StatusHandler)
}
