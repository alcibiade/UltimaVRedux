package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// CombatMonsterManager manages the collection of monsters in combat.
// Provides CRUD operations and query capabilities for combat monsters.
type CombatMonsterManager struct {
	monsters []*CombatMonster
}

// NewCombatMonsterManager creates a new empty monster manager.
func NewCombatMonsterManager() *CombatMonsterManager {
	return &CombatMonsterManager{
		monsters: make([]*CombatMonster, 0),
	}
}

// --- CRUD Operations ---

// AddMonster adds a new monster to the combat.
// Returns the index of the added monster.
func (m *CombatMonsterManager) AddMonster(monster *CombatMonster) int {
	index := len(m.monsters)
	m.monsters = append(m.monsters, monster)

	logging.Debug(logging.CategoryGameState,
		"Added combat monster: type=%s hp=%d index=%d",
		monster.MonsterType().Name(), monster.CurrentHP(), index)

	return index
}

// RemoveMonster removes a monster by index.
// Returns the removed monster, or nil if index is invalid.
func (m *CombatMonsterManager) RemoveMonster(index int) *CombatMonster {
	if index < 0 || index >= len(m.monsters) {
		return nil
	}

	removed := m.monsters[index]
	m.monsters = append(m.monsters[:index], m.monsters[index+1:]...)

	logging.Debug(logging.CategoryGameState,
		"Removed combat monster: type=%s index=%d",
		removed.MonsterType().Name(), index)

	return removed
}

// GetMonster returns the monster at the given index.
// Returns nil if index is invalid.
func (m *CombatMonsterManager) GetMonster(index int) *CombatMonster {
	if index < 0 || index >= len(m.monsters) {
		return nil
	}

	return m.monsters[index]
}

// GetMonsterByID returns the monster with the immutable encounter-scoped identity.
func (m *CombatMonsterManager) GetMonsterByID(id CombatantID) *CombatMonster {
	if id == 0 {
		return nil
	}
	for _, monster := range m.monsters {
		if monster.CombatantID() == id {
			return monster
		}
	}

	return nil
}

// Count returns the total number of monsters (including dead ones).
func (m *CombatMonsterManager) Count() int {
	return len(m.monsters)
}

// Clear removes all monsters from the manager.
func (m *CombatMonsterManager) Clear() {
	m.monsters = m.monsters[:0]
}

// --- Query Operations ---

// GetActiveMonsters returns all active (non-dead) monsters.
func (m *CombatMonsterManager) GetActiveMonsters() []*CombatMonster {
	active := make([]*CombatMonster, 0)
	for _, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() {
			active = append(active, monster)
		}
	}

	return active
}

// CountActive returns the number of active (non-dead) monsters.
func (m *CombatMonsterManager) CountActive() int {
	count := 0
	for _, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() {
			count++
		}
	}

	return count
}

// GetActiveHostileMonsters returns active monsters loyal to the opposing side.
func (m *CombatMonsterManager) GetActiveHostileMonsters() []*CombatMonster {
	hostile := make([]*CombatMonster, 0)
	for _, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() && monster.IsHostile() {
			hostile = append(hostile, monster)
		}
	}

	return hostile
}

// CountActiveHostile returns the number of active monsters loyal to the opposing side.
func (m *CombatMonsterManager) CountActiveHostile() int {
	count := 0
	for _, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() && monster.IsHostile() {
			count++
		}
	}

	return count
}

// GetActivePartyLoyalMonsters returns active monsters that support the party.
func (m *CombatMonsterManager) GetActivePartyLoyalMonsters() []*CombatMonster {
	partyLoyal := make([]*CombatMonster, 0)
	for _, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() && monster.IsPartyLoyal() {
			partyLoyal = append(partyLoyal, monster)
		}
	}

	return partyLoyal
}

// CountActivePartyLoyal returns the number of active monsters that support the party.
func (m *CombatMonsterManager) CountActivePartyLoyal() int {
	count := 0
	for _, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() && monster.IsPartyLoyal() {
			count++
		}
	}

	return count
}

// GetDeadMonsters returns all dead monsters.
func (m *CombatMonsterManager) GetDeadMonsters() []*CombatMonster {
	dead := make([]*CombatMonster, 0)
	for _, monster := range m.monsters {
		if monster.IsDead() {
			dead = append(dead, monster)
		}
	}

	return dead
}

// ForEachActive calls fn for each active monster.
func (m *CombatMonsterManager) ForEachActive(fn func(index int, monster *CombatMonster)) {
	for i, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() {
			fn(i, monster)
		}
	}
}

// ForEach calls fn for all monsters.
func (m *CombatMonsterManager) ForEach(fn func(index int, monster *CombatMonster)) {
	for i, monster := range m.monsters {
		fn(i, monster)
	}
}

