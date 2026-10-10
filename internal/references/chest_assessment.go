package references

// ChestAssessment is party-shared knowledge about one chest's lifetime, not its
// physical trap state. Zero is deliberately uninspected for new/legacy objects.
type ChestAssessment uint8

// Chest knowledge progresses only through inspection or actual disarming.
const (
	ChestUninspected ChestAssessment = iota
	ChestUncertain
	ChestSafe
	ChestTrapped
)

// IsValid reports whether the assessment is a defined semantic value.
func (assessment ChestAssessment) IsValid() bool {
	return assessment <= ChestTrapped
}

// ValidForChest rejects invented certainty inconsistent with physical state.
func (assessment ChestAssessment) ValidForChest(armed bool) bool {
	return assessment.IsValid() &&
		(assessment != ChestSafe || !armed) &&
		(assessment != ChestTrapped || armed)
}
