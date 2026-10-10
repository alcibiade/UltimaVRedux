package references

import (
	"errors"
	"fmt"
)

var (
	errUnsupportedMerchantCode     = errors.New("unsupported merchant dialog code")
	errMerchantNotFound            = errors.New("no active merchant")
	errMerchantNPCMismatch         = errors.New("merchant NPC index mismatch")
	errInvalidMerchantDefinition   = errors.New("invalid merchant definition")
	errDuplicateMerchantDefinition = errors.New("duplicate merchant definition")
)

// MerchantID is the stable identity of an original merchant.
type MerchantID string

// MerchantFamily is the original merchant Talk code.
type MerchantFamily byte

// Merchant family constants are the only supported production Talk codes.
const (
	MerchantFamilyArms    MerchantFamily = 0x81
	MerchantFamilyTavern  MerchantFamily = 0x82
	MerchantFamilyHorse   MerchantFamily = 0x83
	MerchantFamilyShip    MerchantFamily = 0x84
	MerchantFamilyReagent MerchantFamily = 0x85
	MerchantFamilyGuild   MerchantFamily = 0x86
	MerchantFamilyHealer  MerchantFamily = 0x87
	MerchantFamilyInn     MerchantFamily = 0x88
)

// Stable merchant identity constants cover every active original shop row.
const (
	MerchantArmsBritain         MerchantID = "arms.britain"
	MerchantArmsJhelom          MerchantID = "arms.jhelom"
	MerchantArmsYew             MerchantID = "arms.yew"
	MerchantArmsMinoc           MerchantID = "arms.minoc"
	MerchantArmsTrinsic         MerchantID = "arms.trinsic"
	MerchantArmsCastleBritannia MerchantID = "arms.castle-britannia"
	MerchantArmsBuccaneersDen   MerchantID = "arms.buccaneers-den"
	MerchantArmsBordermarch     MerchantID = "arms.bordermarch"
	MerchantArmsSerpentsHold    MerchantID = "arms.serpents-hold"
	MerchantTavernMoonglow      MerchantID = "tavern.moonglow"
	MerchantTavernBritain       MerchantID = "tavern.britain"
	MerchantTavernJhelom        MerchantID = "tavern.jhelom"
	MerchantTavernYew           MerchantID = "tavern.yew"
	MerchantTavernNewMagincia   MerchantID = "tavern.new-magincia"
	MerchantTavernWestBritanny  MerchantID = "tavern.west-britanny"
	MerchantTavernPaws          MerchantID = "tavern.paws"
	MerchantTavernBuccaneersDen MerchantID = "tavern.buccaneers-den"
	MerchantTavernLycaeum       MerchantID = "tavern.lycaeum"
	MerchantHorseTrinsic        MerchantID = "horse.trinsic"
	MerchantHorseNorthBritanny  MerchantID = "horse.north-britanny"
	MerchantHorsePaws           MerchantID = "horse.paws"
	MerchantShipJhelom          MerchantID = "ship.jhelom"
	MerchantShipMinoc           MerchantID = "ship.minoc"
	MerchantShipEastBritanny    MerchantID = "ship.east-britanny"
	MerchantShipBuccaneersDen   MerchantID = "ship.buccaneers-den"
	MerchantReagentMoonglow     MerchantID = "reagent.moonglow"
	MerchantReagentYew          MerchantID = "reagent.yew"
	MerchantReagentSkaraBrae    MerchantID = "reagent.skara-brae"
	MerchantReagentCove         MerchantID = "reagent.cove"
	MerchantReagentLycaeum      MerchantID = "reagent.lycaeum"
	MerchantGuildNewMagincia    MerchantID = "guild.new-magincia"
	MerchantGuildPaws           MerchantID = "guild.paws"
	MerchantGuildBuccaneersDen  MerchantID = "guild.buccaneers-den"
	MerchantHealerMinoc         MerchantID = "healer.minoc"
	MerchantHealerTrinsic       MerchantID = "healer.trinsic"
	MerchantHealerSkaraBrae     MerchantID = "healer.skara-brae"
	MerchantHealerEastBritanny  MerchantID = "healer.east-britanny"
	MerchantHealerCove          MerchantID = "healer.cove"
	MerchantHealerLycaeum       MerchantID = "healer.lycaeum"
	MerchantHealerEmpathAbbey   MerchantID = "healer.empath-abbey"
	MerchantInnBritain          MerchantID = "inn.britain"
	MerchantInnJhelom           MerchantID = "inn.jhelom"
	MerchantInnSkaraBrae        MerchantID = "inn.skara-brae"
	MerchantInnNorthBritanny    MerchantID = "inn.north-britanny"
	MerchantInnPaws             MerchantID = "inn.paws"
	MerchantInnBuccaneersDen    MerchantID = "inn.buccaneers-den"
)

