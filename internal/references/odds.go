package references

// TheOdds holds tunable chances for world generation.
type TheOdds struct {
	monsterGenerationRollMaximum int
}

// DefaultMonsterGenerationRollMaximum is the inclusive world-generation draw bound; ref:ENEMY-0004.
const DefaultMonsterGenerationRollMaximum = 30

// NewDefaultTheOdds returns the original-game odds.
func NewDefaultTheOdds() TheOdds {
	theOdds := TheOdds{
		monsterGenerationRollMaximum: DefaultMonsterGenerationRollMaximum,
	}

	return theOdds
}

// GetMonsterGenerationRollMaximum returns the inclusive generation-roll upper bound.
func (o *TheOdds) GetMonsterGenerationRollMaximum() int {
	return o.monsterGenerationRollMaximum
}

// SetMonsterGenerationRollMaximum changes the inclusive generation-roll upper bound.
func (o *TheOdds) SetMonsterGenerationRollMaximum(maximum int) {
	o.monsterGenerationRollMaximum = maximum
}
