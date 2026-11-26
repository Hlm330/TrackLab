package handlers

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse définit le format commun des erreurs de l'API.
type ErrorResponse struct {
	Error string `json:"error"`
}

// JSON envoie une réponse JSON avec le statut HTTP approprié.
func JSON(w http.ResponseWriter, status int, data any) {
	content, err := json.Marshal(data)

	if err != nil {
		status = http.StatusInternalServerError
		content = []byte(`{"error":"Erreur interne du serveur"}`)
	}

	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)

	_, _ = w.Write(content)
}

// Error envoie une erreur au format JSON.
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, ErrorResponse{
		Error: message,
	})
}
