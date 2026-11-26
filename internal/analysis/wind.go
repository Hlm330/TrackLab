package analysis

import (
	"errors"
	"math"
)

const (
	// Seuil réglementaire indicatif pour les épreuves concernées.
	LegalTailwindLimit = 2.0

	// Tolérance utilisée pour neutraliser les erreurs numériques.
	windEpsilon = 1e-9
)

// WindInput représente les conditions de vent et
// l'orientation de déplacement de l'athlète.
type WindInput struct {
	Speed          float64 `json:"speed"`
	Direction      float64 `json:"direction"`
	TrackDirection float64 `json:"track_direction"`
}

// WindResult contient les composantes vectorielles du vent.
// Toutes les vitesses calculées sont exprimées en m/s.
type WindResult struct {
	Speed                    float64 `json:"speed"`
	Longitudinal             float64 `json:"longitudinal"`
	Lateral                  float64 `json:"lateral"`
	Tailwind                 float64 `json:"tailwind"`
	Headwind                 float64 `json:"headwind"`
	Crosswind                float64 `json:"crosswind"`
	LongitudinalType         string  `json:"longitudinal_type"`
	LateralType              string  `json:"lateral_type"`
	RelativeAngle            float64 `json:"relative_angle"`
	OptimalRunningDirection  float64 `json:"optimal_running_direction"`
	OppositeRunningDirection float64 `json:"opposite_running_direction"`
	ExceedsTailwindLimit     bool    `json:"exceeds_tailwind_limit"`
}

// Validate vérifie la cohérence des données de vent.
// La vitesse d'entrée est exprimée en km/h.
// Les directions sont exprimées en degrés géographiques.
func (input WindInput) Validate() error {
	values := []float64{
		input.Speed,
		input.Direction,
		input.TrackDirection,
	}

	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return errors.New("donnée de vent non valide")
		}
	}

	if input.Speed < 0 {
		return errors.New("vitesse du vent négative")
	}

	if input.Direction < 0 || input.Direction > 360 {
		return errors.New("direction du vent hors limites")
	}

	if input.TrackDirection < 0 ||
		input.TrackDirection > 360 {
		return errors.New("orientation de course hors limites")
	}

	return nil
}

// normalizeAngle ramène un angle dans l'intervalle [0, 360[.
func normalizeAngle(angle float64) float64 {
	normalized := math.Mod(angle, 360)

	if normalized < 0 {
		normalized += 360
	}

	if normalized == 0 {
		return 0
	}

	return normalized
}

// normalizeWindValue élimine les résidus numériques
// issus des fonctions trigonométriques.
func normalizeWindValue(value float64) float64 {
	if math.Abs(value) < windEpsilon {
		return 0
	}

	return value
}

// AnalyzeWind décompose le vent selon les axes longitudinal
// et latéral de l'athlète.
//
// Convention longitudinale :
//
//	positive : vent arrière
//	négative : vent de face
//
// Convention latérale :
//
//	positive : vent provenant de la gauche
//	négative : vent provenant de la droite
func AnalyzeWind(input WindInput) (WindResult, error) {
	if err := input.Validate(); err != nil {
		return WindResult{}, err
	}

	speed := input.Speed / 3.6

	direction := normalizeAngle(input.Direction)
	trackDirection := normalizeAngle(input.TrackDirection)

	relativeAngle := normalizeAngle(
		direction - trackDirection,
	)

	radians := relativeAngle * math.Pi / 180

	longitudinal := normalizeWindValue(
		-speed * math.Cos(radians),
	)

	lateral := normalizeWindValue(
		-speed * math.Sin(radians),
	)

	result := WindResult{
		Speed:                    speed,
		Longitudinal:             longitudinal,
		Lateral:                  lateral,
		Tailwind:                 max(0, longitudinal),
		Headwind:                 max(0, -longitudinal),
		Crosswind:                math.Abs(lateral),
		LongitudinalType:         "neutral",
		LateralType:              "neutral",
		RelativeAngle:            relativeAngle,
		OptimalRunningDirection:  normalizeAngle(direction + 180),
		OppositeRunningDirection: direction,
		ExceedsTailwindLimit: longitudinal >
			LegalTailwindLimit+windEpsilon,
	}

	switch {
	case longitudinal > 0:
		result.LongitudinalType = "tailwind"

	case longitudinal < 0:
		result.LongitudinalType = "headwind"
	}

	switch {
	case lateral > 0:
		result.LateralType = "from_left"

	case lateral < 0:
		result.LateralType = "from_right"
	}

	// En l'absence de vent, aucune orientation
	// n'est plus favorable qu'une autre.
	if speed == 0 {
		result.OptimalRunningDirection = trackDirection
		result.OppositeRunningDirection = normalizeAngle(
			trackDirection + 180,
		)
	}

	return result, nil
}
