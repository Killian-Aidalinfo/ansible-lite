package endpoints

import (
    "encoding/json"
    "net/http"
    "aidalinfo/ansible-lite/internal/config"
    "aidalinfo/ansible-lite/internal/db"
)

// Handler pour récupérer les détails des exécutions avec nom et URL du dépôt
func ExecutionDetailsHandler(w http.ResponseWriter, r *http.Request, cfg *config.GlobalConfig) {
    // Récupérer les détails des exécutions
    executionDetails, err := db.GetExecutionDetails(cfg.Global.DBPath)
    if err != nil {
        http.Error(w, "Erreur lors de la récupération des détails des exécutions", http.StatusInternalServerError)
        return
    }

    // Encoder les détails des exécutions en JSON et les renvoyer
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(executionDetails)
}
