package legacy_save

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/bradhannah/UltimaVRedux/internal/environment"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
)

const (
	originalAmuletEffect      = 0x0e
	originalCrownEffect       = 0x1c
	originalBadgeEffect       = 0x1d
	originalPlayerShape       = 0x1c
	originalRiderShape        = 0x12
	originalCarpetShape       = 0x14
	originalShipShape         = 0x20
	originalAnchorShape       = 0x24
	originalSkiffShape        = 0x28
	originalPirateShape       = 0x2c
	originalNPCShape          = 0x40
	originalChestShape        = 0x01
	originalCarpetObjectShape = 0x1b
	originalDeadPCShape       = 0x1e
	originalDeadMonsterShape  = 0x1f
	originalGoldShape         = 0x02
	originalPotionShape       = 0x03
	originalScrollShape       = 0x04
	originalWeaponShape       = 0x05
	originalShieldShape       = 0x06
	originalKeyShape          = 0x07
	originalGemShape          = 0x08
	originalHelmShape         = 0x09
	originalRingShape         = 0x0a
	originalArmorShape        = 0x0b
	originalAmuletShape       = 0x0c
	originalTorchShape        = 0x0d
	originalFoodShape         = 0x0f
	originalMoonstoneShape    = 0x19
	originalShardShape        = 0xb4
	originalCrownShape        = 0xb5
	originalSceptreShape      = 0xb6
	originalQuestAmuletShape  = 0xb7
	originalNothing           = 0xff
	originalOpenMarker        = 0x80
	originalHMSCapeRiggedMask = 0x80
	originalDirectionMask     = 0x03
	originalSkiffCountMask    = 0x3f
	originalKeyQuantityMask   = 0x7f
	originalObjectBaseMask    = 0xfc
	originalHorseObjectShape  = 0x10
	originalMonsterFrames     = 4
	originalDeliveryMarker    = 0x40
	originalFrigateDelivery   = 0x80
	originalShardPayloadBase  = 0xf0
	bitsPerByte               = 8
	npcFlagBytesPerLocation   = 4
	conversionWarningCapacity = 8
	defaultShipHull           = 99
	originalDirectionNorth    = 0
	originalDirectionEast     = 1
	originalDirectionSouth    = 2
	originalDirectionWest     = 3
)

// ConversionWarning describes one deterministic, non-fatal loss or safe
// default applied while translating original state into the native schema.
type ConversionWarning struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ConversionResult contains a validated native snapshot and stable warnings.
type ConversionResult struct {
	Snapshot save_snapshot.SaveSnapshot
	Warnings []ConversionWarning
}

type canonicalBundle struct {
	Game    SavedGame    `json:"game"`
	Objects SavedObjects `json:"objects"`
}

