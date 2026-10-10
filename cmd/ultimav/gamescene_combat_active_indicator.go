package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

type combatActivePCIndicatorStyle int

const (
	combatActivePCIndicatorSolidGreen combatActivePCIndicatorStyle = iota
	combatActivePCIndicatorMarchingContrast
	combatActivePCIndicatorInwardArrows
	combatActivePCIndicatorSelectionTile
)

const (
	activeCombatPCIndicatorStyle      = combatActivePCIndicatorSelectionTile
	combatActivePCMarchingIntervalMs  = 400
	combatActivePCMarchingPhaseCount  = 2
	combatActivePCArrowIntervalMs     = 350
	combatActivePCArrowPhaseCount     = 2
	combatActivePCArrowLeftBaseX      = 2
	combatActivePCArrowLeftTipX       = 6
	combatActivePCArrowRightBaseX     = 16
	combatActivePCArrowRightTipX      = 12
	combatActivePCArrowTopY           = 4
	combatActivePCArrowMiddleY        = 8
	combatActivePCArrowBottomY        = 11
	combatActivePCArrowShadowStroke   = 2.0
	combatActivePCArrowFaceStroke     = 1.25
	combatActivePCSelectionLeft       = 1
	combatActivePCSelectionTop        = 1
	combatActivePCSelectionSize       = 16
	combatActivePCSelectionShadow     = 2.0
	combatActivePCSelectionFace       = 1.0
	combatActivePCSelectionIntervalMs = 450
	combatActivePCSelectionPhaseCount = 2
	combatTargetSelectionElapsedMs    = combatActivePCSelectionIntervalMs
)

type combatActivePCArrowGeometry struct {
	LeftBaseX, LeftTipX    float64
	RightBaseX, RightTipX  float64
	TopY, MiddleY, BottomY float64
}

type combatActivePCSelectionGeometry struct {
	Left, Top, Width, Height float64
}

type combatSelectionActors struct {
	ActivePartyIndex   int
	ActiveMonsterIndex int
	TargetPartyIndex   int
}

func (g *GameScene) applyCombatActivePCIndicator(sprite *ebiten.Image, elapsedMs int64) *ebiten.Image {
	processor := g.spriteSheet.GetTileEffectProcessor()
	switch activeCombatPCIndicatorStyle {
	case combatActivePCIndicatorSolidGreen:
		return processor.ApplyDynamicSimpleWithIntensity(
			sprite,
			sprites.CombatActivePCOutline,
			activePCOutlinePulse(elapsedMs),
		)
	case combatActivePCIndicatorMarchingContrast:
		return processor.ApplyDynamicMarchingOutline(sprite, combatActivePCMarchingPhase(elapsedMs))
	case combatActivePCIndicatorInwardArrows:
		return processor.ApplyDynamicSimple(sprite, sprites.CombatInactivePCOutline)
	case combatActivePCIndicatorSelectionTile:
		return processor.ApplyDynamicSimple(sprite, sprites.CombatInactivePCOutline)
	default:
		panic("unsupported active combat PC indicator style")
	}
}

func drawCombatSelectionFrame(
	screen *ebiten.Image,
	drawOptions *ebiten.DrawImageOptions,
	elapsedMs int64,
	faceColor ucolor.EGAColor,
) {
	geometry := combatActivePCSelectionSourceGeometry(combatActivePCSelectionPhase(elapsedMs))
	left, top := drawOptions.GeoM.Apply(geometry.Left, geometry.Top)
	right, bottom := drawOptions.GeoM.Apply(
		geometry.Left+geometry.Width,
		geometry.Top+geometry.Height,
	)
	sourcePixelScale := combatActivePCSourcePixelScale(drawOptions)
	drawSelectionFrame := func(strokeWidth float32, clr ucolor.EGAColor) {
		vector.StrokeRect(
			screen,
			float32(left), float32(top),
			float32(right-left), float32(bottom-top),
			strokeWidth*sourcePixelScale,
			clr,
			false,
		)
	}
	drawSelectionFrame(combatActivePCSelectionShadow, ucolor.EGABlack)
	drawSelectionFrame(combatActivePCSelectionFace, faceColor)
}

