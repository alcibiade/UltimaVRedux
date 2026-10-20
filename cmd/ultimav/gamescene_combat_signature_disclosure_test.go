package main

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func nextSignatureMonsterStep(t *testing.T, scene *GameScene, monsterIndex int) game_state.CombatPresentationStep {
	t.Helper()
	state := scene.gameState
	state.BeginCombatTurns(game_state.CombatTurnsPresented)
	for range 128 {
		if step, ok := state.ReviewScheduledCombatStep(); ok {
			require.Equal(t, game_state.CombatPresentationMonsterTurn, step.Kind)
			require.Equal(t, monsterIndex, step.Actor.EntityIndex)

			return step
		}
		require.False(t, state.CombatScene.HasScheduledCombatStep())
		state.FinishCombatAction()
	}
	t.Fatal("monster presentation step was not reached")

	return game_state.CombatPresentationStep{}
}

func TestCombatSignaturePossessionCapturesOriginDisclosureBeforeRetirement(t *testing.T) {
	for _, test := range []struct {
		name   string
		kind   references.MonsterType
		hidden bool
	}{
		{"invisible Shadowlord", references.MonsterShadowlord, true},
		{"visible retired daemon", references.MonsterDevil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			scene, screen := newCombatConcealmentTestScene(t)
			state := scene.gameState
			state.CombatScene.MonsterManager.Clear()
			state.CombatScene.TurnQueue.RemoveAllMonsters()
			from := references.Position{X: 8, Y: 1}
			to := references.Position{X: 1, Y: 8}
			state.CombatScene.SetPartyMemberPosition(0, to)
			setCombatConcealmentTerrain(state, to, indexes.BrickFloor)
			state.PartyState.GetCharacter(0).SetStatus(party_state.Good)
			state.PartyState.GetCharacter(0).Intelligence = 1
			state.CombatScene.PartySleeping[0] = false
			index, monster := signatureMonster(t, scene, test.kind, from)
			monster.SetInvisible(test.hidden)
			state.SystemCallbacks.Combat = game_state.NewCombatCallbacks(scene.recordCombatEvent, scene.recordCombatConclusion)
			step := nextSignatureMonsterStep(t, scene, index)
			seed := possessionPresentationSeed(t, monster.EnemyRef().Intelligence)
			state.SetRandomSeed(seed)
			_, err := state.ResolveScheduledCombatStep(step.Token)
			require.NoError(t, err)
			require.True(t, state.CombatScene.PartyPossessed[0], "the real hostile action must possess the victim")
			require.Len(t, scene.combatSignatureEffects, 1)
			effect := scene.combatSignatureEffects[0]
			assert.Equal(t, !test.hidden, effect.OriginDisclosed)
			if !test.hidden {
				require.False(t, monster.IsActive(), "Daemon retirement follows possession publication")
			}
			// Recycle the caster's manager position and change visibility. Neither
			// a revealed replacement nor retirement may rewrite origin disclosure.
			monster.SetInvisible(!test.hidden)
			state.CombatScene.MonsterManager.RemoveMonster(index)
			replacement := game_state.NewCombatMonster(references.MonsterRat,
				&(*state.GameReferences.EnemyReferences)[references.MonsterRat], from, state.AsRandomProvider())
			replacement.SetInvisible(!test.hidden)
			state.CombatScene.MonsterManager.AddMonster(replacement)
			state.ElapsedMs = effect.StartedMs + 100
			scene.updateCombatSignatureEffects()
			scene.drawCombatSignatureEffects(screen)
			originPixels := combatSignatureRegion(scene, screen, from)
			victimPixels := combatSignatureRegion(scene, screen, to)
			if test.hidden {
				assert.Zero(t, opaquePixelCount(originPixels))
				// A victim-only reference also proves no link/mote leaked between endpoints.
				reference := newPresentationRaster(screen.Bounds().Dx(), screen.Bounds().Dy())
				t.Cleanup(reference.Dispose)
				target := scene.combatPartyDrawOptions(to, 0)
				drawCombatCornerCue(reference, target, (1-float32(effect.progress(state.ElapsedMs)))*5, ucolor.CombatPossessed)
				assert.Equal(t, signaturePixels(reference), signaturePixels(screen))
			} else {
				assert.Positive(t, opaquePixelCount(originPixels))
			}
			assert.Positive(t, opaquePixelCount(victimPixels))
			scene.clearCombatSignatureEffects()
		})
	}
}

