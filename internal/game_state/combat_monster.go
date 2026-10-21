package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// Combat monster constants from original combat initialization and original game reference definitions.
// See ref:ENEMY-0003.
const (
	dexModifierRange = 8  // Original getrandom(7) is inclusive: 0-7.
	dexModifierMin   = -4 // -4 to +3 range.
	maxDexterity     = 30 // Candidates outside unsigned-byte 0-30 reset to base DEX.
	turnDelayBaseDex = 36 // turn = 36 - dex (higher dex = shorter delay)
)

// CombatMonsterLoyalty identifies which side an encounter-local monster supports.
type CombatMonsterLoyalty uint8

const (
	// CombatMonsterLoyaltyMonster marks normal hostile monsters.
	CombatMonsterLoyaltyMonster CombatMonsterLoyalty = iota
	// CombatMonsterLoyaltyParty marks summoned or charmed monsters that support the party.
	CombatMonsterLoyaltyParty
)

// CombatMonster represents a monster in active combat.
// All fields are private - use getters/setters for access.
// Analogous to original combat queue[] struct in original game reference definitions.
// See ref:ENEMY-0003.
type CombatMonster struct {
	// Identity (immutable after creation)
	combatantID CombatantID
	monsterType references.MonsterType
	enemyRef    *references.EnemyReference

	// Combat state (mutable via setters)
	currentHP   int
	currentDex  int
	dexModifier int
	position    references.Position
	loyalty     CombatMonsterLoyalty
	// Encounter-only provenance, set only by successful charm/summon producers.
	charmed  bool
	summoned bool
	// divisionConsumed limits Redux-mode SPLITS without changing source enemy data.
	divisionConsumed bool
	corpserExposed   bool

	// Condition flags
	isActive    bool
	isInvisible bool
	conditions  CombatConditions

	// Turn tracking
	turnDelay int
	hasActed  bool
}

// CombatConditions tracks status effects on a combat monster.
type CombatConditions struct {
	isPoisoned bool
	isAsleep   bool
	isFleeing  bool
}

// --- Identity Getters (immutable) ---

// MonsterType returns the monster type identity.
func (m *CombatMonster) MonsterType() references.MonsterType { return m.monsterType }

// CombatantID returns the immutable encounter-scoped monster identity assigned on registration.
func (m *CombatMonster) CombatantID() CombatantID { return m.combatantID }

// EnemyRef returns the enemy reference for stats and abilities.
func (m *CombatMonster) EnemyRef() *references.EnemyReference { return m.enemyRef }

// --- HP Getters and Setters ---

// CurrentHP returns the monster's current hit points.
func (m *CombatMonster) CurrentHP() int { return m.currentHP }

// MaxHP returns the monster's maximum hit points.
func (m *CombatMonster) MaxHP() int { return m.enemyRef.HitPoints }

// IsDead returns true if the monster has 0 or fewer HP.
func (m *CombatMonster) IsDead() bool { return m.currentHP <= 0 }

// SetCurrentHP sets HP with clamping to [0, MaxHP].
// Returns the actual HP value set (may differ from input due to clamping).
func (m *CombatMonster) SetCurrentHP(hitPoints int) int {
	maxHP := m.MaxHP()
	switch {
	case hitPoints < 0:
		m.currentHP = 0
	case hitPoints > maxHP:
		m.currentHP = maxHP
	default:
		m.currentHP = hitPoints
	}

	return m.currentHP
}

// TakeDamage reduces HP by damage amount, clamped to 0.
// Returns true if the monster died from this damage.
func (m *CombatMonster) TakeDamage(damage int) bool {
	if damage < 0 {
		damage = 0 // Prevent healing via negative damage
	}
	m.currentHP -= damage
	if m.currentHP < 0 {
		m.currentHP = 0
	}

	return m.currentHP == 0
}

