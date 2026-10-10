package save_snapshot

// saveSnapshotV12 and its nested DTOs freeze the complete version 12 wire
// contract independently from the mutable current snapshot types.
type saveSnapshotV12 struct {
	Version      int                      `json:"version"`
	Party        partySnapshotV12         `json:"party"`
	Inventory    inventorySnapshotV12     `json:"inventory"`
	World        worldSnapshotV12         `json:"world"`
	Progression  progressionSnapshotV12   `json:"progression"`
	Effects      effectsSnapshotV12       `json:"effects"`
	Wind         windSnapshotV12          `json:"wind"`
	Random       randomSnapshotV12        `json:"random"`
	Redux        reduxSnapshotV12         `json:"redux"`
	Dungeon      *dungeonSnapshotV12      `json:"dungeon"`
	PartyVehicle *vehicleSnapshotV12      `json:"partyVehicle"`
	MapUnits     []mapUnitSnapshotV12     `json:"mapUnits"`
	ItemStacks   []itemStackSnapshotV12   `json:"itemStacks"`
	MapMutations []mapMutationSnapshotV12 `json:"mapMutations"`
}

type dungeonSnapshotV12 struct {
	Location          int                         `json:"location"`
	Cells             []int                       `json:"cells"`
	Facing            int                         `json:"facing"`
	LastDirection     int                         `json:"lastDirection"`
	Wanderer          *dungeonWandererSnapshotV12 `json:"wanderer"`
	QuicknessSkipNext bool                        `json:"quicknessSkipNext"`
}

type dungeonWandererSnapshotV12 struct {
	Monster   int  `json:"monster"`
	Floor     int  `json:"floor"`
	X         int  `json:"x"`
	Y         int  `json:"y"`
	PreviousX int  `json:"previousX"`
	PreviousY int  `json:"previousY"`
	OnCeiling bool `json:"onCeiling"`
}

type partySnapshotV12 struct {
	Roster   [RosterSlotCount]characterSnapshotV12 `json:"roster"`
	Karma    int                                   `json:"karma"`
	MetNPCs  []npcFlagSnapshotV12                  `json:"metNpcs"`
	DeadNPCs []npcFlagSnapshotV12                  `json:"deadNpcs"`
}

type npcFlagSnapshotV12 struct {
	Location int `json:"location"`
	NPC      int `json:"npc"`
}

type characterSnapshotV12 struct {
	Occupied     bool   `json:"occupied"`
	Name         string `json:"name"`
	Gender       int    `json:"gender"`
	Class        int    `json:"class"`
	Status       int    `json:"status"`
	Strength     int    `json:"strength"`
	Dexterity    int    `json:"dexterity"`
	Intelligence int    `json:"intelligence"`
	CurrentMP    int    `json:"currentMp"`
	CurrentHP    int    `json:"currentHp"`
	MaximumHP    int    `json:"maximumHp"`
	Experience   int    `json:"experience"`
	Level        int    `json:"level"`
	MonthsAtInn  int    `json:"monthsAtInn"`
	ArmorClass   int    `json:"armorClass"`
	Helmet       int    `json:"helmet"`
	Armor        int    `json:"armor"`
	Weapon       int    `json:"weapon"`
	Shield       int    `json:"shield"`
	Ring         int    `json:"ring"`
	Amulet       int    `json:"amulet"`
	PartyMap     int    `json:"partyMap"`
}

type inventorySnapshotV12 struct {
	Food       uint16                  `json:"food"`
	Gold       uint16                  `json:"gold"`
	Gems       uint16                  `json:"gems"`
	Torches    uint16                  `json:"torches"`
	Keys       uint16                  `json:"keys"`
	SkullKeys  uint16                  `json:"skullKeys"`
	Equipment  [EquipmentCount]uint16  `json:"equipment"`
	Spells     [SpellCount]uint16      `json:"spells"`
	Scrolls    [ConsumableCount]uint16 `json:"scrolls"`
	Potions    [ConsumableCount]uint16 `json:"potions"`
	Reagents   [ConsumableCount]uint16 `json:"reagents"`
	QuestItems [QuestItemCount]uint16  `json:"questItems"`
	Moonstones [MoonstoneCount]uint16  `json:"moonstones"`
}

