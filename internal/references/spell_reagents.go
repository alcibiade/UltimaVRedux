package references

// GetSpellReagents returns the reagent bitmask required for a given spell.
// Returns 0 if the spell is unknown or has no reagents.
// Reagents:
// SulfurAsh(0), Ginseng(1), Garlic(2), SpiderSilk(3)
// BloodMoss(4), BlackPearl(5), NightShade(6), MandrakeRoot(7)
func GetSpellReagents(spell Spell) uint16 {
	switch spell {
	// Circle 1
	case AnNox: // Ginseng, Garlic
		return (1 << Ginseng) | (1 << Garlic)
	case AnZu: // Ginseng, Garlic
		return (1 << Ginseng) | (1 << Garlic)
	case GravPor: // SulfurAsh, BlackPearl
		return (1 << SulfurAsh) | (1 << BlackPearl)
	case Mani: // Ginseng, SpiderSilk
		return (1 << Ginseng) | (1 << SpiderSilk)
	case AnYlem: // Garlic, BloodMoss
		return (1 << Garlic) | (1 << BloodMoss)
	case InLor: // SulfurAsh
		return (1 << SulfurAsh)

	// Circle 2
	case AnSanct: // SulfurAsh, BloodMoss
		return (1 << SulfurAsh) | (1 << BloodMoss)
	case AnXenCorp: // Garlic, SulfurAsh
		return (1 << Garlic) | (1 << SulfurAsh)
	case InWis: // NightShade
		return (1 << NightShade)
	case KalXen: // SpiderSilk, MandrakeRoot
		return (1 << SpiderSilk) | (1 << MandrakeRoot)
	case RelHur: // SulfurAsh, BloodMoss
		return (1 << SulfurAsh) | (1 << BloodMoss)
	case InXenMani: // Ginseng, Garlic, MandrakeRoot
		return (1 << Ginseng) | (1 << Garlic) | (1 << MandrakeRoot)

	// Circle 3
	case InFlamGrav: // SpiderSilk, BlackPearl, SulfurAsh
		return (1 << SpiderSilk) | (1 << BlackPearl) | (1 << SulfurAsh)
	case InNoxGrav: // SpiderSilk, BlackPearl, NightShade
		return (1 << SpiderSilk) | (1 << BlackPearl) | (1 << NightShade)
	case InZuGrav: // SpiderSilk, BlackPearl, Ginseng
		return (1 << SpiderSilk) | (1 << BlackPearl) | (1 << Ginseng)
	case InPor: // SpiderSilk, BloodMoss
		return (1 << SpiderSilk) | (1 << BloodMoss)
	case VasFlam: // SulfurAsh, BlackPearl
		return (1 << SulfurAsh) | (1 << BlackPearl)
	case VasLor: // SulfurAsh, MandrakeRoot
		return (1 << SulfurAsh) | (1 << MandrakeRoot)

	// Circle 4
	case AnGrav: // SulfurAsh, BlackPearl
		return (1 << SulfurAsh) | (1 << BlackPearl)
	case InSanct: // SulfurAsh, Ginseng, Garlic
		return (1 << SulfurAsh) | (1 << Ginseng) | (1 << Garlic)
	case InSanctGrav: // SpiderSilk, BlackPearl, MandrakeRoot
		return (1 << SpiderSilk) | (1 << BlackPearl) | (1 << MandrakeRoot)
	case UusPor: // SpiderSilk, BloodMoss
		return (1 << SpiderSilk) | (1 << BloodMoss)
	case DesPor: // SpiderSilk, BloodMoss
		return (1 << SpiderSilk) | (1 << BloodMoss)
	case WisQuas: // SpiderSilk, NightShade
		return (1 << SpiderSilk) | (1 << NightShade)

	// Circle 5
	case AnExPor: // SulfurAsh, Garlic, BloodMoss
		return (1 << SulfurAsh) | (1 << Garlic) | (1 << BloodMoss)
	case InExPor: // SulfurAsh, BloodMoss
		return (1 << SulfurAsh) | (1 << BloodMoss)
	case InBetXen: // SulfurAsh, SpiderSilk, BloodMoss
		return (1 << SulfurAsh) | (1 << SpiderSilk) | (1 << BloodMoss)
	case InZu: // Ginseng, SpiderSilk, NightShade
		return (1 << Ginseng) | (1 << SpiderSilk) | (1 << NightShade)
	case RelTym: // SulfurAsh, BloodMoss, MandrakeRoot
		return (1 << SulfurAsh) | (1 << BloodMoss) | (1 << MandrakeRoot)
	case VasMani: // Ginseng, SpiderSilk, MandrakeRoot
		return (1 << Ginseng) | (1 << SpiderSilk) | (1 << MandrakeRoot)

	// Circle 6
	case AnXenEx: // SpiderSilk, BlackPearl, NightShade
		return (1 << SpiderSilk) | (1 << BlackPearl) | (1 << NightShade)
	case InAn: // SulfurAsh, Garlic, MandrakeRoot
		return (1 << SulfurAsh) | (1 << Garlic) | (1 << MandrakeRoot)
	case InVasPorYlem: // SulfurAsh, BloodMoss, MandrakeRoot
		return (1 << SulfurAsh) | (1 << BloodMoss) | (1 << MandrakeRoot)
	case QuasAnWis: // MandrakeRoot, NightShade
		return (1 << MandrakeRoot) | (1 << NightShade)
	case RelXenBet: // SulfurAsh, SpiderSilk, MandrakeRoot, NightShade
		return (1 << SulfurAsh) | (1 << SpiderSilk) | (1 << MandrakeRoot) | (1 << NightShade)
	case WisAnYlem: // SulfurAsh, MandrakeRoot
		return (1 << SulfurAsh) | (1 << MandrakeRoot)

	// Circle 7
	case InNoxHur: // SulfurAsh, BloodMoss, NightShade
		return (1 << SulfurAsh) | (1 << BloodMoss) | (1 << NightShade)
	case InQuasCorp: // Garlic, MandrakeRoot, NightShade
		return (1 << Garlic) | (1 << MandrakeRoot) | (1 << NightShade)
	case InQuasWis: // MandrakeRoot, NightShade
		return (1 << MandrakeRoot) | (1 << NightShade)
	case InQuasXen: // SulfurAsh, Ginseng, SpiderSilk, BloodMoss, MandrakeRoot
		return (1 << SulfurAsh) | (1 << Ginseng) | (1 << SpiderSilk) | (1 << BloodMoss) | (1 << MandrakeRoot)
	case SanctLor: // BloodMoss, MandrakeRoot, NightShade
		return (1 << BloodMoss) | (1 << MandrakeRoot) | (1 << NightShade)
	case XenCorp: // BlackPearl, NightShade
		return (1 << BlackPearl) | (1 << NightShade)

	// Circle 8
	case AnTym: // Garlic, BloodMoss, MandrakeRoot
		return (1 << Garlic) | (1 << BloodMoss) | (1 << MandrakeRoot)
	case InFlamHur: // SulfurAsh, BloodMoss, MandrakeRoot
		return (1 << SulfurAsh) | (1 << BloodMoss) | (1 << MandrakeRoot)
	case InManiCorp: // SulfurAsh, Garlic, Ginseng, SpiderSilk, BloodMoss, MandrakeRoot
		return (1 << SulfurAsh) | (1 << Garlic) | (1 << Ginseng) | (1 << SpiderSilk) | (1 << BloodMoss) | (1 << MandrakeRoot)
	case InVasGravCorp: // SulfurAsh, MandrakeRoot, NightShade
		return (1 << SulfurAsh) | (1 << MandrakeRoot) | (1 << NightShade)
	case KalXenCorp: // Garlic, SpiderSilk, BloodMoss, MandrakeRoot
		return (1 << Garlic) | (1 << SpiderSilk) | (1 << BloodMoss) | (1 << MandrakeRoot)
	case VasRelPor: // SulfurAsh, BlackPearl, MandrakeRoot
		return (1 << SulfurAsh) | (1 << BlackPearl) | (1 << MandrakeRoot)

	// Nox (Special/Monster?)
	case Nox: // Probably Nightshade + Garlic?
		// Standard Nox spell isn't usually mixable by players in U5 (An Nox is cure), but if it's there...
		// U5 doesn't let players mix "Nox".
		return 0

	case InvalidSpell:
	}

	return 0
}
