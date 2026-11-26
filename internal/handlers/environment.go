package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"tracklab/internal/api"
)

// EnvironmentHandler gère les données environnementales.
type EnvironmentHandler struct {
	client *api.Client
}

// NewEnvironmentHandler initialise les routes environnementales.
func NewEnvironmentHandler(client *api.Client) *EnvironmentHandler {
	return &EnvironmentHandler{
		client: client,
	}
}

// Register enregistre les routes HTTP du module.
func (h *EnvironmentHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/cities", h.cities)
	mux.HandleFunc("GET /api/weather", h.weather)
	mux.HandleFunc("GET /api/air-quality", h.airQuality)
}

// coordinates extrait et valide les coordonnées de la requête.
func coordinates(r *http.Request) (float64, float64, error) {
	latitude, err := strconv.ParseFloat(
		r.URL.Query().Get("lat"), 64,
	)

	if err != nil {
		return 0, 0, errors.New("latitude invalide")
	}

	longitude, err := strconv.ParseFloat(
		r.URL.Query().Get("lon"), 64,
	)

	if err != nil {
		return 0, 0, errors.New("longitude invalide")
	}

	if err := api.ValidateCoordinates(latitude, longitude); err != nil {
		return 0, 0, err
	}

	return latitude, longitude, nil
}

func (h *EnvironmentHandler) cities(
	w http.ResponseWriter,
	r *http.Request,
) {
	name := r.URL.Query().Get("q")

	data, err := h.client.SearchCities(name)
	if err != nil {
		Error(w, http.StatusBadGateway,
			"Recherche de villes impossible")
		return
	}

	JSON(w, http.StatusOK, data)
}

func (h *EnvironmentHandler) weather(
	w http.ResponseWriter,
	r *http.Request,
) {
	latitude, longitude, err := coordinates(r)
	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}

	data, err := h.client.GetWeather(latitude, longitude)
	if err != nil {
		Error(w, http.StatusBadGateway,
			"Données météorologiques indisponibles")
		return
	}

	JSON(w, http.StatusOK, data)
}

func (h *EnvironmentHandler) airQuality(
	w http.ResponseWriter,
	r *http.Request,
) {
	latitude, longitude, err := coordinates(r)
	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}

	data, err := h.client.GetAirQuality(latitude, longitude)
	if err != nil {
		Error(w, http.StatusBadGateway,
			"Données de qualité de l'air indisponibles")
		return
	}

	JSON(w, http.StatusOK, data)
}
