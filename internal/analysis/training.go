package analysis

import (
	"errors"
	"math"
	"time"
)

// TrainingInput contient les indicateurs disponibles
// pour évaluer la préparation de l'athlète.
//
// Les pointeurs permettent de distinguer une donnée
// absente d'une valeur réellement égale à zéro.
type TrainingInput struct {
	SleepHours        *float64 `json:"sleep_hours"`
	SleepScore        *float64 `json:"sleep_score"`
	BodyBattery       *float64 `json:"body_battery"`
	RestingHeartRate  *float64 `json:"resting_heart_rate"`
	BaselineHeartRate *float64 `json:"baseline_heart_rate"`
	HRV               *float64 `json:"hrv"`
	BaselineHRV       *float64 `json:"baseline_hrv"`
	Fatigue           *float64 `json:"fatigue"`
}

// TrainingResult représente une estimation heuristique
// de la préparation à l'entraînement.
type TrainingResult struct {
	Available           bool           `json:"available"`
	Score               *int           `json:"score"`
	Level               string         `json:"level"`
	AvailableIndicators int            `json:"available_indicators"`
	TotalIndicators     int            `json:"total_indicators"`
	DataCoverage        int            `json:"data_coverage"`
	Rules               []AnalysisRule `json:"rules"`
}

// TrainingSession représente une séance enregistrée.
type TrainingSession struct {
	Date            string  `json:"date"`
	DurationMinutes float64 `json:"duration_minutes"`
	RPE             float64 `json:"rpe"`
}

// TrainingLoadResult contient les charges cumulées.
type TrainingLoadResult struct {
	Load7Days      float64 `json:"load_7_days"`
	Load28Days     float64 `json:"load_28_days"`
	WeeklyAverage  float64 `json:"weekly_average"`
	Sessions7Days  int     `json:"sessions_7_days"`
	Sessions28Days int     `json:"sessions_28_days"`
}

// validateOptional valide une donnée uniquement
// lorsqu'elle est disponible.
func validateOptional(
	value *float64,
	minimum float64,
	maximum float64,
) bool {
	if value == nil {
		return true
	}

	return !math.IsNaN(*value) &&
		!math.IsInf(*value, 0) &&
		*value >= minimum &&
		*value <= maximum
}

func (input TrainingInput) Validate() error {
	checks := []struct {
		value   *float64
		minimum float64
		maximum float64
	}{
		{input.SleepHours, 0, 24},
		{input.SleepScore, 0, 100},
		{input.BodyBattery, 0, 100},
		{input.RestingHeartRate, 20, 250},
		{input.BaselineHeartRate, 20, 250},
		{input.HRV, 1, 1000},
		{input.BaselineHRV, 1, 1000},
		{input.Fatigue, 1, 10},
	}

	for _, check := range checks {
		if !validateOptional(
			check.value,
			check.minimum,
			check.maximum,
		) {
			return errors.New(
				"indicateur de préparation hors limites",
			)
		}
	}

	return nil
}