type worldSnapshotV12 struct {
	Position             positionSnapshotV12 `json:"position"`
	LastLargeMapPosition positionSnapshotV12 `json:"lastLargeMapPosition"`
	Calendar             calendarSnapshotV12 `json:"calendar"`
	ElapsedMs            int64               `json:"elapsedMs"`
	LastProcessedHour    int                 `json:"lastProcessedHour"`
	LastCamped           int                 `json:"lastCamped"`
	LastWandererMonth    int                 `json:"lastWandererMonth"`
	CombatSequence       uint64              `json:"combatSequence"`
	IsSailing            bool                `json:"isSailing"`
}

type positionSnapshotV12 struct {
	Location int `json:"location"`
	X        int `json:"x"`
	Y        int `json:"y"`
	Floor    int `json:"floor"`
}

type calendarSnapshotV12 struct {
	Year   int    `json:"year"`
	Month  int    `json:"month"`
	Day    int    `json:"day"`
	Hour   int    `json:"hour"`
	Minute int    `json:"minute"`
	Turns  uint32 `json:"turns"`
}

type progressionSnapshotV12 struct {
	SearchFound           [SearchBitmaskBytes]byte             `json:"searchFound"`
	SkullKeysFoundDay     byte                                 `json:"skullKeysFoundDay"`
	ReagentCachesFoundDay [3]byte                              `json:"reagentCachesFoundDay"`
	Moonstones            [MoonstoneCount]moonstoneSnapshotV12 `json:"moonstones"`
	Shrines               [ShrineCount]shrineSnapshotV12       `json:"shrines"`
	Dungeons              [DungeonCount]bool                   `json:"dungeons"`
	DungeonRooms          [DungeonRoomProgressBytes]byte       `json:"dungeonRooms"`
	Shadowlords           shadowlordsSnapshotV12               `json:"shadowlords"`
	Recipes               [SpellCount]recipeSnapshotV12        `json:"recipes"`
	HMSCapeActivated      bool                                 `json:"hmsCapeActivated"`
}

type recipeSnapshotV12 struct {
	Known bool   `json:"known"`
	Mask  uint16 `json:"mask"`
}

type moonstoneSnapshotV12 struct {
	X       byte `json:"x"`
	Y       byte `json:"y"`
	Map     byte `json:"map"`
	Level   byte `json:"level"`
	State   byte `json:"state"`
	IsFound bool `json:"isFound"`
}

type shrineSnapshotV12 struct {
	Active    bool `json:"active"`
	Completed bool `json:"completed"`
	Ruined    bool `json:"ruined"`
}

type shadowlordsSnapshotV12 struct {
	Alive             [ShadowlordCount]bool `json:"alive"`
	TownAssignments   [ShadowlordCount]int  `json:"townAssignments"`
	LastCalled        int                   `json:"lastCalled"`
	Summoned          bool                  `json:"summoned"`
	AutomaticPresence int                   `json:"automaticPresence"`
}

type effectsSnapshotV12 struct {
	DurationType                 byte              `json:"durationType"`
	DurationTurns                int               `json:"durationTurns"`
	DurationToggleOnly           bool              `json:"durationToggleOnly"`
	MagicLightMinutesRemaining   int               `json:"magicLightMinutesRemaining"`
	CarriedTorchMinutesRemaining int               `json:"carriedTorchMinutesRemaining"`
	QuicknessWorldActorSkip      bool              `json:"quicknessWorldActorSkip"`
	Amulet                       toggleSnapshotV12 `json:"amulet"`
	Crown                        toggleSnapshotV12 `json:"crown"`
	Badge                        toggleSnapshotV12 `json:"badge"`
	Drunk                        int               `json:"drunk"`
	Smashed                      int               `json:"smashed"`
}

type toggleSnapshotV12 struct {
	Active bool `json:"active"`
	Turns  int  `json:"turns"`
}

type windSnapshotV12 struct {
	Direction     int   `json:"direction"`
	UpdateCounter int   `json:"updateCounter"`
	NextTickMs    int64 `json:"nextTickMs"`
}

type randomSnapshotV12 struct {
	Seed        uint64 `json:"seed"`
	SourceCalls uint64 `json:"sourceCalls"`
}