// Convert translates a validated original bundle into the current native snapshot.
func Convert(bundle Bundle) (ConversionResult, error) {
	warnings := make([]ConversionWarning, 0, conversionWarningCapacity)
	var snapshot save_snapshot.SaveSnapshot
	snapshot.Version = save_snapshot.CurrentVersion
	snapshot.Presentation = journal.EmptyPresentation()
	convertParty(bundle.Game, &snapshot, &warnings)
	convertContinuation(bundle.Game, &snapshot, &warnings)
	convertInventory(bundle.Game, &snapshot)
	convertWorld(bundle.Game, &snapshot, &warnings)
	convertProgression(bundle.Game, &snapshot, &warnings)
	// Progression conversion establishes the carried moonstone quantities.
	snapshot.BackfillMoonstonePossession()
	convertEffects(bundle.Game, &snapshot, &warnings)
	convertDungeon(bundle.Game, &snapshot)
	snapshot.Wind = save_snapshot.WindSnapshot{
		Direction:     int(bundle.Game.Wind),
		UpdateCounter: int(bundle.Game.WindUpdateCount),
		NextTickMs:    environment.WindTickIntervalMs,
	}
	snapshot.Random = save_snapshot.RandomSnapshot{Seed: sourceSeed(bundle), SourceCalls: 0}
	snapshot.PartyVehicle = convertPartyVehicle(bundle.Game, &warnings)
	snapshot.ItemStacks = make([]save_snapshot.ItemStackSnapshot, 0)
	convertMapObjects(bundle, &snapshot, &warnings)
	convertBasementChestHistory(bundle.Game, &snapshot, &warnings)
	snapshot.MapMutations = make([]save_snapshot.MapMutationSnapshot, 0)
	snapshot.Redux.DungeonDiscoveries = make([]save_snapshot.DungeonDiscoverySnapshot, 0)
	snapshot.Redux.PlayerKnowledge = save_snapshot.InferLegacyPlayerKnowledge(snapshot.Inventory, snapshot.Progression)
	snapshot.Redux.QuestJournal = save_snapshot.InferLegacyQuestJournal(snapshot.Progression)
	snapshot.Redux.LordBritishPassageOpen = snapshot.Inventory.QuestItems[references.WoodenBox] > 0 ||
		hasNPCFlag(snapshot.Party.DeadNPCs, lordBritishNPCFlagLocation, lordBritishSandalwoodBoxNPC)
	if snapshot.Inventory.QuestItems[references.WoodenBox] > 0 &&
		!hasNPCFlag(snapshot.Party.DeadNPCs, lordBritishNPCFlagLocation, lordBritishSandalwoodBoxNPC) {
		snapshot.Party.DeadNPCs = append(snapshot.Party.DeadNPCs, save_snapshot.NPCFlagSnapshot{
			Location: lordBritishNPCFlagLocation,
			NPC:      lordBritishSandalwoodBoxNPC,
		})
		sort.Slice(snapshot.Party.DeadNPCs, func(i, j int) bool {
			if snapshot.Party.DeadNPCs[i].Location != snapshot.Party.DeadNPCs[j].Location {
				return snapshot.Party.DeadNPCs[i].Location < snapshot.Party.DeadNPCs[j].Location
			}

			return snapshot.Party.DeadNPCs[i].NPC < snapshot.Party.DeadNPCs[j].NPC
		})
	}
	addConversionWarning(
		&warnings,
		"redux",
		"legacy_defaults",
		"Redux-only elapsed-clock, combat-sequence, actor-phase, and recipe-discovery state use safe defaults",
	)
	sortConversionWarnings(warnings)

	if issues := save_snapshot.Validate(snapshot); len(issues) != 0 {
		return ConversionResult{}, &save_snapshot.ValidationError{Issues: issues}
	}

	return ConversionResult{Snapshot: snapshot, Warnings: warnings}, nil
}

const (
	lordBritishNPCFlagLocation  = int(references.LordBritishsCastle - references.Moonglow)
	lordBritishSandalwoodBoxNPC = 31
)

func hasNPCFlag(flags []save_snapshot.NPCFlagSnapshot, location, npc int) bool {
	for _, flag := range flags {
		if flag.Location == location && flag.NPC == npc {
			return true
		}
	}

	return false
}

func convertParty(
	game SavedGame,
	snapshot *save_snapshot.SaveSnapshot,
	warnings *[]ConversionWarning,
) {
	partyMapNormalized := false
	for index, source := range game.Characters {
		if !source.Occupied {
			continue
		}
		partyMap := source.PartyMap
		if index < int(game.PartyCount) && partyMap != PartyMapActive {
			partyMap = PartyMapActive
			partyMapNormalized = true
		} else if index >= int(game.PartyCount) && partyMap == PartyMapActive {
			partyMap = PartyMapUnrecruited
			partyMapNormalized = true
		}
		snapshot.Party.Roster[index] = save_snapshot.CharacterSnapshot{
			Occupied:     true,
			Name:         source.Name,
			Gender:       int(source.Gender),
			Class:        int(source.Class),
			Status:       int(source.Status),
			Strength:     int(source.Strength),
			Dexterity:    int(source.Dexterity),
			Intelligence: int(source.Intelligence),
			CurrentMP:    int(source.CurrentMP),
			CurrentHP:    int(source.CurrentHP),
			MaximumHP:    int(source.MaximumHP),
			Experience:   int(source.Experience),
			Level:        int(source.Level),
			MonthsAtInn:  int(source.MonthsAtInn),
			ArmorClass:   int(source.ArmorClass),
			Helmet:       int(source.Equipment.Helmet),
			Armor:        int(source.Equipment.Armor),
			Weapon:       int(source.Equipment.Hands[0]),
			Shield:       int(source.Equipment.Hands[1]),
			Ring:         int(source.Equipment.Ring),
			Amulet:       int(source.Equipment.Amulet),
			PartyMap:     int(partyMap),
		}
	}
	if partyMapNormalized {
		addConversionWarning(
			warnings,
			"party.roster",
			"party_prefix_pmap_normalized",
			"character pmap values were normalized to the authoritative original party-size prefix",
		)
	}
	snapshot.Party.Karma = int(game.Karma)
	snapshot.Party.MetNPCs = expandNPCFlags(game.MetNPCs)
	snapshot.Party.DeadNPCs = expandNPCFlags(game.DeadNPCs)
}

