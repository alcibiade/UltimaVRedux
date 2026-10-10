package debugging

// DurationEffectDescription supplies inspector-only explanations. Effect identity,
// duration and display names come from the authoritative gameplay status.
func DurationEffectDescription(code string) string {
	switch code {
	case "L":
		return "Magical illumination active"
	case "P":
		return "+3 defense bonus active"
	case "N":
		return "Combat magic and magical enemy abilities suppressed for both sides"
	case "T":
		return "Time is frozen"
	case "Q":
		return "Monsters lose half their combat actions; travel takes half time"
	case "C":
		return "Monsters attack randomly"
	default:
		return ""
	}
}