type reduxSnapshotV12 struct {
	ReconstructMapUnits    bool                           `json:"reconstructMapUnits"`
	DungeonDiscoveries     []dungeonDiscoverySnapshotV12  `json:"dungeonDiscoveries"`
	PlayerKnowledge        []playerKnowledgeSnapshotV12   `json:"playerKnowledge"`
	QuestJournal           []questJournalEventSnapshotV12 `json:"questJournal"`
	LordBritishPassageOpen bool                           `json:"lordBritishPassageOpen"`
}

type questJournalEventSnapshotV12 struct {
	Sequence     uint64                     `json:"sequence"`
	Kind         string                     `json:"kind"`
	Subject      string                     `json:"subject"`
	OccurredOn   *britannianDateSnapshotV12 `json:"occurredOn"`
	Source       string                     `json:"source"`
	Acknowledged bool                       `json:"acknowledged"`
}

type playerKnowledgeSnapshotV12 struct {
	Category     string                            `json:"category"`
	Key          string                            `json:"key"`
	Source       knowledgeSourceSnapshotV12        `json:"source"`
	DiscoveredOn *britannianDateSnapshotV12        `json:"discoveredOn"`
	Unread       bool                              `json:"unread"`
	SpellFormula *spellFormulaKnowledgeSnapshotV12 `json:"spellFormula"`
}

type knowledgeSourceSnapshotV12 struct {
	Kind        string `json:"kind"`
	DisplayName string `json:"displayName"`
}

type britannianDateSnapshotV12 struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	Day   int `json:"day"`
}

type spellFormulaKnowledgeSnapshotV12 struct {
	State          string   `json:"state"`
	ConfirmedMask  uint16   `json:"confirmedMask"`
	CandidateMasks []uint16 `json:"candidateMasks"`
}