func convertContinuation(
	game SavedGame,
	snapshot *save_snapshot.SaveSnapshot,
	warnings *[]ConversionWarning,
) {
	snapshot.Continuation.ActivePlayerIndex = -1
	snapshot.Continuation.BeggarKarmaCounter = int(game.LastGave)
	snapshot.Continuation.OpeningCampaignEncounterReduction = game.Easy != 0
	snapshot.Continuation.CombatActionTimeCounter = int(game.CombatActionTimeCount)
	if game.ActivePlayer == originalNothing {
		return
	}

	index := int(game.ActivePlayer)
	if index < len(snapshot.Party.Roster) && index < int(game.PartyCount) {
		character := snapshot.Party.Roster[index]
		if character.Occupied && character.PartyMap == int(PartyMapActive) &&
			character.CurrentHP > 0 &&
			character.Status != int(StatusDead) && character.Status != int(StatusAsleep) {
			snapshot.Continuation.ActivePlayerIndex = index

			return
		}
	}
	addConversionWarning(
		warnings,
		"continuation.activePlayerIndex",
		"invalid_source_value",
		"the original active-player index does not identify an eligible active-party character and was reset to none",
	)
}

func convertInventory(game SavedGame, snapshot *save_snapshot.SaveSnapshot) {
	source := game.Inventory
	destination := &snapshot.Inventory
	destination.Food = source.Food
	destination.Gold = source.Gold
	destination.Gems = uint16(source.Gems)
	destination.Torches = uint16(source.Torches)
	destination.Keys = uint16(source.Keys)
	destination.SkullKeys = uint16(source.SkullKeys)
	copyByteQuantities(destination.Equipment[:], source.Equipment[:])
	copyByteQuantities(destination.Spells[:], source.PreparedSpells[:])
	copyByteQuantities(destination.Scrolls[:], source.Scrolls[:])
	copyByteQuantities(destination.Potions[:], source.Potions[:])
	copyByteQuantities(destination.Reagents[:], source.Reagents[:])

	destination.QuestItems[references.Amulet] = uniqueItemQuantity(source.Amulet)
	destination.QuestItems[references.Crown] = uniqueItemQuantity(source.Crown)
	destination.QuestItems[references.Sceptre] = uniqueItemQuantity(source.Sceptre)
	destination.QuestItems[references.ShardOfFalsehood] = uniqueItemQuantity(source.Shards.Falsehood)
	destination.QuestItems[references.ShardOfHatred] = uniqueItemQuantity(source.Shards.Hatred)
	destination.QuestItems[references.ShardOfCowardice] = uniqueItemQuantity(source.Shards.Cowardice)
	destination.QuestItems[references.Carpet] = uint16(source.Carpets)
	destination.QuestItems[references.Grapple] = uint16(source.Grapples)
	destination.QuestItems[references.Spyglass] = uniqueItemQuantity(source.SpecialItems.Spyglass)
	destination.QuestItems[references.HMSCape] = uniqueItemQuantity(source.SpecialItems.HMSCapePlans)
	destination.QuestItems[references.PocketWatch] = uniqueItemQuantity(source.SpecialItems.PocketWatch)
	destination.QuestItems[references.BlackBadge] = uniqueItemQuantity(source.SpecialItems.BlackBadge)
	destination.QuestItems[references.WoodenBox] = uniqueItemQuantity(source.SpecialItems.WoodenBox)
	destination.QuestItems[references.Sextant] = uniqueItemQuantity(source.SpecialItems.Sextant)
}

