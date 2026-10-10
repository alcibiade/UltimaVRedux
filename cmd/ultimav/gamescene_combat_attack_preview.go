package main

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func (g *GameScene) combatAttackPreview() game_state.CombatAttackPreview {
	frame := &g.combatRenderFrame
	if !frame.active {
		return g.computeCombatAttackPreview()
	}
	if !frame.previewReady {
		frame.preview = g.computeCombatAttackPreview()
		frame.previewReady = true
	}

	return frame.preview
}

func (g *GameScene) computeCombatAttackPreview() game_state.CombatAttackPreview {
	combat := g.gameState.CombatScene
	if combat == nil || !combat.CombatTurnsStarted() || g.inputBlocked ||
		(g.secondaryKeyState != PrimaryInput && g.secondaryKeyState != CombatAttackTargetInput) ||
		combatTurnIsAutoPass(combat, combat.GetCurrentTurnIdentity()) {
		return game_state.CombatAttackPreview{}
	}

	return g.gameState.PreviewCombatAttack()
}

func (g *GameScene) drawCombatAttackEligibility(screen *ebiten.Image) {
	preview := g.combatAttackPreview()
	for _, target := range preview.Targets {
		if target.Actor == nil {
			continue // Assistance legality does not confer renderer disclosure.
		}
		if preview.Aim != nil && target.Position == preview.Aim.Position {
			continue // The existing aiming reticle owns the selected square.
		}
		drawCombatAttackBrackets(screen, g.combatEntityDrawOptions(target.Position, false, target.MonsterIndex, 0))
	}
}

// Approved foreground foot cues stay inside each 16-pixel cell. Their complete
// backing ends before the health-frame outline at y=12 and remains inward of flee/status
// marks. Draw after every actor, but before the aiming reticle, so neighboring
// sprites cannot erase either two-pixel arm or its one-pixel black border.
func combatAttackBracketFaces() [4]image.Rectangle {
	return [4]image.Rectangle{
		image.Rect(5, 9, 6, 11), image.Rect(5, 10, 7, 11),
		image.Rect(11, 9, 12, 11), image.Rect(10, 10, 12, 11),
	}
}

func drawCombatAttackBrackets(screen *ebiten.Image, options *ebiten.DrawImageOptions) {
	faces := combatAttackBracketFaces()
	for _, border := range []bool{true, false} {
		for _, rect := range faces {
			clr := ucolor.CombatAttackEligibleFace
			if border {
				rect = rect.Inset(-1)
				clr = ucolor.CombatAttackEligibleBorder
			}
			left, top := options.GeoM.Apply(float64(rect.Min.X), float64(rect.Min.Y))
			right, bottom := options.GeoM.Apply(float64(rect.Max.X), float64(rect.Max.Y))
			left, top = math.Round(left), math.Round(top)
			right, bottom = math.Round(right), math.Round(bottom)
			vector.FillRect(screen, float32(left), float32(top), float32(right-left), float32(bottom-top), clr, false)
		}
	}
}

func appendCombatAttackReadout(output *text.Output, preview game_state.CombatAttackPreview) {
	for _, source := range preview.Sources {
		name := source.Equipment.String()
		if source.Equipment == references.NoEquipment {
			name = "Bare Hands"
		}
		output.AppendLine().White(name).LightGrayf(" (R%d)", source.MaxRange).Done()
	}
	if preview.Aim == nil {
		return
	}
	target := preview.Aim.Name
	if target == "" {
		target = "No one seen"
	}
	if preview.Aim.Status == game_state.CombatShotChooseTarget {
		target = "Choose target"
	}
	distanceSeparator := " "
	if preview.Aim.Actor != nil && preview.Aim.Actor.Identity.Slot != 0 {
		// Keep the full nine-character PC name plus Clone and distance in the
		// fixed pane. The differently coloured parenthesis remains a separator.
		distanceSeparator = ""
	}
	output.AppendLine().White(target).LightGrayf("%s(D%d)", distanceSeparator, preview.Aim.Distance).Done()
	label := combatAttackActionLabel
	separator := ": "
	if preview.Aim.Actor != nil {
		label += " " + combatAllegianceText(*preview.Aim.Actor)
		separator = ":"
	}
	output.AppendLine().Text(label+separator, widgets.DialogLabelColor).Text(combatShotStatusText(preview.Aim.Status), widgets.DialogValueColor).Done()
}

func combatShotStatusText(status game_state.CombatShotStatus) string {
	if status == game_state.CombatShotChooseTarget {
		return "Aim"
	}
	if status == game_state.CombatShotTerrainBypass {
		return "Bypass"
	}
	if status == game_state.CombatShotOutOfRange {
		return "Too far"
	}
	if status == game_state.CombatShotInterference {
		return "Engaged"
	}
	if status == game_state.CombatShotUnready {
		return "Unready"
	}

	return string(status)
}
