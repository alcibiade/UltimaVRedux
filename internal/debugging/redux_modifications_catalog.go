package debugging

import "github.com/bradhannah/UltimaVRedux/internal/config"

const (
	categoryReduxModifications = "Redux Modifications"

	// CommandReduxModificationsStatus reports persisted Redux gameplay choices.
	CommandReduxModificationsStatus = "redux.modifications.status"
	// CommandReduxCombatRepeatedMonsterDivisionSet changes repeated SPLITS behavior.
	CommandReduxCombatRepeatedMonsterDivisionSet = "redux.combat.repeated-monster-division.set"
	// CommandReduxDungeonsPersistHiddenDiscoveriesSet changes hidden-feature retention between visits.
	CommandReduxDungeonsPersistHiddenDiscoveriesSet = "redux.dungeons.persist-hidden-discoveries.set"
	// CommandReduxDungeonsSubduedUnlitVisibilitySet changes cosmetic unlit dungeon visibility.
	CommandReduxDungeonsSubduedUnlitVisibilitySet = "redux.dungeons.subdued-unlit-visibility.set"
	// CommandReduxPresentationSmoothSurfaceMovementSet changes optional surface camera scrolling.
	CommandReduxPresentationSmoothSurfaceMovementSet = "redux.presentation.smooth-surface-movement.set"
	// CommandReduxNavigationFlexibleSextantUseSet changes optional sextant navigation policy.
	CommandReduxNavigationFlexibleSextantUseSet = "redux.navigation.flexible_sextant_use.set"
	// CommandReduxConfirmQuickSaveSet persists quick-save confirmation.
	CommandReduxConfirmQuickSaveSet = "redux.saves.confirm-quick-save.set"
	// CommandReduxCombatHPBarsSet persists battlefield health-bar visibility.
	CommandReduxCombatHPBarsSet = "redux.combat.health-bars.set"
	// CommandReduxActionPopupsSet persists the maximum simultaneous action cards.
	CommandReduxActionPopupsSet = "redux.presentation.action-popups.set"
)