func convertWorld(game SavedGame, snapshot *save_snapshot.SaveSnapshot, warnings *[]ConversionWarning) {
	position := save_snapshot.PositionSnapshot{
		Location: int(game.Position.Location),
		X:        int(game.Position.X),
		Y:        int(game.Position.Y),
		Floor:    signedFloor(game.Position.Floor),
	}
	var lastLargeMapPosition save_snapshot.PositionSnapshot
	if game.Position.Location == 0 {
		lastLargeMapPosition = position
	} else {
		addConversionWarning(
			warnings,
			"world.lastLargeMapPosition",
			"source_value_unavailable",
			"the original save does not retain the last large-map coordinates; Britannia origin is used",
		)
	}
	snapshot.World = save_snapshot.WorldSnapshot{
		Position:             position,
		LastLargeMapPosition: lastLargeMapPosition,
		Calendar: save_snapshot.CalendarSnapshot{
			Year:   int(game.Calendar.Year),
			Month:  int(game.Calendar.Month),
			Day:    int(game.Calendar.Day),
			Hour:   int(game.Calendar.Hour),
			Minute: int(game.Calendar.Minute),
			Turns:  0,
		},
		ElapsedMs:         0,
		LastProcessedHour: int(game.OldHour),
		LastCamped:        int(game.LastCamped),
		LastWandererMonth: int(game.LastWandererMonth),
		CombatSequence:    0,
		IsSailing:         false,
	}
	if game.Ship.Sailing != 0 {
		addConversionWarning(
			warnings,
			"world.isSailing",
			"automatic_sailing_not_restored",
			"automatic sailing is workflow-owned and is not resumed from an original save",
		)
	}
}

func convertProgression(game SavedGame, snapshot *save_snapshot.SaveSnapshot, warnings *[]ConversionWarning) {
	convertSearchProgress(game, snapshot)
	convertMoonstones(game, snapshot)
	convertShrinesAndDungeons(game, snapshot)
	convertShadowlords(game, snapshot, warnings)
	snapshot.Progression.HMSCapeActivated = game.Inventory.SpecialItems.HMSCapePlans&originalHMSCapeRiggedMask != 0
}

// convertSearchProgress copies search flags and cache days, clearing flag bits beyond the known search locations.
func convertSearchProgress(game SavedGame, snapshot *save_snapshot.SaveSnapshot) {
	copy(snapshot.Progression.SearchFound[:], game.SearchFlags[:])
	for bit := references.SearchLocationCount; bit < len(snapshot.Progression.SearchFound)*8; bit++ {
		snapshot.Progression.SearchFound[bit/bitsPerByte] &^= 1 << (bit & (bitsPerByte - 1))
	}
	snapshot.Progression.SkullKeysFoundDay = game.SkullKeysFoundDay
	copy(snapshot.Progression.ReagentCachesFoundDay[:], game.ReagentCacheDays[:])
}

// convertMoonstones converts moonstone placements and marks buried-nowhere stones as held.
func convertMoonstones(game SavedGame, snapshot *save_snapshot.SaveSnapshot) {
	for index, source := range game.Moonstones {
		state := byte(0)
		if source.Map == originalNothing {
			state = originalNothing
			snapshot.Inventory.Moonstones[index] = 1
		}
		snapshot.Progression.Moonstones[index] = save_snapshot.MoonstoneSnapshot{
			X:       source.X,
			Y:       source.Y,
			Map:     source.Map,
			Level:   source.Level,
			State:   state,
			IsFound: false,
		}
	}
}

// convertShrinesAndDungeons converts shrine bits, open dungeons, and visited dungeon rooms.
func convertShrinesAndDungeons(game SavedGame, snapshot *save_snapshot.SaveSnapshot) {
	for index := range snapshot.Progression.Shrines {
		mask := byte(1 << index)
		snapshot.Progression.Shrines[index] = save_snapshot.ShrineSnapshot{
			Active:    game.Shrines.ActiveBits&mask != 0,
			Completed: game.Shrines.CompletedBits&mask != 0,
			Ruined:    game.Shrines.Closed[index]&originalOpenMarker != 0,
		}
		snapshot.Progression.Dungeons[index] = game.OpenDungeons[index]&originalOpenMarker != 0
	}
	copy(snapshot.Progression.DungeonRooms[:], game.DungeonRooms[:])
}

