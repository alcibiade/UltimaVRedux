package main

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const merchantUnimplementedReason = "This merchant family is not implemented."

var errMerchantTesterSpriteUnavailable = errors.New("canonical merchant sprite is unavailable")

func (g *GameScene) executeMerchantTesterDebugCommand(
	request debugging.Request,
) (debugging.Result, bool) {
	switch request.CommandID {
	case debugging.CommandMerchantTesterCatalog:
		data := g.newMerchantTesterCatalogData()

		return debugSuccessResult(
			request,
			fmt.Sprintf("Merchant Tester catalog loaded: %d canonical merchants.", len(data.Merchants)),
			data,
		), true
	case debugging.CommandMerchantTesterPreviewOpen:
		return g.executeMerchantTesterPreview(request), true
	case debugging.CommandMerchantTesterLiveOpen:
		return g.executeMerchantTesterLive(request), true
	default:
		return debugging.Result{}, false
	}
}

func (g *GameScene) executeMerchantTesterPreview(request debugging.Request) debugging.Result {
	merchant, failure := g.merchantTesterDefinition(request)
	if failure != nil {
		return *failure
	}
	if code, message, blocked := g.merchantTesterLaunchBlock(); blocked {
		return debugging.FailureResult(request, code, message)
	}
	switch merchant.Family {
	case references.MerchantFamilyArms:
		review, err := g.prepareArmsMerchantPreview(merchant.ID)
		if err != nil {
			return debugging.FailureResult(request, "merchant_unavailable", err.Error())
		}
		g.startArmsMerchantPreviewWorkflow(review)
	case references.MerchantFamilyGuild:
		review, err := g.prepareGuildMerchantPreview(merchant.ID)
		if err != nil {
			return debugging.FailureResult(request, "merchant_unavailable", err.Error())
		}
		g.startGuildMerchantPreviewWorkflow(review)
	case references.MerchantFamilyHealer:
		review, err := g.prepareHealerMerchantPreview(merchant.ID)
		if err != nil {
			return debugging.FailureResult(request, "merchant_unavailable", err.Error())
		}
		g.startHealerMerchantPreviewWorkflow(review)
	case references.MerchantFamilyReagent:
		review, err := g.prepareReagentMerchantPreview(merchant.ID)
		if err != nil {
			return debugging.FailureResult(request, "merchant_unavailable", err.Error())
		}
		g.startReagentMerchantPreviewWorkflow(review)
	case references.MerchantFamilyHorse:
		review, err := g.prepareHorseMerchantPreview(merchant.ID)
		if err != nil {
			return debugging.FailureResult(request, "merchant_unavailable", err.Error())
		}
		g.startHorseMerchantPreviewWorkflow(review)
	case references.MerchantFamilyShip:
		review, err := g.prepareShipMerchantPreview(merchant.ID)
		if err != nil {
			return debugging.FailureResult(request, "merchant_unavailable", err.Error())
		}
		g.startShipMerchantPreviewWorkflow(review)
	case references.MerchantFamilyInn:
		review, err := g.prepareInnMerchantPreview(merchant.ID)
		if err != nil {
			return debugging.FailureResult(request, "merchant_unavailable", err.Error())
		}
		g.startInnMerchantPreviewWorkflow(review)
	case references.MerchantFamilyTavern:
		review, err := g.prepareTavernMerchantPreview(merchant.ID)
		if err != nil {
			return debugging.FailureResult(request, "merchant_unavailable", err.Error())
		}
		g.startTavernMerchantPreviewWorkflow(review)
	default:
		panic("merchant tester preview reached an unknown family")
	}

	return debugSuccessResultWithMode(
		request,
		debugging.ExecutionWorkflow,
		"Merchant preview workflow queued; no transaction has been committed.",
		newMerchantTesterOpenData(g.gameState, merchant, true, false),
	)
}

