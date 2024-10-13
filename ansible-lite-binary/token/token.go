package token

import (
	"crypto/rand"
	"encoding/hex"
	"log"
)

// Générer un token aléatoire de longueur donnée
func GenerateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		log.Println("Erreur lors de la génération du token:", err)
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