type dungeonDiscoverySnapshotV12 struct {
	Location int    `json:"location"`
	Floor    int    `json:"floor"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Kind     string `json:"kind"`
}

type vehicleSnapshotV12 struct {
	Type              int  `json:"type"`
	CurrentDirection  int  `json:"currentDirection"`
	PreviousDirection int  `json:"previousDirection"`
	Skiffs            int  `json:"skiffs"`
	OwnedByNPC        bool `json:"ownedByNpc"`
	Hull              int  `json:"hull"`
	SailState         int  `json:"sailState"`
	HasHMSCape        bool `json:"hasHmsCape"`
	PendingDelivery   bool `json:"pendingDelivery"`
}

type mapUnitSnapshotV12 struct {
	World                     int                   `json:"world"`
	ControllerOrder           int                   `json:"controllerOrder"`
	Kind                      string                `json:"kind"`
	ReferenceID               int                   `json:"referenceId"`
	NPCNum                    int                   `json:"npcNum"`
	Position                  positionSnapshotV12   `json:"position"`
	Visible                   bool                  `json:"visible"`
	SkipNextTurn              bool                  `json:"skipNextTurn"`
	WantsToAutoTalk           bool                  `json:"wantsToAutoTalk"`
	OverriddenAI              int                   `json:"overriddenAi"`
	AIOverrideProvenance      string                `json:"aiOverrideProvenance"`
	ArrivedScheduleSlot       *int                  `json:"arrivedScheduleSlot,omitempty"`
	ConsumedTalk3ScheduleSlot *int                  `json:"consumedTalk3ScheduleSlot,omitempty"`
	CurrentPath               []positionSnapshotV12 `json:"currentPath"`
	Vehicle                   *vehicleSnapshotV12   `json:"vehicle"`
	EnemyVariant              string                `json:"enemyVariant"`
	EnemyCurrentHP            int                   `json:"enemyCurrentHp"`
	EnemyFacing               int                   `json:"enemyFacing"`
	Shadowlord                *int                  `json:"shadowlord"`
	SurfaceChestNumber        *int                  `json:"surfaceChestNumber,omitempty"`
}

type itemStackSnapshotV12 struct {
	Position positionSnapshotV12 `json:"position"`
	Items    []itemSnapshotV12   `json:"items"`
}

type itemSnapshotV12 struct {
	Type     int    `json:"type"`
	ID       int    `json:"id"`
	Quantity uint16 `json:"quantity"`
}

type mapMutationSnapshotV12 struct {
	Position positionSnapshotV12 `json:"position"`
	Kind     string              `json:"kind"`
	Value    int                 `json:"value"`
}

func (snapshot saveSnapshotV12) promote() SaveSnapshot {
	promoted := snapshot.toCurrent()
	promoted.Version = CurrentVersion
	promoted.Continuation = ContinuationSnapshot{
		ActivePlayerIndex:                 -1,
		IsCriminal:                        false,
		HMSCapeAutomaticWaitActorSkip:     false,
		BeggarKarmaCounter:                100,
		OpeningCampaignEncounterReduction: snapshot.World.Calendar.Year == 139 && snapshot.World.Calendar.Month == 4,
	}
	promoted.promoteHistoricalSurfaceCadence()
	promoted.promoteHistoricalJournal()

	return promoted
}

func (snapshot saveSnapshotV12) validationView() SaveSnapshot {
	return snapshot.toCurrent()
}

func (snapshot saveSnapshotV12) toCurrent() SaveSnapshot {
	promoted := SaveSnapshot{
		Version:      snapshot.Version,
		Party:        promotePartyV12(snapshot.Party),
		Inventory:    promoteInventoryV12(snapshot.Inventory),
		World:        promoteWorldV12(snapshot.World),
		Progression:  promoteProgressionV12(snapshot.Progression),
		Effects:      promoteEffectsV12(snapshot.Effects),
		Wind:         WindSnapshot{Direction: snapshot.Wind.Direction, UpdateCounter: snapshot.Wind.UpdateCounter, NextTickMs: snapshot.Wind.NextTickMs},
		Random:       RandomSnapshot{Seed: snapshot.Random.Seed, SourceCalls: snapshot.Random.SourceCalls},
		Redux:        promoteReduxV12(snapshot.Redux),
		Dungeon:      promoteDungeonV12(snapshot.Dungeon),
		PartyVehicle: promoteVehicleV12(snapshot.PartyVehicle),
		MapUnits:     promoteMapUnitsV12(snapshot.MapUnits),
		ItemStacks:   promoteItemStacksV12(snapshot.ItemStacks),
		MapMutations: promoteMapMutationsV12(snapshot.MapMutations),
	}
	if snapshot.Redux.ReconstructMapUnits {
		promoted.MarkMapUnitsForReconstruction()
	}

	return promoted
}

//nolint:staticcheck // Explicit field mapping keeps the frozen v12 DTO independent from current types.
func promotePartyV12(party partySnapshotV12) PartySnapshot {
	promoted := PartySnapshot{
		Karma:    party.Karma,
		MetNPCs:  promoteNPCFlagsV12(party.MetNPCs),
		DeadNPCs: promoteNPCFlagsV12(party.DeadNPCs),
	}
	for index, character := range party.Roster {
		promoted.Roster[index] = CharacterSnapshot{
			Occupied: character.Occupied, Name: character.Name, Gender: character.Gender,
			Class: character.Class, Status: character.Status, Strength: character.Strength,
			Dexterity: character.Dexterity, Intelligence: character.Intelligence,
			CurrentMP: character.CurrentMP, CurrentHP: character.CurrentHP, MaximumHP: character.MaximumHP,
			Experience: character.Experience, Level: character.Level, MonthsAtInn: character.MonthsAtInn,
			ArmorClass: character.ArmorClass, Helmet: character.Helmet, Armor: character.Armor,
			Weapon: character.Weapon, Shield: character.Shield, Ring: character.Ring,
			Amulet: character.Amulet, PartyMap: character.PartyMap,
		}
	}

	return promoted
}

//nolint:staticcheck // Explicit field mapping keeps the frozen v12 DTO independent from current types.
func promoteNPCFlagsV12(flags []npcFlagSnapshotV12) []NPCFlagSnapshot {
	if flags == nil {
		return nil
	}
	promoted := make([]NPCFlagSnapshot, len(flags))
	for index, flag := range flags {
		promoted[index] = NPCFlagSnapshot{Location: flag.Location, NPC: flag.NPC}
	}

	return promoted
}

//nolint:staticcheck // Explicit field mapping keeps the frozen v12 DTO independent from current types.
func promoteInventoryV12(inventory inventorySnapshotV12) InventorySnapshot {
	return InventorySnapshot{
		Food: inventory.Food, Gold: inventory.Gold, Gems: inventory.Gems,
		Torches: inventory.Torches, Keys: inventory.Keys, SkullKeys: inventory.SkullKeys,
		Equipment: inventory.Equipment, Spells: inventory.Spells, Scrolls: inventory.Scrolls,
		Potions: inventory.Potions, Reagents: inventory.Reagents, QuestItems: inventory.QuestItems,
		Moonstones: inventory.Moonstones,
	}
}

func promoteWorldV12(world worldSnapshotV12) WorldSnapshot {
	return WorldSnapshot{
		Position: promotePositionV12(world.Position), LastLargeMapPosition: promotePositionV12(world.LastLargeMapPosition),
		Calendar: CalendarSnapshot{
			Year: world.Calendar.Year, Month: world.Calendar.Month, Day: world.Calendar.Day,
			Hour: world.Calendar.Hour, Minute: world.Calendar.Minute, Turns: world.Calendar.Turns,
		},
		ElapsedMs: world.ElapsedMs, LastProcessedHour: world.LastProcessedHour,
		LastCamped: world.LastCamped, LastWandererMonth: world.LastWandererMonth,
		CombatSequence: world.CombatSequence, IsSailing: world.IsSailing,
	}
}

//nolint:staticcheck // Explicit field mapping keeps the frozen v12 DTO independent from current types.
func promotePositionV12(position positionSnapshotV12) PositionSnapshot {
	return PositionSnapshot{Location: position.Location, X: position.X, Y: position.Y, Floor: position.Floor}
}

//nolint:staticcheck // Explicit field mapping keeps the frozen v12 DTO independent from current types.
func promoteProgressionV12(progression progressionSnapshotV12) ProgressionSnapshot {
	promoted := ProgressionSnapshot{
		SearchFound: progression.SearchFound, SkullKeysFoundDay: progression.SkullKeysFoundDay,
		ReagentCachesFoundDay: progression.ReagentCachesFoundDay, Dungeons: progression.Dungeons,
		DungeonRooms: progression.DungeonRooms,
		Shadowlords: ShadowlordsSnapshot{
			Alive: progression.Shadowlords.Alive, TownAssignments: progression.Shadowlords.TownAssignments,
			LastCalled: progression.Shadowlords.LastCalled, Summoned: progression.Shadowlords.Summoned,
			AutomaticPresence: progression.Shadowlords.AutomaticPresence,
		},
		HMSCapeActivated: progression.HMSCapeActivated,
	}
	for index, moonstone := range progression.Moonstones {
		promoted.Moonstones[index] = MoonstoneSnapshot{
			X: moonstone.X, Y: moonstone.Y, Map: moonstone.Map, Level: moonstone.Level,
			State: moonstone.State, IsFound: moonstone.IsFound,
		}
	}
	for index, shrine := range progression.Shrines {
		promoted.Shrines[index] = ShrineSnapshot{Active: shrine.Active, Completed: shrine.Completed, Ruined: shrine.Ruined}
	}
	for index, recipe := range progression.Recipes {
		promoted.Recipes[index] = RecipeSnapshot{Known: recipe.Known, Mask: recipe.Mask}
	}

	return promoted
}

func promoteEffectsV12(effects effectsSnapshotV12) EffectsSnapshot {
	return EffectsSnapshot{
		DurationType: effects.DurationType, DurationTurns: effects.DurationTurns,
		DurationToggleOnly:           effects.DurationToggleOnly,
		MagicLightMinutesRemaining:   effects.MagicLightMinutesRemaining,
		CarriedTorchMinutesRemaining: effects.CarriedTorchMinutesRemaining,
		QuicknessWorldActorSkip:      effects.QuicknessWorldActorSkip,
		Amulet:                       ToggleSnapshot{Active: effects.Amulet.Active, Turns: effects.Amulet.Turns},
		Crown:                        ToggleSnapshot{Active: effects.Crown.Active, Turns: effects.Crown.Turns},
		Badge:                        ToggleSnapshot{Active: effects.Badge.Active, Turns: effects.Badge.Turns},
		Drunk:                        effects.Drunk, Smashed: effects.Smashed,
	}
}

func promoteReduxV12(redux reduxSnapshotV12) ReduxSnapshot {
	return ReduxSnapshot{
		DungeonDiscoveries:     promoteDungeonDiscoveriesV12(redux.DungeonDiscoveries),
		PlayerKnowledge:        promotePlayerKnowledgeV12(redux.PlayerKnowledge),
		QuestJournal:           promoteQuestJournalV12(redux.QuestJournal),
		LordBritishPassageOpen: redux.LordBritishPassageOpen,
	}
}

//nolint:staticcheck // Explicit field mapping keeps the frozen v12 DTO independent from current types.
func promoteDungeonDiscoveriesV12(discoveries []dungeonDiscoverySnapshotV12) []DungeonDiscoverySnapshot {
	if discoveries == nil {
		return nil
	}
	promoted := make([]DungeonDiscoverySnapshot, len(discoveries))
	for index, discovery := range discoveries {
		promoted[index] = DungeonDiscoverySnapshot{
			Location: discovery.Location, Floor: discovery.Floor, X: discovery.X, Y: discovery.Y, Kind: discovery.Kind,
		}
	}

	return promoted
}

func promotePlayerKnowledgeV12(entries []playerKnowledgeSnapshotV12) []PlayerKnowledgeSnapshot {
	if entries == nil {
		return nil
	}
	promoted := make([]PlayerKnowledgeSnapshot, len(entries))
	for index, entry := range entries {
		promoted[index] = PlayerKnowledgeSnapshot{
			Category: entry.Category, Key: entry.Key,
			Source:       KnowledgeSourceSnapshot{Kind: entry.Source.Kind, DisplayName: entry.Source.DisplayName},
			DiscoveredOn: promoteBritannianDateV12(entry.DiscoveredOn), Unread: entry.Unread,
			SpellFormula: promoteSpellFormulaV12(entry.SpellFormula),
		}
	}

	return promoted
}

func promoteQuestJournalV12(events []questJournalEventSnapshotV12) []QuestJournalEventSnapshot {
	if events == nil {
		return nil
	}
	promoted := make([]QuestJournalEventSnapshot, len(events))
	for index, event := range events {
		promoted[index] = QuestJournalEventSnapshot{
			Sequence: event.Sequence, Kind: event.Kind, Subject: event.Subject,
			OccurredOn: promoteBritannianDateV12(event.OccurredOn), Source: event.Source,
			Acknowledged: event.Acknowledged,
		}
	}

	return promoted
}

func promoteBritannianDateV12(date *britannianDateSnapshotV12) *BritannianDateSnapshot {
	if date == nil {
		return nil
	}

	return &BritannianDateSnapshot{Year: date.Year, Month: date.Month, Day: date.Day}
}

func promoteSpellFormulaV12(formula *spellFormulaKnowledgeSnapshotV12) *SpellFormulaKnowledgeSnapshot {
	if formula == nil {
		return nil
	}
	var candidates []uint16
	if formula.CandidateMasks != nil {
		candidates = make([]uint16, len(formula.CandidateMasks))
		copy(candidates, formula.CandidateMasks)
	}

	return &SpellFormulaKnowledgeSnapshot{
		State: formula.State, ConfirmedMask: formula.ConfirmedMask, CandidateMasks: candidates,
	}
}

func promoteDungeonV12(dungeon *dungeonSnapshotV12) *DungeonSnapshot {
	if dungeon == nil {
		return nil
	}
	var cells []int
	if dungeon.Cells != nil {
		cells = make([]int, len(dungeon.Cells))
		copy(cells, dungeon.Cells)
	}
	var wanderer *DungeonWandererSnapshot
	if dungeon.Wanderer != nil {
		wanderer = &DungeonWandererSnapshot{
			Monster: dungeon.Wanderer.Monster, Floor: dungeon.Wanderer.Floor,
			X: dungeon.Wanderer.X, Y: dungeon.Wanderer.Y,
			PreviousX: dungeon.Wanderer.PreviousX, PreviousY: dungeon.Wanderer.PreviousY,
			OnCeiling: dungeon.Wanderer.OnCeiling,
		}
	}

	return &DungeonSnapshot{
		Location: dungeon.Location, Cells: cells, Facing: dungeon.Facing,
		LastDirection: dungeon.LastDirection, Wanderer: wanderer,
		QuicknessSkipNext: dungeon.QuicknessSkipNext,
	}
}

func promoteVehicleV12(vehicle *vehicleSnapshotV12) *VehicleSnapshot {
	if vehicle == nil {
		return nil
	}

	return &VehicleSnapshot{
		Type: vehicle.Type, CurrentDirection: vehicle.CurrentDirection,
		PreviousDirection: vehicle.PreviousDirection, Skiffs: vehicle.Skiffs,
		OwnedByNPC: vehicle.OwnedByNPC, Hull: vehicle.Hull, SailState: vehicle.SailState,
		HasHMSCape: vehicle.HasHMSCape, PendingDelivery: vehicle.PendingDelivery,
	}
}

func promoteMapUnitsV12(units []mapUnitSnapshotV12) []MapUnitSnapshot {
	if units == nil {
		return nil
	}
	promoted := make([]MapUnitSnapshot, len(units))
	for index, unit := range units {
		promoted[index] = MapUnitSnapshot{
			World: unit.World, ControllerOrder: unit.ControllerOrder, Kind: unit.Kind,
			ReferenceID: unit.ReferenceID, NPCNum: unit.NPCNum, Position: promotePositionV12(unit.Position),
			Visible: unit.Visible, SkipNextTurn: unit.SkipNextTurn, LegacyWantsToAutoTalk: unit.WantsToAutoTalk,
			OverriddenAI: unit.OverriddenAI, AIOverrideProvenance: unit.AIOverrideProvenance,
			ArrivedScheduleSlot:       promoteIntPointerV12(unit.ArrivedScheduleSlot),
			ConsumedTalk3ScheduleSlot: promoteIntPointerV12(unit.ConsumedTalk3ScheduleSlot),
			CurrentPath:               promotePositionsV12(unit.CurrentPath), Vehicle: promoteVehicleV12(unit.Vehicle),
			EnemyVariant: unit.EnemyVariant, EnemyCurrentHP: unit.EnemyCurrentHP,
			EnemyFacing: unit.EnemyFacing, Shadowlord: promoteIntPointerV12(unit.Shadowlord),
			SurfaceChestNumber: promoteIntPointerV12(unit.SurfaceChestNumber),
		}
	}

	return promoted
}

func promotePositionsV12(positions []positionSnapshotV12) []PositionSnapshot {
	if positions == nil {
		return nil
	}
	promoted := make([]PositionSnapshot, len(positions))
	for index, position := range positions {
		promoted[index] = promotePositionV12(position)
	}

	return promoted
}

func promoteIntPointerV12(value *int) *int {
	if value == nil {
		return nil
	}
	promoted := *value

	return &promoted
}

func promoteItemStacksV12(stacks []itemStackSnapshotV12) []ItemStackSnapshot {
	if stacks == nil {
		return nil
	}
	promoted := make([]ItemStackSnapshot, len(stacks))
	for index, stack := range stacks {
		promoted[index] = ItemStackSnapshot{
			Position: promotePositionV12(stack.Position),
			Items:    promoteItemsV12(stack.Items),
		}
	}

	return promoted
}

//nolint:staticcheck // Explicit field mapping keeps the frozen v12 DTO independent from current types.
func promoteItemsV12(items []itemSnapshotV12) []ItemSnapshot {
	if items == nil {
		return nil
	}
	promoted := make([]ItemSnapshot, len(items))
	for index, item := range items {
		promoted[index] = ItemSnapshot{Type: item.Type, ID: item.ID, Quantity: item.Quantity}
	}

	return promoted
}

func promoteMapMutationsV12(mutations []mapMutationSnapshotV12) []MapMutationSnapshot {
	if mutations == nil {
		return nil
	}
	promoted := make([]MapMutationSnapshot, len(mutations))
	for index, mutation := range mutations {
		promoted[index] = MapMutationSnapshot{
			Position: promotePositionV12(mutation.Position),
			Kind:     MapMutationKind(mutation.Kind),
			Value:    mutation.Value,
		}
	}

	return promoted
}
