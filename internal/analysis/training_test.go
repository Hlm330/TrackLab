package analysis

import (
	"math"
	"testing"
	"time"
)

func trainingValue(value float64) *float64 {
	return &value
}

func TestAnalyzeTraining(t *testing.T) {
	tests := []struct {
		name         string
		input        TrainingInput
		wantScore    int
		wantCoverage int
		wantRules    int
	}{
		{
			"préparation favorable",
			TrainingInput{
				SleepHours:        trainingValue(9),
				SleepScore:        trainingValue(90),
				BodyBattery:       trainingValue(90),
				RestingHeartRate:  trainingValue(55),
				BaselineHeartRate: trainingValue(55),
				HRV:               trainingValue(70),
				BaselineHRV:       trainingValue(70),
				Fatigue:           trainingValue(2),
			},
			100, 100, 0,
		},
		{
			"sommeil très court",
			TrainingInput{SleepHours: trainingValue(4)},
			65, 20, 1,
		},
		{
			"sommeil court",
			TrainingInput{SleepHours: trainingValue(5.5)},
			75, 20, 1,
		},
		{
			"sommeil insuffisant",
			TrainingInput{SleepHours: trainingValue(6.5)},
			85, 20, 1,
		},
		{
			"sommeil légèrement réduit",
			TrainingInput{SleepHours: trainingValue(7.5)},
			95, 20, 1,
		},
		{
			"score sommeil très faible",
			TrainingInput{SleepScore: trainingValue(30)},
			65, 20, 1,
		},
		{
			"score sommeil faible",
			TrainingInput{SleepScore: trainingValue(50)},
			80, 20, 1,
		},
		{
			"score sommeil modéré",
			TrainingInput{SleepScore: trainingValue(70)},
			90, 20, 1,
		},
		{
			"pas de double pénalité sommeil",
			TrainingInput{
				SleepHours: trainingValue(4),
				SleepScore: trainingValue(30),
			},
			65, 20, 1,
		},
		{
			"énergie très faible",
			TrainingInput{BodyBattery: trainingValue(10)},
			65, 20, 1,
		},
		{
			"énergie faible",
			TrainingInput{BodyBattery: trainingValue(30)},
			80, 20, 1,
		},
		{
			"énergie modérée",
			TrainingInput{BodyBattery: trainingValue(50)},
			90, 20, 1,
		},
		{
			"énergie réellement nulle",
			TrainingInput{BodyBattery: trainingValue(0)},
			65, 20, 1,
		},
		{
			"fréquence cardiaque élevée",
			TrainingInput{
				RestingHeartRate:  trainingValue(65),
				BaselineHeartRate: trainingValue(55),
			},
			70, 20, 1,
		},
		{
			"fréquence cardiaque modérée",
			TrainingInput{
				RestingHeartRate:  trainingValue(60),
				BaselineHeartRate: trainingValue(55),
			},
			85, 20, 1,
		},
		{
			"HRV fortement réduite",
			TrainingInput{
				HRV:         trainingValue(40),
				BaselineHRV: trainingValue(80),
			},
			70, 20, 1,
		},
		{
			"HRV modérément réduite",
			TrainingInput{
				HRV:         trainingValue(64),
				BaselineHRV: trainingValue(80),
			},
			85, 20, 1,
		},
		{
			"fatigue très importante",
			TrainingInput{Fatigue: trainingValue(10)},
			65, 20, 1,
		},
		{
			"fatigue importante",
			TrainingInput{Fatigue: trainingValue(7)},
			75, 20, 1,
		},
		{
			"fatigue modérée",
			TrainingInput{Fatigue: trainingValue(5)},
			90, 20, 1,
		},
		{
			"cumul des contraintes",
			TrainingInput{
				SleepHours:  trainingValue(3),
				BodyBattery: trainingValue(10),
				Fatigue:     trainingValue(10),
			},
			0, 60, 3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := AnalyzeTraining(test.input)

			if err != nil {
				t.Fatal(err)
			}

			if !result.Available || result.Score == nil {
				t.Fatal("score indisponible")
			}

			if *result.Score != test.wantScore {
				t.Errorf(
					"score obtenu : %d, attendu : %d",
					*result.Score,
					test.wantScore,
				)
			}

			if result.DataCoverage != test.wantCoverage {
				t.Error("couverture des données incorrecte")
			}

			if len(result.Rules) != test.wantRules {
				t.Error("nombre de règles incorrect")
			}
		})
	}
}