func (g *GameScene) executeMerchantTesterLive(request debugging.Request) debugging.Result {
	merchant, failure := g.merchantTesterDefinition(request)
	if failure != nil {
		return *failure
	}
	if request.ExpectedSceneGeneration == 0 {
		return debugging.FailureResult(
			request,
			"stale_scene",
			"an expected scene generation is required for a live merchant launch",
		)
	}
	if code, message, blocked := g.merchantTesterLaunchBlock(); blocked {
		return debugging.FailureResult(request, code, message)
	}
	if !reviewedWorldPositionMatches(request, g.gameState) ||
		mustDebugStringArgument(request, "expectedSourceFingerprint") != debugWorldSourceFingerprint(g.gameState) {
		return debugging.FailureResult(
			request,
			"stale_merchant_context",
			"world state changed after the Merchant Tester catalog was reviewed",
		)
	}
	if err := g.validateMerchantTesterPreview(merchant); err != nil {
		return debugging.FailureResult(request, "merchant_unavailable", err.Error())
	}

	if merchant.Location == g.gameState.MapState.PlayerLocation.Location {
		if _, err := g.gameState.ReviewDebugMerchantSessionByID(merchant.ID); err != nil {
			return debugging.FailureResult(request, "merchant_unavailable", err.Error())
		}
		g.startDebugMerchantTalkWorkflow(merchant.ID)

		return debugSuccessResultWithMode(
			request,
			debugging.ExecutionWorkflow,
			"Live merchant workflow queued; no transaction has completed.",
			newMerchantTesterOpenData(g.gameState, merchant, false, false),
		)
	}

	if err := g.gameState.ValidateDebugMerchantLocationTransition(merchant.ID); err != nil {
		return debugging.FailureResult(request, "merchant_context_unsupported", err.Error())
	}
	operation, err := g.gameState.DebugEnterMerchantLocation(merchant.ID)
	if err != nil {
		return debugging.FailureResult(request, "merchant_setup_failed", err.Error())
	}
	g.startDebugMerchantTalkWorkflow(merchant.ID)
	result := debugSuccessResultWithMode(
		request,
		debugging.ExecutionAlteration,
		"Merchant town setup and live workflow queued; no transaction has completed.",
		newMerchantTesterOpenData(g.gameState, merchant, false, true),
	)
	appendWorldSetupChanges(&result.Changes, operation)

	return result
}

func (g *GameScene) merchantTesterDefinition(
	request debugging.Request,
) (references.MerchantDefinition, *debugging.Result) {
	if g.gameState == nil || g.gameState.GameReferences == nil ||
		g.gameState.GameReferences.MerchantReferences == nil {
		result := debugging.FailureResult(request, "merchant_unavailable", "merchant references are unavailable")

		return references.MerchantDefinition{}, &result
	}
	id := references.MerchantID(mustDebugStringArgument(request, "merchantId"))
	merchant, found := g.gameState.GameReferences.MerchantReferences.Definition(id)
	if !found {
		result := debugging.FailureResult(request, "merchant_unknown", "unknown merchant ID")

		return references.MerchantDefinition{}, &result
	}
	if !merchantTesterImplemented(merchant) {
		result := debugging.FailureResult(request, "merchant_unimplemented", merchantUnimplementedReason)

		return references.MerchantDefinition{}, &result
	}

	return merchant, nil
}

func (g *GameScene) prepareGuildMerchantPreview(
	id references.MerchantID,
) (game_state.GuildShopReview, error) {
	review, err := g.gameState.ReviewGuildShopByMerchantID(id)
	if err != nil {
		return game_state.GuildShopReview{}, fmt.Errorf("reviewing Guild merchant: %w", err)
	}
	if _, err = g.resolveGuildShopProse(review.State); err != nil {
		return game_state.GuildShopReview{}, err
	}
	if g.spriteSheet == nil || g.spriteSheet.GetSprite(review.MerchantSpriteIndex) == nil {
		return game_state.GuildShopReview{}, errMerchantTesterSpriteUnavailable
	}

	return review, nil
}

func (g *GameScene) prepareArmsMerchantPreview(
	id references.MerchantID,
) (game_state.ArmsShopReview, error) {
	review, err := g.gameState.ReviewArmsShopByMerchantID(id)
	if err != nil {
		return game_state.ArmsShopReview{}, fmt.Errorf("reviewing Arms merchant: %w", err)
	}
	if _, err = g.resolveArmsShopProse(review.State); err != nil {
		return game_state.ArmsShopReview{}, err
	}
	if g.spriteSheet == nil || g.spriteSheet.GetSprite(review.MerchantSpriteIndex) == nil {
		return game_state.ArmsShopReview{}, errMerchantTesterSpriteUnavailable
	}

	return review, nil
}