// Heal restores HP, clamped to MaxHP.
// Returns the actual HP restored.
func (m *CombatMonster) Heal(amount int) int {
	if amount < 0 {
		return 0 // Prevent damage via negative heal
	}
	oldHP := m.currentHP
	m.SetCurrentHP(m.currentHP + amount)

	return m.currentHP - oldHP
}

// --- DEX Getters ---

// CurrentDex returns the monster's current dexterity (with random modifier).
func (m *CombatMonster) CurrentDex() int { return m.currentDex }

// BaseDex returns the monster's base dexterity from EnemyRef.
func (m *CombatMonster) BaseDex() int { return m.enemyRef.Dexterity }

// DexterityModifier returns the randomized DEX modifier selected at construction.
func (m *CombatMonster) DexterityModifier() int { return m.dexModifier }

// --- Position Getters and Setters ---

// Position returns the monster's position on the combat map.
func (m *CombatMonster) Position() references.Position { return m.position }

// SetPosition updates the monster's combat map position.
func (m *CombatMonster) SetPosition(pos references.Position) {
	m.position = pos
}

// Loyalty returns the side this monster supports during the encounter.
func (m *CombatMonster) Loyalty() CombatMonsterLoyalty { return m.loyalty }

// SetLoyalty changes the side this monster supports during the encounter.
func (m *CombatMonster) SetLoyalty(loyalty CombatMonsterLoyalty) {
	m.loyalty = loyalty
}

// IsHostile returns whether the monster is loyal to the opposing monster side.
func (m *CombatMonster) IsHostile() bool {
	return m.loyalty == CombatMonsterLoyaltyMonster
}

// IsPartyLoyal returns whether the monster supports the party.
func (m *CombatMonster) IsPartyLoyal() bool {
	return m.loyalty == CombatMonsterLoyaltyParty
}

// --- Active State Getters and Setters ---

// IsActive returns whether the monster is participating in combat.
func (m *CombatMonster) IsActive() bool { return m.isActive }

// SetActive marks the monster as participating in combat.
func (m *CombatMonster) SetActive(active bool) {
	m.isActive = active
}

// --- Invisibility Getters and Setters ---

// IsInvisible returns the monster's current invisibility state.
func (m *CombatMonster) IsInvisible() bool { return m.isInvisible }

// SetInvisible updates the monster's invisibility state.
func (m *CombatMonster) SetInvisible(invisible bool) {
	m.isInvisible = invisible
}

// IsCorpserHidden reports the original untargetable base-shape Corpser state.
func (m *CombatMonster) IsCorpserHidden() bool {
	return m.monsterType == references.MonsterCorpser && !m.corpserExposed
}

// IsCorpserExposed reports whether a Corpser has begun its adjacent attack animation.
func (m *CombatMonster) IsCorpserExposed() bool {
	return m.monsterType == references.MonsterCorpser && m.corpserExposed
}

// SetCorpserExposed changes Corpser attack-animation state. Other monster types ignore it.
func (m *CombatMonster) SetCorpserExposed(exposed bool) {
	if m.monsterType == references.MonsterCorpser {
		m.corpserExposed = exposed
	}
}

// --- Turn Tracking Getters and Setters ---

// --- Condition Getters and Setters ---

// IsPoisoned returns whether the monster is poisoned.
func (m *CombatMonster) IsPoisoned() bool { return m.conditions.isPoisoned }

// IsAsleep returns whether the monster is asleep.
func (m *CombatMonster) IsAsleep() bool { return m.conditions.isAsleep }

// SetAsleep sets the monster's sleep state.
func (m *CombatMonster) SetAsleep(asleep bool) {
	m.conditions.isAsleep = asleep
}

// IsFleeing reports whether the monster is trying to escape this encounter.
func (m *CombatMonster) IsFleeing() bool { return m.conditions.isFleeing }

