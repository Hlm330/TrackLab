package analysis

import (
	"math"
	"testing"
)

func TestEventDiscipline(t *testing.T) {
	tests := []struct {
		event      Event
		discipline Discipline
		regulated  bool
	}{
		{Event100m, DisciplineSprint, true},
		{Event200m, DisciplineSprint, true},
		{Event400m, DisciplineSprint, false},
		{Event100mHurdles, DisciplineHurdles, true},
		{Event110mHurdles, DisciplineHurdles, true},
		{Event400mHurdles, DisciplineHurdles, false},
		{EventLongJump, DisciplineJumps, true},
		{EventTripleJump, DisciplineJumps, true},
		{EventHighJump, DisciplineJumps, false},
		{EventPoleVault, DisciplineJumps, false},
		{EventJavelin, DisciplineThrows, false},
		{EventDiscus, DisciplineThrows, false},
		{EventShotPut, DisciplineThrows, false},
		{EventHammer, DisciplineThrows, false},
		{Event800m, DisciplineMiddle, false},
		{Event1500m, DisciplineMiddle, false},
		{Event3000m, DisciplineMiddle, false},
		{Event5000m, DisciplineLong, false},
		{Event10000m, DisciplineLong, false},
		{EventHalfMarathon, DisciplineLong, false},
		{EventMarathon, DisciplineLong, false},
	}

	for _, test := range tests {
		t.Run(string(test.event), func(t *testing.T) {
			discipline, err := test.event.Discipline()

			if err != nil {
				t.Fatal(err)
			}

			if discipline != test.discipline {
				t.Error("discipline incorrecte")
			}

			if test.event.IsWindRegulated() != test.regulated {
				t.Error("application réglementaire incorrecte")
			}
		})
	}
}

func TestWindRegulation(t *testing.T) {
	tests := []struct {
		event      Event
		wind       float64
		applicable bool
		exceeded   bool
	}{
		{Event100m, 1.9, true, false},
		{Event100m, 2.0, true, false},
		{Event100m, 2.1, true, true},
		{Event110mHurdles, 2.0, true, false},
		{Event110mHurdles, 2.1, true, true},
		{EventLongJump, 3, true, true},
		{EventTripleJump, 3, true, true},
		{Event200m, 3, true, true},
		{Event100mHurdles, 3, true, true},
		{Event400m, 5, false, false},
		{Event400mHurdles, 5, false, false},
		{EventPoleVault, 5, false, false},
		{EventHighJump, 5, false, false},
		{EventJavelin, 5, false, false},
		{Event100m, -5, true, false},
	}

	for _, test := range tests {
		direction := 180.0

		if test.wind < 0 {
			direction = 0
		}

		result, err := AnalyzeWindImpact(WindImpactInput{
			Event:     test.event,
			Objective: ObjectiveCompetition,
			Wind: WindInput{
				Speed:          math.Abs(test.wind) * 3.6,
				Direction:      direction,
				TrackDirection: 0,
			},
		})

		if err != nil {
			t.Fatal(err)
		}

		if result.Regulation.Applicable != test.applicable {
			t.Errorf("%s : applicabilité incorrecte", test.event)
		}

		if result.Regulation.Exceeded != test.exceeded {
			t.Errorf("%s : dépassement incorrect", test.event)
		}

		if !result.Regulation.Indicative {
			t.Error("le résultat doit rester indicatif")
		}
	}
}

