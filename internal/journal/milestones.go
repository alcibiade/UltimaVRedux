package journal

// ItemMilestoneTitle is the finite, disclosure-safe acquisition catalog.
func ItemMilestoneTitle(key string) string {
	switch key {
	case "crown":
		return "Crown of Lord British"
	case "sceptre":
		return "Sceptre of Lord British"
	case "amulet":
		return "Amulet of Lord British"
	case "shard.falsehood":
		return "Shard of Falsehood"
	case "shard.hatred":
		return "Shard of Hatred"
	case "shard.cowardice":
		return "Shard of Cowardice"
	case "sandalwood-box":
		return "Sandalwood box"
	case "minoc-odd-keys":
		return "Odd keys in Minoc"
	default:
		return ""
	}
}