// MerchantDefinition identifies one active original merchant. NPCIndex and
// CatalogRow are zero-based indexes in the location NPC table and family data.
type MerchantDefinition struct {
	ID         MerchantID
	Family     MerchantFamily
	Location   Location
	NPCIndex   int
	CatalogRow int
	OwnerName  string
	ShopName   string
}

// MerchantReferences owns immutable merchant definitions and SHOPPE text.
// Retrieval methods return values or fresh slices, never internal maps.
type MerchantReferences struct {
	definitions     []MerchantDefinition
	definitionsByID map[MerchantID]MerchantDefinition
	byLocation      map[merchantLocationKey]MerchantDefinition
	byCatalog       map[merchantCatalogKey]MerchantDefinition
	texts           map[MerchantTextID]string
}

type merchantLocationKey struct {
	family   MerchantFamily
	location Location
}

type merchantCatalogKey struct {
	family MerchantFamily
	row    int
}

// Definitions returns a detached snapshot of all active original merchants.
func (r *MerchantReferences) Definitions() []MerchantDefinition {
	return append([]MerchantDefinition(nil), r.definitions...)
}

// Definition returns one merchant by stable identity.
func (r *MerchantReferences) Definition(id MerchantID) (MerchantDefinition, bool) {
	definition, ok := r.definitionsByID[id]

	return definition, ok
}

// DefinitionForCatalog returns the active merchant assigned to a family row.
// Source rows with no active merchant are deliberately absent.
func (r *MerchantReferences) DefinitionForCatalog(
	family MerchantFamily,
	row int,
) (MerchantDefinition, bool) {
	definition, ok := r.byCatalog[merchantCatalogKey{family: family, row: row}]

	return definition, ok
}

// Resolve validates an original merchant Talk code, location, and local NPC index.
func (r *MerchantReferences) Resolve(
	dialogNumber DialogNumber,
	location Location,
	npcIndex int,
) (MerchantDefinition, error) {
	family, err := MerchantFamilyFromDialog(dialogNumber)
	if err != nil {
		return MerchantDefinition{}, err
	}

	definition, ok := r.byLocation[merchantLocationKey{family: family, location: location}]
	if !ok {
		return MerchantDefinition{}, fmt.Errorf(
			"%w for family 0x%02x at location 0x%02x",
			errMerchantNotFound,
			byte(family), byte(location),
		)
	}
	if definition.NPCIndex != npcIndex {
		return MerchantDefinition{}, fmt.Errorf(
			"%w for %q: expected %d, got %d",
			errMerchantNPCMismatch,
			definition.ID, definition.NPCIndex, npcIndex,
		)
	}

	return definition, nil
}

// MerchantFamilyFromDialog accepts only the eight production merchant codes.
func MerchantFamilyFromDialog(dialogNumber DialogNumber) (MerchantFamily, error) {
	if dialogNumber < DialogWeaponDealer || dialogNumber > DialogInnkeeper {
		return 0, fmt.Errorf("%w 0x%02x", errUnsupportedMerchantCode, byte(dialogNumber))
	}

	return MerchantFamily(dialogNumber), nil
}

