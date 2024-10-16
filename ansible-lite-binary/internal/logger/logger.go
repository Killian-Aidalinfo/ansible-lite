package logger

import (
    "fmt"
    "log"
    "time"
)
func init() {
    // Désactiver l'ajout automatique du timestamp par log.Printf
    log.SetFlags(0)
}
// Logger personnalisé qui ajoute un timestamp ISO 8601 et un niveau de log (INFO, ERROR)
func Log(level string, format string, v ...interface{}) {
    timestamp := time.Now().Format(time.RFC3339)
    log.Printf(fmt.Sprintf("[%s] [%s] %s", timestamp, level, format), v...)
}