func (g *GameScene) drawCombatSelectionUnderlays(screen *ebiten.Image) {
	combatScene := g.gameState.CombatScene
	if combatScene == nil {
		return
	}
	actors := resolveCombatSelectionActors(
		combatScene.ActiveMonsterIndex,
		combatScene.GetCurrentTurnMemberIndex(),
		combatScene.ActiveTargetPCIndex,
	)

	if activeCombatPCIndicatorStyle == combatActivePCIndicatorSelectionTile {
		g.drawActiveCombatantSelectionFrame(screen, actors)
	}
	if combatScene.ActiveTargetIdentitySet && combatScene.ActiveTargetIdentity.IsPartyMember {
		g.drawMonsterTargetPCSelectionFrame(screen, combatScene.ActiveTargetIdentity)
	} else {
		g.drawMonsterTargetSelectionFrame(screen, actors.TargetPartyIndex)
	}
	if combatScene.ActiveTargetIdentitySet && !combatScene.ActiveTargetIdentity.IsPartyMember {
		g.drawMonsterTargetMonsterSelectionFrame(screen, combatScene.ActiveTargetIdentity.EntityIndex)
	}
}

func (g *GameScene) drawActiveCombatantSelectionFrame(
	screen *ebiten.Image,
	actors combatSelectionActors,
) {
	combatScene := g.gameState.CombatScene
	if actors.ActiveMonsterIndex != game_state.NoActiveMonster {
		monster := combatScene.MonsterManager.GetMonster(actors.ActiveMonsterIndex)
		if monster == nil || !monster.IsActive() || monster.IsDead() || monster.IsInvisible() {
			return
		}
		position, visible := g.combatSelectionPresentation(
			game_state.CombatTurnIdentity{EntityIndex: actors.ActiveMonsterIndex}, monster.Position())
		if !visible || !g.gameState.CombatTerrainAllowsObjectDisplay(position) {
			return
		}
		drawOptions := g.combatMonsterDrawOptions(position, actors.ActiveMonsterIndex)
		drawCombatSelectionFrame(
			screen,
			drawOptions,
			g.getElapsedMs(),
			combatActiveSelectionFaceColor(actors.ActiveMonsterIndex),
		)

		return
	}

	identity := combatScene.GetCurrentTurnIdentity()
	activeIndex := identity.EntityIndex
	if actors.ActivePartyIndex < 0 || !identity.IsPartyMember ||
		activeIndex < 0 || activeIndex >= g.gameState.PartyState.GetCharacterCount() ||
		!combatScene.IsCombatPCActive(identity) {
		return
	}
	character := g.gameState.PartyState.GetCharacter(activeIndex)
	position, visible := g.combatSelectionPresentation(identity, combatScene.GetCombatantPosition(identity))
	if visible && !character.IsDead() && character.IsInParty() && g.gameState.CombatTerrainAllowsObjectDisplay(position) {
		drawOptions := g.combatPartyIdentityDrawOptions(position, identity)
		faceColor := combatActiveSelectionFaceColor(game_state.NoActiveMonster)
		if combatScene.IsCombatPCPossessed(identity) {
			faceColor = ucolor.CombatActiveMonsterOutline
		}
		if combatScene.IsPartyTurnAutoPass(activeIndex) {
			faceColor = ucolor.EGALightGray
		}
		drawCombatSelectionFrame(
			screen,
			drawOptions,
			g.getElapsedMs(),
			faceColor,
		)
	}
}