func newMerchantReferenceIndex(
	definitions []MerchantDefinition,
	texts map[MerchantTextID]string,
) (*MerchantReferences, error) {
	refs := &MerchantReferences{
		definitions:     append([]MerchantDefinition(nil), definitions...),
		definitionsByID: make(map[MerchantID]MerchantDefinition, len(definitions)),
		byLocation:      make(map[merchantLocationKey]MerchantDefinition, len(definitions)),
		byCatalog:       make(map[merchantCatalogKey]MerchantDefinition, len(definitions)),
		texts:           make(map[MerchantTextID]string, len(texts)),
	}
	for id, text := range texts {
		refs.texts[id] = text
	}

	for _, definition := range definitions {
		if definition.ID == "" || definition.OwnerName == "" || definition.ShopName == "" {
			return nil, fmt.Errorf("%w: incomplete row %+v", errInvalidMerchantDefinition, definition)
		}
		if _, err := MerchantFamilyFromDialog(DialogNumber(definition.Family)); err != nil {
			return nil, fmt.Errorf("merchant %q: %w", definition.ID, err)
		}
		if definition.NPCIndex < 0 || definition.NPCIndex >= npcsPerTown {
			return nil, fmt.Errorf(
				"%w: merchant %q has NPC index %d",
				errInvalidMerchantDefinition, definition.ID, definition.NPCIndex,
			)
		}
		catalogRows := merchantCatalogRowCount(definition.Family)
		if definition.CatalogRow < 0 || definition.CatalogRow >= catalogRows {
			return nil, fmt.Errorf(
				"%w: merchant %q has invalid catalog row %d",
				errInvalidMerchantDefinition, definition.ID, definition.CatalogRow,
			)
		}

		locationKey := merchantLocationKey{family: definition.Family, location: definition.Location}
		catalogKey := merchantCatalogKey{family: definition.Family, row: definition.CatalogRow}
		if _, exists := refs.definitionsByID[definition.ID]; exists {
			return nil, fmt.Errorf("%w: duplicate merchant ID %q", errDuplicateMerchantDefinition, definition.ID)
		}
		if _, exists := refs.byLocation[locationKey]; exists {
			return nil, fmt.Errorf(
				"%w: family 0x%02x at location 0x%02x",
				errDuplicateMerchantDefinition,
				byte(definition.Family), byte(definition.Location),
			)
		}
		if _, exists := refs.byCatalog[catalogKey]; exists {
			return nil, fmt.Errorf(
				"%w: family 0x%02x catalog row %d",
				errDuplicateMerchantDefinition,
				byte(definition.Family), definition.CatalogRow,
			)
		}

		refs.definitionsByID[definition.ID] = definition
		refs.byLocation[locationKey] = definition
		refs.byCatalog[catalogKey] = definition
	}

	return refs, nil
}

func merchantCatalogRowCount(family MerchantFamily) int {
	switch family {
	case MerchantFamilyArms, MerchantFamilyTavern:
		return 9
	case MerchantFamilyHorse, MerchantFamilyGuild:
		return 3
	case MerchantFamilyShip:
		return 4
	case MerchantFamilyReagent:
		return 5
	case MerchantFamilyHealer:
		return 7
	case MerchantFamilyInn:
		return 6
	default:
		return 0
	}
}

