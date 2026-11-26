package analysis

import (
	"math"
)

// ScoringInput regroupe les données nécessaires
// à l'évaluation globale d'une séance.
type ScoringInput struct {
	Event     Event            `json:"event"`
	Objective SessionObjective `json:"objective"`
	Weather   WeatherInput     `json:"weather"`
	Wind      WindInput        `json:"wind"`
	Training  TrainingInput    `json:"training"`
}

// ScoreWeights représente les pondérations des moteurs.
type ScoreWeights struct {
	Weather  float64 `json:"weather"`
	Wind     float64 `json:"wind"`
	Training float64 `json:"training"`
}

// ScoringResult contient l'analyse complète de la séance.
type ScoringResult struct {
	Event             Event            `json:"event"`
	Discipline        Discipline       `json:"discipline"`
	Objective         SessionObjective `json:"objective"`
	Score             int              `json:"score"`
	RawScore          int              `json:"raw_score"`
	Level             string           `json:"level"`
	Weights           ScoreWeights     `json:"weights"`
	Weather           WeatherResult    `json:"weather"`
	Wind              WindImpactResult `json:"wind"`
	Training          TrainingResult   `json:"training"`
	EnvironmentalOnly bool             `json:"environmental_only"`
	LimitedData       bool             `json:"limited_data"`
	ScoreCapped       bool             `json:"score_capped"`
	HighSeverityAlert bool             `json:"high_severity_alert"`
	Recommendation    string           `json:"recommendation"`
	Rules             []AnalysisRule   `json:"rules"`
}

// objectiveWeights définit les pondérations heuristiques
// propres à chaque objectif de séance.
func objectiveWeights(
	objective SessionObjective,
) ScoreWeights {
	switch objective {
	case ObjectivePerformance:
		return ScoreWeights{30, 30, 40}

	case ObjectiveTechnique:
		return ScoreWeights{30, 45, 25}

	case ObjectiveResistance:
		return ScoreWeights{35, 25, 40}

	case ObjectiveCompetition:
		return ScoreWeights{25, 40, 35}

	case ObjectiveRecovery:
		return ScoreWeights{35, 15, 50}

	default:
		return ScoreWeights{}
	}
}

// effectiveWeights réduit l'influence des données
// personnelles lorsque leur couverture est incomplète.
func effectiveWeights(
	weights ScoreWeights,
	coverage int,
) ScoreWeights {
	weights.Training *= float64(coverage) / 100

	total := weights.Weather +
		weights.Wind +
		weights.Training

	if total == 0 {
		return ScoreWeights{}
	}

	return ScoreWeights{
		Weather:  weights.Weather / total,
		Wind:     weights.Wind / total,
		Training: weights.Training / total,
	}
}

// AnalyzeSession combine les moteurs d'analyse.
//
// Le score représente une compatibilité estimée.
// Il ne constitue pas une garantie de sécurité,
// une prédiction de performance ou un diagnostic.
func AnalyzeSession(
	input ScoringInput,
) (ScoringResult, error) {
	discipline, err := input.Event.Discipline()

	if err != nil {
		return ScoringResult{}, err
	}

	if err := input.Objective.Validate(); err != nil {
		return ScoringResult{}, err
	}

	// L'épreuve sélectionnée constitue la source
	// de vérité concernant la discipline.
	input.Weather.Discipline = discipline

	weather, err := AnalyzeWeather(input.Weather)

	if err != nil {
		return ScoringResult{}, err
	}

	wind, err := AnalyzeWindImpact(WindImpactInput{
		Event:     input.Event,
		Objective: input.Objective,
		Wind:      input.Wind,
	})

	if err != nil {
		return ScoringResult{}, err
	}

	training, err := AnalyzeTraining(input.Training)

	if err != nil {
		return ScoringResult{}, err
	}

	weights := effectiveWeights(
		objectiveWeights(input.Objective),
		training.DataCoverage,
	)

	weightedScore := float64(weather.Score)*weights.Weather +
		float64(wind.Score)*weights.Wind

	if training.Available && training.Score != nil {
		weightedScore += float64(*training.Score) *
			weights.Training
	}

	rawScore := max(
		0,
		min(100, int(math.Round(weightedScore))),
	)

	result := ScoringResult{
		Event:             input.Event,
		Discipline:        discipline,
		Objective:         input.Objective,
		Score:             rawScore,
		RawScore:          rawScore,
		Weights:           weights,
		Weather:           weather,
		Wind:              wind,
		Training:          training,
		EnvironmentalOnly: !training.Available,
		LimitedData:       training.DataCoverage < 60,
		Rules:             make([]AnalysisRule, 0),
	}

	result.Rules = append(result.Rules, weather.Rules...)
	result.Rules = append(result.Rules, wind.Rules...)
	result.Rules = append(result.Rules, training.Rules...)

	// Certaines contraintes environnementales ne doivent
	// pas être masquées par une moyenne pondérée favorable.
	for _, rule := range result.Rules {
		if rule.Severity == "high" {
			result.HighSeverityAlert = true
		}

		switch rule.Code {
		case "EXTREME_HEAT",
			"AIR_VERY_POOR",
			"GUSTS_HIGH",
			"WIND_STRONG":

			result.Score = min(result.Score, 49)
		}
	}

	// En compétition, un dépassement réglementaire
	// doit rester visible dans le score global.
	if input.Objective == ObjectiveCompetition &&
		wind.Regulation.Exceeded {
		result.Score = min(result.Score, 69)
	}

	result.ScoreCapped = result.Score < result.RawScore
	result.Level = weatherLevel(result.Score)

	result.Recommendation = sessionRecommendation(result)

	return result, nil
}

// sessionRecommendation produit une recommandation
// déterministe adaptée aux résultats de l'analyse.
func sessionRecommendation(
	result ScoringResult,
) string {
	if result.ScoreCapped && result.Score < 50 {
		return "Contraintes environnementales importantes : envisager un autre créneau ou adapter la séance."
	}

	if result.Objective == ObjectiveCompetition &&
		result.Wind.Regulation.Exceeded {
		return "Le seuil de vent favorable estimé est dépassé. Vérifier les conditions officielles de compétition."
	}

	if result.HighSeverityAlert {
		return "Examiner les alertes importantes avant de décider du contenu de la séance."
	}

	switch {
	case result.Score >= 85:
		if result.LimitedData {
			return "Conditions estimées favorables, mais données personnelles insuffisantes pour évaluer complètement la préparation."
		}

		return "Conditions estimées favorables à l'objectif de séance."

	case result.Score >= 70:
		return "Conditions globalement favorables, avec quelques adaptations à prévoir."

	case result.Score >= 50:
		return "Conditions modérées : adapter le contenu ou comparer d'autres créneaux."

	case result.Score >= 30:
		return "Conditions défavorables : envisager une séance adaptée ou un autre créneau."

	default:
		return "Conditions très défavorables : réévaluer la pertinence de la séance prévue."
	}
}
