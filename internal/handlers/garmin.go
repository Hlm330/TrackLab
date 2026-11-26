package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"tracklab/internal/api"
)

type GarminHandler struct {
	client *api.GarminClient
}

func NewGarminHandler(
	client *api.GarminClient,
) *GarminHandler {
	return &GarminHandler{
		client: client,
	}
}

// Enregistre les routes de consultation des données Garmin.
func (h *GarminHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/garmin/status", h.status)

	mux.HandleFunc(
		"GET /api/garmin/activities",
		h.protect(h.activities),
	)

	mux.HandleFunc(
		"GET /api/garmin/sleep",
		h.protect(h.daily(h.client.GetSleep)),
	)

	mux.HandleFunc(
		"GET /api/garmin/body-battery",
		h.protect(h.daily(h.client.GetBodyBattery)),
	)

	mux.HandleFunc(
		"GET /api/garmin/heart-rate",
		h.protect(h.daily(h.client.GetHeartRate)),
	)

	mux.HandleFunc(
		"GET /api/garmin/hrv",
		h.protect(h.daily(h.client.GetHRV)),
	)

	mux.HandleFunc(
		"GET /api/garmin/summary",
		h.protect(h.daily(h.client.GetDailySummary)),
	)
}

// Empêche la consultation des données depuis une origine externe.
func (h *GarminHandler) protect(
	next http.HandlerFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")

		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			Error(w, http.StatusForbidden, "Accès externe interdit.")
			return
		}

		origin := r.Header.Get("Origin")

		if origin != "" && origin != "http://"+r.Host {
			Error(w, http.StatusForbidden, "Origine non autorisée.")
			return
		}

		if !h.client.Configured() {
			Error(
				w,
				http.StatusServiceUnavailable,
				"Garmin n'est pas configuré.",
			)
			return
		}

		next(w, r)
	}
}

// Indique uniquement si une clé personnelle est configurée.
func (h *GarminHandler) status(
	w http.ResponseWriter,
	r *http.Request,
) {
	JSON(w, http.StatusOK, map[string]any{
		"configured": h.client.Configured(),
		"provider":   "Garmin",
		"mode":       "personal",
	})
}

// Récupère les dernières activités sportives.
func (h *GarminHandler) activities(
	w http.ResponseWriter,
	r *http.Request,
) {
	limit := 20

	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)

		if err != nil || parsed < 1 || parsed > 100 {
			Error(
				w,
				http.StatusBadRequest,
				"Le nombre d'activités doit être compris entre 1 et 100.",
			)
			return
		}

		limit = parsed
	}

	data, err := h.client.GetActivities(limit)

	h.respond(w, data, err)
}

// Mutualise la validation des dates et les requêtes quotidiennes.
func (h *GarminHandler) daily(
	fetch func(string) (json.RawMessage, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		date := r.URL.Query().Get("date")

		if date == "" {
			date = time.Now().Format("2006-01-02")
		}

		parsed, err := time.ParseInLocation(
			"2006-01-02",
			date,
			time.Local,
		)

		if err != nil {
			Error(
				w,
				http.StatusBadRequest,
				"Format de date attendu : AAAA-MM-JJ.",
			)
			return
		}

		if parsed.After(time.Now()) {
			Error(
				w,
				http.StatusBadRequest,
				"La date ne peut pas être future.",
			)
			return
		}

		data, err := fetch(date)

		h.respond(w, data, err)
	}
}

// Centralise les réponses sans exposer les erreurs internes.
func (h *GarminHandler) respond(
	w http.ResponseWriter,
	data json.RawMessage,
	err error,
) {
	if err != nil {
		Error(
			w,
			http.StatusBadGateway,
			"Les données Garmin sont momentanément indisponibles.",
		)
		return
	}

	JSON(w, http.StatusOK, data)
}