func (g *GameScene) prepareHealerMerchantPreview(
	id references.MerchantID,
) (game_state.HealerShopReview, error) {
	review, err := g.gameState.ReviewHealerShopByMerchantID(id)
	if err != nil {
		return game_state.HealerShopReview{}, fmt.Errorf("reviewing healer merchant: %w", err)
	}
	if _, err = g.resolveHealerShopProse(review.State); err != nil {
		return game_state.HealerShopReview{}, err
	}
	if g.spriteSheet == nil || g.spriteSheet.GetSprite(review.MerchantSpriteIndex) == nil {
		return game_state.HealerShopReview{}, errMerchantTesterSpriteUnavailable
	}

	return review, nil
}

func (g *GameScene) prepareReagentMerchantPreview(
	id references.MerchantID,
) (game_state.ReagentShopReview, error) {
	review, err := g.gameState.ReviewReagentShopByMerchantID(id)
	if err != nil {
		return game_state.ReagentShopReview{}, fmt.Errorf("reviewing reagent merchant: %w", err)
	}
	if _, err = g.resolveReagentShopProse(review.State); err != nil {
		return game_state.ReagentShopReview{}, err
	}
	if g.spriteSheet == nil || g.spriteSheet.GetSprite(review.MerchantSpriteIndex) == nil {
		return game_state.ReagentShopReview{}, errMerchantTesterSpriteUnavailable
	}

	return review, nil
}

func (g *GameScene) prepareHorseMerchantPreview(
	id references.MerchantID,
) (game_state.HorseShopReview, error) {
	review, err := g.gameState.ReviewHorseShopByMerchantID(id)
	if err != nil {
		return game_state.HorseShopReview{}, fmt.Errorf("reviewing horse merchant: %w", err)
	}
	if _, err = g.resolveHorseShopProse(review.State); err != nil {
		return game_state.HorseShopReview{}, err
	}
	if g.spriteSheet == nil || g.spriteSheet.GetSprite(review.MerchantSpriteIndex) == nil ||
		g.spriteSheet.GetSprite(indexes.HorseRight) == nil {
		return game_state.HorseShopReview{}, errMerchantTesterSpriteUnavailable
	}

	return review, nil
}

func (g *GameScene) prepareShipMerchantPreview(
	id references.MerchantID,
) (game_state.ShipShopReview, error) {
	review, err := g.gameState.ReviewShipShopByMerchantID(id)
	if err != nil {
		return game_state.ShipShopReview{}, fmt.Errorf("reviewing ship merchant: %w", err)
	}
	if _, err = g.resolveShipShopProse(review.State); err != nil {
		return game_state.ShipShopReview{}, err
	}
	if g.spriteSheet == nil || g.spriteSheet.GetSprite(review.MerchantSpriteIndex) == nil ||
		g.spriteSheet.GetSprite(indexes.ShipNoSailsRight) == nil ||
		g.spriteSheet.GetSprite(indexes.SkiffRight) == nil {
		return game_state.ShipShopReview{}, errMerchantTesterSpriteUnavailable
	}

	return review, nil
}

func (g *GameScene) prepareInnMerchantPreview(
	id references.MerchantID,
) (game_state.InnShopReview, error) {
	review, err := g.gameState.ReviewInnShopByMerchantID(id)
	if err != nil {
		return game_state.InnShopReview{}, fmt.Errorf("reviewing Inn merchant: %w", err)
	}
	if _, err = g.resolveInnShopProse(review.State); err != nil {
		return game_state.InnShopReview{}, err
	}
	if g.spriteSheet == nil || g.spriteSheet.GetSprite(review.MerchantSpriteIndex) == nil {
		return game_state.InnShopReview{}, errMerchantTesterSpriteUnavailable
	}

	return review, nil
}

func (g *GameScene) prepareTavernMerchantPreview(
	id references.MerchantID,
) (game_state.TavernShopReview, error) {
	review, err := g.gameState.ReviewTavernShopByMerchantID(id)
	if err != nil {
		return game_state.TavernShopReview{}, fmt.Errorf("reviewing Tavern merchant: %w", err)
	}
	if _, err = g.resolveTavernShopProse(review.State); err != nil {
		return game_state.TavernShopReview{}, err
	}
	if g.spriteSheet == nil || g.spriteSheet.GetSprite(review.MerchantSpriteIndex) == nil {
		return game_state.TavernShopReview{}, errMerchantTesterSpriteUnavailable
	}

	return review, nil
}

