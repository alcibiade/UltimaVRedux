package references

func (m MissileType) GetStringName() string {
	switch m {
	case MissileNone:
		return "None"
	case MissileArrow:
		return "Arrow"
	case MissileCannonBall:
		return "Cannon Ball"
	case MissileAxe:
		return "Axe"
	case MissileRed:
		return "Red"
	case MissileBlue:
		return "Blue"
	case MissileGreen:
		return "Green"
	case MissileViolet:
		return "Violet"
	case MissileRock:
		return "Rock"
	default:
		return "Unknown"
	}
}
