package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

// A one-source-pixel exterior surrounds the unchanged x2/y13/w13/h4 black
// health frame. Drawing four disjoint strips never paints over its HP interior.
func combatHealthFrameBounds() image.Rectangle { return image.Rect(1, 12, 16, 18) }

func (g *GameScene) showCombatHPBars() bool {
	return g.gameConfig == nil || g.gameConfig.GetShowCombatHPBars()
}

func drawCombatHealthFrameOutline(screen *ebiten.Image, options *ebiten.DrawImageOptions, loyalty game_state.CombatMonsterLoyalty) {
	clr := ucolor.CombatAllegianceEnemy
	if loyalty == game_state.CombatMonsterLoyaltyParty {
		clr = ucolor.CombatAllegianceAlly
	}
	for _, rect := range [4]image.Rectangle{
		image.Rect(1, 12, 16, 13), image.Rect(1, 17, 16, 18),
		image.Rect(1, 13, 2, 17), image.Rect(15, 13, 16, 17),
	} {
		// Use the HP renderer's transform/raster rules, including noninteger
		// scaling. Independently rounding these edges could invade its black rim.
		left, top := options.GeoM.Apply(float64(rect.Min.X), float64(rect.Min.Y))
		right, bottom := options.GeoM.Apply(float64(rect.Max.X), float64(rect.Max.Y))
		vector.FillRect(screen, float32(left), float32(top), float32(right-left), float32(bottom-top), clr, false)
	}
}

func (g *GameScene) drawCombatAllegianceHealthOutline(screen *ebiten.Image, options *ebiten.DrawImageOptions, identity game_state.CombatTurnIdentity) {
	if !g.showCombatHPBars() {
		return
	}
	if actor, disclosed := g.gameState.ProjectCombatActor(identity, 0); disclosed {
		drawCombatHealthFrameOutline(screen, options, actor.Loyalty)
	}
}

// Draw HP and allegiance together after every actor/status/effect silhouette.
// Neighboring 18-pixel sprites overlap 16-pixel cells and must not erase either
// the HP fill or its exterior. This path is combat-only; world/vehicle consumers
// of drawCombatHealthBar retain their original rendering.
func (g *GameScene) drawCombatHealthFrames(screen *ebiten.Image) {
	g.gameState.ForEachDisclosedCombatActor(func(actor game_state.CombatActorPresentation) {
		if !g.combatAllegiancePresented(actor) || g.combatSignatureHidesActor(actor.Identity) {
			return
		}
		options := g.combatEntityDrawOptions(actor.Position, actor.Identity.IsPartyMember, actor.Identity.EntityIndex, actor.Identity.Slot)
		g.drawCombatActorHealthFrame(screen, options, actor)
	})
	for _, effect := range g.combatSignatureEffects {
		options, visible := g.combatSignatureHealthOptions(effect)
		if !visible {
			continue
		}
		// The initial signature pass excludes this region. Apply the same mask
		// and opacity here exactly once, not over an already blended HP underlay.
		g.prepareCombatSignatureLayers(effect)
		screen.DrawImage(effect.HealthFrame, options)
	}
}

func (g *GameScene) drawCombatActorHealthFrame(screen *ebiten.Image, options *ebiten.DrawImageOptions, actor game_state.CombatActorPresentation) {
	if actor.Identity.IsPartyMember {
		if !g.showCombatHPBars() {
			return
		}
		character := g.gameState.PartyState.GetCharacter(actor.Identity.EntityIndex)
		drawCombatHealthFrameOutline(screen, options, actor.Loyalty)
		drawCombatHealthBar(screen, options, int(character.GetCurrentHp()), int(character.GetMaxHp()))

		return
	}
	stack := g.combatPartyInsectStack(actor.Position)
	if len(stack) > 0 {
		if stack[0] != actor.Identity.EntityIndex {
			return
		}
		if g.showCombatHPBars() {
			drawCombatHealthFrameOutline(screen, options, actor.Loyalty)
			drawCombatHealthBar(screen, options, 0, 1) // Black separator around the aggregate's unchanged segments.
			g.drawCombatInsectStackHealth(screen, options, stack)
		}
		g.drawCombatInsectStackIndicators(screen, options, stack)

		return
	}
	if !g.showCombatHPBars() {
		return
	}
	monster := g.gameState.CombatScene.MonsterManager.GetMonster(actor.Identity.EntityIndex)
	drawCombatHealthFrameOutline(screen, options, actor.Loyalty)
	drawCombatHealthBar(screen, options, monster.CurrentHP(), monster.MaxHP())
}

func (g *GameScene) combatSignatureHealthOptions(effect *combatSignatureEffect) (*ebiten.DrawImageOptions, bool) {
	if !g.combatSignatureEffectCurrent(effect) || effect.Kind == game_state.CombatEventPossession ||
		(effect.Kind == game_state.CombatEventGating && effect.GatePhase == combatGateArrival) {
		return nil, false
	}
	position, opacity := effect.To, float32(1)
	if effect.Kind == game_state.CombatEventTeleport {
		phase, progress := effect.blink(max(g.gameState.ElapsedMs-effect.StartedMs, 0))
		if phase == combatTeleportHidden {
			return nil, false
		}
		if phase == combatTeleportDeparture {
			position, opacity = effect.From, 1-progress
		} else {
			opacity = progress
		}
	}
	if !g.gameState.CombatTerrainAllowsObjectDisplay(position) {
		return nil, false
	}
	options := g.combatEntityDrawOptions(position, effect.Actor.IsPartyMember, effect.Actor.EntityIndex, effect.Actor.Slot)
	options.ColorScale.ScaleAlpha(opacity)

	return options, true
}

func (g *GameScene) combatSignatureImage(effect *combatSignatureEffect) *ebiten.Image {
	frame := g.composeCombatSignatureFrame(effect)
	if effect.Kind != game_state.CombatEventGating {
		return spriteWithDissolveMask(frame, &effect.Dissolve)
	}

	return frame
}

func (g *GameScene) prepareCombatSignatureLayers(effect *combatSignatureEffect) {
	frame := g.combatSignatureImage(effect)
	// Even with bars hidden this region contains sprite feet/status pixels.
	// Always partition and draw both layers; only HP artwork is conditional.
	// Keep both layers at the full source dimensions and use the same transform.
	// Cropped-source quads can choose a different nearest texel at noninteger
	// scaling ties. Partition at integral source pixels before scaling instead.
	effect.BodyFrame.Clear()
	effect.BodyFrame.DrawImage(frame, &ebiten.DrawImageOptions{})
	combatHealthFrameRegion(effect.BodyFrame).Clear()
	effect.HealthFrame.Clear()
	combatHealthFrameRegion(effect.HealthFrame).DrawImage(frame, &ebiten.DrawImageOptions{})
}

func combatHealthFrameRegion(frame *ebiten.Image) *ebiten.Image {
	region, ok := frame.SubImage(combatHealthFrameBounds()).(*ebiten.Image)
	if !ok {
		panic("combat health frame region is not an Ebitengine image")
	}

	return region
}