func TestTrainingUnavailable(t *testing.T) {
	inputs := []TrainingInput{
		{},
		{RestingHeartRate: trainingValue(55)},
		{HRV: trainingValue(70)},
		{BaselineHRV: trainingValue(70)},
	}

	for _, input := range inputs {
		result, err := AnalyzeTraining(input)

		if err != nil {
			t.Fatal(err)
		}

		if result.Available ||
			result.Score != nil ||
			result.DataCoverage != 0 {
			t.Error("aucun score ne doit être produit")
		}
	}
}

func TestTrainingValidation(t *testing.T) {
	tests := []TrainingInput{
		{SleepHours: trainingValue(-1)},
		{SleepHours: trainingValue(25)},
		{SleepScore: trainingValue(101)},
		{BodyBattery: trainingValue(-1)},
		{RestingHeartRate: trainingValue(0)},
		{BaselineHeartRate: trainingValue(300)},
		{HRV: trainingValue(0)},
		{BaselineHRV: trainingValue(0)},
		{Fatigue: trainingValue(0)},
		{Fatigue: trainingValue(11)},
		{SleepHours: trainingValue(math.NaN())},
		{HRV: trainingValue(math.Inf(1))},
	}

	for _, input := range tests {
		if _, err := AnalyzeTraining(input); err == nil {
			t.Error("une erreur de validation était attendue")
		}
	}
}

func TestCalculateSessionLoad(t *testing.T) {
	tests := []struct {
		duration float64
		rpe      float64
		want     float64
	}{
		{120, 8, 960},
		{60, 5, 300},
		{30, 3, 90},
		{45, 7.5, 337.5},
		{60, 0, 0},
	}

	for _, test := range tests {
		load, err := CalculateSessionLoad(
			test.duration,
			test.rpe,
		)

		if err != nil || load != test.want {
			t.Errorf(
				"charge obtenue : %f, attendue : %f",
				load,
				test.want,
			)
		}
	}
}

func TestSessionLoadValidation(t *testing.T) {
	tests := [][2]float64{
		{0, 5},
		{-1, 5},
		{1441, 5},
		{60, -1},
		{60, 11},
		{math.NaN(), 5},
		{60, math.Inf(1)},
	}

	for _, test := range tests {
		if _, err := CalculateSessionLoad(
			test[0],
			test[1],
		); err == nil {
			t.Error("une erreur était attendue")
		}
	}
}

func TestAnalyzeTrainingLoad(t *testing.T) {
	reference := time.Date(
		2026, 11, 26,
		12, 0, 0, 0,
		time.UTC,
	)

	sessions := []TrainingSession{
		{"2026-11-26", 120, 8},
		{"2026-11-24", 60, 5},
		{"2026-11-20", 60, 5},
		{"2026-11-19", 60, 5},
		{"2026-10-30", 60, 5},
		{"2026-10-29", 60, 5},
	}

	result, err := AnalyzeTrainingLoad(
		sessions,
		reference,
	)

	if err != nil {
		t.Fatal(err)
	}

	if result.Load7Days != 1560 {
		t.Error("charge sur 7 jours incorrecte")
	}

	if result.Load28Days != 2160 {
		t.Error("charge sur 28 jours incorrecte")
	}

	if result.WeeklyAverage != 540 {
		t.Error("moyenne hebdomadaire incorrecte")
	}

	if result.Sessions7Days != 3 ||
		result.Sessions28Days != 5 {
		t.Error("nombre de séances incorrect")
	}
}

func TestTrainingLoadValidation(t *testing.T) {
	reference := time.Date(
		2026, 11, 26,
		12, 0, 0, 0,
		time.UTC,
	)

	tests := []TrainingSession{
		{"invalid", 60, 5},
		{"2026-11-27", 60, 5},
		{"2026-11-26", -1, 5},
		{"2026-11-26", 60, 11},
	}

	for _, session := range tests {
		if _, err := AnalyzeTrainingLoad(
			[]TrainingSession{session},
			reference,
		); err == nil {
			t.Error("une erreur était attendue")
		}
	}
}

func TestEmptyTrainingHistory(t *testing.T) {
	result, err := AnalyzeTrainingLoad(nil, time.Now())

	if err != nil {
		t.Fatal(err)
	}

	if result != (TrainingLoadResult{}) {
		t.Error("historique vide incorrect")
	}
}
