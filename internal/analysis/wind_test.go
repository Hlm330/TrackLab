package analysis

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-8
}

func TestAnalyzeWind(t *testing.T) {
	tests := []struct {
		name         string
		input        WindInput
		longitudinal float64
		lateral      float64
		longType     string
		latType      string
		exceedsLimit bool
	}{
		{
			"vent nul",
			WindInput{0, 90, 180},
			0, 0,
			"neutral", "neutral", false,
		},
		{
			"vent arrière",
			WindInput{18, 270, 90},
			5, 0,
			"tailwind", "neutral", true,
		},
		{
			"vent de face",
			WindInput{18, 90, 90},
			-5, 0,
			"headwind", "neutral", false,
		},
		{
			"vent venant de gauche",
			WindInput{18, 0, 90},
			0, 5,
			"neutral", "from_left", false,
		},
		{
			"vent venant de droite",
			WindInput{18, 180, 90},
			0, -5,
			"neutral", "from_right", false,
		},
		{
			"vent arrière diagonal gauche",
			WindInput{18, 315, 90},
			5 / math.Sqrt2, 5 / math.Sqrt2,
			"tailwind", "from_left", true,
		},
		{
			"vent arrière diagonal droite",
			WindInput{18, 225, 90},
			5 / math.Sqrt2, -5 / math.Sqrt2,
			"tailwind", "from_right", true,
		},
		{
			"vent de face diagonal gauche",
			WindInput{18, 45, 90},
			-5 / math.Sqrt2, 5 / math.Sqrt2,
			"headwind", "from_left", false,
		},
		{
			"vent de face diagonal droite",
			WindInput{18, 135, 90},
			-5 / math.Sqrt2, -5 / math.Sqrt2,
			"headwind", "from_right", false,
		},
		{
			"seuil réglementaire exact",
			WindInput{7.2, 270, 90},
			2, 0,
			"tailwind", "neutral", false,
		},
		{
			"seuil réglementaire dépassé",
			WindInput{7.56, 270, 90},
			2.1, 0,
			"tailwind", "neutral", true,
		},
		{
			"direction 360 degrés",
			WindInput{18, 360, 0},
			-5, 0,
			"headwind", "neutral", false,
		},
		{
			"orientation 360 degrés",
			WindInput{18, 180, 360},
			5, 0,
			"tailwind", "neutral", true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := AnalyzeWind(test.input)

			if err != nil {
				t.Fatal(err)
			}

			if !almostEqual(
				result.Longitudinal,
				test.longitudinal,
			) {
				t.Errorf(
					"vent longitudinal : %f, attendu : %f",
					result.Longitudinal,
					test.longitudinal,
				)
			}

			if !almostEqual(result.Lateral, test.lateral) {
				t.Errorf(
					"vent latéral : %f, attendu : %f",
					result.Lateral,
					test.lateral,
				)
			}

			if result.LongitudinalType != test.longType {
				t.Errorf(
					"type longitudinal incorrect : %s",
					result.LongitudinalType,
				)
			}

			if result.LateralType != test.latType {
				t.Errorf(
					"type latéral incorrect : %s",
					result.LateralType,
				)
			}

			if result.ExceedsTailwindLimit != test.exceedsLimit {
				t.Error("indicateur réglementaire incorrect")
			}

			// Vérifie la conservation de la norme du vecteur.
			magnitude := math.Hypot(
				result.Longitudinal,
				result.Lateral,
			)

			if !almostEqual(magnitude, result.Speed) {
				t.Error("norme du vecteur non conservée")
			}

			if !almostEqual(
				result.Tailwind,
				max(0, test.longitudinal),
			) {
				t.Error("vent arrière incorrect")
			}

			if !almostEqual(
				result.Headwind,
				max(0, -test.longitudinal),
			) {
				t.Error("vent de face incorrect")
			}

			if !almostEqual(
				result.Crosswind,
				math.Abs(test.lateral),
			) {
				t.Error("vent transversal incorrect")
			}
		})
	}
}

func TestOptimalRunningDirection(t *testing.T) {
	tests := []struct {
		input    WindInput
		optimal  float64
		opposite float64
	}{
		{WindInput{18, 0, 90}, 180, 0},
		{WindInput{18, 90, 0}, 270, 90},
		{WindInput{18, 180, 0}, 0, 180},
		{WindInput{18, 270, 0}, 90, 270},
		{WindInput{18, 360, 0}, 180, 0},
		{WindInput{0, 180, 45}, 45, 225},
	}

	for _, test := range tests {
		result, err := AnalyzeWind(test.input)

		if err != nil {
			t.Fatal(err)
		}

		if !almostEqual(
			result.OptimalRunningDirection,
			test.optimal,
		) {
			t.Error("orientation optimale incorrecte")
		}

		if !almostEqual(
			result.OppositeRunningDirection,
			test.opposite,
		) {
			t.Error("orientation opposée incorrecte")
		}
	}
}

func TestWindValidation(t *testing.T) {
	tests := []struct {
		name  string
		input WindInput
	}{
		{"vitesse négative", WindInput{-1, 0, 0}},
		{"direction négative", WindInput{10, -1, 0}},
		{"direction excessive", WindInput{10, 361, 0}},
		{"orientation négative", WindInput{10, 0, -1}},
		{"orientation excessive", WindInput{10, 0, 361}},
		{"vitesse NaN", WindInput{math.NaN(), 0, 0}},
		{"direction infinie", WindInput{10, math.Inf(1), 0}},
		{"orientation NaN", WindInput{10, 0, math.NaN()}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := AnalyzeWind(test.input); err == nil {
				t.Error("une erreur était attendue")
			}
		})
	}
}

func TestNormalizeAngle(t *testing.T) {
	tests := []struct {
		input float64
		want  float64
	}{
		{0, 0},
		{90, 90},
		{360, 0},
		{720, 0},
		{-90, 270},
		{-360, 0},
		{-450, 270},
	}

	for _, test := range tests {
		if got := normalizeAngle(test.input); got != test.want {
			t.Errorf(
				"angle %f : obtenu %f, attendu %f",
				test.input,
				got,
				test.want,
			)
		}
	}
}
