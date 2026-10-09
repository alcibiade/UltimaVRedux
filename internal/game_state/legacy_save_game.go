package game_state

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const savedGamFileSize = 4192

var ErrInvalidSaveGameSize = errors.New("invalid save game file size, expected 4192 bytes")

type (
	StartingMemoryAddressUb  uint16
	StartingMemoryAddressU16 uint16
)

// ReadLegacySaveFile reads and validates an original 4192-byte SAVED.GAM file.
//
//nolint:nonamedreturns // Deferred close preserves validation failures and rejects an unsuccessful close.
func ReadLegacySaveFile(savedGamFilePath string) (data []byte, err error) {
	file, err := os.Open(savedGamFilePath)
	if err != nil {
		return nil, fmt.Errorf("opening legacy save file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			data = nil
			err = errors.Join(err, fmt.Errorf("closing legacy save file: %w", closeErr))
		}
	}()

	buffer := make([]byte, savedGamFileSize)
	bytesRead, err := io.ReadFull(file, buffer)
	if err != nil {
		return nil, fmt.Errorf("%w: got %d bytes: %w", ErrInvalidSaveGameSize, bytesRead, err)
	}

	var extra [1]byte
	extraBytes, err := io.ReadFull(file, extra[:])
	if err == nil || extraBytes != 0 {
		return nil, fmt.Errorf("%w: got more than %d bytes", ErrInvalidSaveGameSize, savedGamFileSize)
	}
	if !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("checking legacy save file size: %w", err)
	}

	return buffer, nil
}

