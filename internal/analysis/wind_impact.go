package analysis

import "math"

// WindImpactInput associe les conditions de vent
// à une épreuve et à un objectif sportif.
type WindImpactInput struct {
	Event     Event            `json:"event"`
	Objective SessionObjective `json:"objective"`
	Wind      WindInput        `json:"wind"`
}

// WindRegulation décrit la compatibilité indicative
// du vent estimé avec la réglementation.
type WindRegulation struct {
	Applicable bool    `json:"applicable"`
	Limit      float64 `json:"limit"`
	Exceeded   bool    `json:"exceeded"`
	Status     string  `json:"status"`
	Indicative bool    `json:"indicative"`
	Message    string  `json:"message"`
}

// WindImpactResult contient l'interprétation sportive.
type WindImpactResult struct {
	Event                 Event            `json:"event"`
	Discipline            Discipline       `json:"discipline"`
	Objective             SessionObjective `json:"objective"`
	Wind                  WindResult       `json:"wind"`
	Score                 int              `json:"score"`
	Level                 string           `json:"level"`
	LongitudinalIntensity string           `json:"longitudinal_intensity"`
	LateralIntensity      string           `json:"lateral_intensity"`
	PerformanceEffect     string           `json:"performance_effect"`
	TechnicalEffect       string           `json:"technical_effect"`
	Regulation            WindRegulation   `json:"regulation"`
	Rules                 []AnalysisRule   `json:"rules"`
}

// windIntensity classe l'intensité d'une composante.
func windIntensity(speed float64) string {
	speed = math.Abs(speed)

	switch {
	case speed <= 0.5:
		return "negligible"
	case speed <= 1:
		return "low"
	case speed <= 2:
		return "moderate"
	case speed <= 4:
		return "significant"
	case speed <= 6:
		return "strong"
	default:
		return "very_strong"
	}
}

// evaluateWindRegulation ne constitue jamais
// une décision officielle d'homologation.
func evaluateWindRegulation(
	event Event,
	wind WindResult,
) WindRegulation {
	result := WindRegulation{
		Applicable: event.IsWindRegulated(),
		Indicative: true,
		Status:     "not_applicable",
		Message:    "Seuil réglementaire non applicable à cette épreuve.",
	}

	if !result.Applicable {
		return result
	}

	result.Limit = LegalTailwindLimit
	result.Exceeded = wind.ExceedsTailwindLimit

	if result.Exceeded {
		result.Status = "threshold_exceeded"
		result.Message = "Vent favorable estimé supérieur à +2 m/s."
	} else {
		result.Status = "within_threshold"
		result.Message = "Vent estimé compatible avec le seuil de +2 m/s."
	}

	return result
}

// describeWindEffects distingue l'effet aérodynamique
// de l'effet sur les repères techniques.
func describeWindEffects(
	event Event,
	discipline Discipline,
	longitudinal float64,
) (string, string) {
	if math.Abs(longitudinal) <= 0.5 {
		return "Influence longitudinale négligeable.",
			"Repères peu affectés par la composante longitudinale."
	}

	if discipline == DisciplineThrows {
		switch event {
		case EventJavelin, EventDiscus:
			return "Influence variable sur la trajectoire et le vol de l'engin.",
				"Adapter les paramètres de lâcher aux conditions de vent."

		default:
			return "Influence aérodynamique dépendante de l'engin.",
				"Surveiller la stabilité et la qualité du geste."
		}
	}

	if longitudinal < 0 {
		switch discipline {
		case DisciplineSprint:
			return "Résistance aérodynamique accrue.",
				"Vitesse de course et sensations potentiellement modifiées."

		case DisciplineHurdles:
			return "Progression rendue plus difficile par le vent de face.",
				"Surveiller le rythme inter-haies et les distances d'attaque."

		case DisciplineJumps:
			return "Vitesse d'élan potentiellement réduite.",
				"Contrôler les marques et les repères d'appel."

		default:
			return "Coût énergétique accru à allure comparable.",
				"Adapter l'allure à l'effort réellement fourni."
		}
	}

	switch discipline {
	case DisciplineSprint:
		return "Vitesse facilitée par le vent arrière.",
			"Surveiller les modifications de foulée et de sensations."

	case DisciplineHurdles:
		return "Vitesse d'approche potentiellement facilitée.",
			"Surveiller les appuis, le rythme et les distances d'attaque."

	case DisciplineJumps:
		return "Vitesse d'élan potentiellement facilitée.",
			"Vérifier les marques pour éviter un appel mal positionné."

	default:
		return "Allure facilitée à effort comparable.",
			"Éviter de surestimer sa capacité de maintien d'allure."
	}
}