// convertShadowlords converts Shadowlord life, town assignments, and the last-called lord.
func convertShadowlords(game SavedGame, snapshot *save_snapshot.SaveSnapshot, warnings *[]ConversionWarning) {
	assignedShadowlordTowns := make(map[byte]struct{}, save_snapshot.ShadowlordCount)
	for index, marker := range game.ShadowlordLords {
		switch {
		case marker == originalNothing:
			snapshot.Progression.Shadowlords.Alive[index] = false
		case marker == 0:
			snapshot.Progression.Shadowlords.Alive[index] = true
		case marker >= 1 && marker <= 8:
			snapshot.Progression.Shadowlords.Alive[index] = true
			if _, duplicate := assignedShadowlordTowns[marker]; duplicate {
				addConversionWarning(
					warnings,
					fmt.Sprintf("progression.shadowlords.townAssignments[%d]", index),
					"duplicate_source_value",
					"the original Shadowlord town duplicates an earlier assignment and was reset to unassigned",
				)

				continue
			}
			assignedShadowlordTowns[marker] = struct{}{}
			snapshot.Progression.Shadowlords.TownAssignments[index] = int(marker)
		default:
			snapshot.Progression.Shadowlords.Alive[index] = true
			addConversionWarning(
				warnings,
				fmt.Sprintf("progression.shadowlords.townAssignments[%d]", index),
				"unsupported_source_value",
				"the original Shadowlord town value is outside 0 through 8 and was reset to unassigned",
			)
		}
	}
	snapshot.Progression.Shadowlords.LastCalled = -1
	if game.LastCalledShadowlord < save_snapshot.ShadowlordCount {
		snapshot.Progression.Shadowlords.LastCalled = int(game.LastCalledShadowlord)
	} else if game.LastCalledShadowlord != originalNothing {
		addConversionWarning(
			warnings,
			"progression.shadowlords.lastCalled",
			"unsupported_source_value",
			"the original last-called Shadowlord value is outside 0 through 2 and was reset to none",
		)
	}
	snapshot.Progression.Shadowlords.Summoned = false
	snapshot.Progression.Shadowlords.AutomaticPresence = -1
}

func convertEffects(game SavedGame, snapshot *save_snapshot.SaveSnapshot, warnings *[]ConversionWarning) {
	duration := int(game.SpellDuration)
	switch game.ActiveSpell {
	case 0:
	case 'P', 'N', 'T', 'Q', 'C':
		if duration == 0 {
			addConversionWarning(
				warnings,
				"effects.activeSpell",
				"invalid_duration_omitted",
				"the original active duration effect had no remaining duration and was omitted",
			)

			break
		}
		snapshot.Effects.DurationType = game.ActiveSpell
		snapshot.Effects.DurationTurns = duration
	case originalAmuletEffect:
		snapshot.Effects.Amulet = save_snapshot.ToggleSnapshot{Active: true, Turns: duration}
	case originalCrownEffect:
		snapshot.Effects.Crown = save_snapshot.ToggleSnapshot{Active: true, Turns: duration}
	case originalBadgeEffect:
		snapshot.Effects.Badge = save_snapshot.ToggleSnapshot{Active: true, Turns: duration}
	default:
		addConversionWarning(
			warnings,
			"effects.activeSpell",
			"unsupported_source_value",
			"the original active-effect code is unsupported and was omitted",
		)
	}
	snapshot.Effects.MagicLightMinutesRemaining = int(game.MagicLight)
	snapshot.Effects.CarriedTorchMinutesRemaining = int(game.TorchTurns)
	snapshot.Effects.QuicknessWorldActorSkip = false
	snapshot.Effects.Drunk = 0
	snapshot.Effects.Smashed = int(game.Smashed)
	if game.Spell != 0 || game.Magic != 0 {
		addConversionWarning(
			warnings,
			"effects",
			"transient_action_flags_omitted",
			"original transient spell and magical-attack flags are not durable native state",
		)
	}
}