func reduxModificationCommands() []CommandDefinition {
	spec := config.ActionPopupsSpec()
	minimum, maximum, step := int(spec.Min), int(spec.Max), int(spec.Step)

	return []CommandDefinition{
		{
			ID:             CommandReduxModificationsStatus,
			Category:       categoryReduxModifications,
			Name:           "Redux Gameplay Modifications",
			Description:    "Inspect persisted Redux gameplay rules and original-behavior overrides.",
			Aliases:        []string{"redux settings", "gameplay modifications"},
			Parameters:     nil,
			ExecutionModes: []ExecutionMode{ExecutionQuery},
			Destructive:    false,
		},
		{
			ID:          CommandReduxCombatRepeatedMonsterDivisionSet,
			Category:    categoryReduxModifications,
			Name:        "Set Repeated Monster Division",
			Description: "Persist whether surviving Slimes, Gargoyles, and their children may divide repeatedly.",
			Aliases:     []string{"repeated splits", "original monster division"},
			Parameters: []ParameterDefinition{{
				ID: parameterEnabled, Name: "Allow Repeated Division",
				Description: "Enable repeated source-authentic division instead of the Redux one-child limit.",
				Kind:        ParameterBoolean, Required: true, ControlHint: controlHintSelect,
			}},
			ExecutionModes:   []ExecutionMode{ExecutionLiveSafe},
			Destructive:      false,
			ScenarioExcluded: true,
		},
		{
			ID: CommandReduxConfirmQuickSaveSet, Category: categoryReduxModifications, Name: "Confirm Quick Save",
			Description:    "Persist confirmation prompts for Q quick saves; Off still reports success.",
			Parameters:     []ParameterDefinition{{ID: parameterEnabled, Name: "Confirm Quick Save", Kind: ParameterBoolean, Required: true, ControlHint: controlHintSelect}},
			ExecutionModes: []ExecutionMode{ExecutionLiveSafe}, ScenarioExcluded: true,
		},
		{
			ID: CommandReduxCombatHPBarsSet, Category: categoryReduxModifications, Name: "Show Combat HP Bars",
			Description:    "Persist battlefield health bars; Off adds monster injury descriptions. Fleeing indicators remain visible.",
			Parameters:     []ParameterDefinition{{ID: parameterEnabled, Name: "Show Combat HP Bars", Kind: ParameterBoolean, Required: true, ControlHint: controlHintSelect}},
			ExecutionModes: []ExecutionMode{ExecutionLiveSafe}, ScenarioExcluded: true,
		},
		{
			ID: CommandReduxActionPopupsSet, Category: categoryReduxModifications, Name: "Maximum Action Popups",
			Description:    "Persist the simultaneous action-popup limit; zero is Off and messages remain in the output panel.",
			Parameters:     []ParameterDefinition{{ID: "count", Name: "Maximum Action Popups", Kind: ParameterInteger, Required: true, Minimum: &minimum, Maximum: &maximum, Step: &step, ControlHint: controlHintNumber}},
			ExecutionModes: []ExecutionMode{ExecutionLiveSafe}, ScenarioExcluded: true,
		},
		{
			ID:          CommandReduxDungeonsPersistHiddenDiscoveriesSet,
			Category:    categoryReduxModifications,
			Name:        "Set Persistent Hidden Dungeon Discoveries",
			Description: "Persist revealed hidden dungeon doors and pits between fresh visits.",
			Aliases:     []string{"persistent dungeon discoveries", "remember hidden doors"},
			Parameters: []ParameterDefinition{{
				ID: parameterEnabled, Name: "Persist Hidden Discoveries",
				Description: "Enable Redux persistence instead of original per-visit discovery reset.",
				Kind:        ParameterBoolean, Required: true, ControlHint: controlHintSelect,
			}},
			ExecutionModes:   []ExecutionMode{ExecutionLiveSafe},
			Destructive:      false,
			ScenarioExcluded: true,
		},
		{
			ID:          CommandReduxDungeonsSubduedUnlitVisibilitySet,
			Category:    categoryReduxModifications,
			Name:        "Set Subdued Unlit Dungeon Visibility",
			Description: "Show deterministic dim dungeon geometry when no real light is active.",
			Aliases:     []string{"dim dungeons", "unlit dungeon visibility"},
			Parameters: []ParameterDefinition{{
				ID: parameterEnabled, Name: "Show Subdued Unlit Dungeons",
				Description: "Enable Redux subdued visibility instead of original binary black.",
				Kind:        ParameterBoolean, Required: true, ControlHint: controlHintSelect,
			}},
			ExecutionModes:   []ExecutionMode{ExecutionLiveSafe},
			Destructive:      false,
			ScenarioExcluded: true,
		},
		{
			ID:          CommandReduxPresentationSmoothSurfaceMovementSet,
			Category:    categoryReduxModifications,
			Name:        "Set Smooth Surface Movement",
			Description: "Persist optional camera scrolling while walking, riding, flying or boating; mounts animate 25% faster and automatic sailing follows its cadence. Gameplay timing, combat and dungeons are unchanged.",
			Aliases:     []string{"smooth movement", "surface scrolling"},
			Parameters: []ParameterDefinition{{
				ID: parameterEnabled, Name: "Smooth Surface Movement",
				Description: "Enable optional Redux camera scrolling; disabled by default.",
				Kind:        ParameterBoolean, Required: true, ControlHint: controlHintSelect,
			}},
			ExecutionModes:   []ExecutionMode{ExecutionLiveSafe},
			Destructive:      false,
			ScenarioExcluded: true,
		},
		{
			ID:          CommandReduxNavigationFlexibleSextantUseSet,
			Category:    categoryReduxModifications,
			Name:        "Set Flexible Sextant Use",
			Description: "Persist optional daytime and settlement sextant use on all floors, including cellars; dungeons, Underworld and ownership requirements remain excluded.",
			Aliases:     []string{"flexible sextant", "sextant navigation"},
			Parameters: []ParameterDefinition{{
				ID: parameterEnabled, Name: "Flexible sextant use",
				Description: "Enable flexible navigation instead of original nighttime world-only use; disabled by default.",
				Kind:        ParameterBoolean, Required: true, ControlHint: controlHintSelect,
			}},
			ExecutionModes:   []ExecutionMode{ExecutionLiveSafe},
			Destructive:      false,
			ScenarioExcluded: true,
		},
	}
}