func (g *GameScene) validateMerchantTesterPreview(merchant references.MerchantDefinition) error {
	switch merchant.Family {
	case references.MerchantFamilyArms:
		_, err := g.prepareArmsMerchantPreview(merchant.ID)

		return err
	case references.MerchantFamilyGuild:
		_, err := g.prepareGuildMerchantPreview(merchant.ID)

		return err
	case references.MerchantFamilyHealer:
		_, err := g.prepareHealerMerchantPreview(merchant.ID)

		return err
	case references.MerchantFamilyReagent:
		_, err := g.prepareReagentMerchantPreview(merchant.ID)

		return err
	case references.MerchantFamilyHorse:
		_, err := g.prepareHorseMerchantPreview(merchant.ID)

		return err
	case references.MerchantFamilyShip:
		_, err := g.prepareShipMerchantPreview(merchant.ID)

		return err
	case references.MerchantFamilyInn:
		_, err := g.prepareInnMerchantPreview(merchant.ID)

		return err
	case references.MerchantFamilyTavern:
		_, err := g.prepareTavernMerchantPreview(merchant.ID)

		return err
	default:
		panic("merchant tester validation reached an unknown family")
	}
}

func (g *GameScene) merchantTesterLaunchBlock() (string, string, bool) {
	switch {
	case g.gameState == nil:
		return "gameplay_unavailable", "Merchant Tester requires active gameplay", true
	case g.dialogStack.HasOpenDialog():
		return "dialog_active", "Merchant Tester is unavailable while a dialog is open", true
	case g.gameState.EventQueueManager.HasActiveWorkflows():
		return "gameplay_workflow_active", "Merchant Tester is unavailable during a gameplay workflow", true
	case g.miniMapScene != nil || g.gameState.ShrineScene != nil:
		return "scripted_scene_active", "Merchant Tester is unavailable during a scripted scene", true
	case g.gameState.IsInCombat() ||
		g.gameState.MapState.PlayerLocation.Location.GetMapType() == references.CombatMapType ||
		g.gameState.MapState.PlayerLocation.Location.GetMapType() == references.CampingMapType:
		return "gameplay_context_unsupported", "Merchant Tester is unavailable in combat or camping", true
	case g.timedOverlayDialog != nil || g.secondaryKeyState != PrimaryInput || g.inputBlocked ||
		len(g.transientActionFeedback) != 0 || len(g.visualEffects) != 0 ||
		len(g.combatImpactEffects) != 0 || len(g.combatVanishEffects) != 0 || len(g.combatSignatureEffects) != 0 || g.combatMovement.Active ||
		(g.autoOpenDoorState != nil && g.autoOpenDoorState.IsActive()) ||
		g.avatarDissolve.IsActive() || g.wandererDissolve.IsActive() || len(g.activeTileDissolves) != 0 ||
		g.mapFadeLevel != 0 || g.screenFlashIntensity != 0 || g.screenShakeOffsetX != 0 ||
		g.screenShakeOffsetY != 0 || g.screenInvertIntensity != 0 || g.xRayVisionActive || g.activeProjectile != nil:
		return "transient_context_active", "Merchant Tester is unavailable during transient gameplay presentation", true
	default:
		return "", "", false
	}
}

func (g *GameScene) newMerchantTesterCatalogData() debugging.MerchantTesterCatalogData {
	current := g.gameState.MapState.PlayerLocation
	data := debugging.MerchantTesterCatalogData{
		SourceFingerprint: debugWorldSourceFingerprint(g.gameState),
		CurrentContext: debugging.MerchantTesterContextData{
			LocationID: int(current.Location), LocationName: current.Location.String(),
			Floor: int(current.Floor), X: int(current.Position.X), Y: int(current.Position.Y),
		},
		Merchants: g.merchantTesterRows(),
	}

	return data
}

