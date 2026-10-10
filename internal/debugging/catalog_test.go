package debugging

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestInitialCatalogContainsReadOnlySlice(t *testing.T) {
	catalog := NewInitialCatalog()
	definitions := catalog.Definitions()

	require.Len(t, definitions, 156)
	assert.Equal(t, CommandScenarioRunCancel, definitions[155].ID)
	assert.Equal(t, []ExecutionMode{ExecutionLiveSafe}, definitions[155].ExecutionModes)
	assert.True(t, definitions[155].ScenarioExcluded)
	_, legacySavePublished := catalog.Get("scenario.definition.save")
	assert.False(t, legacySavePublished)
	assert.Equal(t, CommandEnvironmentWindStatus, definitions[0].ID)
	assert.Equal(t, []ExecutionMode{ExecutionQuery}, definitions[0].ExecutionModes)
	assert.Equal(t, CommandEnvironmentWindSet, definitions[1].ID)
	assert.Equal(t, CommandCombatMapCatalog, definitions[2].ID)
	assert.Equal(t, CommandLoggingStatus, definitions[3].ID)
	assert.Equal(t, CommandShrineReferenceInspect, definitions[4].ID)
	assert.Equal(t, CommandShrineStatus, definitions[5].ID)
	assert.Equal(t, CommandShrineProgressUpdate, definitions[6].ID)
	assert.Equal(t, CommandPartyCharacterInspect, definitions[7].ID)
	assert.Equal(t, CommandPartyCharacterUpdate, definitions[8].ID)
	assert.Equal(t, []ExecutionMode{ExecutionLiveSafe}, definitions[8].ExecutionModes)
	assert.Equal(t, OptionProviderActivePartyCharacters, definitions[8].Parameters[0].OptionsProvider)
	assert.Equal(t, ScenarioValueResolverActivePartyPosition, definitions[8].Parameters[0].ScenarioValueResolver)
	require.Len(t, definitions[8].Parameters[0].ScenarioChoices, 6)
	assert.Equal(t, Choice{Value: "1", Label: "Party Position 1"}, definitions[8].Parameters[0].ScenarioChoices[0])
	assert.Equal(t, "level", definitions[8].Parameters[7].ID)
	assert.Equal(t, OptionProviderActivePartyPositions, definitions[8].Parameters[9].OptionsProvider)
	assert.Equal(t, CommandPartyMaximizeLevels, definitions[9].ID)
	assert.Equal(t, CommandPartyStatusSet, definitions[10].ID)
	assert.Equal(t, ParameterIntegerList, definitions[10].Parameters[1].Kind)
	assert.Equal(t, CommandPartyFill, definitions[11].ID)
	assert.Equal(t, CommandPartyMemberAdd, definitions[12].ID)
	assert.Equal(t, CommandCalendarStatus, definitions[13].ID)
	assert.Equal(t, CommandCalendarSetTime, definitions[14].ID)
	assert.Equal(t, CommandCalendarSetDay, definitions[15].ID)
	assert.Equal(t, CommandSearchStatus, definitions[16].ID)
	assert.Equal(t, CommandSearchItemStackAdd, definitions[17].ID)
	assert.Equal(t, CommandSearchProgressReset, definitions[18].ID)
	assert.True(t, definitions[18].Destructive)
	assert.Equal(t, CommandQuestItemStatus, definitions[19].ID)
	assert.Equal(t, CommandQuestItemEnsure, definitions[20].ID)
	assert.Equal(t, CommandQuestItemRemove, definitions[21].ID)
	assert.True(t, definitions[21].Destructive)
	assert.Equal(t, CommandWorldPositionStatus, definitions[22].ID)
	assert.Equal(t, CommandMovementTeleport, definitions[23].ID)
	assert.Equal(t, []ExecutionMode{ExecutionSynchronized}, definitions[23].ExecutionModes)
	assert.Equal(t, CommandMovementEnterLocation, definitions[24].ID)
	assert.Equal(t, CommandMovementFloorSet, definitions[25].ID)
	assert.Equal(t, CommandMovementLandmarkTravel, definitions[26].ID)
	assert.Contains(t, definitions[26].Parameters[0].Choices, Choice{
		Value: "lord_british_secret_room", Label: "Lord British's Secret Room",
	})
	assert.Equal(t, CommandMoongateStatus, definitions[27].ID)
	assert.Equal(t, CommandConversationScriptCatalog, definitions[28].ID)
	assert.Equal(t, CommandConversationScriptInspect, definitions[29].ID)
	assert.Equal(t, CommandCombatEncounterOptions, definitions[30].ID)
	assert.Equal(t, CommandCombatEncounterStart, definitions[31].ID)
	assert.True(t, definitions[31].ScenarioTerminal)
	assert.Equal(t, []ExecutionMode{ExecutionSynchronized}, definitions[31].ExecutionModes)
	assert.Equal(t, CommandVFXProjectile, definitions[34].ID)
	assert.Equal(t, []ExecutionMode{ExecutionPresentation}, definitions[34].ExecutionModes)
	assert.Equal(t, CommandEnvironmentEarthquake, definitions[35].ID)
	assert.Equal(t, CommandMonsterClearMapEnemies, definitions[38].ID)
	assert.True(t, definitions[38].Destructive)
	assert.Equal(t, CommandVehicleStatus, definitions[41].ID)
	assert.Equal(t, CommandVehicleSpawn, definitions[42].ID)
	assert.Equal(t, CommandVehicleForceBoard, definitions[43].ID)
	assert.Equal(t, CommandShipDamageRaw, definitions[44].ID)
	assert.Equal(t, CommandShipRepairFull, definitions[45].ID)
	assert.Equal(t, CommandShipSkiffSet, definitions[46].ID)
	assert.Equal(t, CommandInventoryLockerStatus, definitions[47].ID)
	assert.Equal(t, CommandInventoryProvisionSet, definitions[48].ID)
	assert.Equal(t, []ExecutionMode{ExecutionSynchronized}, definitions[48].ExecutionModes)
	assert.Equal(t, CommandInventoryEquipmentClear, definitions[59].ID)
	assert.True(t, definitions[59].Destructive)
	assert.Equal(t, CommandInventorySpellStatus, definitions[60].ID)
	assert.Equal(t, CommandInventorySpellSet, definitions[61].ID)
	assert.Equal(t, CommandInventorySpellPresetMax, definitions[62].ID)
	assert.Equal(t, CommandSpellEffectSetRemaining, definitions[63].ID)
	assert.Equal(t, CommandSpellEffectActivate, definitions[64].ID)
	assert.Equal(t, CommandCombatStatus, definitions[65].ID)
	assert.Equal(t, CommandCombatMonsterSleepSet, definitions[66].ID)
	assert.Equal(t, []ExecutionMode{ExecutionSynchronized}, definitions[66].ExecutionModes)
	assert.Equal(t, CommandCombatExitDebug, definitions[67].ID)
	assert.Equal(t, []ExecutionMode{ExecutionSynchronized}, definitions[67].ExecutionModes)
	assert.True(t, definitions[67].Destructive)
	assert.Equal(t, CommandCombatMonsterSpawn, definitions[68].ID)
	assert.Equal(t, []ExecutionMode{ExecutionSynchronized}, definitions[68].ExecutionModes)
	assert.Equal(t, CommandRNGSeedSet, definitions[69].ID)
	assert.Equal(t, CommandCombatMonsterDamageRaw, definitions[70].ID)
	assert.Equal(t, CommandCombatMonsterKill, definitions[71].ID)
	assert.Equal(t, CommandCombatMonsterKillAll, definitions[72].ID)
	assert.Equal(t, CommandCombatMonsterClear, definitions[73].ID)
	assert.True(t, definitions[72].Destructive)
	assert.True(t, definitions[73].Destructive)
	assert.Equal(t, CommandCombatDungeonRoomClear, definitions[148].ID)
	assert.Equal(t, CommandMerchantTesterCatalog, definitions[149].ID)
	assert.Equal(t, CommandMerchantTesterPreviewOpen, definitions[150].ID)
	assert.Equal(t, CommandMerchantTesterLiveOpen, definitions[151].ID)
	assert.Equal(t, CommandScenarioBlackthornInterrogation, definitions[152].ID)
	assert.Equal(t, []ExecutionMode{ExecutionWorkflow}, definitions[152].ExecutionModes)
	assert.True(t, definitions[152].Destructive)
	assert.False(t, definitions[152].ScenarioTerminal)
	assert.Equal(t, CommandScenarioLordBritishFinale, definitions[153].ID)
	assert.Equal(t, []ExecutionMode{ExecutionWorkflow}, definitions[153].ExecutionModes)
	assert.True(t, definitions[153].Destructive)
	assert.True(t, definitions[153].ScenarioTerminal)
	assert.True(t, definitions[152].Parameters[0].ScenarioRuntime)
	assert.Equal(t, ParameterBoolean, definitions[153].Parameters[1].Kind)
	assert.Equal(t, CommandDisplayModeSet, definitions[154].ID)
	assert.Equal(t, CommandLoggingEnabledSet, definitions[74].ID)
	assert.Equal(t, CommandLoggingCategorySet, definitions[77].ID)
	assert.Equal(t, CommandLoggingCategoriesSetAll, definitions[78].ID)
	assert.Equal(t, CommandLightingOverlaySet, definitions[79].ID)
	assert.Equal(t, CommandCombatPresentationSpeedSet, definitions[82].ID)
	assert.Equal(t, CommandGuardsStatus, definitions[83].ID)
	assert.Equal(t, CommandShadowlordStatus, definitions[84].ID)
	assert.Equal(t, CommandDeathStatus, definitions[85].ID)
	assert.Equal(t, CommandDebugOptionsStatus, definitions[86].ID)
	assert.Equal(t, CommandMovementFreeMoveSet, definitions[87].ID)
	assert.Equal(t, CommandTextPlaceholdersSet, definitions[88].ID)
	assert.Equal(t, CommandDialogBoundsSet, definitions[89].ID)
	assert.Equal(t, CommandUnlimitedCastingResourcesSet, definitions[90].ID)
	assert.Equal(t, CommandSpellLevelRestrictionsBypassSet, definitions[91].ID)
	assert.Equal(t, CommandDisplayStatus, definitions[92].ID)
	assert.Equal(t, CommandDisplayResolutionSet, definitions[93].ID)
	assert.Equal(t, CommandDisplayFullscreenSet, definitions[94].ID)
	assert.Equal(t, CommandPartyKarmaSet, definitions[95].ID)
	assert.Equal(t, CommandScenarioStatus, definitions[96].ID)
	assert.Equal(t, CommandScenarioStarvationTrigger, definitions[97].ID)
	assert.Equal(t, CommandScenarioStarvationHourly, definitions[98].ID)
	assert.Equal(t, CommandScenarioGuardsActivate, definitions[99].ID)
	assert.Equal(t, CommandScenarioWaterfallSetup, definitions[100].ID)
	assert.Equal(t, CommandScenarioBridgeTroll, definitions[101].ID)
	assert.Equal(t, []ExecutionMode{ExecutionWorkflow}, definitions[101].ExecutionModes)
	assert.True(t, definitions[101].ScenarioTerminal)
	assert.Equal(t, CommandScenarioWordSetup, definitions[102].ID)
	assert.Equal(t, []ExecutionMode{ExecutionSynchronized}, definitions[102].ExecutionModes)
	assert.Equal(t, CommandScenarioDefinitionList, definitions[103].ID)
	assert.Equal(t, []ExecutionMode{ExecutionQuery}, definitions[103].ExecutionModes)
	assert.Equal(t, CommandScenarioDefinitionGet, definitions[104].ID)
	assert.Equal(t, []ExecutionMode{ExecutionQuery}, definitions[104].ExecutionModes)
	assert.Equal(t, CommandScenarioDefinitionValidate, definitions[105].ID)
	assert.Equal(t, CommandScenarioDefinitionCreate, definitions[106].ID)
	assert.Equal(t, CommandScenarioDefinitionUpdate, definitions[107].ID)
	assert.Equal(t, CommandScenarioDefinitionDelete, definitions[108].ID)
	assert.Equal(t, CommandScenarioDefinitionDuplicate, definitions[109].ID)
	assert.Equal(t, CommandScenarioRunStart, definitions[110].ID)
	assert.Equal(t, []ExecutionMode{ExecutionWorkflow}, definitions[110].ExecutionModes)
	assert.Equal(t, CommandScenarioRunStatus, definitions[111].ID)
	assert.Equal(t, []ExecutionMode{ExecutionQuery}, definitions[111].ExecutionModes)
	assert.Equal(t, CommandTestHoleUp, definitions[112].ID)
	assert.Equal(t, CommandTestSearch, definitions[113].ID)
	assert.Equal(t, CommandMoongateTravel, definitions[114].ID)
	assert.Equal(t, []ExecutionMode{ExecutionWorkflow}, definitions[114].ExecutionModes)
	assert.Equal(t, CommandQuestShardUse, definitions[115].ID)
	assert.Equal(t, []ExecutionMode{ExecutionWorkflow}, definitions[115].ExecutionModes)
	assert.Equal(t, CommandQuestInventoryPresetUsable, definitions[116].ID)
	assert.Equal(t, []ExecutionMode{ExecutionAlteration}, definitions[116].ExecutionModes)
	assert.Equal(t, CommandTestTelescope, definitions[117].ID)
	assert.Equal(t, []ExecutionMode{ExecutionWorkflow}, definitions[117].ExecutionModes)
	assert.Equal(t, CommandTestSpyglass, definitions[118].ID)
	assert.Equal(t, CommandTestCrystalBall, definitions[119].ID)
	assert.Equal(t, CommandDeathKillParty, definitions[120].ID)
	assert.Equal(t, []ExecutionMode{ExecutionSynchronized}, definitions[120].ExecutionModes)
	assert.True(t, definitions[120].Destructive)
	assert.Equal(t, CommandDeathTestSequence, definitions[121].ID)
	assert.Equal(t, []ExecutionMode{ExecutionWorkflow}, definitions[121].ExecutionModes)
	assert.True(t, definitions[121].Destructive)
	assert.Equal(t, CommandCodexQuestStateSetAll, definitions[122].ID)
	assert.Equal(t, []ExecutionMode{ExecutionAlteration}, definitions[122].ExecutionModes)
	assert.True(t, definitions[122].Destructive)
	assert.Equal(t, CommandCodexEquipmentCatalog, definitions[123].ID)
	assert.Equal(t, CommandCodexMonsterCatalog, definitions[124].ID)
	assert.Equal(t, CommandCodexSpellCatalog, definitions[125].ID)
	assert.Equal(t, CommandCodexTileCatalog, definitions[126].ID)
	assert.Equal(t, CommandCodexLocationCatalog, definitions[127].ID)
	assert.Equal(t, CommandCodexNPCCatalog, definitions[128].ID)
	assert.Equal(t, CommandCodexQuestCatalog, definitions[129].ID)
	assert.Equal(t, CommandCodexSaveFormat, definitions[130].ID)
	assert.Equal(t, CommandCodexSearchLocations, definitions[131].ID)
	assert.Equal(t, CommandCodexConsumables, definitions[132].ID)
	assert.Equal(t, CommandSaveSnapshotList, definitions[133].ID)
	assert.Equal(t, CommandSaveSnapshotGet, definitions[134].ID)
	assert.Equal(t, CommandSaveSnapshotCapture, definitions[135].ID)
	assert.Equal(t, []ExecutionMode{ExecutionAlteration}, definitions[135].ExecutionModes)
	assert.Equal(t, CommandSaveSnapshotLoad, definitions[136].ID)
	assert.Equal(t, []ExecutionMode{ExecutionSceneRestart}, definitions[136].ExecutionModes)
	assert.True(t, definitions[136].Destructive)
	assert.Equal(t, CommandReduxModificationsStatus, definitions[139].ID)
	assert.Equal(t, []ExecutionMode{ExecutionQuery}, definitions[139].ExecutionModes)
	assert.Equal(t, CommandReduxCombatRepeatedMonsterDivisionSet, definitions[140].ID)
	assert.Equal(t, []ExecutionMode{ExecutionLiveSafe}, definitions[140].ExecutionModes)
	assert.True(t, definitions[140].ScenarioExcluded)
	require.Len(t, definitions[140].Parameters, 1)
	assert.Equal(t, ParameterBoolean, definitions[140].Parameters[0].Kind)
	assert.Equal(t, CommandReduxDungeonsPersistHiddenDiscoveriesSet, definitions[141].ID)
	assert.Equal(t, CommandReduxDungeonsSubduedUnlitVisibilitySet, definitions[142].ID)
	assert.Equal(t, CommandReduxPresentationSmoothSurfaceMovementSet, definitions[143].ID)
	assert.Equal(t, []ExecutionMode{ExecutionLiveSafe}, definitions[143].ExecutionModes)
	assert.True(t, definitions[143].ScenarioExcluded)
	require.Len(t, definitions[143].Parameters, 1)
	assert.Equal(t, ParameterBoolean, definitions[143].Parameters[0].Kind)
	assert.Equal(t, CommandReduxNavigationFlexibleSextantUseSet, definitions[144].ID)
	assert.Equal(t, []ExecutionMode{ExecutionLiveSafe}, definitions[144].ExecutionModes)
	assert.True(t, definitions[144].ScenarioExcluded)
	assert.False(t, definitions[144].Destructive)
	require.Len(t, definitions[144].Parameters, 1)
	assert.Equal(t, ParameterBoolean, definitions[144].Parameters[0].Kind)
	assert.Equal(t, CommandWorldPositionSet, definitions[145].ID)
	assert.Equal(t, []ExecutionMode{ExecutionSynchronized}, definitions[145].ExecutionModes)
	assert.Equal(t, CommandDungeonExplorerStatus, definitions[146].ID)
	assert.Equal(t, []ExecutionMode{ExecutionQuery}, definitions[146].ExecutionModes)
	assert.Equal(t, CommandDungeonExplorerSetup, definitions[147].ID)
	assert.Equal(t, []ExecutionMode{ExecutionSynchronized}, definitions[147].ExecutionModes)
	assert.Equal(t, CommandSaveSnapshotDelete, definitions[137].ID)
	assert.True(t, definitions[137].Destructive)
	assert.Equal(t, CommandPartyEquipmentLoadoutUpdate, definitions[138].ID)
	assert.Equal(t, []ExecutionMode{ExecutionAlteration}, definitions[138].ExecutionModes)
	assert.True(t, definitions[138].Destructive)
}