func (g *GameState) LoadLegacySaveGameFromBytes(rawSaveData []byte) error {
	if !g.CanAcceptGameplayCommands() {
		return fmt.Errorf("loading legacy save during Lord British finale: %w", ErrSaveSnapshotNotRepresentable)
	}
	if len(rawSaveData) != savedGamFileSize {
		return fmt.Errorf("%w: got %d bytes", ErrInvalidSaveGameSize, len(rawSaveData))
	}

	// var saveGame = GameState{}
	g.RawSave = [savedGamFileSize]byte(rawSaveData)

	// Overlay player characters over memory rawSaveData to easily consume data
	g.PartyState = *party_state.LoadFromRaw(g.RawSave)
	const preparedSpellsOffset = 0x024a
	for spell := references.InLor; spell <= references.AnTym; spell++ {
		g.PartyState.Inventory.Spells.Set(spell, uint16(rawSaveData[preparedSpellsOffset+spell.ID()]))
	}

	// world and position
	const lbLocation = 0x2ED
	const lbX = 0x2F0
	const lbY = 0x2F1
	const lbFloor = 0x2EF
	g.MapState.PlayerLocation.Location = references.Location(rawSaveData[lbLocation])
	g.MapState.PlayerLocation.Position = references.Position{X: references.Coordinate(rawSaveData[lbX]), Y: references.Coordinate(rawSaveData[lbY])}
	g.MapState.PlayerLocation.Floor = references.FloorNumber(rawSaveData[lbFloor])

	// Date/Time
	const lsYear = 0x2CE
	const lbMonth = 0x2D7
	const lbDay = 0x2D8
	const lbHour = 0x2D9
	const lbMinute = 0x2DB
	g.DateTime.Year = getUint16(&rawSaveData, lsYear)
	g.DateTime.Month = rawSaveData[lbMonth]
	g.DateTime.Day = rawSaveData[lbDay]
	g.DateTime.Hour = rawSaveData[lbHour]
	g.DateTime.Minute = rawSaveData[lbMinute]
	g.combatActionTimeCounter = int(rawSaveData[0x02dc])
	g.openingCampaignEncounterReduction = rawSaveData[0x03b3] != 0

	// Various Things
	const lbKarma = 0x2e2
	const lbLastGave = 0x2e5
	const lsGold = 0x204
	const lbSmashed = 0x03b1
	g.PartyState.Karma = party_state.NewKarma(rawSaveData[lbKarma])
	g.beggarKarmaCounter = int(rawSaveData[lbLastGave])
	g.PartyState.Inventory.Gold.Set(getUint16(&rawSaveData, lsGold))
	g.drunk = 0
	g.smashed = int(rawSaveData[lbSmashed])
	g.isCriminal = false
	g.hmsCapeAutomaticWaitActorSkip = false

	const (
		shadowlordLordsOffset = 0x0322
		lastCalledOffset      = 0x0325
		originalNothing       = 0xff
	)
	g.ShadowlordState = references.NewShadowlordState()
	assignedTowns := make(map[byte]struct{}, references.ShadowlordCount)
	for index, marker := range rawSaveData[shadowlordLordsOffset : shadowlordLordsOffset+references.ShadowlordCount] {
		switch {
		case marker == originalNothing:
			g.ShadowlordState.Alive[index] = false
		case marker >= ShadowlordTownMinLocation && marker <= ShadowlordTownMaxLocation:
			if _, duplicate := assignedTowns[marker]; duplicate {
				logging.Warn(logging.CategoryGameState, "legacy Shadowlord %d duplicates town marker %d; treating as unassigned", index, marker)

				continue
			}
			assignedTowns[marker] = struct{}{}
			g.ShadowlordState.TownAssignments[index] = references.Location(marker)
		case marker != 0:
			logging.Warn(logging.CategoryGameState, "legacy Shadowlord %d has unsupported town marker %d; treating as unassigned", index, marker)
		}
	}
	lastCalled := rawSaveData[lastCalledOffset]
	if lastCalled < references.ShadowlordCount {
		g.ShadowlordState.LastCalled = references.Shadowlord(lastCalled)
	}

	// ProvisionsQuantity
	const lsFood = 0x202
	const lbKeys = 0x206
	const lbGems = 0x207
	const lbTorches = 0x208
	const lbSkullKeys = 0x20B

	g.PartyState.Inventory.Provisions.Food.Set(getUint16(&rawSaveData, lsFood))
	g.PartyState.Inventory.Provisions.Gems.Set(uint16(rawSaveData[lbGems]))
	g.PartyState.Inventory.Provisions.Torches.Set(uint16(rawSaveData[lbTorches]))
	g.PartyState.Inventory.Provisions.Keys.Set(uint16(rawSaveData[lbKeys]))
	g.PartyState.Inventory.Provisions.SkullKeys.Set(uint16(rawSaveData[lbSkullKeys]))

	// Special Items (offsets 0x0214-0x0219)
	const (
		lbSpyglass    = 0x0214
		lbHMSCape     = 0x0215 // "Plans" in original - the HMS Cape plans
		lbSextant     = 0x0216
		lbPocketWatch = 0x0217
		lbBlackBadge  = 0x0218
		lbWoodenBox   = 0x0219
	)

	if rawSaveData[lbSpyglass] > 0 {
		g.PartyState.Inventory.QuestItems.SetHasOne(references.Spyglass)
	}

	if rawSaveData[lbHMSCape] > 0 {
		g.PartyState.Inventory.QuestItems.SetHasOne(references.HMSCape)
	}
	g.hmsCapeActivated = rawSaveData[lbHMSCape]&0x80 != 0

	if rawSaveData[lbSextant] > 0 {
		g.PartyState.Inventory.QuestItems.SetHasOne(references.Sextant)
	}

	if rawSaveData[lbPocketWatch] > 0 {
		g.PartyState.Inventory.QuestItems.SetHasOne(references.PocketWatch)
	}

	if rawSaveData[lbBlackBadge] > 0 {
		g.PartyState.Inventory.QuestItems.SetHasOne(references.BlackBadge)
	}

	if rawSaveData[lbWoodenBox] > 0 {
		g.PartyState.Inventory.QuestItems.SetHasOne(references.WoodenBox)
		g.PartyState.SetDeadNpc(references.LordBritishsCastle, lordBritishSandalwoodBoxNPC)
		g.lordBritishPassageOpen = true
	}

	g.MapState.LayeredMaps = *map_state.NewLayeredMaps(g.GameReferences.TileReferences,
		g.GameReferences.OverworldLargeMapReference,
		g.GameReferences.UnderworldLargeMapReference,
		g.MapState.XTilesVisibleOnGameScreen,
		g.MapState.YTilesVisibleOnGameScreen)

	g.LargeMapNPCAIController = make(map[references.World]*ai.NPCAIControllerLargeMap)
	overworldNPCAIInput := ai.NewNPCAIControllerLargeMapInput{
		World:           references.OVERWORLD,
		TileRefs:        g.GameReferences.TileReferences,
		MapState:        &g.MapState,
		DebugOptions:    &g.DebugOptions,
		TheOdds:         &g.TheOdds,
		EnemyReferences: g.GameReferences.EnemyReferences,
		DateTime:        &g.DateTime,
		WindProvider:    g,
		RandomProvider:  g.AsRandomProvider(),
	}
	g.LargeMapNPCAIController[references.OVERWORLD] = ai.NewNPCAIControllerLargeMap(overworldNPCAIInput)
	// TODO: load the npcs from the save game
	underworldNPCAIInput := ai.NewNPCAIControllerLargeMapInput{
		World:           references.UNDERWORLD,
		TileRefs:        g.GameReferences.TileReferences,
		MapState:        &g.MapState,
		DebugOptions:    &g.DebugOptions,
		TheOdds:         &g.TheOdds,
		EnemyReferences: g.GameReferences.EnemyReferences,
		DateTime:        &g.DateTime,
		WindProvider:    g,
		RandomProvider:  g.AsRandomProvider(),
	}
	g.LargeMapNPCAIController[references.UNDERWORLD] = ai.NewNPCAIControllerLargeMap(underworldNPCAIInput)
	// TODO: load the npcs from the save game

	switch g.MapState.PlayerLocation.Location.GetMapType() {
	case references.LargeMapType:
		g.CurrentNPCAIController = g.GetCurrentLargeMapNPCAIController()
		// Place visible quest items on large maps during game initialization
		g.RefreshQuestItemsOnMap()
	case references.SmallMapType:
		g.UpdateSmallMap(g.GameReferences.TileReferences, g.GameReferences.LocationReferences)
	case references.CombatMapType, references.DungeonMapType, references.CampingMapType:
		panic("legacy save cannot initialize transient map controller")
	}

	g.ItemStacksMap = *references.NewItemStacksMap()

	g.TheOdds = references.NewDefaultTheOdds()
	g.DebugOptions = references.DebugOptions{
		FreeMove:   false,
		MonsterGen: true,
	}

	const lbActivePlayer = 0x02d5
	legacyActivePlayer := int(rawSaveData[lbActivePlayer])
	if legacyActivePlayer == 0xff {
		legacyActivePlayer = -1
	}
	if !g.SetActivePlayer(legacyActivePlayer) {
		logging.Warn(logging.CategoryGameState, "legacy active player %d is unavailable; clearing selection", legacyActivePlayer)
		g.activePlayerIndex = -1
	}

	// Load moonstone data from SAVED.GAM
	const stonesXOffset = 0x028A
	const stonesYOffset = 0x0292
	const stonesMOffset = 0x029A
	const stonesLOffset = 0x02A2
	const objFoundOffset = 0x02B6
	const skullKeysDayOffset = 0x020C
	const reagentCacheDaysOffset = 0x02B2

	stonesX := rawSaveData[stonesXOffset : stonesXOffset+8]
	stonesY := rawSaveData[stonesYOffset : stonesYOffset+8]
	stonesM := rawSaveData[stonesMOffset : stonesMOffset+8]
	stonesL := rawSaveData[stonesLOffset : stonesLOffset+8]
	objFound := rawSaveData[objFoundOffset : objFoundOffset+24]

	g.moonstoneSystem.LoadFromSaveData(stonesX, stonesY, stonesM, stonesL)
	var found [24]byte
	copy(found[:], objFound)
	for bit := references.SearchLocationCount; bit < len(found)*8; bit++ {
		found[bit/8] &^= 1 << (bit & 7)
	}
	var reagentDays [3]byte
	copy(reagentDays[:], rawSaveData[reagentCacheDaysOffset:reagentCacheDaysOffset+len(reagentDays)])
	g.SearchManager.restoreState(found, rawSaveData[skullKeysDayOffset], reagentDays)
	g.restoreSurfaceCorpsesForCurrentMap()

	// Initialize dungeon states from save file data
	g.InitializeDungeonStatesFromSave(rawSaveData)

	// Initialize lastProcessedHour from loaded DateTime for food consumption tracking.
	// Matches original U5 original party-effect rules "last processed hour = hour" pattern.
	// See ref:SAVE-0004.
	g.lastProcessedHour = int(g.DateTime.Hour)
	g.initializeMoongateStage()
	g.MagicState = NewMagicState()
	g.playerKnowledge = newPlayerKnowledge()
	g.inferMigratedHiddenSpellKnowledge()
	g.questJournal = newQuestJournal()
	g.inferMigratedQuestJournal()

	return nil
}

func getBytesAsUint16(data0, data1 byte) uint16 {
	res := uint16(data0) | (uint16(data1) << 8)

	return res
}

func getUint16(bytes *[]byte, address uint16) uint16 {
	return getBytesAsUint16((*bytes)[address], (*bytes)[address+1])
}