func possessionPresentationSeed(t *testing.T, intelligence int) uint64 {
	t.Helper()
	for seed := uint64(0); seed < 10000; seed++ {
		rng := beradrng.New(seed)
		if rng.Intn(32) == 0 && game_state.CombatHitCheck(intelligence, 1, false, rng).Outcome != game_state.CombatHitMiss {
			return seed
		}
	}
	t.Fatal("no possession seed")

	return 0
}

func combatSignatureRegion(scene *GameScene, screen *ebiten.Image, position references.Position) *ebiten.Image {
	options := scene.combatMonsterDrawOptions(position, game_state.NoActiveMonster)
	left, top := options.GeoM.Apply(-3, -3)
	right, bottom := options.GeoM.Apply(21, 21)

	return screen.SubImage(image.Rect(int(left), int(top), int(right), int(bottom))).(*ebiten.Image)
}

func signaturePixels(screen *ebiten.Image) []byte {
	pixels := make([]byte, screen.Bounds().Dx()*screen.Bounds().Dy()*4)
	screen.ReadPixels(pixels)

	return pixels
}

func TestCombatReportDisclosureFreezesHiddenTeleportAndTypedVanish(t *testing.T) {
	for _, hidden := range []bool{true, false} {
		for _, kind := range []game_state.CombatEventKind{game_state.CombatEventTeleport, game_state.CombatEventVanished, game_state.CombatEventRetired} {
			scene, _ := newCombatUseTestScene(t)
			scene.spriteSheet = sharedPresentationTestSpriteSheet()
			from, to := references.Position{X: 2, Y: 2}, references.Position{X: 8, Y: 7}
			index, monster := signatureMonster(t, scene, references.MonsterShadowlord, to)
			setCombatConcealmentTerrain(scene.gameState, from, indexes.BrickFloor)
			monster.SetInvisible(hidden)
			if kind == game_state.CombatEventVanished {
				monster.SetCurrentHP(0)
			}
			if kind == game_state.CombatEventRetired {
				monster.SetActive(false)
			}
			event := game_state.CombatEvent{Kind: kind, From: from, To: to,
				Attacker: game_state.CombatFeedbackEntity{Index: index, Position: to},
				Vanished: game_state.CombatVanishedMonster{CombatantID: monster.CombatantID(), Index: index,
					MonsterType: monster.MonsterType(), Position: to}}
			scene.recordCombatEvent(event)
			slot := scene.combatHUD.historyCount - 1
			scene.combatHUD.detail = true
			scene.combatHUD.eventOutput = text.NewOutput(nil, 1, 20, 120)
			disclosure := scene.combatHUD.identities[slot]
			assert.Equal(t, !hidden, disclosure.DetailsDisclosed)
			if kind == game_state.CombatEventVanished {
				assert.Equal(t, !hidden, disclosure.VanishedDisclosed)
				assert.Equal(t, !hidden, len(scene.combatVanishEffects) > 0)
			}
			var original string
			scene.withCombatHistoryEvent(slot, func(event game_state.CombatEvent) {
				original = scene.combatEventPlainText(event) + " " + scene.combatEventDetailText(event)
			})
			// The report owns disclosure, independently of the live slot's later contents.
			scene.gameState.CombatScene.MonsterManager.RemoveMonster(index)
			replacement := game_state.NewCombatMonster(references.MonsterRat,
				&(*scene.gameState.GameReferences.EnemyReferences)[references.MonsterRat], to, scene.gameState.AsRandomProvider())
			replacement.SetInvisible(!hidden)
			scene.gameState.CombatScene.MonsterManager.AddMonster(replacement)
			scene.withCombatHistoryEvent(slot, func(event game_state.CombatEvent) {
				assert.Equal(t, original, scene.combatEventPlainText(event)+" "+scene.combatEventDetailText(event))
				scene.appendCombatEvent(event)
			})
			output := scene.combatHUD.eventOutput.GetOutputStr(false)
			if hidden {
				assert.Contains(t, output, combatUnknownEntityLabel)
				assert.NotContains(t, output, "Shadowlord")
				assert.NotContains(t, output, "(2,2)")
				assert.NotContains(t, output, "(8,7)")
			} else {
				assert.Contains(t, output, "Shadowlord")
				assert.Contains(t, output, "(")
			}
			assert.NotContains(t, output, "Rat")
			scene.clearCombatSignatureEffects()
			scene.clearCombatVanishEffects()
		}
	}
}
