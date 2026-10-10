package game_state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestReadyReplacementUsesFinalLoadoutAndReturnsBothHands(t *testing.T) {
	state, character, request := newReadyExplorationTestState(t)
	character.Strength = 20
	character.SetEquippedItem(references.SlotArmour, references.LeatherArmour)
	character.SetEquippedItem(references.SlotWeapon, references.SilverSword)
	character.SetEquippedItem(references.SlotShield, references.SmallShield)
	state.PartyState.Inventory.Equipment.Set(references.Halberd, 1)
	transaction := readyTransaction(request, character, references.Halberd)
	beforeRNG, beforeTime := state.GetRNG().Calls(), state.DateTime
	preview, err := state.PreviewReadyEquipment(transaction)
	require.NoError(t, err)
	require.Equal(t, ReadyEquipmentTotals{Attack: 12, Defense: 4, Strength: 12}, preview.Current)
	require.Equal(t, ReadyEquipmentTotals{Attack: 30, Defense: 2, Strength: 20}, preview.Resulting)
	require.True(t, preview.Removes(references.SlotWeapon))
	require.True(t, preview.Removes(references.SlotShield))
	require.Equal(t, references.SilverSword, character.GetEquippedItem(references.SlotWeapon))
	_, err = state.ApplyReadyEquipment(transaction)
	require.ErrorIs(t, err, ErrReadyEquipment)
	transaction.Confirmed = &preview
	result, err := state.ApplyReadyEquipment(transaction)
	require.NoError(t, err)
	require.True(t, result.Changed)
	require.Equal(t, references.Halberd, character.GetEquippedItem(references.SlotWeapon))
	require.Equal(t, references.NoEquipment, character.GetEquippedItem(references.SlotShield))
	require.Equal(t, uint16(1), state.PartyState.Inventory.Equipment.Get(references.SilverSword))
	require.Equal(t, uint16(1), state.PartyState.Inventory.Equipment.Get(references.SmallShield))
	require.Zero(t, state.PartyState.Inventory.Equipment.Get(references.Halberd))
	require.Equal(t, beforeRNG, state.GetRNG().Calls())
	require.Equal(t, beforeTime, state.DateTime)
}

func TestReadyReplacementNamedHandAndInvalidStrengthPreview(t *testing.T) {
	state, character, request := newReadyExplorationTestState(t)
	character.Strength = 20
	character.SetEquippedItem(references.SlotArmour, references.LeatherArmour)
	character.SetEquippedItem(references.SlotWeapon, references.SilverSword)
	character.SetEquippedItem(references.SlotShield, references.SmallShield)
	state.PartyState.Inventory.Equipment.Set(references.LongSword, 1)
	transaction := readyTransaction(request, character, references.LongSword)
	transaction.TargetHand = references.SlotWeapon
	preview, err := state.PreviewReadyEquipment(transaction)
	require.NoError(t, err)
	require.Equal(t, 3, preview.Resulting.Attack-preview.Current.Attack)
	require.Equal(t, 1, preview.Resulting.Strength-preview.Current.Strength)
	require.Equal(t, references.SmallShield, preview.After[references.SlotShield])
	transaction.TargetHand = references.SlotShield
	preview, err = state.PreviewReadyEquipment(transaction)
	require.NoError(t, err)
	require.Equal(t, references.SilverSword, preview.After[references.SlotWeapon])
	require.Equal(t, 19, preview.Resulting.Strength)
	character.Strength = 18
	preview, err = state.PreviewReadyEquipment(transaction)
	require.ErrorContains(t, err, "Needs 1 more Strength.")
	require.ErrorIs(t, err, ErrReadyEquipmentStrength)
	require.ErrorIs(t, err, ErrReadyEquipment)
	require.Equal(t, 19, preview.Resulting.Strength, "invalid preview must still explain the final allowance")
}

func TestReadyReplacementRejectsStaleReviewAndOverflowAtomically(t *testing.T) {
	for _, cause := range []string{"equipment", "strength", "stock", "overflow", "duplicate returns"} {
		t.Run(cause, func(t *testing.T) {
			state, character, request := newReadyExplorationTestState(t)
			character.SetEquippedItem(references.SlotWeapon, references.Dagger)
			character.SetEquippedItem(references.SlotShield, references.Dagger)
			state.PartyState.Inventory.Equipment.Set(references.Halberd, 1)
			transaction := readyTransaction(request, character, references.Halberd)
			preview, err := state.PreviewReadyEquipment(transaction)
			require.NoError(t, err)
			transaction.Confirmed = &preview
			switch cause {
			case "equipment":
				character.SetEquippedItem(references.SlotShield, references.SmallShield)
			case "strength":
				character.Strength--
			case "stock":
				state.PartyState.Inventory.Equipment.Set(references.Halberd, 0)
			case "overflow":
				state.PartyState.Inventory.Equipment.Set(references.Dagger, 99)
			case "duplicate returns":
				state.PartyState.Inventory.Equipment.Set(references.Dagger, 98)
			}
			before := equipmentLoadout(character)
			inventory := state.PartyState.Inventory.LockerSnapshot()
			calls := state.GetRNG().Calls()
			_, err = state.ApplyReadyEquipment(transaction)
			require.ErrorIs(t, err, ErrReadyEquipment)
			require.Equal(t, before, equipmentLoadout(character))
			require.Equal(t, inventory, state.PartyState.Inventory.LockerSnapshot())
			require.Equal(t, calls, state.GetRNG().Calls())
		})
	}
}

