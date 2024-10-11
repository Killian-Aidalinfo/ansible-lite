package logger

import (
    "fmt"
    "log"
    "time"
)

// Logger personnalisé qui ajoute un timestamp ISO 8601 et un niveau de log (INFO, ERROR)
func Log(level string, format string, v ...interface{}) {
    timestamp := time.Now().Format(time.RFC3339)
    log.Printf(fmt.Sprintf("[%s] [%s] %s", timestamp, level, format), v...)
}