// SetFleeing changes the monster's encounter-local fleeing condition.
func (m *CombatMonster) SetFleeing(fleeing bool) {
	m.conditions.isFleeing = fleeing
}

// --- CombatantProvider Interface ---

// GetDexterity implements CombatantProvider for turn queue ordering.
// Returns the monster's current effective dexterity (base + random modifier).
func (m *CombatMonster) GetDexterity() int {
	return m.currentDex
}

// IsAlive implements CombatantProvider.
func (m *CombatMonster) IsAlive() bool {
	return m.isActive && !m.IsDead()
}

// CanTakeTurn implements CombatantProvider.
// Returns false if the monster is dead or asleep.
func (m *CombatMonster) CanTakeTurn() bool {
	return m.CanAct()
}

// IsPC implements CombatantProvider. Always returns false for monsters.
func (m *CombatMonster) IsPC() bool {
	return false
}

// --- Ability Helpers ---

// HasAbility checks if the monster has a specific ability.
func (m *CombatMonster) HasAbility(ability references.EnemyAbility) bool {
	if m.enemyRef == nil {
		return false
	}

	return m.enemyRef.HasAbility(ability)
}

// CanAct returns true if the monster can take an action this turn.
// Monsters cannot act if dead or asleep.
func (m *CombatMonster) CanAct() bool {
	return m.IsAlive() && !m.conditions.isAsleep
}

// ApplyDamageModifiers adjusts damage for undead and immortal targets.
// Replicates the relevant parts of damage application from original combat damage rules.
//
// Original behavior:
//   - Immortal: damage = 0
//   - Undead: damage /= 2 if not magic weapon
//
// See ref:ENEMY-0003.
func (m *CombatMonster) ApplyDamageModifiers(damage int, weapon references.Equipment) int {
	isUndead := m.HasAbility(references.Undead)
	isImmortal := m.HasAbility(references.Immortal)
	isMagicWeapon := IsMagicWeapon(weapon)

	return CombatApplyDamageModifiers(damage, isUndead, isImmortal, isMagicWeapon)
}

// --- Constructor ---

// NewCombatMonster creates a combat monster with randomized DEX modifier.
// Implements original combat initialization monster initialization:
//   - HP = base HP from EnemyRef
//   - DEX = base DEX + random(-4, +3), reset to base when the byte candidate exceeds 30
//   - Turn delay = 36 - DEX
//
// See ref:ENEMY-0003.
func NewCombatMonster(
	monsterType references.MonsterType,
	enemyRef *references.EnemyReference,
	position references.Position,
	rng references.RandomProvider,
) *CombatMonster {
	monster := &CombatMonster{
		combatantID:      0,
		monsterType:      monsterType,
		enemyRef:         enemyRef,
		position:         position,
		currentHP:        enemyRef.HitPoints,
		currentDex:       enemyRef.Dexterity,
		dexModifier:      0,
		loyalty:          CombatMonsterLoyaltyMonster,
		divisionConsumed: false,
		corpserExposed:   false,
		isActive:         true,
		isInvisible:      false,
		conditions: CombatConditions{
			isPoisoned: false,
			isAsleep:   false,
			isFleeing:  false,
		},
		turnDelay: 0,
		hasActed:  false,
	}

	// Randomized encounter Dexterity uses an inclusive bounded draw.
	// Negative Go candidates model byte underflow, which also exceeds the
	// original maximum Dexterity check and resets to the base value. ref:ENEMY-0003
	dexMod := rng.Intn(dexModifierRange) + dexModifierMin
	monster.dexModifier = dexMod
	dexCandidate := enemyRef.Dexterity + dexMod
	monster.currentDex = dexCandidate
	if dexCandidate < 0 || dexCandidate > maxDexterity {
		monster.currentDex = enemyRef.Dexterity
	}

	// Turn delay based on DEX
	// See ref:ENEMY-0003.
	monster.turnDelay = turnDelayBaseDex - monster.currentDex

	return monster
}
