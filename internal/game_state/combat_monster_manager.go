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

	logging.Debugf(logging.CategoryGameState,
		"Added combat monster: type=%s hp=%d index=%d",
		monster.MonsterType().Name(), monster.CurrentHP(), index)

	return index
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

// --- Turn Order Operations ---

// --- Position Operations ---

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
