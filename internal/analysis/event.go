package analysis

import "errors"

// Event identifie une épreuve athlétique précise.
type Event string

const (
	Event100m         Event = "100m"
	Event200m         Event = "200m"
	Event400m         Event = "400m"
	Event100mHurdles  Event = "100m_hurdles"
	Event110mHurdles  Event = "110m_hurdles"
	Event400mHurdles  Event = "400m_hurdles"
	EventLongJump     Event = "long_jump"
	EventTripleJump   Event = "triple_jump"
	EventHighJump     Event = "high_jump"
	EventPoleVault    Event = "pole_vault"
	EventJavelin      Event = "javelin"
	EventDiscus       Event = "discus"
	EventShotPut      Event = "shot_put"
	EventHammer       Event = "hammer"
	Event800m         Event = "800m"
	Event1500m        Event = "1500m"
	Event3000m        Event = "3000m"
	Event5000m        Event = "5000m"
	Event10000m       Event = "10000m"
	EventHalfMarathon Event = "half_marathon"
	EventMarathon     Event = "marathon"
)

// Discipline retourne la famille sportive de l'épreuve.
func (e Event) Discipline() (Discipline, error) {
	switch e {
	case Event100m, Event200m, Event400m:
		return DisciplineSprint, nil

	case Event100mHurdles, Event110mHurdles, Event400mHurdles:
		return DisciplineHurdles, nil

	case EventLongJump, EventTripleJump, EventHighJump, EventPoleVault:
		return DisciplineJumps, nil

	case EventJavelin, EventDiscus, EventShotPut, EventHammer:
		return DisciplineThrows, nil

	case Event800m, Event1500m, Event3000m:
		return DisciplineMiddle, nil

	case Event5000m, Event10000m, EventHalfMarathon, EventMarathon:
		return DisciplineLong, nil

	default:
		return "", errors.New("épreuve inconnue")
	}
}

// IsWindRegulated indique si l'épreuve fait partie
// des épreuves individuelles extérieures gérées par TrackLab
// soumises au seuil de vent favorable de +2 m/s.
func (e Event) IsWindRegulated() bool {
	switch e {
	case Event100m,
		Event200m,
		Event100mHurdles,
		Event110mHurdles,
		EventLongJump,
		EventTripleJump:
		return true

	default:
		return false
	}
}

// SessionObjective identifie l'objectif de l'entraînement.
type SessionObjective string

const (
	ObjectivePerformance SessionObjective = "performance"
	ObjectiveTechnique   SessionObjective = "technique"
	ObjectiveResistance  SessionObjective = "resistance"
	ObjectiveCompetition SessionObjective = "competition"
	ObjectiveRecovery    SessionObjective = "recovery"
)

func (o SessionObjective) Validate() error {
	switch o {
	case ObjectivePerformance,
		ObjectiveTechnique,
		ObjectiveResistance,
		ObjectiveCompetition,
		ObjectiveRecovery:
		return nil

	default:
		return errors.New("objectif de séance inconnu")
	}
}