func TestScenarioRunCancelCatalogRequiresOnlyRunIdentity(t *testing.T) {
	catalog := NewInitialCatalog()
	definition, found := catalog.Get(CommandScenarioRunCancel)
	require.True(t, found)
	require.Len(t, definition.Parameters, 1)
	assert.Equal(t, "runId", definition.Parameters[0].ID)
	assert.True(t, definition.ScenarioExcluded)
	assert.False(t, definition.Destructive)
	assert.Equal(t, []ExecutionMode{ExecutionLiveSafe}, definition.ExecutionModes)
	for _, value := range []string{`"run-1"`, `""`, `" "`, `0`, `null`} {
		_, err := catalog.NormalizeRequest(Request{SessionID: "test", ID: "cancel", CommandID: CommandScenarioRunCancel,
			Arguments: map[string]json.RawMessage{"runId": json.RawMessage(value)}})
		if value == `"run-1"` {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
	_, err := catalog.NormalizeRequest(Request{SessionID: "test", ID: "cancel", CommandID: CommandScenarioRunCancel})
	require.Error(t, err)
}

func TestInitialCatalogValidatesDisplayModeChoices(t *testing.T) {
	catalog := NewInitialCatalog()
	request := Request{
		SessionID: "test", ID: "display-mode", CommandID: CommandDisplayModeSet,
		Arguments: map[string]json.RawMessage{"mode": json.RawMessage(`"maximized"`)},
	}

	_, err := catalog.NormalizeRequest(request)
	require.NoError(t, err)

	request.Arguments["mode"] = json.RawMessage(`"exclusive"`)
	_, err = catalog.NormalizeRequest(request)
	assert.Error(t, err)
}

func TestMerchantTesterCatalogDefinitionsAndTypedParameters(t *testing.T) {
	catalog := NewInitialCatalog()

	status, ok := catalog.Get(CommandMerchantTesterCatalog)
	require.True(t, ok)
	assert.Empty(t, status.Parameters)
	assert.Equal(t, []ExecutionMode{ExecutionQuery}, status.ExecutionModes)
	assert.True(t, status.ScenarioExcluded)

	preview, ok := catalog.Get(CommandMerchantTesterPreviewOpen)
	require.True(t, ok)
	require.Len(t, preview.Parameters, 1)
	assert.Equal(t, "merchantId", preview.Parameters[0].ID)
	assert.Equal(t, ParameterString, preview.Parameters[0].Kind)
	assert.True(t, preview.Parameters[0].Required)
	assert.Equal(t, OptionProviderMerchantTesterMerchants, preview.Parameters[0].OptionsProvider)
	assert.Equal(t, []ExecutionMode{ExecutionWorkflow}, preview.ExecutionModes)
	assert.True(t, preview.ScenarioExcluded)

	live, ok := catalog.Get(CommandMerchantTesterLiveOpen)
	require.True(t, ok)
	require.Len(t, live.Parameters, 6)
	assert.Equal(t, []string{
		"merchantId", "expectedLocationId", "expectedFloor", "expectedX", "expectedY", "expectedSourceFingerprint",
	}, []string{
		live.Parameters[0].ID, live.Parameters[1].ID, live.Parameters[2].ID,
		live.Parameters[3].ID, live.Parameters[4].ID, live.Parameters[5].ID,
	})
	assert.Equal(t, []ExecutionMode{ExecutionWorkflow, ExecutionAlteration}, live.ExecutionModes)
	assert.True(t, live.Destructive)
	assert.True(t, live.ScenarioExcluded)

	require.NoError(t, catalog.ValidateRequest(Request{
		SessionID: "test", ID: "preview", CommandID: CommandMerchantTesterPreviewOpen,
		Arguments: map[string]json.RawMessage{"merchantId": json.RawMessage(`"guild.paws"`)},
	}))
	require.NoError(t, catalog.ValidateRequest(Request{
		SessionID: "test", ID: "live", CommandID: CommandMerchantTesterLiveOpen,
		Arguments: map[string]json.RawMessage{
			"merchantId":         json.RawMessage(`"unknown.merchant"`),
			"expectedLocationId": json.RawMessage(`2`), "expectedFloor": json.RawMessage(`0`),
			"expectedX": json.RawMessage(`10`), "expectedY": json.RawMessage(`11`),
			"expectedSourceFingerprint": json.RawMessage(`"reviewed"`),
		},
	}), "unknown identities must reach update-thread merchant_unknown handling")
}

func TestWorldDungeonExplorerCatalogDefinitionsAndNormalization(t *testing.T) {
	catalog := NewInitialCatalog()

	world, ok := catalog.Get(CommandWorldPositionSet)
	require.True(t, ok)
	assert.Equal(t, "World", world.Category)
	assert.Contains(t, world.SearchTerms, "navigator")
	require.Len(t, world.Parameters, 11)
	assert.Equal(t, ParameterEnum, world.Parameters[0].Kind)
	assert.Equal(t, []Choice{
		{Value: WorldPlacementExact, Label: "Arbitrary exact position"},
		{Value: WorldPlacementDungeonOutside, Label: "Outside dungeon entrance"},
		{Value: WorldPlacementDungeonInsideStart, Label: "Inside dungeon start"},
		{Value: WorldPlacementDungeonUnderworldExit, Label: "Dungeon Underworld-exit ladder"},
	}, world.Parameters[0].Choices)
	assert.Equal(t, ParameterEnum, world.Parameters[1].Kind)
	assert.Equal(t, []Choice{
		{Value: WorldRouteBritannia, Label: "Britannia"},
		{Value: WorldRouteUnderworld, Label: "Underworld"},
	}, world.Parameters[1].Choices)
	assert.Equal(t, ParameterEnum, world.Parameters[2].Kind)
	require.Len(t, world.Parameters[2].Choices, int(references.Doom)+1)
	assert.Equal(t, Choice{Value: "0", Label: "Britannia / Underworld"}, world.Parameters[2].Choices[0])
	assert.Equal(t, Choice{Value: "40", Label: "Doom"}, world.Parameters[2].Choices[40])
	assert.Equal(t, ParameterEnum, world.Parameters[3].Kind)
	require.Len(t, world.Parameters[3].Choices, references.DungeonFloorCount+1)
	assert.Equal(t, "-1", world.Parameters[3].Choices[0].Value)
	assert.Equal(t, "7", world.Parameters[3].Choices[8].Value)
	assert.Equal(t, []string{"expectedLocationId", "expectedFloor", "expectedX", "expectedY", "expectedSourceFingerprint"}, []string{
		world.Parameters[6].ID,
		world.Parameters[7].ID,
		world.Parameters[8].ID,
		world.Parameters[9].ID,
		world.Parameters[10].ID,
	})
	for _, parameter := range world.Parameters[6:] {
		assert.True(t, parameter.ScenarioRuntime, parameter.ID)
	}
	require.NoError(t, catalog.ValidateRequest(Request{
		SessionID: "test", ID: "world", CommandID: CommandWorldPositionSet,
		Arguments: map[string]json.RawMessage{
			"placement": json.RawMessage(`"exact"`), "route": json.RawMessage(`"underworld"`),
			"location": json.RawMessage(`"40"`), "floor": json.RawMessage(`"7"`),
			"x": json.RawMessage(`7`), "y": json.RawMessage(`0`),
			"expectedLocationId": json.RawMessage(`2`), "expectedFloor": json.RawMessage(`0`),
			"expectedX": json.RawMessage(`10`), "expectedY": json.RawMessage(`11`),
			"expectedSourceFingerprint": json.RawMessage(`"reviewed-world"`),
		},
	}))
	routeRequest, err := catalog.NormalizeRequest(Request{
		SessionID: "test", ID: "world-route", CommandID: CommandWorldPositionSet,
		Arguments: map[string]json.RawMessage{
			"placement": json.RawMessage(`"dungeon_underworld_exit"`), "route": json.RawMessage(`"underworld"`),
			"location":           json.RawMessage(`"33"`),
			"expectedLocationId": json.RawMessage(`2`), "expectedFloor": json.RawMessage(`0`),
			"expectedX": json.RawMessage(`10`), "expectedY": json.RawMessage(`11`),
			"expectedSourceFingerprint": json.RawMessage(`"reviewed-world"`),
		},
	})
	require.NoError(t, err)
	assert.JSONEq(t, `"dungeon_underworld_exit"`, string(routeRequest.Arguments["placement"]))
	assert.JSONEq(t, `0`, string(routeRequest.Arguments["x"]))

	dungeon, ok := catalog.Get(CommandDungeonExplorerSetup)
	require.True(t, ok)
	assert.Equal(t, "Dungeon Explorer", dungeon.Category)
	assert.Contains(t, dungeon.SearchTerms, "facing")
	require.Len(t, dungeon.Parameters, 12)
	require.Len(t, dungeon.Parameters[0].Choices, 8)
	assert.Equal(t, Choice{Value: "33", Label: "Deceit"}, dungeon.Parameters[0].Choices[0])
	assert.Equal(t, Choice{Value: "40", Label: "Doom"}, dungeon.Parameters[0].Choices[7])
	assert.Equal(t, ParameterInteger, dungeon.Parameters[1].Kind)
	assert.Equal(t, 0, *dungeon.Parameters[1].Minimum)
	assert.Equal(t, 7, *dungeon.Parameters[1].Maximum)
	assert.Equal(t, []Choice{
		{Value: "north", Label: "North"},
		{Value: "east", Label: "East"},
		{Value: "south", Label: "South"},
		{Value: "west", Label: "West"},
	}, dungeon.Parameters[4].Choices)
	assert.Equal(t, ParameterBoolean, dungeon.Parameters[5].Kind)
	assert.Equal(t, "expectedSourceFingerprint", dungeon.Parameters[10].ID)
	assert.Equal(t, "expectedDungeonRevision", dungeon.Parameters[11].ID)
	for _, parameter := range dungeon.Parameters[6:] {
		assert.True(t, parameter.ScenarioRuntime, parameter.ID)
	}

	normalized, err := catalog.NormalizeRequest(Request{
		SessionID: "test", ID: "dungeon", CommandID: CommandDungeonExplorerSetup,
		Arguments: map[string]json.RawMessage{
			"dungeon":            json.RawMessage(`"33"`),
			"expectedLocationId": json.RawMessage(`2`), "expectedFloor": json.RawMessage(`0`),
			"expectedX": json.RawMessage(`10`), "expectedY": json.RawMessage(`11`),
			"expectedSourceFingerprint": json.RawMessage(`"reviewed-world"`),
			"expectedDungeonRevision":   json.RawMessage(`0`),
		},
	})
	require.NoError(t, err)
	assert.JSONEq(t, `0`, string(normalized.Arguments["floor"]))
	assert.JSONEq(t, `4`, string(normalized.Arguments["x"]))
	assert.JSONEq(t, `4`, string(normalized.Arguments["y"]))
	assert.JSONEq(t, `"east"`, string(normalized.Arguments["facing"]))
	assert.JSONEq(t, `true`, string(normalized.Arguments["lit"]))

	normalized.Arguments["dungeon"] = json.RawMessage(`"32"`)
	require.ErrorIs(t, catalog.ValidateRequest(normalized), ErrInvalidRequest)
	normalized.Arguments["dungeon"] = json.RawMessage(`"33"`)
	normalized.Arguments["facing"] = json.RawMessage(`"up"`)
	require.ErrorIs(t, catalog.ValidateRequest(normalized), ErrInvalidRequest)
}

func TestPartyEquipmentLoadoutCatalogUsesSlotSpecificAtomicChoices(t *testing.T) {
	definition, ok := NewInitialCatalog().Get(CommandPartyEquipmentLoadoutUpdate)
	require.True(t, ok)
	require.Len(t, definition.Parameters, 8)
	assert.Equal(t, OptionProviderEquipmentLoadoutTargets, definition.Parameters[0].OptionsProvider)
	assert.Equal(t, ScenarioValueResolverActivePartyPosition, definition.Parameters[0].ScenarioValueResolver)
	assert.Equal(t, Choice{Value: "all", Label: "All active party members"}, definition.Parameters[0].ScenarioChoices[0])
	assert.Equal(t, "helm", definition.Parameters[1].ID)
	assert.Equal(t, "amulet", definition.Parameters[6].ID)
	assert.Equal(t, inventoryExpectedSnapshotID, definition.Parameters[7].ID)
	assert.True(t, definition.Parameters[7].ScenarioRuntime)
	assert.Contains(t, definition.Parameters[3].Choices, choice("inventory:30", "Equip Long Sword from inventory"))
	assert.Contains(t, definition.Parameters[3].Choices, choice("manufacture:39", "Manufacture and equip Glass Sword"))
	assert.NotContains(t, definition.Parameters[3].Choices, choice("inventory:27", "Equip Arrows from inventory"))
}

func TestInitialCatalogValidatesSaveSnapshotOperations(t *testing.T) {
	catalog := NewInitialCatalog()
	revision := json.RawMessage(`"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`)

	require.NoError(t, catalog.ValidateRequest(Request{SessionID: "test", ID: "list", CommandID: CommandSaveSnapshotList}))
	require.NoError(t, catalog.ValidateRequest(Request{SessionID: "test", ID: "get", CommandID: CommandSaveSnapshotGet, Arguments: map[string]json.RawMessage{"reference": json.RawMessage(`"rollback:automatic-1"`)}}))
	capture, err := catalog.NormalizeRequest(Request{SessionID: "test", ID: "capture", CommandID: CommandSaveSnapshotCapture, Arguments: map[string]json.RawMessage{
		"id": json.RawMessage(`"before-test"`), "name": json.RawMessage(`"Before test"`),
	}})
	require.NoError(t, err)
	assert.JSONEq(t, `"personal"`, string(capture.Arguments["source"]))
	assert.JSONEq(t, `""`, string(capture.Arguments["description"]))

	load := Request{SessionID: "test", ID: "load", CommandID: CommandSaveSnapshotLoad, Arguments: map[string]json.RawMessage{"reference": json.RawMessage(`"project:before-test"`), "expectedRevision": revision}}
	require.NoError(t, catalog.ValidateRequest(load))
	remove := Request{SessionID: "test", ID: "delete", CommandID: CommandSaveSnapshotDelete, Arguments: map[string]json.RawMessage{"reference": json.RawMessage(`"personal:before-test"`), "expectedRevision": revision}}
	require.NoError(t, catalog.ValidateRequest(remove))
	remove.Arguments["reference"] = json.RawMessage(`"rollback:automatic-1"`)
	require.ErrorIs(t, catalog.ValidateRequest(remove), ErrInvalidRequest)
	load.Arguments["expectedRevision"] = json.RawMessage(`"stale"`)
	require.ErrorIs(t, catalog.ValidateRequest(load), ErrInvalidRequest)
}

func TestInitialCatalogValidatesDurableScenarioOperations(t *testing.T) {
	catalog := NewInitialCatalog()

	for _, commandID := range []string{CommandScenarioDefinitionList, CommandScenarioRunStatus} {
		require.NoError(t, catalog.ValidateRequest(Request{SessionID: "test", ID: commandID, CommandID: commandID}))
		require.ErrorIs(t, catalog.ValidateRequest(Request{
			SessionID: "test", ID: commandID, CommandID: commandID,
			Arguments: map[string]json.RawMessage{"unexpected": json.RawMessage(`true`)},
		}), ErrInvalidRequest)
	}

	get := Request{SessionID: "test", ID: "get", CommandID: CommandScenarioDefinitionGet, Arguments: map[string]json.RawMessage{
		"reference": json.RawMessage(`"project:starvation-check"`),
	}}
	require.NoError(t, catalog.ValidateRequest(get))
	get.Arguments["reference"] = json.RawMessage(`"starvation-check"`)
	require.ErrorIs(t, catalog.ValidateRequest(get), ErrInvalidRequest)
	get.Arguments["reference"] = json.RawMessage(`"personal:starvation-check"`)
	require.NoError(t, catalog.ValidateRequest(get))
	delete(get.Arguments, "reference")
	require.ErrorIs(t, catalog.ValidateRequest(get), ErrInvalidRequest)

	validate := Request{SessionID: "test", ID: "validate", CommandID: CommandScenarioDefinitionValidate, Arguments: map[string]json.RawMessage{
		"definition": json.RawMessage(`"{\"schemaVersion\":2}"`),
	}}
	require.NoError(t, catalog.ValidateRequest(validate))

	create := Request{SessionID: "test", ID: "create", CommandID: CommandScenarioDefinitionCreate, Arguments: map[string]json.RawMessage{
		"source": json.RawMessage(`"personal"`), "definition": json.RawMessage(`"{\"schemaVersion\":2}"`),
	}}
	require.NoError(t, catalog.ValidateRequest(create))

	revision := `"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`
	update := Request{SessionID: "test", ID: "update", CommandID: CommandScenarioDefinitionUpdate, Arguments: map[string]json.RawMessage{
		"reference": json.RawMessage(`"project:starvation-check"`), "expectedRevision": json.RawMessage(revision),
		"definition": json.RawMessage(`"{\"schemaVersion\":2}"`),
	}}
	require.NoError(t, catalog.ValidateRequest(update))
	update.Arguments["expectedRevision"] = json.RawMessage(`"stale"`)
	require.ErrorIs(t, catalog.ValidateRequest(update), ErrInvalidRequest)

	remove := Request{SessionID: "test", ID: "delete", CommandID: CommandScenarioDefinitionDelete, Arguments: map[string]json.RawMessage{
		"reference": json.RawMessage(`"personal:starvation-check"`), "expectedRevision": json.RawMessage(revision),
	}}
	require.NoError(t, catalog.ValidateRequest(remove))

	duplicate := Request{SessionID: "test", ID: "duplicate", CommandID: CommandScenarioDefinitionDuplicate, Arguments: map[string]json.RawMessage{
		"sourceReference": json.RawMessage(`"project:starvation-check"`), "sourceRevision": json.RawMessage(revision),
		"destinationSource": json.RawMessage(`"personal"`), "destinationId": json.RawMessage(`"starvation-copy"`),
		"destinationName": json.RawMessage(`"Starvation Copy"`),
	}}
	require.NoError(t, catalog.ValidateRequest(duplicate))
	delete(duplicate.Arguments, "destinationName")
	require.ErrorIs(t, catalog.ValidateRequest(duplicate), ErrInvalidRequest)

	start := Request{SessionID: "test", ID: "start", CommandID: CommandScenarioRunStart, Arguments: map[string]json.RawMessage{
		"reference": json.RawMessage(`"personal:starvation-check"`), "expectedRevision": json.RawMessage(revision),
		"reviewedSceneGeneration": json.RawMessage(`7`),
	}}
	normalized, err := catalog.NormalizeRequest(start)
	require.NoError(t, err)
	assert.JSONEq(t, `"{}"`, string(normalized.Arguments["parameterValues"]))
	assert.JSONEq(t, `""`, string(normalized.Arguments["expectedBaselineRevision"]))
	delete(start.Arguments, "expectedRevision")
	require.ErrorIs(t, catalog.ValidateRequest(start), ErrInvalidRequest)
	start.Arguments["expectedRevision"] = json.RawMessage(revision)
	delete(start.Arguments, "reviewedSceneGeneration")
	require.ErrorIs(t, catalog.ValidateRequest(start), ErrInvalidRequest)
	start.Arguments["reviewedSceneGeneration"] = json.RawMessage(`7`)

	start.Arguments["reference"] = json.RawMessage(`"starvation-check"`)
	require.ErrorIs(t, catalog.ValidateRequest(start), ErrInvalidRequest)
	start.Arguments["reference"] = json.RawMessage(`"project:starvation-check"`)
	start.Arguments["parameterValues"] = json.RawMessage(`"{\"seed\":77}"`)
	require.NoError(t, catalog.ValidateRequest(start))
	start.Arguments["parameterValues"] = json.RawMessage(`{"seed":77}`)
	require.ErrorIs(t, catalog.ValidateRequest(start), ErrInvalidRequest)
	start.Arguments["parameterValues"] = json.RawMessage(`"{\"seed\":77}"`)
	start.Arguments["unknown"] = json.RawMessage(`true`)
	require.ErrorIs(t, catalog.ValidateRequest(start), ErrInvalidRequest)
}

func TestInitialCatalogValidatesEighthBatchCombatArguments(t *testing.T) {
	catalog := NewInitialCatalog()
	base := map[string]json.RawMessage{
		"encounterId": json.RawMessage(`"4"`), "monsterId": json.RawMessage(`"7"`),
		"expectedCurrentHp": json.RawMessage(`20`), "expectedSnapshot": json.RawMessage(`"reviewed"`),
	}
	damage := Request{SessionID: "test", ID: "damage", CommandID: CommandCombatMonsterDamageRaw, Arguments: base}
	damage.Arguments["damage"] = json.RawMessage(`999`)
	require.NoError(t, catalog.ValidateRequest(damage))
	damage.Arguments["damage"] = json.RawMessage(`1000`)
	require.ErrorIs(t, catalog.ValidateRequest(damage), ErrInvalidRequest)

	for _, commandID := range []string{CommandCombatMonsterKillAll, CommandCombatMonsterClear} {
		require.NoError(t, catalog.ValidateRequest(Request{
			SessionID: "test", ID: commandID, CommandID: commandID,
			Arguments: map[string]json.RawMessage{"encounterId": json.RawMessage(`"4"`), "expectedSnapshot": json.RawMessage(`"reviewed"`)},
		}))
	}
}

func TestInitialCatalogValidatesSeventhBatchReviewedShrineState(t *testing.T) {
	catalog := NewInitialCatalog()
	request := Request{
		SessionID: "test", ID: "set-all-shrines", CommandID: CommandCodexQuestStateSetAll,
		Arguments: map[string]json.RawMessage{
			"targetState":         json.RawMessage(`"completed"`),
			"expectedFingerprint": json.RawMessage(`"reviewed-shrines"`),
		},
	}

	_, err := catalog.NormalizeRequest(request)
	require.NoError(t, err)

	request.Arguments["targetState"] = json.RawMessage(`"assigned"`)
	_, err = catalog.NormalizeRequest(request)
	assert.Error(t, err)

	request.Arguments["targetState"] = json.RawMessage(`"not_assigned"`)
	delete(request.Arguments, "expectedFingerprint")
	_, err = catalog.NormalizeRequest(request)
	assert.Error(t, err)
}

func TestInitialCatalogValidatesSixthBatchReviewedFingerprint(t *testing.T) {
	catalog := NewInitialCatalog()
	for _, commandID := range []string{CommandDeathKillParty, CommandDeathTestSequence} {
		request := Request{SessionID: "test", ID: commandID, CommandID: commandID, Arguments: map[string]json.RawMessage{
			"expectedFingerprint": json.RawMessage(`"reviewed-party"`),
		}}
		_, err := catalog.NormalizeRequest(request)
		require.NoError(t, err)

		delete(request.Arguments, "expectedFingerprint")
		_, err = catalog.NormalizeRequest(request)
		assert.Error(t, err)
	}
}

func TestInitialCatalogMatchesBrowserContractFixture(t *testing.T) {
	want, err := os.ReadFile("webui/client/src/initial_catalog.fixture.json")
	require.NoError(t, err)
	got, err := json.Marshal(NewInitialCatalog().Definitions())
	require.NoError(t, err)

	assert.JSONEq(t, string(want), string(got))
}

func TestInitialCatalogValidatesIndependentSpellEffectReviewedState(t *testing.T) {
	catalog := NewInitialCatalog()
	setRequest := Request{SessionID: "test", ID: "set-effect", CommandID: CommandSpellEffectSetRemaining, Arguments: map[string]json.RawMessage{
		"effectKind": json.RawMessage(`"magic_light"`), "expectedEffect": json.RawMessage(`"P"`),
		"expectedTurns": json.RawMessage(`20`), "expectedLightMinutes": json.RawMessage(`30`), "remaining": json.RawMessage(`5`),
	}}
	require.NoError(t, catalog.ValidateRequest(setRequest))
	delete(setRequest.Arguments, "expectedLightMinutes")
	require.Error(t, catalog.ValidateRequest(setRequest))

	activateRequest := Request{SessionID: "test", ID: "activate", CommandID: CommandSpellEffectActivate, Arguments: map[string]json.RawMessage{
		"effect": json.RawMessage(`"light"`), "expectedEffect": json.RawMessage(`""`),
		"expectedTurns": json.RawMessage(`0`), "expectedLightMinutes": json.RawMessage(`0`),
	}}
	require.NoError(t, catalog.ValidateRequest(activateRequest))
}

func TestInitialCatalogNormalizesCombatMapSectionDefault(t *testing.T) {
	catalog := NewInitialCatalog()
	request := Request{SessionID: "test-session", ID: "request-1", CommandID: CommandCombatMapCatalog}

	normalized, err := catalog.NormalizeRequest(request)

	require.NoError(t, err)
	assert.JSONEq(t, `"all"`, string(normalized.Arguments["section"]))
}

func TestInitialCatalogValidatesThirdBatchParameters(t *testing.T) {
	catalog := NewInitialCatalog()
	holeUp := Request{SessionID: "test", ID: "hole-up", CommandID: CommandTestHoleUp, Arguments: map[string]json.RawMessage{
		"hours": json.RawMessage(`1`), "expectedPartyFingerprint": json.RawMessage(`"party"`),
	}}
	normalized, err := catalog.NormalizeRequest(holeUp)
	require.NoError(t, err)
	assert.JSONEq(t, `false`, string(normalized.Arguments["forceWanderer"]))
	assert.JSONEq(t, `false`, string(normalized.Arguments["forceAmbush"]))
	holeUp.Arguments["hours"] = json.RawMessage(`25`)
	require.Error(t, catalog.ValidateRequest(holeUp))

	search := Request{SessionID: "test", ID: "search", CommandID: CommandTestSearch, Arguments: map[string]json.RawMessage{"direction": json.RawMessage(`"up"`)}}
	require.NoError(t, catalog.ValidateRequest(search))
	search.Arguments["direction"] = json.RawMessage(`"north"`)
	require.Error(t, catalog.ValidateRequest(search))
}

func TestInitialCatalogValidatesFourthBatchReviewedParameters(t *testing.T) {
	catalog := NewInitialCatalog()
	shard := Request{SessionID: "test", ID: "shard", CommandID: CommandQuestShardUse, Arguments: map[string]json.RawMessage{
		"shard": json.RawMessage(`"shard_hatred"`), "expectedQuantity": json.RawMessage(`1`),
		"expectedLocationId": json.RawMessage(`31`), "expectedX": json.RawMessage(`15`), "expectedY": json.RawMessage(`3`),
		"expectedFloor": json.RawMessage(`1`), "expectedFlame": json.RawMessage(`"love"`),
		"expectedShadowlordSummoned": json.RawMessage(`true`), "expectedShadowlord": json.RawMessage(`"astaroth"`),
		"expectedShadowlordAlive": json.RawMessage(`true`),
	}}
	require.NoError(t, catalog.ValidateRequest(shard))
	shard.Arguments["shard"] = json.RawMessage(`"amulet"`)
	require.Error(t, catalog.ValidateRequest(shard))

	require.NoError(t, catalog.ValidateRequest(Request{SessionID: "test", ID: "preset", CommandID: CommandQuestInventoryPresetUsable, Arguments: map[string]json.RawMessage{
		"expectedFingerprint": json.RawMessage(`"reviewed"`),
	}}))
}

func TestInitialCatalogValidatesPartyRosterIndexLists(t *testing.T) {
	catalog := NewInitialCatalog()
	request := Request{SessionID: "test", ID: "status", CommandID: CommandPartyStatusSet, Arguments: map[string]json.RawMessage{
		"expectedFingerprint": json.RawMessage(`"reviewed"`),
		"rosterIndices":       json.RawMessage(`[0,5]`),
		"status":              json.RawMessage(`"dead"`),
	}}
	require.NoError(t, catalog.ValidateRequest(request))

	request.Arguments["rosterIndices"] = json.RawMessage(`[]`)
	require.Error(t, catalog.ValidateRequest(request))
	request.Arguments["rosterIndices"] = json.RawMessage(`[16]`)
	require.Error(t, catalog.ValidateRequest(request))
	request.Arguments["rosterIndices"] = json.RawMessage(`"0"`)
	require.Error(t, catalog.ValidateRequest(request))
}

func TestInitialCatalogValidatesConcreteQueryParameters(t *testing.T) {
	catalog := NewInitialCatalog()

	for _, section := range []string{CombatMapSectionAll, CombatMapSectionBritannia, CombatMapSectionDungeons} {
		err := catalog.ValidateRequest(Request{
			SessionID: "test-session",
			ID:        "combat-" + section,
			CommandID: CommandCombatMapCatalog,
			Arguments: map[string]json.RawMessage{"section": json.RawMessage(`"` + section + `"`)},
		})
		require.NoError(t, err)
	}

	for _, shrineIndex := range []int{0, 7} {
		err := catalog.ValidateRequest(Request{
			SessionID: "test-session",
			ID:        "shrine",
			CommandID: CommandShrineReferenceInspect,
			Arguments: map[string]json.RawMessage{"index": json.RawMessage(strconv.Quote(strconv.Itoa(shrineIndex)))},
		})
		require.NoError(t, err)
	}
}

func TestInitialCatalogRejectsInvalidConcreteQueryParameters(t *testing.T) {
	catalog := NewInitialCatalog()
	tests := []Request{
		{
			SessionID: "test-session",
			ID:        "combat-invalid",
			CommandID: CommandCombatMapCatalog,
			Arguments: map[string]json.RawMessage{"section": json.RawMessage(`"unknown"`)},
		},
		{
			SessionID: "test-session",
			ID:        "shrine-low",
			CommandID: CommandShrineReferenceInspect,
			Arguments: map[string]json.RawMessage{"index": json.RawMessage(`"-1"`)},
		},
		{
			SessionID: "test-session",
			ID:        "shrine-high",
			CommandID: CommandShrineReferenceInspect,
			Arguments: map[string]json.RawMessage{"index": json.RawMessage(`"8"`)},
		},
	}

	for _, request := range tests {
		err := catalog.ValidateRequest(request)
		require.ErrorIs(t, err, ErrInvalidRequest)
	}
}

func TestInitialCatalogValidatesApprovedToolBoundaries(t *testing.T) {
	catalog := NewInitialCatalog()
	valid := []Request{
		{SessionID: "test", ID: "time", CommandID: CommandCalendarSetTime, Arguments: map[string]json.RawMessage{"hour": json.RawMessage(`23`), "minute": json.RawMessage(`59`)}},
		{SessionID: "test", ID: "day", CommandID: CommandCalendarSetDay, Arguments: map[string]json.RawMessage{"day": json.RawMessage(`28`)}},
		{SessionID: "test", ID: "stack", CommandID: CommandSearchItemStackAdd, Arguments: map[string]json.RawMessage{"x": json.RawMessage(`255`), "y": json.RawMessage(`0`), "item": json.RawMessage(`"gold"`), "quantity": json.RawMessage(`999`)}},
		{SessionID: "test", ID: "quest", CommandID: CommandQuestItemEnsure, Arguments: map[string]json.RawMessage{"item": json.RawMessage(`"sextant"`)}},
		{SessionID: "test", ID: "teleport", CommandID: CommandMovementTeleport, Arguments: map[string]json.RawMessage{"x": json.RawMessage(`0`), "y": json.RawMessage(`255`)}},
	}
	for _, request := range valid {
		require.NoError(t, catalog.ValidateRequest(request), request.ID)
	}

	invalid := []Request{
		{SessionID: "test", ID: "time", CommandID: CommandCalendarSetTime, Arguments: map[string]json.RawMessage{"hour": json.RawMessage(`24`), "minute": json.RawMessage(`0`)}},
		{SessionID: "test", ID: "day", CommandID: CommandCalendarSetDay, Arguments: map[string]json.RawMessage{"day": json.RawMessage(`0`)}},
		{SessionID: "test", ID: "stack", CommandID: CommandSearchItemStackAdd, Arguments: map[string]json.RawMessage{"x": json.RawMessage(`0`), "y": json.RawMessage(`0`), "item": json.RawMessage(`"gems"`), "quantity": json.RawMessage(`1`)}},
		{SessionID: "test", ID: "quest", CommandID: CommandQuestItemRemove, Arguments: map[string]json.RawMessage{"item": json.RawMessage(`"skull_key"`)}},
	}
	for _, request := range invalid {
		require.ErrorIs(t, catalog.ValidateRequest(request), ErrInvalidRequest, request.ID)
	}
}

func TestInitialCatalogNormalizesSearchResetScope(t *testing.T) {
	normalized, err := NewInitialCatalog().NormalizeRequest(Request{SessionID: "test", ID: "reset", CommandID: CommandSearchProgressReset})
	require.NoError(t, err)
	assert.JSONEq(t, `"one_time_locations"`, string(normalized.Arguments["scope"]))
}

func TestInitialCatalogNormalizesSecondBatchDefaults(t *testing.T) {
	catalog := NewInitialCatalog()

	conversation, err := catalog.NormalizeRequest(Request{SessionID: "test", ID: "conversation", CommandID: CommandConversationScriptCatalog})
	require.NoError(t, err)
	assert.JSONEq(t, `"towne"`, string(conversation.Arguments["source"]))

	encounter, err := catalog.NormalizeRequest(Request{
		SessionID: "test", ID: "encounter", CommandID: CommandCombatEncounterStart,
		Arguments: map[string]json.RawMessage{
			"territory": json.RawMessage(`"britannia"`), "britanniaMap": json.RawMessage(`2`),
			"entryDirection": json.RawMessage(`"north"`), "monsterType": json.RawMessage(`"32"`),
			"countMode": json.RawMessage(`"automatic"`),
		},
	})
	require.NoError(t, err)
	assert.JSONEq(t, `true`, string(encounter.Arguments["friends"]))
}

func TestInitialCatalogValidatesSecondBatchBounds(t *testing.T) {
	catalog := NewInitialCatalog()
	valid := []Request{
		{SessionID: "test", ID: "script", CommandID: CommandConversationScriptInspect, Arguments: map[string]json.RawMessage{"source": json.RawMessage(`"dwelling"`), "index": json.RawMessage(`14`)}},
		{SessionID: "test", ID: "generation", CommandID: CommandMonsterGenerationSet, Arguments: map[string]json.RawMessage{"enabled": json.RawMessage(`true`), "odds": json.RawMessage(`1000`)}},
		{SessionID: "test", ID: "damage", CommandID: CommandShipDamageRaw, Arguments: map[string]json.RawMessage{"amount": json.RawMessage(`98`)}},
		{SessionID: "test", ID: "projectile", CommandID: CommandVFXProjectile, Arguments: map[string]json.RawMessage{"fromMode": json.RawMessage(`"player"`), "toX": json.RawMessage(`255`), "toY": json.RawMessage(`0`), "projectileType": json.RawMessage(`"violet"`)}},
		{SessionID: "test", ID: "spawn", CommandID: CommandVehicleSpawn, Arguments: map[string]json.RawMessage{"type": json.RawMessage(`"frigate"`), "placement": json.RawMessage(`"dock"`), "dock": json.RawMessage(`"east_britanny"`)}},
		{SessionID: "test", ID: "board", CommandID: CommandVehicleForceBoard, Arguments: map[string]json.RawMessage{"type": json.RawMessage(`"carpet"`)}},
	}
	for _, request := range valid {
		require.NoError(t, catalog.ValidateRequest(request), request.ID)
	}

	invalid := []Request{
		{SessionID: "test", ID: "script", CommandID: CommandConversationScriptInspect, Arguments: map[string]json.RawMessage{"source": json.RawMessage(`"village"`), "index": json.RawMessage(`0`)}},
		{SessionID: "test", ID: "generation", CommandID: CommandMonsterGenerationSet, Arguments: map[string]json.RawMessage{"enabled": json.RawMessage(`true`), "odds": json.RawMessage(`0`)}},
		{SessionID: "test", ID: "damage", CommandID: CommandShipDamageRaw, Arguments: map[string]json.RawMessage{"amount": json.RawMessage(`99`)}},
		{SessionID: "test", ID: "projectile", CommandID: CommandVFXProjectile, Arguments: map[string]json.RawMessage{"fromMode": json.RawMessage(`"player"`), "toX": json.RawMessage(`0`), "toY": json.RawMessage(`0`), "projectileType": json.RawMessage(`"fireball"`)}},
		{SessionID: "test", ID: "spawn", CommandID: CommandVehicleSpawn, Arguments: map[string]json.RawMessage{"type": json.RawMessage(`"horse"`), "placement": json.RawMessage(`"avatar"`)}},
		{SessionID: "test", ID: "board", CommandID: CommandVehicleForceBoard, Arguments: map[string]json.RawMessage{"type": json.RawMessage(`"none"`)}},
	}
	for _, request := range invalid {
		require.ErrorIs(t, catalog.ValidateRequest(request), ErrInvalidRequest, request.ID)
	}
}

func TestInitialCatalogValidatesInventoryLockerArguments(t *testing.T) {
	catalog := NewInitialCatalog()
	valid := []Request{
		{SessionID: "test", ID: "set", CommandID: CommandInventoryProvisionSet, Arguments: map[string]json.RawMessage{"provision": json.RawMessage(`"food"`), "targetQuantity": json.RawMessage(`0`), "expectedQuantity": json.RawMessage(`9999`)}},
		{SessionID: "test", ID: "reagent", CommandID: CommandInventoryReagentSet, Arguments: map[string]json.RawMessage{"reagent": json.RawMessage(`"7"`), "targetQuantity": json.RawMessage(`99`), "expectedQuantity": json.RawMessage(`0`)}},
		{SessionID: "test", ID: "equipment", CommandID: CommandInventoryEquipmentSet, Arguments: map[string]json.RawMessage{"equipment": json.RawMessage(`"47"`), "targetQuantity": json.RawMessage(`0`), "expectedQuantity": json.RawMessage(`99`)}},
		{SessionID: "test", ID: "potion", CommandID: CommandInventoryPotionAdd, Arguments: map[string]json.RawMessage{"potion": json.RawMessage(`"7"`), "amount": json.RawMessage(`1`)}},
		{SessionID: "test", ID: "preset", CommandID: CommandInventoryReagentPreset, Arguments: map[string]json.RawMessage{"preset": json.RawMessage(`"reagents_all99"`), "expectedSnapshot": json.RawMessage(`"snapshot"`)}},
		{SessionID: "test", ID: "grant", CommandID: CommandInventoryEquipmentGrantPreset, Arguments: map[string]json.RawMessage{"preset": json.RawMessage(`"weapons5"`), "expectedSnapshot": json.RawMessage(`"snapshot"`)}},
		{SessionID: "test", ID: "clear", CommandID: CommandInventoryEquipmentClear, Arguments: map[string]json.RawMessage{"expectedSnapshot": json.RawMessage(`"snapshot"`)}},
		{SessionID: "test", ID: "spells", CommandID: CommandInventorySpellPresetMax, Arguments: map[string]json.RawMessage{"expectedSnapshot": json.RawMessage(`"snapshot"`)}},
		{SessionID: "test", ID: "spell-set", CommandID: CommandInventorySpellSet, Arguments: map[string]json.RawMessage{"spell": json.RawMessage(`"47"`), "targetQuantity": json.RawMessage(`99`), "expectedQuantity": json.RawMessage(`0`)}},
	}
	for _, request := range valid {
		require.NoError(t, catalog.ValidateRequest(request), request.ID)
	}

	invalid := []Request{
		{SessionID: "test", ID: "set", CommandID: CommandInventoryProvisionSet, Arguments: map[string]json.RawMessage{"provision": json.RawMessage(`"food"`), "targetQuantity": json.RawMessage(`10000`), "expectedQuantity": json.RawMessage(`0`)}},
		{SessionID: "test", ID: "potion", CommandID: CommandInventoryPotionAdd, Arguments: map[string]json.RawMessage{"potion": json.RawMessage(`"8"`), "amount": json.RawMessage(`0`)}},
		{SessionID: "test", ID: "preset", CommandID: CommandInventoryScrollPreset, Arguments: map[string]json.RawMessage{"preset": json.RawMessage(`"scrolls_all1"`)}},
		{SessionID: "test", ID: "reagent", CommandID: CommandInventoryReagentSet, Arguments: map[string]json.RawMessage{"reagent": json.RawMessage(`"8"`), "targetQuantity": json.RawMessage(`0`), "expectedQuantity": json.RawMessage(`0`)}},
		{SessionID: "test", ID: "equipment", CommandID: CommandInventoryEquipmentSet, Arguments: map[string]json.RawMessage{"equipment": json.RawMessage(`"0"`), "targetQuantity": json.RawMessage(`100`), "expectedQuantity": json.RawMessage(`0`)}},
		{SessionID: "test", ID: "clear", CommandID: CommandInventoryEquipmentClear},
		{SessionID: "test", ID: "spells", CommandID: CommandInventorySpellPresetMax},
		{SessionID: "test", ID: "spell-set", CommandID: CommandInventorySpellSet, Arguments: map[string]json.RawMessage{"spell": json.RawMessage(`"48"`), "targetQuantity": json.RawMessage(`100`), "expectedQuantity": json.RawMessage(`0`)}},
	}
	for _, request := range invalid {
		require.ErrorIs(t, catalog.ValidateRequest(request), ErrInvalidRequest, request.ID)
	}
}

func TestInventoryEquipmentSetCatalogHasAllAuthoritativeEquipment(t *testing.T) {
	definition, ok := NewInitialCatalog().Get(CommandInventoryEquipmentSet)
	require.True(t, ok)
	require.Len(t, definition.Parameters, 3)
	require.Len(t, definition.Parameters[0].Choices, 48)

	for id, choice := range definition.Parameters[0].Choices {
		assert.Equal(t, strconv.Itoa(id), choice.Value)
		assert.Equal(t, references.Equipment(id).String(), choice.Label)
	}
}

func TestInitialCatalogValidatesActiveCombatConditionArguments(t *testing.T) {
	catalog := NewInitialCatalog()
	valid := Request{
		SessionID: "test", ID: "sleep", CommandID: CommandCombatMonsterSleepSet,
		Arguments: map[string]json.RawMessage{
			"encounterId":      json.RawMessage(`"4"`),
			"monsterId":        json.RawMessage(`"2"`),
			"asleep":           json.RawMessage(`true`),
			"expectedAsleep":   json.RawMessage(`false`),
			"expectedSnapshot": json.RawMessage(`"snapshot"`),
		},
	}
	require.NoError(t, catalog.ValidateRequest(valid))

	delete(valid.Arguments, "expectedSnapshot")
	require.ErrorIs(t, catalog.ValidateRequest(valid), ErrInvalidRequest)
}

func TestInitialCatalogValidatesActiveCombatExitArguments(t *testing.T) {
	catalog := NewInitialCatalog()
	valid := Request{
		SessionID: "test", ID: "exit", CommandID: CommandCombatExitDebug,
		Arguments: map[string]json.RawMessage{
			"encounterId":      json.RawMessage(`"4"`),
			"expectedSnapshot": json.RawMessage(`"snapshot"`),
		},
	}
	require.NoError(t, catalog.ValidateRequest(valid))

	valid.Arguments["encounterId"] = json.RawMessage(`"0"`)
	require.ErrorIs(t, catalog.ValidateRequest(valid), ErrInvalidRequest)
	valid.Arguments["encounterId"] = json.RawMessage(`"4"`)
	delete(valid.Arguments, "expectedSnapshot")
	require.ErrorIs(t, catalog.ValidateRequest(valid), ErrInvalidRequest)
}

func TestInitialCatalogValidatesActiveCombatSpawnArguments(t *testing.T) {
	catalog := NewInitialCatalog()
	valid := Request{
		SessionID: "test", ID: "spawn", CommandID: CommandCombatMonsterSpawn,
		Arguments: map[string]json.RawMessage{
			"encounterId": json.RawMessage(`"4"`), "expectedSnapshot": json.RawMessage(`"snapshot"`),
			"monsterType": json.RawMessage(`32`), "countMode": json.RawMessage(`"exact"`),
			"exactCount": json.RawMessage(`2`), "friends": json.RawMessage(`true`), "seed": json.RawMessage(`0`),
		},
	}
	require.NoError(t, catalog.ValidateRequest(valid))

	valid.Arguments["monsterType"] = json.RawMessage(`48`)
	require.ErrorIs(t, catalog.ValidateRequest(valid), ErrInvalidRequest)
	valid.Arguments["monsterType"] = json.RawMessage(`32`)
	valid.Arguments["exactCount"] = json.RawMessage(`17`)
	require.ErrorIs(t, catalog.ValidateRequest(valid), ErrInvalidRequest)
}

func TestInitialCatalogValidatesLoggingAndPresentationArguments(t *testing.T) {
	catalog := NewInitialCatalog()
	for _, request := range []Request{
		{SessionID: "test", ID: "category", CommandID: CommandLoggingCategorySet, Arguments: map[string]json.RawMessage{"category": json.RawMessage(`"combat"`), "enabled": json.RawMessage(`true`)}},
		{SessionID: "test", ID: "level", CommandID: CommandLoggingLevelSet, Arguments: map[string]json.RawMessage{"level": json.RawMessage(`"WARN"`)}},
		{SessionID: "test", ID: "overlay", CommandID: CommandSearchOverlaySet, Arguments: map[string]json.RawMessage{"enabled": json.RawMessage(`true`)}},
		{SessionID: "test", ID: "speed", CommandID: CommandCombatPresentationSpeedSet, Arguments: map[string]json.RawMessage{"percent": json.RawMessage(`400`)}},
	} {
		require.NoError(t, catalog.ValidateRequest(request))
	}

	invalid := Request{SessionID: "test", ID: "invalid", CommandID: CommandLoggingCategorySet, Arguments: map[string]json.RawMessage{"category": json.RawMessage(`"unknown"`), "enabled": json.RawMessage(`true`)}}
	require.ErrorIs(t, catalog.ValidateRequest(invalid), ErrInvalidRequest)
	invalid = Request{SessionID: "test", ID: "invalid", CommandID: CommandCombatPresentationSpeedSet, Arguments: map[string]json.RawMessage{"percent": json.RawMessage(`401`)}}
	require.ErrorIs(t, catalog.ValidateRequest(invalid), ErrInvalidRequest)
}

func TestCatalogRejectsDuplicateCommandID(t *testing.T) {
	definition := validTestDefinition()

	_, err := NewCatalog([]CommandDefinition{definition, definition})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate command ID")
}

func TestCatalogValidatesTypedArguments(t *testing.T) {
	minimum := 1
	maximum := 3
	definition := validTestDefinition()
	definition.Parameters = []ParameterDefinition{
		{
			ID:       "count",
			Name:     "Count",
			Kind:     ParameterInteger,
			Required: true,
			Minimum:  &minimum,
			Maximum:  &maximum,
		},
		{
			ID:       "mode",
			Name:     "Mode",
			Kind:     ParameterEnum,
			Required: true,
			Choices: []Choice{
				{Value: "one", Label: "One", Metadata: nil},
				{Value: "two", Label: "Two", Metadata: nil},
			},
		},
	}
	catalog, err := NewCatalog([]CommandDefinition{definition})
	require.NoError(t, err)

	err = catalog.ValidateRequest(Request{
		SessionID: "test-session",
		ID:        "request-1",
		CommandID: definition.ID,
		Arguments: map[string]json.RawMessage{
			"count": json.RawMessage(`2`),
			"mode":  json.RawMessage(`"two"`),
		},
	})
	require.NoError(t, err)

	err = catalog.ValidateRequest(Request{
		SessionID: "test-session",
		ID:        "request-2",
		CommandID: definition.ID,
		Arguments: map[string]json.RawMessage{
			"count": json.RawMessage(`4`),
			"mode":  json.RawMessage(`"three"`),
		},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidRequest)
}

func TestCatalogRejectsUnknownCommand(t *testing.T) {
	catalog := NewInitialCatalog()

	err := catalog.ValidateRequest(Request{SessionID: "test-session", ID: "request-1", CommandID: "unknown.command"})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCommandNotFound))
}

func TestCatalogReturnsDefensiveCopies(t *testing.T) {
	minimum := 1
	definitionWithBounds := validTestDefinition()
	definitionWithBounds.Parameters = []ParameterDefinition{{
		ID:      "count",
		Name:    "Count",
		Kind:    ParameterInteger,
		Minimum: &minimum,
	}}
	catalog, err := NewCatalog([]CommandDefinition{definitionWithBounds})
	require.NoError(t, err)
	minimum = 10
	definitions := catalog.Definitions()
	*definitions[0].Parameters[0].Minimum = 20

	definition, ok := catalog.Get(definitionWithBounds.ID)

	require.True(t, ok)
	assert.Equal(t, 1, *definition.Parameters[0].Minimum)
}

func TestCatalogNormalizesDefaultsWithoutMutatingRequest(t *testing.T) {
	definition := validTestDefinition()
	definition.Parameters = []ParameterDefinition{{
		ID:      "count",
		Name:    "Count",
		Kind:    ParameterInteger,
		Default: json.RawMessage(`2`),
	}}
	catalog, err := NewCatalog([]CommandDefinition{definition})
	require.NoError(t, err)
	request := Request{SessionID: "test-session", ID: "request-1", CommandID: definition.ID}

	normalized, err := catalog.NormalizeRequest(request)

	require.NoError(t, err)
	assert.Nil(t, request.Arguments)
	assert.JSONEq(t, `2`, string(normalized.Arguments["count"]))
}

func TestNewNormalizedInvocationDefensivelyCopiesTypedArguments(t *testing.T) {
	argument := json.RawMessage(`{"count":2,"enabled":true}`)
	request := Request{
		CommandID: "test.command",
		Arguments: map[string]json.RawMessage{"settings": argument},
	}

	invocation := NewNormalizedInvocation(request)
	argument[9] = '9'
	delete(request.Arguments, "settings")

	assert.Equal(t, "test.command", invocation.CommandID)
	assert.JSONEq(t, `{"count":2,"enabled":true}`, string(invocation.Arguments["settings"]))
	empty := NewNormalizedInvocation(Request{CommandID: "test.empty"})
	assert.NotNil(t, empty.Arguments)
	assert.Empty(t, empty.Arguments)
}

func TestCatalogRejectsNullAndInvalidDefaults(t *testing.T) {
	definition := validTestDefinition()
	definition.Parameters = []ParameterDefinition{{
		ID:      "count",
		Name:    "Count",
		Kind:    ParameterInteger,
		Default: json.RawMessage(`null`),
	}}

	_, err := NewCatalog([]CommandDefinition{definition})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidCatalog)
}