func TestWindImpactScores(t *testing.T) {
	tests := []struct {
		name      string
		event     Event
		objective SessionObjective
		speed     float64
		direction float64
		wantScore int
	}{
		{"vent nul", Event100m, ObjectivePerformance, 0, 0, 100},
		{"vent négligeable", Event100m, ObjectivePerformance, 0.5, 180, 100},
		{"sprint face", Event100m, ObjectivePerformance, 3, 0, 80},
		{"sprint dos", Event100m, ObjectivePerformance, 3, 180, 100},
		{"sprint dos fort", Event100m, ObjectivePerformance, 5, 180, 94},
		{"haies technique face", Event110mHurdles, ObjectiveTechnique, 3, 0, 80},
		{"haies technique dos", Event110mHurdles, ObjectiveTechnique, 3, 180, 80},
		{"résistance face", Event100m, ObjectiveResistance, 3, 0, 100},
		{"résistance face fort", Event100m, ObjectiveResistance, 5, 0, 84},
		{"résistance dos", Event100m, ObjectiveResistance, 3, 180, 87},
		{"compétition face", Event100m, ObjectiveCompetition, 3, 0, 85},
		{"compétition vent légal", Event100m, ObjectiveCompetition, 2, 180, 100},
		{"compétition vent excessif", Event100m, ObjectiveCompetition, 3, 180, 65},
		{"compétition dos fort", Event100m, ObjectiveCompetition, 5, 180, 59},
		{"400m sans seuil", Event400m, ObjectiveCompetition, 3, 180, 100},
		{"récupération", Event1500m, ObjectiveRecovery, 3, 0, 87},
		{"javelot face", EventJavelin, ObjectivePerformance, 3, 0, 90},
		{"javelot dos", EventJavelin, ObjectivePerformance, 3, 180, 90},
		{"disque", EventDiscus, ObjectiveTechnique, 3, 180, 90},
		{"poids", EventShotPut, ObjectivePerformance, 3, 0, 90},
		{"sauts face", EventLongJump, ObjectivePerformance, 3, 0, 80},
		{"sauts dos", EventLongJump, ObjectivePerformance, 3, 180, 100},
		{"haies performance face", Event110mHurdles, ObjectivePerformance, 3, 0, 80},
		{"haies performance dos", Event110mHurdles, ObjectivePerformance, 3, 180, 100},
		{"fond face", Event5000m, ObjectivePerformance, 3, 0, 80},
		{"fond dos", Event5000m, ObjectivePerformance, 3, 180, 100},
		{"haies latéral", Event110mHurdles, ObjectiveTechnique, 3, 90, 75},
		{"sprint latéral", Event100m, ObjectivePerformance, 3, 90, 85},
		{"fond latéral", Event5000m, ObjectivePerformance, 3, 90, 90},
		{"vent fort", Event100m, ObjectivePerformance, 8, 180, 66},
		{"vent extrême", Event100m, ObjectivePerformance, 50, 0, 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := AnalyzeWindImpact(WindImpactInput{
				Event:     test.event,
				Objective: test.objective,
				Wind: WindInput{
					Speed:          test.speed * 3.6,
					Direction:      test.direction,
					TrackDirection: 0,
				},
			})

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

			if result.PerformanceEffect == "" ||
				result.TechnicalEffect == "" {
				t.Error("interprétation sportive manquante")
			}
		})
	}
}

func TestWindIntensity(t *testing.T) {
	tests := []struct {
		speed float64
		want  string
	}{
		{0, "negligible"},
		{0.5, "negligible"},
		{0.6, "low"},
		{1, "low"},
		{1.1, "moderate"},
		{2, "moderate"},
		{2.1, "significant"},
		{4, "significant"},
		{4.1, "strong"},
		{6, "strong"},
		{6.1, "very_strong"},
		{-3, "significant"},
	}

	for _, test := range tests {
		if got := windIntensity(test.speed); got != test.want {
			t.Errorf(
				"intensité obtenue : %s, attendue : %s",
				got,
				test.want,
			)
		}
	}
}

func TestWindImpactValidation(t *testing.T) {
	tests := []WindImpactInput{
		{
			Event:     "unknown",
			Objective: ObjectivePerformance,
		},
		{
			Event:     Event100m,
			Objective: "unknown",
		},
		{
			Event:     Event100m,
			Objective: ObjectivePerformance,
			Wind:      WindInput{Speed: -1},
		},
	}

	for _, input := range tests {
		if _, err := AnalyzeWindImpact(input); err == nil {
			t.Error("une erreur de validation était attendue")
		}
	}
}
