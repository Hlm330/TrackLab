package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"

	"tracklab/internal/analysis"
)

const maxAnalysisBody = 1024 * 1024

type AnalysisHandler struct{}

func NewAnalysisHandler() *AnalysisHandler {
	return &AnalysisHandler{}
}

func (h *AnalysisHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/analysis/session", h.session)
	mux.HandleFunc("POST /api/analysis/wind", h.wind)
	mux.HandleFunc("POST /api/analysis/training", h.training)
	mux.HandleFunc("POST /api/analysis/load", h.load)
}

// decodeAnalysis sécurise et valide le corps JSON.
func decodeAnalysis(
	w http.ResponseWriter,
	r *http.Request,
	target any,
) bool {
	w.Header().Set("Cache-Control", "no-store")

	origin := r.Header.Get("Origin")

	if r.Header.Get("Sec-Fetch-Site") == "cross-site" ||
		(origin != "" && origin != "http://"+r.Host) {
		Error(w, http.StatusForbidden, "origine non autorisée")
		return false
	}

	mediaType, _, err := mime.ParseMediaType(
		r.Header.Get("Content-Type"),
	)

	if err != nil || mediaType != "application/json" {
		Error(
			w,
			http.StatusUnsupportedMediaType,
			"contenu JSON requis",
		)
		return false
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxAnalysisBody,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		var sizeError *http.MaxBytesError

		if errors.As(err, &sizeError) {
			Error(
				w,
				http.StatusRequestEntityTooLarge,
				"requête trop volumineuse",
			)
		} else {
			Error(
				w,
				http.StatusBadRequest,
				"corps JSON invalide",
			)
		}

		return false
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		Error(
			w,
			http.StatusBadRequest,
			"un seul objet JSON est attendu",
		)
		return false
	}

	return true
}

// Les pointeurs permettent de vérifier que les données
// environnementales obligatoires sont présentes.
type sessionRequest struct {
	Event     analysis.Event            `json:"event"`
	Objective analysis.SessionObjective `json:"objective"`
	Weather   *weatherRequest           `json:"weather"`
	Wind      *windRequest              `json:"wind"`
	Training  analysis.TrainingInput    `json:"training"`
}

type weatherRequest struct {
	Temperature              *float64 `json:"temperature"`
	ApparentTemperature      *float64 `json:"apparent_temperature"`
	Humidity                 *float64 `json:"humidity"`
	Precipitation            *float64 `json:"precipitation"`
	PrecipitationProbability *float64 `json:"precipitation_probability"`
	UVIndex                  *float64 `json:"uv_index"`
	WindGusts                *float64 `json:"wind_gusts"`
	AirQualityIndex          *float64 `json:"air_quality_index"`
}

func (v *weatherRequest) complete() bool {
	return v != nil &&
		v.Temperature != nil &&
		v.ApparentTemperature != nil &&
		v.Humidity != nil &&
		v.Precipitation != nil &&
		v.PrecipitationProbability != nil &&
		v.UVIndex != nil &&
		v.WindGusts != nil &&
		v.AirQualityIndex != nil
}

func (v *weatherRequest) input() analysis.WeatherInput {
	return analysis.WeatherInput{
		Temperature:              *v.Temperature,
		ApparentTemperature:      *v.ApparentTemperature,
		Humidity:                 *v.Humidity,
		Precipitation:            *v.Precipitation,
		PrecipitationProbability: *v.PrecipitationProbability,
		UVIndex:                  *v.UVIndex,
		WindGusts:                *v.WindGusts,
		AirQualityIndex:          *v.AirQualityIndex,
	}
}

type windRequest struct {
	Speed          *float64 `json:"speed"`
	Direction      *float64 `json:"direction"`
	TrackDirection *float64 `json:"track_direction"`
}

func (v *windRequest) complete() bool {
	return v != nil &&
		v.Speed != nil &&
		v.Direction != nil &&
		v.TrackDirection != nil
}

func (v *windRequest) input() analysis.WindInput {
	return analysis.WindInput{
		Speed:          *v.Speed,
		Direction:      *v.Direction,
		TrackDirection: *v.TrackDirection,
	}
}

func (h *AnalysisHandler) session(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request sessionRequest

	if !decodeAnalysis(w, r, &request) {
		return
	}

	if !request.Weather.complete() ||
		!request.Wind.complete() {
		Error(
			w,
			http.StatusBadRequest,
			"données environnementales incomplètes",
		)
		return
	}

	result, err := analysis.AnalyzeSession(
		analysis.ScoringInput{
			Event:     request.Event,
			Objective: request.Objective,
			Weather:   request.Weather.input(),
			Wind:      request.Wind.input(),
			Training:  request.Training,
		},
	)

	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}

	JSON(w, http.StatusOK, result)
}

func (h *AnalysisHandler) wind(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request struct {
		Event     analysis.Event            `json:"event"`
		Objective analysis.SessionObjective `json:"objective"`
		Wind      *windRequest              `json:"wind"`
	}

	if !decodeAnalysis(w, r, &request) {
		return
	}

	if !request.Wind.complete() {
		Error(
			w,
			http.StatusBadRequest,
			"données de vent incomplètes",
		)
		return
	}

	result, err := analysis.AnalyzeWindImpact(
		analysis.WindImpactInput{
			Event:     request.Event,
			Objective: request.Objective,
			Wind:      request.Wind.input(),
		},
	)

	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}

	JSON(w, http.StatusOK, result)
}

func (h *AnalysisHandler) training(
	w http.ResponseWriter,
	r *http.Request,
) {
	var input analysis.TrainingInput

	if !decodeAnalysis(w, r, &input) {
		return
	}

	result, err := analysis.AnalyzeTraining(input)

	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}

	JSON(w, http.StatusOK, result)
}

func (h *AnalysisHandler) load(
	w http.ResponseWriter,
	r *http.Request,
) {
	var input struct {
		Sessions      []analysis.TrainingSession `json:"sessions"`
		ReferenceDate string                     `json:"reference_date"`
	}

	if !decodeAnalysis(w, r, &input) {
		return
	}

	if len(input.Sessions) > 5000 {
		Error(
			w,
			http.StatusBadRequest,
			"historique trop volumineux",
		)
		return
	}

	reference := time.Now()

	if input.ReferenceDate != "" {
		date, err := time.ParseInLocation(
			"2006-01-02",
			input.ReferenceDate,
			time.Local,
		)

		if err != nil {
			Error(
				w,
				http.StatusBadRequest,
				"date de référence invalide",
			)
			return
		}

		reference = date
	}

	result, err := analysis.AnalyzeTrainingLoad(
		input.Sessions,
		reference,
	)

	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}

	JSON(w, http.StatusOK, result)
}