func (g *GameScene) drawMonsterTargetSelectionFrame(screen *ebiten.Image, targetIndex int) {
	combatScene := g.gameState.CombatScene
	if targetIndex < 0 || targetIndex >= g.gameState.PartyState.GetCharacterCount() ||
		!combatScene.IsPartyMemberActive(targetIndex) {
		return
	}
	identity := game_state.CombatTurnIdentity{IsPartyMember: true, EntityIndex: targetIndex}
	character := g.gameState.PartyState.GetCharacter(targetIndex)
	position, visible := g.combatSelectionPresentation(identity, combatScene.GetPartyMemberPosition(targetIndex))
	if !visible || character.IsDead() || !character.IsInParty() || !g.gameState.CombatTerrainAllowsObjectDisplay(position) {
		return
	}

	drawOptions := g.combatPartyDrawOptions(position, targetIndex)
	drawCombatSelectionFrame(screen, drawOptions, combatTargetSelectionElapsedMs, ucolor.CombatThreatenedPCOutline)
}

func (g *GameScene) drawMonsterTargetPCSelectionFrame(
	screen *ebiten.Image,
	identity game_state.CombatTurnIdentity,
) {
	combatScene := g.gameState.CombatScene
	if !combatScene.IsCombatPCActive(identity) {
		return
	}
	character := g.gameState.PartyState.GetCharacter(identity.EntityIndex)
	position, visible := g.combatSelectionPresentation(identity, combatScene.GetCombatantPosition(identity))
	if !visible || character.IsDead() || !character.IsInParty() || !g.gameState.CombatTerrainAllowsObjectDisplay(position) {
		return
	}
	drawOptions := g.combatPartyIdentityDrawOptions(position, identity)
	drawCombatSelectionFrame(screen, drawOptions, combatTargetSelectionElapsedMs, ucolor.CombatThreatenedPCOutline)
}

func (g *GameScene) drawMonsterTargetMonsterSelectionFrame(screen *ebiten.Image, targetIndex int) {
	monster := g.gameState.CombatScene.MonsterManager.GetMonster(targetIndex)
	if monster == nil || !monster.IsActive() || monster.IsDead() || monster.IsInvisible() ||
		monster.IsCorpserHidden() {
		return
	}
	position, visible := g.combatSelectionPresentation(game_state.CombatTurnIdentity{EntityIndex: targetIndex}, monster.Position())
	if !visible || !g.gameState.CombatTerrainAllowsObjectDisplay(position) {
		return
	}
	drawOptions := g.combatMonsterDrawOptions(position, targetIndex)
	drawCombatSelectionFrame(screen, drawOptions, combatTargetSelectionElapsedMs, ucolor.CombatThreatenedPCOutline)
}

func resolveCombatSelectionActors(
	activeMonsterIndex, currentPartyIndex, targetPartyIndex int,
) combatSelectionActors {
	return combatSelectionActors{
		ActivePartyIndex:   combatPartyTurnIndex(activeMonsterIndex, currentPartyIndex, false),
		ActiveMonsterIndex: activeMonsterIndex,
		TargetPartyIndex:   targetPartyIndex,
	}
}

func combatActiveSelectionFaceColor(activeMonsterIndex int) ucolor.EGAColor {
	if activeMonsterIndex != game_state.NoActiveMonster {
		return ucolor.CombatActiveMonsterOutline
	}

	return ucolor.CombatActivePCOutline
}

