package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const dungeonFieldSaveSides = 30

func (g *GameState) processDungeonCurrentCellField() {
	cell := g.currentDungeonCell()
	switch cell.Raw() {
	case 0x80, 0x88:
		g.processDungeonSleepField()
	case 0x81, 0x89:
		g.processDungeonPoisonField()
	case 0x82, 0x8a:
		g.SystemCallbacks.Message.AddRowStr("Fire!!")
		g.damageDungeonParty()
	}
}

func (g *GameState) processDungeonSleepField() {
	g.SystemCallbacks.Message.AddRowStr("Sleep spell!")
	g.forEachDungeonPartyFieldRoll(func(character *party_state.PlayerCharacter, roll int) {
		if roll < int(character.Dexterity) || character.IsDead() {
			return
		}
		character.SetStatus(party_state.Sleep)
		g.SystemCallbacks.Audio.PlaySoundEffect(SoundWhiteNoiseDamage)
	})
	g.transformCurrentDungeonCell(func(before references.DungeonCell) references.DungeonCell {
		if before.Raw() != 0x80 && before.Raw() != 0x88 {
			panic("dungeon sleep field changed during resolution")
		}

		return references.DungeonCell(before.Raw() & 0x08)
	})
	g.SystemCallbacks.Screen.UpdateStatsDisplay()
	g.SystemCallbacks.Screen.UpdateDisplay()
}

func (g *GameState) processDungeonPoisonField() {
	g.SystemCallbacks.Message.AddRowStr("Poison!")
	changes := transientStatusChanges{}
	g.forEachDungeonPartyFieldRoll(func(character *party_state.PlayerCharacter, roll int) {
		if roll < int(character.Dexterity) || character.IsDead() {
			return
		}
		statusBefore := character.GetStatus()
		character.SetStatus(party_state.Poisoned)
		changes.observePoison(statusBefore, character)
		g.SystemCallbacks.Audio.PlaySoundEffect(SoundWhiteNoiseDamage)
	})
	g.publishTransientStatusChanges(changes)
	g.SystemCallbacks.Screen.UpdateStatsDisplay()
}

func (g *GameState) forEachDungeonPartyFieldRoll(
	apply func(character *party_state.PlayerCharacter, roll int),
) {
	for index := range party_state.MaxActiveParty {
		character := g.PartyState.GetCharacter(index)
		if character == nil || character.PartyMap != party_state.PartyMapActive {
			continue
		}
		// Original sleep-field resolution and poison-field resolution consume the save roll before
		// checking whether the active party member is dead.
		// See ref:DNG-0006.
		apply(character, g.RandomIntInRange(1, dungeonFieldSaveSides))
	}
}

func (g *GameState) processDungeonElectricField() {
	// REDUX DEVIATION: Original electrical-field bounce temporarily redraws from
	// the field cell and XOR-flashes the viewport before restoring the origin.
	// Redux keeps authoritative position at the origin while preserving the
	// bounce, feedback, damage, and subsequent origin-cell completion.
	// See ref:DNG-0006.
	g.SystemCallbacks.Message.AddRowStr("Ouch!")
	g.SystemCallbacks.Message.AddRowStr("Electric field!")
	g.SystemCallbacks.Audio.PlaySoundEffect(SoundStepOnTrap)
	g.damageDungeonParty()
	g.SystemCallbacks.Screen.UpdateDisplay()
}

func (g *GameState) damageDungeonParty() {
	for _, index := range g.getAllLivingPartyIndices() {
		damage := g.RollDice(8)
		g.PartyState.GetCharacter(index).SubtractHp(uint16(damage)) //nolint:gosec // Source damage is 1d8.
		g.SystemCallbacks.Audio.PlaySoundEffect(SoundWhiteNoiseDamage)
	}
	g.SystemCallbacks.Screen.UpdateStatsDisplay()
}

func (g *GameState) processDungeonPoisonDamage(cache playerUpdateStatusCache) {
	poisoned := false
	for index := range party_state.MaxActiveParty {
		poisoned = poisoned || cache.Active[index] && cache.Statuses[index] == party_state.Poisoned
	}
	g.applyCachedPlayerUpdatePoison(cache, true)
	if poisoned {
		g.SystemCallbacks.Screen.UpdateStatsDisplay()
	}
}