func TestCloneDiscoverySnapshotDetachesDynamicChoiceMetadata(t *testing.T) {
	original := DiscoverySnapshot{
		SceneGeneration: 2,
		OptionsRevision: 3,
		Commands:        NewInitialCatalog().Definitions(),
		OptionSets: []OptionSet{{
			Provider: OptionProviderActivePartyCharacters,
			Choices: []Choice{{
				Value:    "0.QXZhdGFy",
				Label:    "Position 1 · Avatar",
				Metadata: json.RawMessage(`{"strength":20}`),
			}},
			UnavailableReason: "outside combat only",
		}},
	}

	cloned := CloneDiscoverySnapshot(original)
	cloned.Commands[8].Parameters[0].ScenarioChoices[0].Label = "changed scenario choice"
	cloned.OptionSets[0].Choices[0].Metadata[0] = '['
	cloned.OptionSets[0].Choices[0].Label = "changed"
	cloned.OptionSets[0].UnavailableReason = "changed"

	assert.JSONEq(t, `{"strength":20}`, string(original.OptionSets[0].Choices[0].Metadata))
	assert.Equal(t, "Party Position 1", original.Commands[8].Parameters[0].ScenarioChoices[0].Label)
	assert.Equal(t, "Position 1 · Avatar", original.OptionSets[0].Choices[0].Label)
	assert.Equal(t, "outside combat only", original.OptionSets[0].UnavailableReason)
}

func validTestDefinition() CommandDefinition {
	return CommandDefinition{
		ID:             "test.command",
		Category:       "Test",
		Name:           "Test Command",
		Description:    "A test command.",
		ExecutionModes: []ExecutionMode{ExecutionQuery},
	}
}
