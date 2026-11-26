package analysis

import (
	"errors"
	"fmt"
	"math"
)

// WeatherInput représente les conditions environnementales
// d'une séance associée à une discipline athlétique.
type WeatherInput struct {
	Discipline               Discipline `json:"discipline"`
	Temperature              float64    `json:"temperature"`
	ApparentTemperature      float64    `json:"apparent_temperature"`
	Humidity                 float64    `json:"humidity"`
	Precipitation            float64    `json:"precipitation"`
	PrecipitationProbability float64    `json:"precipitation_probability"`
	UVIndex                  float64    `json:"uv_index"`
	WindGusts                float64    `json:"wind_gusts"`
	AirQualityIndex          float64    `json:"air_quality_index"`
}

// AnalysisRule décrit une règle déclenchée par le moteur.
type AnalysisRule struct {
	Code           string `json:"code"`
	Category       string `json:"category"`
	Severity       string `json:"severity"`
	Penalty        int    `json:"penalty"`
	Message        string `json:"message"`
	Recommendation string `json:"recommendation"`
}

// WeatherResult contient les résultats de l'analyse.
type WeatherResult struct {
	Discipline     Discipline     `json:"discipline"`
	ThermalProfile ThermalProfile `json:"thermal_profile"`
	ThermalScore   int            `json:"thermal_score"`
	Score          int            `json:"score"`
	Level          string         `json:"level"`
	Rules          []AnalysisRule `json:"rules"`
}

// Validate vérifie la cohérence des données.
func (input WeatherInput) Validate() error {
	if err := input.Discipline.Validate(); err != nil {
		return err
	}

	values := []float64{
		input.Temperature,
		input.ApparentTemperature,
		input.Humidity,
		input.Precipitation,
		input.PrecipitationProbability,
		input.UVIndex,
		input.WindGusts,
		input.AirQualityIndex,
	}

	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return errors.New("donnée environnementale non valide")
		}
	}

	if input.Humidity < 0 || input.Humidity > 100 {
		return errors.New("humidité hors limites")
	}

	if input.PrecipitationProbability < 0 ||
		input.PrecipitationProbability > 100 {
		return errors.New("probabilité de précipitations hors limites")
	}

	if input.Precipitation < 0 ||
		input.UVIndex < 0 ||
		input.WindGusts < 0 ||
		input.AirQualityIndex < 0 {
		return errors.New("valeur environnementale négative")
	}

	return nil
}

// thermalPenalty calcule l'écart entre la température
// ressentie et le profil thermique de la discipline.
func thermalPenalty(
	temperature float64,
	profile ThermalProfile,
) int {
	penalty := 0.0

	switch {
	case temperature < profile.Minimum:
		penalty = (profile.Minimum - temperature) * 3

	case temperature > profile.Maximum:
		penalty = (temperature - profile.Maximum) * 6
	}

	// Contraintes communes à toutes les disciplines.
	if temperature >= 35 {
		penalty = max(penalty, 40)
	}

	if temperature <= 0 {
		penalty = max(penalty, 25)
	}

	return int(math.Round(min(100, penalty)))
}

