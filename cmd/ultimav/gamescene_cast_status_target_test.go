package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestProductionCombatCastStatusTargetLifecycle(t *testing.T) {
	tests := []struct {
		name          string
		spell         references.Spell
		lastSyllable  ebiten.Key
		status        party_state.CharacterStatus
		label         string
		selectHealthy bool
		cancel        bool
	}{
		{name: "An Nox cures poisoned target", spell: references.AnNox, lastSyllable: ebiten.KeyN, status: party_state.Poisoned, label: "Poisoned"},
		{name: "An Nox healthy warning confirms without effect", spell: references.AnNox, lastSyllable: ebiten.KeyN, status: party_state.Poisoned, label: "Poisoned", selectHealthy: true},
		{name: "An Nox target Escape preserves resources", spell: references.AnNox, lastSyllable: ebiten.KeyN, status: party_state.Poisoned, label: "Poisoned", cancel: true},
		{name: "An Zu awakens sleeping target", spell: references.AnZu, lastSyllable: ebiten.KeyZ, status: party_state.Sleep, label: "Asleep"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, _ := newProductionCombatCastState(t)
			expected, _ := newProductionCombatCastState(t)
			casterIdx := actual.CombatScene.GetActiveCombatantIndex()
			affectedIdx := 1
			if casterIdx == affectedIdx {
				affectedIdx = 2
			}
			actual.PartyState.GetCharacter(0).SetStatus(party_state.Good)
			expected.PartyState.GetCharacter(0).SetStatus(party_state.Good)
			affected := actual.PartyState.GetCharacter(affectedIdx)
			require.True(t, affected.IsInParty())
			affected.SetStatus(test.status)
			expected.PartyState.GetCharacter(affectedIdx).SetStatus(test.status)
			caster := actual.PartyState.GetCharacter(casterIdx)
			mpBefore := caster.GetCurrentMp()
			healthyHPBefore := actual.PartyState.GetCharacter(0).GetCurrentHp()
			actual.PartyState.Inventory.Spells.Set(test.spell, 2)
			scene := newProductionCastTestScene(actual)

			scene.combatMapInputHandler(ebiten.KeyC)
			spellDialog := waitForProductionSpellDialog(t, scene)
			for _, key := range []ebiten.Key{ebiten.KeyA, test.lastSyllable} {
				scene.keyboard.SimulateKeyPress(key)
				spellDialog.Update()
			}
			require.Equal(t, test.spell, spellDialog.GetSelectedSpell())
			scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
			spellDialog.Update()
			require.True(t, spellDialog.HasPendingCastRequest())
			waitForProductionPartySpellTargeting(t, scene)
			require.Equal(t, actual.CombatScene.GetCurrentTurnIdentity(), actual.CombatScene.SpellTargeting.Recipient)
			healthy := actual.ReviewCombatPartySpellRecipient()
			assert.True(t, healthy.Valid)
			assert.Contains(t, healthy.Consequence, "no effect")
			assert.Equal(t, mpBefore, caster.GetCurrentMp(), "resources wait for final confirmation")
			assert.Equal(t, uint16(2), actual.PartyState.Inventory.Spells.Get(test.spell))
			assert.Equal(t, test.status, affected.GetStatus(), "opening the target prompt must not apply the spell")
			assert.Equal(t, expected.CombatScene.GetCurrentTurnIdentity(), actual.CombatScene.GetCurrentTurnIdentity())

			if !test.selectHealthy {
				for range 32 {
					scene.keyboard.SimulateKeyPress(ebiten.KeyTab)
					scene.handleCombatSpellTargetInput()
					if actual.CombatScene.SpellTargeting.Recipient.EntityIndex == affectedIdx {
						break
					}
				}
				preview := actual.ReviewCombatPartySpellRecipient()
				require.Equal(t, affectedIdx, preview.Identity.EntityIndex)
				assert.Contains(t, preview.Status, test.label)
			}
			terminalKey := ebiten.KeyEnter
			if test.cancel {
				terminalKey = ebiten.KeyEscape
			}
			scene.keyboard.SimulateKeyPress(terminalKey)
			scene.handleCombatSpellTargetInput()
			drainProductionCastWorkflow(t, scene)

			wantStatus := test.status
			if !test.cancel && !test.selectHealthy {
				wantStatus = party_state.Good
				if test.spell == references.AnZu {
					require.True(t, expected.AwakenCharacter(affectedIdx))
				} else {
					expected.PartyState.GetCharacter(affectedIdx).SetStatus(party_state.Good)
				}
			}
			if !test.cancel {
				expected.FinishCombatAction()
			}
			assertCombatMatchesSingleFinish(t, actual, expected)
			assert.Equal(t, wantStatus, affected.GetStatus())
			assert.Equal(t, expected.PartyState.GetCharacter(affectedIdx).GetCurrentHp(), affected.GetCurrentHp(),
				"only ordinary turn-boundary poison damage may change HP")
			assert.Equal(t, party_state.Good, actual.PartyState.GetCharacter(0).GetStatus())
			assert.Equal(t, healthyHPBefore, actual.PartyState.GetCharacter(0).GetCurrentHp())
			if test.cancel {
				assert.Equal(t, mpBefore, caster.GetCurrentMp())
				assert.Equal(t, uint16(2), actual.PartyState.Inventory.Spells.Get(test.spell))
			} else {
				assert.Equal(t, mpBefore-1, caster.GetCurrentMp())
				assert.Equal(t, uint16(1), actual.PartyState.Inventory.Spells.Get(test.spell))
			}
			assert.Zero(t, scene.dialogStack.Count())
			assert.False(t, actual.EventQueueManager.HasWorkflow("spell_casting_workflow"))
			assert.False(t, scene.surfaceSpellWorkflowOwnsCycle)
			for range 3 {
				actual.ElapsedMs += 1000
				actual.EventQueueManager.Update()
			}
			assertCombatMatchesSingleFinish(t, actual, expected)
			assert.Zero(t, scene.dialogStack.Count(), "cleanup remains complete on later updates")
		})
	}
}