// Original shop names, owners, and map assignments (ref:MERCH-0001). NPC indexes
// are the matching zero-based records from the four original *.NPC files.
//
//nolint:gochecknoglobals // Canonical original merchant table is immutable after initialization.
var originalMerchantDefinitions = []MerchantDefinition{
	{MerchantArmsBritain, MerchantFamilyArms, Britain, 1, 0, "Gwenneth", "Iolo's Bows"},
	{MerchantArmsJhelom, MerchantFamilyArms, Jhelom, 3, 1, "Nomaan", "Naughty Nomaan's"},
	{MerchantArmsYew, MerchantFamilyArms, Yew, 3, 2, "Ronan", "Arms of Justice"},
	{MerchantArmsMinoc, MerchantFamilyArms, Minoc, 5, 3, "Shenstone", "Darkwatch Armoury"},
	{MerchantArmsTrinsic, MerchantFamilyArms, Trinsic, 2, 4, "Paul", "The Paladin's Protectorate!"},
	{MerchantArmsCastleBritannia, MerchantFamilyArms, LordBritishsCastle, 30, 5, "Max", "North Star Armoury"},
	{MerchantArmsBuccaneersDen, MerchantFamilyArms, BuccaneersDen, 2, 6, "Kitiara", "Buccaneers Booty"},
	{MerchantArmsBordermarch, MerchantFamilyArms, Bordermarch, 3, 7, "Steve", "The Shattered Shield"},
	{MerchantArmsSerpentsHold, MerchantFamilyArms, SerpentsHold, 10, 8, "Thol", "Siege Crafters"},

	{MerchantTavernMoonglow, MerchantFamilyTavern, Moonglow, 1, 0, "Sam", "The Honest Meal"},
	{MerchantTavernBritain, MerchantFamilyTavern, Britain, 3, 1, "Tika", "The Wayfarer Tavern"},
	{MerchantTavernJhelom, MerchantFamilyTavern, Jhelom, 2, 2, "Nicole", "The Sword and Keg"},
	{MerchantTavernYew, MerchantFamilyTavern, Yew, 2, 3, "Duclas", "The Slaughtered Lamb"},
	{MerchantTavernNewMagincia, MerchantFamilyTavern, NewMagincia, 2, 4, "Felicity", "The Humble Palate"},
	{MerchantTavernWestBritanny, MerchantFamilyTavern, WestBritanny, 5, 5, "Jaymes", "The Blue Boar Tavern"},
	{MerchantTavernPaws, MerchantFamilyTavern, Paws, 3, 6, "Dr. Cat", "The Cat's Lair"},
	{MerchantTavernBuccaneersDen, MerchantFamilyTavern, BuccaneersDen, 5, 7, "Nikki", "The Fallen Virgin"},
	{MerchantTavernLycaeum, MerchantFamilyTavern, Lycaeum, 10, 8, "Rob", "The Folley Tap"},

	{MerchantHorseTrinsic, MerchantFamilyHorse, Trinsic, 1, 0, "Hettar", "Horse & Rider"},
	{MerchantHorseNorthBritanny, MerchantFamilyHorse, NorthBritanny, 6, 1, "Theoan", "The Stablehouse"},
	{MerchantHorsePaws, MerchantFamilyHorse, Paws, 4, 2, "Ferru", "Wishing Well Horses"},

	{MerchantShipJhelom, MerchantFamilyShip, Jhelom, 5, 0, "Bantral", "Island Shipwrights"},
	{MerchantShipMinoc, MerchantFamilyShip, Minoc, 4, 1, "Captain Blyth", "The Crow's Nest"},
	{MerchantShipEastBritanny, MerchantFamilyShip, EastBritanny, 2, 2, "Master Hawkins", "The Oaken Oar"},
	{MerchantShipBuccaneersDen, MerchantFamilyShip, BuccaneersDen, 3, 3, "Jones", "The Rusty Bucket"},

	{MerchantReagentMoonglow, MerchantFamilyReagent, Moonglow, 2, 0, "Nilrem", "The Herbalist"},
	{MerchantReagentYew, MerchantFamilyReagent, Yew, 1, 1, "Madam Pendra", "Healers Herbs"},
	{MerchantReagentSkaraBrae, MerchantFamilyReagent, SkaraBrae, 1, 2, "Toama", "The Alchemist"},
	{MerchantReagentCove, MerchantFamilyReagent, Cove, 1, 3, "Enlor", "Mysticism"},
	{MerchantReagentLycaeum, MerchantFamilyReagent, Lycaeum, 12, 4, "Virden", "The Sharper Mage"},

	{MerchantGuildNewMagincia, MerchantFamilyGuild, NewMagincia, 1, 0, "Braunam", "The Den"},
	{MerchantGuildPaws, MerchantFamilyGuild, Paws, 1, 1, "Danfits", "The Guild"},
	{MerchantGuildBuccaneersDen, MerchantFamilyGuild, BuccaneersDen, 1, 2, "Daem", "The Nemesis"},

	{MerchantHealerMinoc, MerchantFamilyHealer, Minoc, 2, 0, "Regina", "The Healers Mission"},
	{MerchantHealerTrinsic, MerchantFamilyHealer, Trinsic, 4, 1, "Leila", "Wounds of Honour"},
	{MerchantHealerSkaraBrae, MerchantFamilyHealer, SkaraBrae, 3, 2, "Temptious", "The Spirit Healers"},
	{MerchantHealerEastBritanny, MerchantFamilyHealer, EastBritanny, 1, 3, "Milan", "Healers' Sanctum"},
	{MerchantHealerCove, MerchantFamilyHealer, Cove, 2, 4, "Jessica", "Sanctuary"},
	{MerchantHealerLycaeum, MerchantFamilyHealer, Lycaeum, 11, 5, "Faye", "The Shield of Truth"},
	{MerchantHealerEmpathAbbey, MerchantFamilyHealer, EmpathAbbey, 9, 6, "Jessip", "The Empath"},

	{MerchantInnBritain, MerchantFamilyInn, Britain, 2, 0, "Donya", "The Wayfarer Inn"},
	{MerchantInnJhelom, MerchantFamilyInn, Jhelom, 1, 1, "Gremnor", "The Warrior's Stead"},
	{MerchantInnSkaraBrae, MerchantFamilyInn, SkaraBrae, 2, 2, "Rogi", "The Haunting Inn"},
	{MerchantInnNorthBritanny, MerchantFamilyInn, NorthBritanny, 1, 3, "Terbor", "Hotel Brittany"},
	{MerchantInnPaws, MerchantFamilyInn, Paws, 2, 4, "Lorien", "The Smugglers' Inn"},
	{MerchantInnBuccaneersDen, MerchantFamilyInn, BuccaneersDen, 4, 5, "Ransack", "The King's Ransom Inn"},
}