// AnalyzeWindImpact interprète les composantes du vent
// selon l'épreuve et l'objectif de séance.
//
// Les coefficients sont des heuristiques TrackLab,
// et non des équations de prédiction des performances.
func AnalyzeWindImpact(
	input WindImpactInput,
) (WindImpactResult, error) {
	discipline, err := input.Event.Discipline()

	if err != nil {
		return WindImpactResult{}, err
	}

	if err := input.Objective.Validate(); err != nil {
		return WindImpactResult{}, err
	}

	wind, err := AnalyzeWind(input.Wind)

	if err != nil {
		return WindImpactResult{}, err
	}

	performance, technique := describeWindEffects(
		input.Event,
		discipline,
		wind.Longitudinal,
	)

	result := WindImpactResult{
		Event:                 input.Event,
		Discipline:            discipline,
		Objective:             input.Objective,
		Wind:                  wind,
		Score:                 100,
		LongitudinalIntensity: windIntensity(wind.Longitudinal),
		LateralIntensity:      windIntensity(wind.Lateral),
		PerformanceEffect:     performance,
		TechnicalEffect:       technique,
		Regulation:            evaluateWindRegulation(input.Event, wind),
		Rules:                 make([]AnalysisRule, 0),
	}

	addRule := func(rule AnalysisRule) {
		result.Score -= rule.Penalty
		result.Rules = append(result.Rules, rule)
	}

	longitudinal := math.Abs(wind.Longitudinal)
	lateral := wind.Crosswind

	longitudinalPenalty := 0.0

	// Un vent longitudinal inférieur ou égal à 0,5 m/s
	// est considéré comme négligeable.
	if longitudinal > 0.5 {
		effective := longitudinal - 0.5

		if discipline == DisciplineThrows {
			// Aucun avantage directionnel universel
			// ne peut être attribué aux lancers.
			longitudinalPenalty = effective * 4
		} else {
			switch input.Objective {
			case ObjectivePerformance:
				if wind.Longitudinal < 0 {
					longitudinalPenalty = effective * 8
				} else {
					longitudinalPenalty = max(
						0,
						longitudinal-4,
					) * 6
				}

			case ObjectiveTechnique:
				longitudinalPenalty = effective * 8

			case ObjectiveResistance:
				if wind.Longitudinal < 0 {
					longitudinalPenalty = max(
						0,
						longitudinal-3,
					) * 8
				} else {
					longitudinalPenalty = effective * 5
				}

			case ObjectiveCompetition:
				if wind.Longitudinal < 0 {
					longitudinalPenalty = effective * 6
				} else {
					longitudinalPenalty = max(
						0,
						longitudinal-4,
					) * 6
				}

			case ObjectiveRecovery:
				longitudinalPenalty = effective * 5
			}
		}

		addRule(AnalysisRule{
			Code:           "WIND_LONGITUDINAL",
			Category:       "wind",
			Severity:       windSeverity(longitudinal),
			Penalty:        boundedWindPenalty(longitudinalPenalty),
			Message:        performance,
			Recommendation: technique,
		})
	}

	// Le vent latéral peut perturber la stabilité,
	// les appuis, l'équilibre et les trajectoires.
	if lateral > 0.5 {
		coefficient := 4.0

		switch discipline {
		case DisciplineHurdles, DisciplineJumps, DisciplineThrows:
			coefficient = 8

		case DisciplineSprint:
			coefficient = 6
		}

		if input.Objective == ObjectiveTechnique {
			coefficient *= 1.25
		}

		penalty := boundedWindPenalty(
			(lateral - 0.5) * coefficient,
		)

		addRule(AnalysisRule{
			Code:           "WIND_CROSS",
			Category:       "wind",
			Severity:       windSeverity(lateral),
			Penalty:        penalty,
			Message:        "Vent latéral susceptible de perturber la stabilité.",
			Recommendation: "Adapter les appuis, les repères techniques et la trajectoire.",
		})
	}

	// Le dépassement réglementaire est toujours signalé.
	// Il pénalise le score uniquement en mode compétition.
	if result.Regulation.Exceeded {
		penalty := 0

		if input.Objective == ObjectiveCompetition {
			penalty = 35
		}

		addRule(AnalysisRule{
			Code:           "WIND_REGULATION",
			Category:       "regulation",
			Severity:       "high",
			Penalty:        penalty,
			Message:        result.Regulation.Message,
			Recommendation: "Vérifier la mesure officielle du vent avant toute homologation.",
		})
	}

	// Une forte vitesse de vent reste une contrainte
	// indépendamment de son orientation.
	if wind.Speed > 6 {
		addRule(AnalysisRule{
			Code:           "WIND_STRONG",
			Category:       "wind",
			Severity:       "high",
			Penalty:        boundedWindPenalty((wind.Speed - 6) * 5),
			Message:        "Vent global important.",
			Recommendation: "Réévaluer la pertinence et la sécurité de la séance.",
		})
	}

	result.Score = max(0, min(100, result.Score))
	result.Level = weatherLevel(result.Score)

	return result, nil
}

func boundedWindPenalty(value float64) int {
	return int(math.Round(max(0, min(100, value))))
}

func windSeverity(speed float64) string {
	switch {
	case speed > 6:
		return "high"
	case speed > 2:
		return "moderate"
	default:
		return "low"
	}
}
