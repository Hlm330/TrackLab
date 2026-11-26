package analysis

import "errors"

// Discipline identifie une famille de disciplines athlétiques.
type Discipline string

const (
	DisciplineSprint  Discipline = "sprint"
	DisciplineHurdles Discipline = "hurdles"
	DisciplineJumps   Discipline = "jumps"
	DisciplineThrows  Discipline = "throws"
	DisciplineMiddle  Discipline = "middle_distance"
	DisciplineLong    Discipline = "long_distance"
)

// ThermalProfile définit la plage thermique privilégiée
// pour une discipline.
type ThermalProfile struct {
	Minimum float64 `json:"minimum"`
	Maximum float64 `json:"maximum"`
}

// Validate vérifie que la discipline est prise en charge.
func (d Discipline) Validate() error {
	_, err := d.ThermalProfile()
	return err
}

// ThermalProfile retourne les seuils heuristiques de TrackLab.
func (d Discipline) ThermalProfile() (ThermalProfile, error) {
	switch d {
	case DisciplineSprint, DisciplineHurdles:
		return ThermalProfile{Minimum: 20, Maximum: 30}, nil

	case DisciplineJumps:
		return ThermalProfile{Minimum: 18, Maximum: 28}, nil

	case DisciplineThrows:
		return ThermalProfile{Minimum: 15, Maximum: 28}, nil

	case DisciplineMiddle:
		return ThermalProfile{Minimum: 12, Maximum: 22}, nil

	case DisciplineLong:
		return ThermalProfile{Minimum: 8, Maximum: 18}, nil

	default:
		return ThermalProfile{}, errors.New("discipline inconnue")
	}
}
