package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

const (
	combatCorpseLootMaxRoll      = 7
	combatCorpseFoodMaxRoll      = 3
	combatCorpseQuantityMaximum  = 3
	combatCorpsePlagueMaxRoll    = 31
	combatCorpsePlagueResult     = 19
	combatCorpseDescriptionCount = 3
)

type combatCorpseSearchOutcome struct {
	message  string
	item     references.Provision
	quantity uint16
	poisoned bool
	hasItem  bool
}

func rollCombatCorpseSearch(randomRange func(int, int) int) combatCorpseSearchOutcome {
	if randomRange(0, combatCorpseLootMaxRoll) == 0 {
		item := references.Gold
		if randomRange(0, combatCorpseFoodMaxRoll) == 0 {
			item = references.Food
		}

		return combatCorpseSearchOutcome{
			message:  combatCorpseItemMessage(item),
			item:     item,
			quantity: combatQuantity(randomRange(1, combatCorpseQuantityMaximum)),
			poisoned: false,
			hasItem:  true,
		}
	}

	if randomRange(0, combatCorpsePlagueMaxRoll) == combatCorpsePlagueResult {
		return combatCorpseSearchOutcome{
			message:  "Plague!",
			item:     references.NoProvision,
			quantity: 0,
			poisoned: true,
			hasItem:  false,
		}
	}

	detail := randomRange(0, randomRange(0, combatCorpseDescriptionCount))
	messages := []string{"nothing!", "worms!", "guts!", "a bloody pulp!"}

	return combatCorpseSearchOutcome{
		message:  messages[detail],
		item:     references.NoProvision,
		quantity: 0,
		poisoned: false,
		hasItem:  false,
	}
}

func combatCorpseItemMessage(item references.Provision) string {
	if item == references.Food {
		return "food!"
	}

	return "gold!"
}
