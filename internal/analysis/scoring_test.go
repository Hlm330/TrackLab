package analysis

import (
	"testing"
)

func optimalScoringInput() ScoringInput {
	return ScoringInput{
		Event:     Event100m,
		Objective: ObjectivePerformance,
		Weather:   optimalWeather(),
		Wind:      WindInput{0, 0, 0},
		Training: TrainingInput{
			SleepHours:        trainingValue(9),
			SleepScore:        trainingValue(90),
			BodyBattery:       trainingValue(90),
			RestingHeartRate:  trainingValue(55),
			BaselineHeartRate: trainingValue(55),
			HRV:               trainingValue(70),
			BaselineHRV:       trainingValue(70),
			Fatigue:           trainingValue(2),
		},
	}
}

func TestObjectiveWeights(t *testing.T) {
	tests := []struct {
		objective SessionObjective
		want      ScoreWeights
	}{
		{ObjectivePerformance, ScoreWeights{30, 30, 40}},
		{ObjectiveTechnique, ScoreWeights{30, 45, 25}},
		{ObjectiveResistance, ScoreWeights{35, 25, 40}},
		{ObjectiveCompetition, ScoreWeights{25, 40, 35}},
		{ObjectiveRecovery, ScoreWeights{35, 15, 50}},
		{"unknown", ScoreWeights{}},
	}

	for _, test := range tests {
		if got := objectiveWeights(test.objective); got != test.want {
			t.Errorf("pondération incorrecte : %+v", got)
		}
	}
}

func TestEffectiveWeights(t *testing.T) {
	tests := []struct {
		coverage int
		want     ScoreWeights
	}{
		{100, ScoreWeights{0.3, 0.3, 0.4}},
		{0, ScoreWeights{0.5, 0.5, 0}},
		{50, ScoreWeights{0.375, 0.375, 0.25}},
	}

	for _, test := range tests {
		got := effectiveWeights(
			ScoreWeights{30, 30, 40},
			test.coverage,
		)

		if !almostEqual(got.Weather, test.want.Weather) ||
			!almostEqual(got.Wind, test.want.Wind) ||
			!almostEqual(got.Training, test.want.Training) {
			t.Errorf("pondération effective incorrecte : %+v", got)
		}

		if !almostEqual(
			got.Weather+got.Wind+got.Training,
			1,
		) {
			t.Error("la somme des pondérations doit être égale à 1")
		}
	}

	if got := effectiveWeights(ScoreWeights{}, 0); got != (ScoreWeights{}) {
		t.Error("pondération vide incorrecte")
	}
}

func TestAnalyzeSession(t *testing.T) {
	tests := []struct {
		name          string
		modify        func(*ScoringInput)
		wantScore     int
		wantCapped    bool
		wantEnvOnly   bool
		wantLimited   bool
		wantHighAlert bool
	}{
		{
			"conditions optimales",
			func(i *ScoringInput) {},
			100, false, false, false, false,
		},
		{
			"sans données personnelles",
			func(i *ScoringInput) {
				i.Training = TrainingInput{}
			},
			100, false, true, true, false,
		},
		{
			"données personnelles partielles",
			func(i *ScoringInput) {
				i.Training = TrainingInput{
					Fatigue: trainingValue(2),
				}
			},
			100, false, false, true, false,
		},
		{
			"météo dégradée",
			func(i *ScoringInput) {
				i.Weather.Precipitation = 1
			},
			97, false, false, false, false,
		},
		{
			"vent de face",
			func(i *ScoringInput) {
				i.Wind = WindInput{10.8, 0, 0}
			},
			94, false, false, false, false,
		},
		{
			"fatigue importante",
			func(i *ScoringInput) {
				i.Training.Fatigue = trainingValue(8)
			},
			90, false, false, false, false,
		},
		{
			"chaleur importante",
			func(i *ScoringInput) {
				i.Weather.ApparentTemperature = 35
			},
			49, true, false, false, true,
		},
		{
			"pollution importante",
			func(i *ScoringInput) {
				i.Weather.AirQualityIndex = 120
			},
			49, true, false, false, true,
		},
		{
			"rafales importantes",
			func(i *ScoringInput) {
				i.Weather.WindGusts = 65
			},
			49, true, false, false, true,
		},
		{
			"vent global important",
			func(i *ScoringInput) {
				i.Wind = WindInput{28.8, 180, 0}
			},
			49, true, false, false, true,
		},
		{
			"compétition vent excessif",
			func(i *ScoringInput) {
				i.Objective = ObjectiveCompetition
				i.Wind = WindInput{10.8, 180, 0}
			},
			69, true, false, false, true,
		},
		{
			"compétition seuil exact",
			func(i *ScoringInput) {
				i.Objective = ObjectiveCompetition
				i.Wind = WindInput{7.2, 180, 0}
			},
			100, false, false, false, false,
		},
		{
			"400m sans seuil réglementaire",
			func(i *ScoringInput) {
				i.Event = Event400m
				i.Objective = ObjectiveCompetition
				i.Wind = WindInput{10.8, 180, 0}
			},
			100, false, false, false, false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := optimalScoringInput()
			test.modify(&input)

			result, err := AnalyzeSession(input)

			if err != nil {
				t.Fatal(err)
			}

			if result.Score != test.wantScore {
				t.Errorf(
					"score obtenu : %d, attendu : %d",
					result.Score,
					test.wantScore,
				)
			}

			if result.ScoreCapped != test.wantCapped {
				t.Error("plafonnement incorrect")
			}

			if result.EnvironmentalOnly != test.wantEnvOnly {
				t.Error("mode environnemental incorrect")
			}

			if result.LimitedData != test.wantLimited {
				t.Error("indicateur de données limitées incorrect")
			}

			if result.HighSeverityAlert != test.wantHighAlert {
				t.Error("indicateur d'alerte incorrect")
			}

			if result.Recommendation == "" {
				t.Error("recommandation manquante")
			}
		})
	}
}

func TestScoringDisciplineConsistency(t *testing.T) {
	input := optimalScoringInput()

	input.Event = Event110mHurdles
	input.Weather.Discipline = DisciplineLong

	result, err := AnalyzeSession(input)

	if err != nil {
		t.Fatal(err)
	}

	if result.Discipline != DisciplineHurdles ||
		result.Weather.Discipline != DisciplineHurdles ||
		result.Wind.Discipline != DisciplineHurdles {
		t.Error("incohérence entre l'épreuve et la discipline")
	}
}

func TestScoringValidation(t *testing.T) {
	tests := []func(*ScoringInput){
		func(i *ScoringInput) {
			i.Event = "unknown"
		},
		func(i *ScoringInput) {
			i.Objective = "unknown"
		},
		func(i *ScoringInput) {
			i.Weather.Humidity = 150
		},
		func(i *ScoringInput) {
			i.Wind.Speed = -1
		},
		func(i *ScoringInput) {
			i.Training.Fatigue = trainingValue(20)
		},
	}

	for _, modify := range tests {
		input := optimalScoringInput()
		modify(&input)

		if _, err := AnalyzeSession(input); err == nil {
			t.Error("une erreur de validation était attendue")
		}
	}
}

func TestSessionRecommendation(t *testing.T) {
	scores := []int{100, 80, 60, 40, 20}

	for _, score := range scores {
		result := ScoringResult{
			Score: score,
		}

		if sessionRecommendation(result) == "" {
			t.Errorf("recommandation absente pour %d", score)
		}
	}

	result := ScoringResult{
		Score:             90,
		HighSeverityAlert: true,
	}

	if sessionRecommendation(result) == "" {
		t.Error("recommandation d'alerte absente")
	}
}
