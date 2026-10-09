package references

// DialogNumber represents the type of dialog/conversation an NPC has.
// Values < 128 are regular NPC conversations indexed into TLK files.
// Values 0x81-0x88 are merchant/service providers with hardcoded logic.
type DialogNumber byte

const (
	// NoResponse indicates an NPC that doesn't respond to talk commands
	NoResponse DialogNumber = 0

	// Regular NPC dialog numbers are 1-127 and index into TLK files
	// The specific TLK file is determined by location type:
	// - CASTLE.TLK for castles
	// - TOWNE.TLK for towns
	// - DWELLING.TLK for dwellings
	// - KEEP.TLK for keeps

	// Special merchant and service provider dialog numbers (0x81-0x88)
	// These trigger special hardcoded merchant/shop logic instead of TLK scripts

	// MerchantThreshold is the boundary immediately before merchant dialog numbers
	MerchantThreshold DialogNumber = 128

	// DialogWeaponDealer sells weapons and armor (dialog 129)
	DialogWeaponDealer DialogNumber = 129

	// DialogBarkeeper sells food and drinks (dialog 130)
	DialogBarkeeper DialogNumber = 130

	// DialogHorseDealer sells horses (dialog 131)
	DialogHorseDealer DialogNumber = 131

	// DialogShipDealer sells ships and skiffs (dialog 132)
	DialogShipDealer DialogNumber = 132

	// DialogMagicDealer sells reagents and magic items (dialog 133)
	DialogMagicDealer DialogNumber = 133

	// DialogGuildMaster provides guild services (dialog 134)
	DialogGuildMaster DialogNumber = 134

	// DialogHealer provides healing services (dialog 135)
	DialogHealer DialogNumber = 135

	// DialogInnkeeper provides room and board (dialog 136)
	DialogInnkeeper DialogNumber = 136

	// Note: There may be additional special dialog numbers beyond 136
	// that trigger other special behaviors

	// DialogGuard harasses player (dialog 255)
	DialogGuard DialogNumber = 255
)

// IsMerchant returns true only for the eight production merchant families.
func (d DialogNumber) IsMerchant() bool {
	return d >= DialogWeaponDealer && d <= DialogInnkeeper
}

// IsRegularNPC returns true if this dialog number represents a regular TLK-based NPC
func (d DialogNumber) IsRegularNPC() bool {
	return d > NoResponse && d < MerchantThreshold
}

// IsSpecial returns true if this dialog number triggers special logic (merchant or guard)
func (d DialogNumber) IsSpecial() bool {
	return d >= MerchantThreshold
}

// GetMerchantType returns a human-readable string for the merchant type
func (d DialogNumber) GetMerchantType() string {
	switch d {
	case DialogWeaponDealer:
		return "Weapon Dealer"
	case DialogBarkeeper:
		return "Barkeeper"
	case DialogHorseDealer:
		return "Horse Dealer"
	case DialogShipDealer:
		return "Ship Dealer"
	case DialogMagicDealer:
		return "Magic Dealer"
	case DialogGuildMaster:
		return "Guild Master"
	case DialogHealer:
		return "Healer"
	case DialogInnkeeper:
		return "Innkeeper"
	case DialogGuard:
		return "Guard"
	default:
		return ""
	}
}

// NPCType to DialogNumber Mapping:
//
// The NPCType values (used in NPC data files) are hexadecimal values that
// happen to have the same decimal value as their corresponding DialogNumber:
//
// NPCType 0x81 (129 decimal) = Blacksmith  -> DialogNumber 129 (DialogWeaponDealer)
// NPCType 0x82 (130 decimal) = Barkeeper   -> DialogNumber 130 (DialogBarkeeper)
// NPCType 0x83 (131 decimal) = HorseSeller -> DialogNumber 131 (DialogHorseDealer)
// NPCType 0x84 (132 decimal) = Shipwright  -> DialogNumber 132 (DialogShipDealer)
// NPCType 0x85 (133 decimal) = MagicSeller -> DialogNumber 133 (DialogMagicDealer)
// NPCType 0x86 (134 decimal) = GuildMaster -> DialogNumber 134 (DialogGuildMaster)
// NPCType 0x87 (135 decimal) = Healer      -> DialogNumber 135 (DialogHealer)
// NPCType 0x88 (136 decimal) = InnKeeper   -> DialogNumber 136 (DialogInnkeeper)
//
// This is not a coincidence - the NPCType values were designed to match the
// dialog numbers used by these special merchant NPCs.