// --- Damage Operations ---

// ApplyDamageTo applies damage to a specific monster.
// Returns true if the monster died from this damage.
func (m *CombatMonsterManager) ApplyDamageTo(index int, damage int) bool {
	monster := m.GetMonster(index)
	if monster == nil {
		return false
	}

	return monster.TakeDamage(damage)
}

// ApplyDamageToAll applies damage to all active monsters.
// Returns the count of monsters killed.
func (m *CombatMonsterManager) ApplyDamageToAll(damage int) int {
	killed := 0
	for _, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() {
			if monster.TakeDamage(damage) {
				killed++
			}
		}
	}

	return killed
}

// --- Turn Order Operations ---

// GetMonstersByInitiative returns monsters sorted by turn delay (lowest first).
// Implements authentic original combat initialization turn order based on DEX.
// See ref:COMBAT-0002.
func (m *CombatMonsterManager) GetMonstersByInitiative() []*CombatMonster {
	active := m.GetActiveMonsters()

	// Sort by turn delay (lower = earlier in turn order)
	// Using simple insertion sort for small lists
	for sortedEnd := 1; sortedEnd < len(active); sortedEnd++ {
		for insertPos := sortedEnd; insertPos > 0 && active[insertPos].TurnDelay() < active[insertPos-1].TurnDelay(); insertPos-- {
			active[insertPos], active[insertPos-1] = active[insertPos-1], active[insertPos]
		}
	}

	return active
}

// ResetTurnStates resets hasActed for all monsters at the start of a new round.
func (m *CombatMonsterManager) ResetTurnStates() {
	for _, monster := range m.monsters {
		monster.SetHasActed(false)
	}
}

// HasMonstersRemaining returns true if any active monsters remain.
func (m *CombatMonsterManager) HasMonstersRemaining() bool {
	return m.CountActive() > 0
}

// --- Position Operations ---

// GetMonstersAtPosition returns every active monster at pos in stable manager order.
func (m *CombatMonsterManager) GetMonstersAtPosition(pos references.Position) []*CombatMonster {
	monsters := make([]*CombatMonster, 0)
	for _, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() && monster.Position() == pos {
			monsters = append(monsters, monster)
		}
	}

	return monsters
}

// GetMonsterIndicesAtPosition returns stable manager indices for every active monster at pos.
func (m *CombatMonsterManager) GetMonsterIndicesAtPosition(pos references.Position) []int {
	indices := make([]int, 0)
	for index, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() && monster.Position() == pos {
			indices = append(indices, index)
		}
	}

	return indices
}

// GetMonsterAtPosition returns the monster at the given position, if any.
// Returns nil if no monster is at that position.
func (m *CombatMonsterManager) GetMonsterAtPosition(pos references.Position) *CombatMonster {
	for _, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() && monster.Position() == pos {
			return monster
		}
	}

	return nil
}

// GetMonsterIndexAtPosition returns the monster and its index at the given position.
// Returns (nil, -1) if no active monster is at that position.
func (m *CombatMonsterManager) GetMonsterIndexAtPosition(pos references.Position) (*CombatMonster, int) {
	for i, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() && monster.Position() == pos {
			return monster, i
		}
	}

	return nil, -1
}

// IsPositionOccupied returns true if any active monster is at the position.
func (m *CombatMonsterManager) IsPositionOccupied(pos references.Position) bool {
	return m.GetMonsterAtPosition(pos) != nil
}

// --- Statistics ---

// GetMonsterTypeCount returns the count of a specific monster type.
func (m *CombatMonsterManager) GetMonsterTypeCount(monsterType references.MonsterType) int {
	count := 0
	for _, monster := range m.monsters {
		if monster.MonsterType() == monsterType {
			count++
		}
	}

	return count
}

// GetTotalExperience sums each killed monster’s maximum-HP-derived award. ref:PROGRESS-0002
//
//nolint:mnd // The kill-award divisor is part of the gameplay formula.
func (m *CombatMonsterManager) GetTotalExperience() int {
	total := 0
	for _, monster := range m.monsters {
		if monster.IsDead() {
			// Experience = HP / 4 + 1
			// See ref:COMBAT-0002.
			exp := monster.MaxHP()/4 + 1
			total += exp
		}
	}

	return total
}

// GetMonsterTypeSummary returns a map of monster types to counts.
func (m *CombatMonsterManager) GetMonsterTypeSummary() map[references.MonsterType]int {
	summary := make(map[references.MonsterType]int)
	for _, monster := range m.monsters {
		if monster.IsActive() && !monster.IsDead() {
			summary[monster.MonsterType()]++
		}
	}

	return summary
}
