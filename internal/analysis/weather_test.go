package analysis

import (
	"math"
	"testing"
)

func optimalWeather() WeatherInput {
	return WeatherInput{
		Discipline:               DisciplineSprint,
		Temperature:              25,
		ApparentTemperature:      25,
		Humidity:                 50,
		Precipitation:            0,
		PrecipitationProbability: 0,
		UVIndex:                  3,
		WindGusts:                10,
		AirQualityIndex:          20,
	}
}

func TestThermalProfiles(t *testing.T) {
	tests := []struct {
		discipline Discipline
		minimum    float64
		maximum    float64
	}{
		{DisciplineSprint, 20, 30},
		{DisciplineHurdles, 20, 30},
		{DisciplineJumps, 18, 28},
		{DisciplineThrows, 15, 28},
		{DisciplineMiddle, 12, 22},
		{DisciplineLong, 8, 18},
	}

	for _, test := range tests {
		t.Run(string(test.discipline), func(t *testing.T) {
			profile, err := test.discipline.ThermalProfile()

			if err != nil {
				t.Fatal(err)
			}

			if profile.Minimum != test.minimum ||
				profile.Maximum != test.maximum {
				t.Errorf("profil thermique incorrect : %+v", profile)
			}
		})
	}
}

func TestThermalCompatibility(t *testing.T) {
	tests := []struct {
		name        string
		discipline  Discipline
		temperature float64
		wantScore   int
	}{
		{"sprint chaud", DisciplineSprint, 29, 100},
		{"haies chaudes", DisciplineHurdles, 30, 100},
		{"sprint limite basse", DisciplineSprint, 20, 100},
		{"sprint froid", DisciplineSprint, 5, 55},
		{"sprint gel", DisciplineSprint, 0, 40},
		{"sprint chaleur élevée", DisciplineSprint, 35, 60},
		{"sprint chaleur extrême", DisciplineSprint, 40, 40},
		{"sauts", DisciplineJumps, 28, 100},
		{"lancers", DisciplineThrows, 15, 100},
		{"demi-fond", DisciplineMiddle, 18, 100},
		{"fond frais", DisciplineLong, 12, 100},
		{"fond chaud", DisciplineLong, 29, 34},
		{"fond gel", DisciplineLong, 0, 75},
		{"froid extrême", DisciplineSprint, -30, 0},
		{"température décimale", DisciplineSprint, 30.5, 97},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := optimalWeather()
			input.Discipline = test.discipline
			input.Temperature = test.temperature
			input.ApparentTemperature = test.temperature

			result, err := AnalyzeWeather(input)

			if err != nil {
				t.Fatal(err)
			}

			if result.ThermalScore != test.wantScore {
				t.Errorf(
					"score thermique obtenu : %d, attendu : %d",
					result.ThermalScore,
					test.wantScore,
				)
			}

			if result.Score != test.wantScore {
				t.Errorf(
					"score global obtenu : %d, attendu : %d",
					result.Score,
					test.wantScore,
				)
			}
		})
	}
}

func TestAnalyzeWeather(t *testing.T) {
	tests := []struct {
		name      string
		modify    func(*WeatherInput)
		wantScore int
		wantRules int
	}{
		{
			"conditions optimales",
			func(input *WeatherInput) {},
			100, 0,
		},
		{
			"chaleur humide",
			func(input *WeatherInput) {
				input.Temperature = 29
				input.ApparentTemperature = 30
				input.Humidity = 80
			},
			90, 1,
		},
		{
			"fortes précipitations",
			func(input *WeatherInput) {
				input.Precipitation = 6
				input.PrecipitationProbability = 90
			},
			75, 1,
		},
		{
			"précipitations modérées",
			func(input *WeatherInput) {
				input.Precipitation = 1
			},
			90, 1,
		},
		{
			"risque de pluie",
			func(input *WeatherInput) {
				input.PrecipitationProbability = 80
			},
			95, 1,
		},
		{
			"UV très élevés",
			func(input *WeatherInput) {
				input.UVIndex = 8
			},
			85, 1,
		},
		{
			"UV élevés",
			func(input *WeatherInput) {
				input.UVIndex = 6
			},
			95, 1,
		},
		{
			"fortes rafales",
			func(input *WeatherInput) {
				input.WindGusts = 65
			},
			70, 1,
		},
		{
			"rafales modérées",
			func(input *WeatherInput) {
				input.WindGusts = 40
			},
			85, 1,
		},
		{
			"air très dégradé",
			func(input *WeatherInput) {
				input.AirQualityIndex = 120
			},
			60, 1,
		},
		{
			"air dégradé",
			func(input *WeatherInput) {
				input.AirQualityIndex = 80
			},
			80, 1,
		},
		{
			"air modéré",
			func(input *WeatherInput) {
				input.AirQualityIndex = 50
			},
			95, 1,
		},
		{
			"cumul des pénalités",
			func(input *WeatherInput) {
				input.ApparentTemperature = 40
				input.Precipitation = 10
				input.WindGusts = 80
				input.AirQualityIndex = 150
			},
			0, 5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := optimalWeather()
			test.modify(&input)

			result, err := AnalyzeWeather(input)

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

			if len(result.Rules) != test.wantRules {
				t.Errorf(
					"règles obtenues : %d, attendues : %d",
					len(result.Rules),
					test.wantRules,
				)
			}
		})
	}
}

func TestWeatherValidation(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*WeatherInput)
	}{
		{"discipline inconnue", func(i *WeatherInput) {
			i.Discipline = "unknown"
		}},
		{"discipline absente", func(i *WeatherInput) {
			i.Discipline = ""
		}},
		{"humidité excessive", func(i *WeatherInput) {
			i.Humidity = 110
		}},
		{"humidité négative", func(i *WeatherInput) {
			i.Humidity = -1
		}},
		{"probabilité excessive", func(i *WeatherInput) {
			i.PrecipitationProbability = 101
		}},
		{"probabilité négative", func(i *WeatherInput) {
			i.PrecipitationProbability = -1
		}},
		{"pluie négative", func(i *WeatherInput) {
			i.Precipitation = -1
		}},
		{"UV négatifs", func(i *WeatherInput) {
			i.UVIndex = -1
		}},
		{"rafales négatives", func(i *WeatherInput) {
			i.WindGusts = -1
		}},
		{"AQI négatif", func(i *WeatherInput) {
			i.AirQualityIndex = -1
		}},
		{"valeur NaN", func(i *WeatherInput) {
			i.Temperature = math.NaN()
		}},
		{"valeur infinie", func(i *WeatherInput) {
			i.WindGusts = math.Inf(1)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := optimalWeather()
			test.modify(&input)

			if _, err := AnalyzeWeather(input); err == nil {
				t.Error("une erreur de validation était attendue")
			}
		})
	}
}

func TestWeatherLevel(t *testing.T) {
	tests := []struct {
		score int
		level string
	}{
		{100, "excellent"},
		{85, "excellent"},
		{84, "good"},
		{70, "good"},
		{69, "moderate"},
		{50, "moderate"},
		{49, "poor"},
		{30, "poor"},
		{29, "very_poor"},
		{0, "very_poor"},
	}

	for _, test := range tests {
		if got := weatherLevel(test.score); got != test.level {
			t.Errorf(
				"score %d : niveau obtenu %s, attendu %s",
				test.score,
				got,
				test.level,
			)
		}
	}
}