// AnalyzeTraining estime la préparation de l'athlète
// uniquement à partir des indicateurs disponibles.
//
// Ce score n'est ni un diagnostic médical,
// ni une autorisation de pratiquer une activité sportive.
func AnalyzeTraining(
	input TrainingInput,
) (TrainingResult, error) {
	if err := input.Validate(); err != nil {
		return TrainingResult{}, err
	}

	result := TrainingResult{
		Level:           "unavailable",
		TotalIndicators: 5,
		Rules:           make([]AnalysisRule, 0),
	}

	totalPenalty := 0

	addRule := func(
		code string,
		category string,
		severity string,
		penalty int,
		message string,
		recommendation string,
	) {
		totalPenalty += penalty

		result.Rules = append(
			result.Rules,
			AnalysisRule{
				Code:           code,
				Category:       category,
				Severity:       severity,
				Penalty:        penalty,
				Message:        message,
				Recommendation: recommendation,
			},
		)
	}

	// Règle 1 : sommeil.
	// La durée et le score constituent un seul indicateur
	// afin de limiter leur double comptabilisation.
	if input.SleepHours != nil || input.SleepScore != nil {
		result.AvailableIndicators++

		penalty := 0

		if input.SleepHours != nil {
			switch {
			case *input.SleepHours < 5:
				penalty = 35
			case *input.SleepHours < 6:
				penalty = 25
			case *input.SleepHours < 7:
				penalty = 15
			case *input.SleepHours < 8:
				penalty = 5
			}
		}

		if input.SleepScore != nil {
			switch {
			case *input.SleepScore < 40:
				penalty = max(penalty, 35)
			case *input.SleepScore < 60:
				penalty = max(penalty, 20)
			case *input.SleepScore < 75:
				penalty = max(penalty, 10)
			}
		}

		if penalty > 0 {
			addRule(
				"SLEEP_RECOVERY",
				"sleep",
				trainingSeverity(penalty),
				penalty,
				"Indicateurs de sommeil moins favorables.",
				"Adapter la charge et privilégier la récupération.",
			)
		}
	}

	// Règle 2 : Body Battery.
	if input.BodyBattery != nil {
		result.AvailableIndicators++

		penalty := 0

		switch {
		case *input.BodyBattery < 20:
			penalty = 35
		case *input.BodyBattery < 40:
			penalty = 20
		case *input.BodyBattery < 60:
			penalty = 10
		}

		if penalty > 0 {
			addRule(
				"BODY_BATTERY_LOW",
				"energy",
				trainingSeverity(penalty),
				penalty,
				"Niveau d'énergie Garmin relativement faible.",
				"Réévaluer l'intensité prévue selon les sensations.",
			)
		}
	}

	// Règle 3 : fréquence cardiaque au repos.
	if input.RestingHeartRate != nil &&
		input.BaselineHeartRate != nil {
		result.AvailableIndicators++

		difference := *input.RestingHeartRate -
			*input.BaselineHeartRate

		penalty := 0

		switch {
		case difference >= 10:
			penalty = 30
		case difference >= 5:
			penalty = 15
		}

		if penalty > 0 {
			addRule(
				"RESTING_HR_ELEVATED",
				"heart_rate",
				trainingSeverity(penalty),
				penalty,
				"Fréquence cardiaque au repos supérieure à la référence personnelle.",
				"Examiner les sensations et les conditions de récupération.",
			)
		}
	}

	// Règle 4 : variabilité cardiaque.
	if input.HRV != nil && input.BaselineHRV != nil {
		result.AvailableIndicators++

		decrease := 100 *
			(*input.BaselineHRV - *input.HRV) /
			*input.BaselineHRV

		penalty := 0

		switch {
		case decrease >= 30:
			penalty = 30
		case decrease >= 15:
			penalty = 15
		}

		if penalty > 0 {
			addRule(
				"HRV_DECREASE",
				"hrv",
				trainingSeverity(penalty),
				penalty,
				"Variabilité cardiaque inférieure à la référence personnelle.",
				"Tenir compte de la tendance et éviter les conclusions sur une mesure isolée.",
			)
		}
	}

	// Règle 5 : fatigue subjective.
	if input.Fatigue != nil {
		result.AvailableIndicators++

		penalty := 0

		switch {
		case *input.Fatigue >= 9:
			penalty = 35
		case *input.Fatigue >= 7:
			penalty = 25
		case *input.Fatigue >= 5:
			penalty = 10
		}

		if penalty > 0 {
			addRule(
				"SUBJECTIVE_FATIGUE",
				"fatigue",
				trainingSeverity(penalty),
				penalty,
				"Fatigue ressentie significative.",
				"Adapter la séance aux sensations réelles.",
			)
		}
	}

	result.DataCoverage = 100 *
		result.AvailableIndicators /
		result.TotalIndicators

	if result.AvailableIndicators == 0 {
		return result, nil
	}

	score := max(0, min(100, 100-totalPenalty))

	result.Available = true
	result.Score = &score
	result.Level = weatherLevel(score)

	return result, nil
}

func trainingSeverity(penalty int) string {
	switch {
	case penalty >= 30:
		return "high"
	case penalty >= 15:
		return "moderate"
	default:
		return "low"
	}
}

// CalculateSessionLoad calcule la charge interne
// selon la méthode session-RPE : durée × difficulté.
//
// Le résultat est exprimé en unités arbitraires.
func CalculateSessionLoad(
	durationMinutes float64,
	rpe float64,
) (float64, error) {
	if !validateOptional(&durationMinutes, 1, 1440) {
		return 0, errors.New("durée de séance invalide")
	}

	if !validateOptional(&rpe, 0, 10) {
		return 0, errors.New("RPE hors limites")
	}

	return durationMinutes * rpe, nil
}

// AnalyzeTrainingLoad calcule les charges cumulées
// sur les 7 et 28 derniers jours calendaires,
// journée de référence incluse.
func AnalyzeTrainingLoad(
	sessions []TrainingSession,
	reference time.Time,
) (TrainingLoadResult, error) {
	result := TrainingLoadResult{}

	location := reference.Location()

	today := time.Date(
		reference.Year(),
		reference.Month(),
		reference.Day(),
		0, 0, 0, 0,
		location,
	)

	start7 := today.AddDate(0, 0, -6)
	start28 := today.AddDate(0, 0, -27)

	for _, session := range sessions {
		date, err := time.ParseInLocation(
			"2006-01-02",
			session.Date,
			location,
		)

		if err != nil {
			return TrainingLoadResult{},
				errors.New("date de séance invalide")
		}

		if date.After(today) {
			return TrainingLoadResult{},
				errors.New("séance située dans le futur")
		}

		load, err := CalculateSessionLoad(
			session.DurationMinutes,
			session.RPE,
		)

		if err != nil {
			return TrainingLoadResult{}, err
		}

		if date.Before(start28) {
			continue
		}

		result.Load28Days += load
		result.Sessions28Days++

		if !date.Before(start7) {
			result.Load7Days += load
			result.Sessions7Days++
		}
	}

	result.WeeklyAverage = result.Load28Days / 4

	return result, nil
}