func (g *GameScene) merchantTesterRows() []debugging.MerchantTesterMerchantData {
	if g.gameState.GameReferences == nil || g.gameState.GameReferences.MerchantReferences == nil {
		return make([]debugging.MerchantTesterMerchantData, 0)
	}
	definitions := g.gameState.GameReferences.MerchantReferences.Definitions()
	rows := make([]debugging.MerchantTesterMerchantData, 0, len(definitions))
	current := g.gameState.MapState.PlayerLocation
	blockCode, blockMessage, blocked := g.merchantTesterLaunchBlock()
	for _, merchant := range definitions {
		familyID, familyLabel := merchantTesterFamilyIdentity(merchant.Family)
		row := debugging.MerchantTesterMerchantData{
			MerchantID: string(merchant.ID), TownID: int(merchant.Location), TownName: merchant.Location.String(),
			ShopName: merchant.ShopName, OwnerName: merchant.OwnerName,
			FamilyID: familyID, FamilyLabel: familyLabel, DialogCode: int(merchant.Family),
			Implemented:       merchantTesterImplemented(merchant),
			LiveExecutionMode: debugging.ExecutionAlteration,
		}
		if merchant.Location == current.Location {
			row.LiveExecutionMode = debugging.ExecutionWorkflow
		}
		switch {
		case !row.Implemented:
			row.UnavailableReason = merchantUnimplementedReason
			row.PreviewUnavailableReason = merchantUnimplementedReason
			row.LiveUnavailableReason = merchantUnimplementedReason
		case blocked:
			reason := blockCode + ": " + blockMessage
			row.PreviewUnavailableReason = reason
			row.LiveUnavailableReason = reason
		default:
			if err := g.validateMerchantTesterPreview(merchant); err != nil {
				row.PreviewUnavailableReason = err.Error()
				row.LiveUnavailableReason = err.Error()

				break
			}
			row.PreviewAvailable = true
			if merchant.Location == current.Location {
				if _, err := g.gameState.ReviewDebugMerchantSessionByID(merchant.ID); err != nil {
					row.LiveUnavailableReason = err.Error()
				} else {
					row.LiveAvailable = true
				}
			} else if err := g.gameState.ValidateDebugMerchantLocationTransition(merchant.ID); err != nil {
				row.LiveUnavailableReason = err.Error()
			} else {
				row.LiveAvailable = true
			}
		}
		rows = append(rows, row)
	}

	return rows
}

func (g *GameScene) merchantTesterOptionSet() debugging.OptionSet {
	if g.gameState == nil || g.gameState.GameReferences == nil ||
		g.gameState.GameReferences.MerchantReferences == nil {
		return debugging.OptionSet{
			Provider: debugging.OptionProviderMerchantTesterMerchants,
			Choices:  make([]debugging.Choice, 0), UnavailableReason: "Merchant references are unavailable.",
		}
	}
	rows := g.merchantTesterRows()
	choices := make([]debugging.Choice, 0, len(rows))
	for _, row := range rows {
		choices = append(choices, debugging.Choice{
			Value:    row.MerchantID,
			Label:    fmt.Sprintf("%s | %s | %s (%s)", row.TownName, row.ShopName, row.OwnerName, row.FamilyLabel),
			Metadata: mustDebugJSON(row),
		})
	}

	return debugging.OptionSet{Provider: debugging.OptionProviderMerchantTesterMerchants, Choices: choices}
}

func merchantTesterImplemented(merchant references.MerchantDefinition) bool {
	switch merchant.Family {
	case references.MerchantFamilyArms, references.MerchantFamilyReagent,
		references.MerchantFamilyGuild, references.MerchantFamilyHealer,
		references.MerchantFamilyHorse, references.MerchantFamilyShip,
		references.MerchantFamilyInn, references.MerchantFamilyTavern:
		return true
	default:
		panic("merchant tester implementation check reached an unknown family")
	}
}

func merchantTesterFamilyIdentity(family references.MerchantFamily) (string, string) {
	switch family {
	case references.MerchantFamilyArms:
		return "arms", "Arms Dealer"
	case references.MerchantFamilyTavern:
		return "tavern", "Tavern"
	case references.MerchantFamilyHorse:
		return merchantHorseSemanticName, "Horse Dealer"
	case references.MerchantFamilyShip:
		return "ship", "Ship Dealer"
	case references.MerchantFamilyReagent:
		return "reagent", "Reagent Vendor"
	case references.MerchantFamilyGuild:
		return "guild", "Guild"
	case references.MerchantFamilyHealer:
		return "healer", "Healer"
	case references.MerchantFamilyInn:
		return "inn", "Inn"
	default:
		panic("unknown production merchant family")
	}
}

func newMerchantTesterOpenData(
	state *game_state.GameState,
	merchant references.MerchantDefinition,
	preview bool,
	transitioned bool,
) debugging.MerchantTesterOpenData {
	current := state.MapState.PlayerLocation

	return debugging.MerchantTesterOpenData{
		MerchantID: string(merchant.ID), TownID: int(merchant.Location), TownName: merchant.Location.String(),
		Preview: preview, Transitioned: transitioned, WorkflowQueued: true,
		LocationID: int(current.Location), Floor: int(current.Floor),
		X: int(current.Position.X), Y: int(current.Position.Y),
	}
}