// AnalyzeWeather applique des règles heuristiques explicables.
//
// Températures : degrés Celsius.
// Précipitations : millimètres par heure.
// Rafales : kilomètres par heure.
// Qualité de l'air : indice européen.
func AnalyzeWeather(input WeatherInput) (WeatherResult, error) {
	if err := input.Validate(); err != nil {
		return WeatherResult{}, err
	}

	profile, err := input.Discipline.ThermalProfile()

	if err != nil {
		return WeatherResult{}, err
	}

	result := WeatherResult{
		Discipline:     input.Discipline,
		ThermalProfile: profile,
		ThermalScore:   100,
		Score:          100,
		Rules:          make([]AnalysisRule, 0),
	}

	addRule := func(rule AnalysisRule) {
		result.Score -= rule.Penalty
		result.Rules = append(result.Rules, rule)
	}

	// Règle 1 : compatibilité thermique de la discipline.
	penalty := thermalPenalty(
		input.ApparentTemperature,
		profile,
	)

	result.ThermalScore = 100 - penalty

	if penalty > 0 {
		severity := "low"

		switch {
		case penalty >= 40:
			severity = "high"
		case penalty >= 15:
			severity = "moderate"
		}

		if input.ApparentTemperature < profile.Minimum {
			addRule(AnalysisRule{
				Code:     "THERMAL_COLD",
				Category: "temperature",
				Severity: severity,
				Penalty:  penalty,
				Message: fmt.Sprintf(
					"Température ressentie inférieure à la plage privilégiée de %.0f à %.0f °C.",
					profile.Minimum,
					profile.Maximum,
				),
				Recommendation: "Adapter l'échauffement et maintenir la température musculaire pendant les récupérations.",
			})
		} else {
			addRule(AnalysisRule{
				Code:     "THERMAL_HEAT",
				Category: "temperature",
				Severity: severity,
				Penalty:  penalty,
				Message: fmt.Sprintf(
					"Température ressentie supérieure à la plage privilégiée de %.0f à %.0f °C.",
					profile.Minimum,
					profile.Maximum,
				),
				Recommendation: "Adapter l'intensité, l'hydratation et les temps de récupération.",
			})
		}
	}

	// Règle 2 : chaleur importante.
	// La pénalité thermique est déjà comptabilisée.
	if input.ApparentTemperature >= 35 {
		addRule(AnalysisRule{
			Code:           "EXTREME_HEAT",
			Category:       "temperature",
			Severity:       "high",
			Penalty:        0,
			Message:        "Contrainte thermique élevée, quelle que soit la discipline.",
			Recommendation: "Privilégier un créneau plus frais pour les efforts intenses.",
		})
	}

	// Règle 3 : humidité associée à la chaleur.
	if input.Humidity >= 75 && input.Temperature >= 25 {
		addRule(AnalysisRule{
			Code:           "HUMID_HEAT",
			Category:       "humidity",
			Severity:       "moderate",
			Penalty:        10,
			Message:        "Humidité élevée associée à la chaleur.",
			Recommendation: "Adapter les récupérations et surveiller les signes de surchauffe.",
		})
	}

	// Règle 4 : précipitations.
	switch {
	case input.Precipitation >= 5:
		addRule(AnalysisRule{
			Code:           "RAIN_HIGH",
			Category:       "precipitation",
			Severity:       "high",
			Penalty:        25,
			Message:        "Précipitations importantes.",
			Recommendation: "Vérifier l'adhérence de la piste et adapter la séance.",
		})

	case input.Precipitation >= 0.5:
		addRule(AnalysisRule{
			Code:           "RAIN_MODERATE",
			Category:       "precipitation",
			Severity:       "moderate",
			Penalty:        10,
			Message:        "Piste potentiellement humide.",
			Recommendation: "Adapter les exercices nécessitant des appuis précis.",
		})

	case input.PrecipitationProbability >= 70:
		addRule(AnalysisRule{
			Code:           "RAIN_PROBABLE",
			Category:       "precipitation",
			Severity:       "low",
			Penalty:        5,
			Message:        "Probabilité élevée de précipitations.",
			Recommendation: "Prévoir une solution de remplacement.",
		})
	}

	// Règle 5 : exposition aux ultraviolets.
	switch {
	case input.UVIndex >= 8:
		addRule(AnalysisRule{
			Code:           "UV_HIGH",
			Category:       "uv",
			Severity:       "high",
			Penalty:        15,
			Message:        "Indice UV très élevé.",
			Recommendation: "Limiter l'exposition et utiliser une protection solaire adaptée.",
		})

	case input.UVIndex >= 6:
		addRule(AnalysisRule{
			Code:           "UV_MODERATE",
			Category:       "uv",
			Severity:       "moderate",
			Penalty:        5,
			Message:        "Indice UV élevé.",
			Recommendation: "Prévoir une protection solaire.",
		})
	}

	// Règle 6 : rafales de vent.
	switch {
	case input.WindGusts >= 60:
		addRule(AnalysisRule{
			Code:           "GUSTS_HIGH",
			Category:       "wind",
			Severity:       "high",
			Penalty:        30,
			Message:        "Rafales de vent importantes.",
			Recommendation: "Réévaluer la sécurité des exercices et des équipements.",
		})

	case input.WindGusts >= 35:
		addRule(AnalysisRule{
			Code:           "GUSTS_MODERATE",
			Category:       "wind",
			Severity:       "moderate",
			Penalty:        15,
			Message:        "Rafales susceptibles de perturber la séance.",
			Recommendation: "Adapter les exercices techniques et les équipements utilisés.",
		})
	}

	// Règle 7 : qualité de l'air.
	switch {
	case input.AirQualityIndex > 100:
		addRule(AnalysisRule{
			Code:           "AIR_VERY_POOR",
			Category:       "air_quality",
			Severity:       "high",
			Penalty:        40,
			Message:        "Qualité de l'air très dégradée.",
			Recommendation: "Privilégier un autre créneau ou un environnement intérieur adapté.",
		})

	case input.AirQualityIndex > 60:
		addRule(AnalysisRule{
			Code:           "AIR_POOR",
			Category:       "air_quality",
			Severity:       "moderate",
			Penalty:        20,
			Message:        "Qualité de l'air dégradée.",
			Recommendation: "Réduire les efforts intenses prolongés et consulter les recommandations locales.",
		})

	case input.AirQualityIndex > 40:
		addRule(AnalysisRule{
			Code:           "AIR_MODERATE",
			Category:       "air_quality",
			Severity:       "low",
			Penalty:        5,
			Message:        "Qualité de l'air modérée.",
			Recommendation: "Rester attentif aux conditions et aux éventuels symptômes.",
		})
	}

	result.Score = max(0, min(100, result.Score))
	result.Level = weatherLevel(result.Score)

	return result, nil
}

// weatherLevel traduit le score en niveau de compatibilité.
func weatherLevel(score int) string {
	switch {
	case score >= 85:
		return "excellent"
	case score >= 70:
		return "good"
	case score >= 50:
		return "moderate"
	case score >= 30:
		return "poor"
	default:
		return "very_poor"
	}
}
