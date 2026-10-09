package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestCombatPartySpellSelectionUsesExactCloneAndFreePreview(t *testing.T) {
	state, source, clone := newSyntheticExactIdentityCastUseState(t)
	state.CombatScene.CurrentTurnIdentity = clone
	state.CombatScene.CurrentTurnIndex = clone.EntityIndex
	state.PartyState.Inventory.Spells.Set(references.Mani, 2)
	pc := state.PartyState.GetCharacter(clone.EntityIndex)
	mp := pc.GetCurrentMp()
	calls := state.GetRNG().Calls()
	require.True(t, state.BeginCombatPartySpellTargeting(CastRequest{Spell: references.Mani, CasterIdx: clone.EntityIndex}).Valid)
	assert.Equal(t, clone, state.ReviewCombatPartySpellRecipient().Identity)
	for range 4 {
		state.CycleCombatPartySpellRecipient()
		require.True(t, state.ReviewCombatPartySpellRecipient().Valid)
	}
	assert.Equal(t, calls, state.GetRNG().Calls())
	assert.Equal(t, mp, pc.GetCurrentMp())
	assert.Equal(t, uint16(2), state.PartyState.Inventory.Spells.Get(references.Mani))
	assert.Contains(t, state.CombatPartySpellRecipients(), source)
	assert.Equal(t, CombatSpellTargetCancelled, state.CancelCombatSpellTargeting())
	assert.Equal(t, clone, state.CombatScene.GetCurrentTurnIdentity())
	assert.Equal(t, mp, pc.GetCurrentMp())
}

func TestCombatPartySpellRecipientVisibilityAndSharedSquareIdentity(t *testing.T) {
	for _, condition := range []string{"sleep", "invisible", "possessed", "underground", "concealed", "fled", "dead"} {
		t.Run(condition, func(t *testing.T) {
			state, source, clone := newSyntheticExactIdentityCastUseState(t)
			position := state.CombatScene.GetCombatantPosition(clone)
			state.CombatScene.SetPartyMemberPosition(source.EntityIndex, position)
			state.PartyState.Inventory.Spells.Set(references.AnNox, 1)
			require.True(t, state.BeginCombatPartySpellTargeting(CastRequest{Spell: references.AnNox, CasterIdx: source.EntityIndex}).Valid)
			state.CycleCombatPartySpellRecipient()
			require.Equal(t, clone, state.ReviewCombatPartySpellRecipient().Identity)
			require.Equal(t, 2, state.ReviewCombatPartySpellRecipient().Occupants)
			switch condition {
			case "sleep":
				state.setCombatPCSleeping(clone, true)
			case "invisible":
				state.CombatScene.setCombatPCInvisible(clone, true)
			case "possessed":
				state.CombatScene.setCombatPCPossessed(clone, true)
			case "underground":
				state.CombatScene.setCombatPCUnderground(clone, true)
			case "concealed":
				state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.BrickWallArchway)
			case "fled":
				state.CombatScene.TurnQueue.markFledIdentity(clone)
			case "dead":
				state.PartyState.GetCharacter(clone.EntityIndex).MarkAsDead()
			}
			valid := condition == "sleep" || condition == "invisible" || condition == "possessed"
			assert.Equal(t, valid, state.ReviewCombatPartySpellRecipient().Valid)
			if !valid {
				assert.Equal(t, CombatSpellTargetInvalid, state.ConfirmCombatPartySpellRecipient().Outcome)
				assert.Equal(t, uint16(1), state.PartyState.Inventory.Spells.Get(references.AnNox))
			}
		})
	}
}

func TestCombatPartyAnZuPreviewMatchesOriginalSleepingSlot(t *testing.T) {
	for _, hiddenOriginal := range []bool{false, true} {
		state, source, clone := newSyntheticExactIdentityCastUseState(t)
		state.CombatScene.CurrentTurnIdentity = clone
		state.CombatScene.CurrentTurnIndex = clone.EntityIndex
		state.setCombatPCSleeping(source, true)
		state.PartyState.Inventory.Spells.Set(references.AnZu, 1)
		if hiddenOriginal {
			position := state.CombatScene.GetCombatantPosition(source)
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.BrickWallArchway)
		}
		require.True(t, state.BeginCombatPartySpellTargeting(CastRequest{Spell: references.AnZu, CasterIdx: clone.EntityIndex}).Valid)
		preview := state.ReviewCombatPartySpellRecipient()
		require.True(t, preview.Valid)
		assert.Equal(t, clone, preview.Identity)
		assert.Contains(t, preview.WakingName, "Original")
		assert.Equal(t, !hiddenOriginal, preview.ShowWakingPosition)
		if hiddenOriginal {
			assert.Zero(t, preview.WakingPosition)
		}
		result := state.ConfirmCombatPartySpellRecipient()
		assert.Equal(t, CombatSpellTargetResolved, result.Outcome)
		assert.True(t, result.Applied)
		assert.False(t, state.CombatScene.isCombatPCSleeping(source))
		assert.Equal(t, CombatSpellTargetIgnored, state.ConfirmCombatPartySpellRecipient().Outcome)
	}
}

func TestCombatPartySpellVasManiGateAndLegalNoEffectCommit(t *testing.T) {
	state, source, _ := newSyntheticExactIdentityCastUseState(t)
	state.PartyState.Inventory.Spells.Set(references.VasMani, 2)
	request := CastRequest{Spell: references.VasMani, CasterIdx: source.EntityIndex}
	assert.Equal(t, "Not until victory!", state.BeginCombatPartySpellTargeting(request).ErrorMsg)
	state.CombatScene.Victory = true
	require.True(t, state.BeginCombatPartySpellTargeting(request).Valid)
	pc := state.PartyState.GetCharacter(source.EntityIndex)
	pc.SetStatus(party_state.Good)
	mp := pc.GetCurrentMp()
	result := state.ConfirmCombatPartySpellRecipient()
	assert.Equal(t, CombatSpellTargetResolved, result.Outcome)
	assert.Zero(t, result.Healed)
	assert.Equal(t, mp-5, pc.GetCurrentMp())
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Spells.Get(references.VasMani))
}