func convertPartyVehicle(game SavedGame, warnings *[]ConversionWarning) *save_snapshot.VehicleSnapshot {
	shape := game.PlayerShape
	vehicleType, direction, ok := originalVehicleIdentity(shape)
	if !ok {
		if shape != 0 && shape != originalPlayerShape {
			addConversionWarning(
				warnings,
				"partyVehicle",
				"unsupported_player_shape",
				"the original player shape does not identify a durable boarded vehicle and was reset to on foot",
			)
		}

		return nil
	}
	vehicle := &save_snapshot.VehicleSnapshot{
		VesselID:          0,
		Type:              vehicleType,
		CurrentDirection:  direction,
		PreviousDirection: direction,
		Skiffs:            0,
		OwnedByNPC:        false,
		Hull:              defaultShipHull,
		SailState:         0,
		HasHMSCape:        false,
		PendingDelivery:   false,
	}
	if vehicle.Type == int(references.FrigateVehicle) {
		vehicle.Hull = int(game.ActiveObjects[0].Number)
		if vehicle.Hull < 1 || vehicle.Hull > defaultShipHull {
			vehicle.Hull = defaultShipHull
			addConversionWarning(
				warnings,
				"partyVehicle.hull",
				"invalid_source_value",
				"boarded frigate slot 0 has invalid hull state and was reset to 99",
			)
		}
		vehicle.Skiffs = int(game.ActiveObjects[0].Misc & originalSkiffCountMask)
		if vehicle.Skiffs > references.MaxFrigateSkiffs {
			vehicle.Skiffs = references.MaxFrigateSkiffs
			addConversionWarning(
				warnings,
				"partyVehicle.skiffs",
				"redux_capacity_clamped",
				fmt.Sprintf(
					"original frigate skiffs exceeded the Redux capacity and were clamped to %d",
					references.MaxFrigateSkiffs,
				),
			)
		}
		if game.Ship.Sailing != 0 {
			vehicle.SailState = 1
		}
	}

	return vehicle
}

func originalVehicleIdentity(shape byte) (int, int, bool) {
	switch {
	case shape == originalRiderShape || shape == originalRiderShape+1:
		return int(references.HorseVehicle), horizontalShapeDirection(shape), true
	case shape == originalCarpetShape || shape == originalCarpetShape+1:
		return int(references.CarpetVehicle), horizontalShapeDirection(shape), true
	case shape&0xfc == originalSkiffShape:
		return int(references.SkiffVehicle), originalDirection(shape & originalDirectionMask), true
	case shape&0xfc == originalShipShape || shape&0xfc == originalAnchorShape:
		return int(references.FrigateVehicle), originalDirection(shape & originalDirectionMask), true
	default:
		return 0, 0, false
	}
}

func expandNPCFlags(region [128]byte) []save_snapshot.NPCFlagSnapshot {
	flags := make([]save_snapshot.NPCFlagSnapshot, 0)
	for location := range 32 {
		for npc := range 32 {
			byteIndex := location*npcFlagBytesPerLocation + npc/bitsPerByte
			if region[byteIndex]&(byte(1)<<uint(npc%bitsPerByte)) != 0 {
				flags = append(flags, save_snapshot.NPCFlagSnapshot{
					Location: location,
					NPC:      npc,
				})
			}
		}
	}

	return flags
}

func copyByteQuantities(destination []uint16, source []byte) {
	for index, quantity := range source {
		destination[index] = uint16(quantity)
	}
}

func uniqueItemQuantity(source byte) uint16 {
	if source == 0 {
		return 0
	}

	return 1
}

func signedFloor(source byte) int {
	if source == originalNothing {
		return -1
	}

	return int(source)
}

func horizontalShapeDirection(shape byte) int {
	if shape&1 == 0 {
		return int(references.Right)
	}

	return int(references.Left)
}

func originalDirection(direction byte) int {
	switch direction {
	case originalDirectionNorth:
		return int(references.Up)
	case originalDirectionEast:
		return int(references.Right)
	case originalDirectionSouth:
		return int(references.Down)
	case originalDirectionWest:
		return int(references.Left)
	default:
		panic("original direction is outside its two-bit representation")
	}
}

func sourceSeed(bundle Bundle) uint64 {
	digest := bundle.sourceDigest
	if digest == ([sha256.Size]byte{}) {
		canonical, err := json.Marshal(canonicalBundle{ //nolint:musttag // Private hash input, not an external JSON contract.
			Game: bundle.Game, Objects: bundle.Objects,
		})
		if err != nil {
			panic("legacy save canonical seed encoding failed: " + err.Error())
		}
		digest = sha256.Sum256(canonical)
	}

	return binary.LittleEndian.Uint64(digest[:8])
}

func addConversionWarning(warnings *[]ConversionWarning, path, code, message string) {
	*warnings = append(*warnings, ConversionWarning{Path: path, Code: code, Message: message})
}

func sortConversionWarnings(warnings []ConversionWarning) {
	sort.Slice(warnings, func(left, right int) bool {
		if warnings[left].Path != warnings[right].Path {
			return warnings[left].Path < warnings[right].Path
		}
		if warnings[left].Code != warnings[right].Code {
			return warnings[left].Code < warnings[right].Code
		}

		return warnings[left].Message < warnings[right].Message
	})
}