func drawCombatActivePCIndicatorOverlay(
	screen *ebiten.Image,
	drawOptions *ebiten.DrawImageOptions,
	elapsedMs int64,
) {
	if activeCombatPCIndicatorStyle != combatActivePCIndicatorInwardArrows {
		return
	}

	geometry := combatActivePCArrowSourceGeometry(combatActivePCArrowPhase(elapsedMs))
	leftTopX, leftTopY := drawOptions.GeoM.Apply(geometry.LeftBaseX, geometry.TopY)
	leftTipX, leftTipY := drawOptions.GeoM.Apply(geometry.LeftTipX, geometry.MiddleY)
	leftBottomX, leftBottomY := drawOptions.GeoM.Apply(geometry.LeftBaseX, geometry.BottomY)
	rightTopX, rightTopY := drawOptions.GeoM.Apply(geometry.RightBaseX, geometry.TopY)
	rightTipX, rightTipY := drawOptions.GeoM.Apply(geometry.RightTipX, geometry.MiddleY)
	rightBottomX, rightBottomY := drawOptions.GeoM.Apply(geometry.RightBaseX, geometry.BottomY)
	sourcePixelScale := combatActivePCSourcePixelScale(drawOptions)

	drawCombatActivePCArrow(
		screen,
		leftTopX, leftTopY,
		leftTipX, leftTipY,
		leftBottomX, leftBottomY,
		sourcePixelScale,
	)
	drawCombatActivePCArrow(
		screen,
		rightTopX, rightTopY,
		rightTipX, rightTipY,
		rightBottomX, rightBottomY,
		sourcePixelScale,
	)
}

func combatActivePCSelectionSourceGeometry(phase int) combatActivePCSelectionGeometry {
	expansion := float64(phase % combatActivePCSelectionPhaseCount)

	return combatActivePCSelectionGeometry{
		Left:   combatActivePCSelectionLeft - expansion,
		Top:    combatActivePCSelectionTop - expansion,
		Width:  combatActivePCSelectionSize + 2*expansion,
		Height: combatActivePCSelectionSize + 2*expansion,
	}
}

func combatActivePCSourcePixelScale(drawOptions *ebiten.DrawImageOptions) float32 {
	unitLeft, unitTop := drawOptions.GeoM.Apply(0, 0)
	unitRight, unitBottom := drawOptions.GeoM.Apply(1, 1)

	return float32(math.Min(unitRight-unitLeft, unitBottom-unitTop))
}

func drawCombatActivePCArrow(
	screen *ebiten.Image,
	topX, topY, tipX, tipY, bottomX, bottomY float64,
	sourcePixelScale float32,
) {
	drawSegment := func(fromX, fromY, toX, toY float64, width float32, clr ucolor.EGAColor) {
		vector.StrokeLine(
			screen,
			float32(fromX), float32(fromY),
			float32(toX), float32(toY),
			width*sourcePixelScale,
			clr,
			false,
		)
	}
	drawSegment(topX, topY, tipX, tipY, combatActivePCArrowShadowStroke, ucolor.EGABlack)
	drawSegment(bottomX, bottomY, tipX, tipY, combatActivePCArrowShadowStroke, ucolor.EGABlack)
	drawSegment(topX, topY, tipX, tipY, combatActivePCArrowFaceStroke, ucolor.EGAWhite)
	drawSegment(bottomX, bottomY, tipX, tipY, combatActivePCArrowFaceStroke, ucolor.EGAWhite)
}

func combatActivePCArrowSourceGeometry(phase int) combatActivePCArrowGeometry {
	inwardOffset := float64(phase % combatActivePCArrowPhaseCount)

	return combatActivePCArrowGeometry{
		LeftBaseX:  combatActivePCArrowLeftBaseX + inwardOffset,
		LeftTipX:   combatActivePCArrowLeftTipX + inwardOffset,
		RightBaseX: combatActivePCArrowRightBaseX - inwardOffset,
		RightTipX:  combatActivePCArrowRightTipX - inwardOffset,
		TopY:       combatActivePCArrowTopY,
		MiddleY:    combatActivePCArrowMiddleY,
		BottomY:    combatActivePCArrowBottomY,
	}
}

func combatActivePCMarchingPhase(elapsedMs int64) int {
	return int(elapsedMs/combatActivePCMarchingIntervalMs) % combatActivePCMarchingPhaseCount
}

func combatActivePCArrowPhase(elapsedMs int64) int {
	return int(elapsedMs/combatActivePCArrowIntervalMs) % combatActivePCArrowPhaseCount
}

func combatActivePCSelectionPhase(elapsedMs int64) int {
	return int(elapsedMs/combatActivePCSelectionIntervalMs) % combatActivePCSelectionPhaseCount
}