func TestReadyReplaceTwoHandedWithExplicitSecondHand(t *testing.T) {
	state, character, request := newReadyExplorationTestState(t)
	character.SetEquippedItem(references.SlotWeapon, references.Halberd)
	state.PartyState.Inventory.Equipment.Set(references.SmallShield, 1)
	transaction := readyTransaction(request, character, references.SmallShield)
	transaction.TargetHand = references.SlotShield
	preview, err := state.PreviewReadyEquipment(transaction)
	require.NoError(t, err)
	require.Equal(t, references.NoEquipment, preview.After[references.SlotWeapon])
	transaction.Confirmed = &preview
	_, err = state.ApplyReadyEquipment(transaction)
	require.NoError(t, err)
	require.Equal(t, references.SmallShield, character.GetEquippedItem(references.SlotShield))
	require.False(t, character.HasTwoHandedWeaponEquipped())
}

func TestReadyReplacementPreservesRingCommitEffects(t *testing.T) {
	state, _ := newReadyCombatTestState(t)
	clearReadyTestEquipment(state)
	request := readyCombatRequest(t, state)
	character := state.PartyState.GetCharacter(request.CharacterIndex)
	character.SetEquippedItem(references.SlotRing, references.RingInvisibility)
	require.True(t, state.CastCombatSanctLor(request.CharacterIndex))
	state.PartyState.Inventory.Equipment.Set(references.RingRegeneration, 1)
	state.SetRandomSeed(readyRingSeed(t, true))
	transaction := readyTransaction(request, character, references.RingRegeneration)
	beforeRNG := state.GetRNG().Calls()
	preview, err := state.PreviewReadyEquipment(transaction)
	require.NoError(t, err)
	require.Equal(t, beforeRNG, state.GetRNG().Calls())
	transaction.Confirmed = &preview
	result, err := state.ApplyReadyEquipment(transaction)
	require.NoError(t, err)
	require.True(t, result.Vanished)
	require.True(t, result.Dismiss)
	require.Equal(t, beforeRNG+1, state.GetRNG().Calls())
	require.False(t, state.CombatScene.IsCombatPCInvisible(request.CombatIdentity))
	require.Equal(t, uint16(1), state.PartyState.Inventory.Equipment.Get(references.RingInvisibility))
	require.Zero(t, state.PartyState.Inventory.Equipment.Get(references.RingRegeneration))
	require.Equal(t, references.NoEquipment, character.GetEquippedItem(references.SlotRing))
}

func TestReadyReplacementRevalidatesAmmoAndCombatOwner(t *testing.T) {
	for _, cause := range []string{"ammo", "owner", "armour"} {
		t.Run(cause, func(t *testing.T) {
			state, _ := newReadyCombatTestState(t)
			clearReadyTestEquipment(state)
			request := readyCombatRequest(t, state)
			character := state.PartyState.GetCharacter(request.CharacterIndex)
			character.Strength = 30
			character.SetEquippedItem(references.SlotWeapon, references.Dagger)
			state.PartyState.Inventory.Equipment.Set(references.Bow, 1)
			state.PartyState.Inventory.Equipment.Set(references.Arrows, 1)
			transaction := readyTransaction(request, character, references.Bow)
			if cause == "armour" {
				state.CombatScene.Victory = true
				character.SetEquippedItem(references.SlotArmour, references.ClothArmour)
				state.PartyState.Inventory.Equipment.Set(references.LeatherArmour, 1)
				transaction.Equipment = references.LeatherArmour
			}
			preview, err := state.PreviewReadyEquipment(transaction)
			require.NoError(t, err)
			transaction.Confirmed = &preview
			switch cause {
			case "ammo":
				state.PartyState.Inventory.Equipment.Set(references.Arrows, 0)
			case "owner":
				transaction.Request.CombatIdentity.Slot++
			case "armour":
				state.CombatScene.Victory = false
			}
			before := equipmentLoadout(character)
			inventory := state.PartyState.Inventory.LockerSnapshot()
			calls := state.GetRNG().Calls()
			_, err = state.ApplyReadyEquipment(transaction)
			require.ErrorIs(t, err, ErrReadyEquipment)
			require.Equal(t, before, equipmentLoadout(character))
			require.Equal(t, inventory, state.PartyState.Inventory.LockerSnapshot())
			require.Equal(t, calls, state.GetRNG().Calls())
		})
	}
}
