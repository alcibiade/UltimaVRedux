package game_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// REDUX DEVIATION: CRAFT-0005: the original has no gathering. With Gameplay.Crafting.Enabled,
// Alt+G on the party's own deed map harvests the adjacent resource node; see ref:CRAFT-0005.

// GatherDeedNode gathers from the deed-map square next to the party, in the given direction
// (ref:CRAFT-0005). It reports whether an attempt was made. An attempt, successful or not, has
// spent the rule's minutes beyond the ordinary turn; the caller then completes the ordinary
// command cycle, so the whole attempt costs exactly the rule's minutes. A refusal spends
// nothing, and the caller must not complete a cycle for it.
//
// Game RNG: a tool or hand harvest draws once for its yield; a bare-handed attempt at a node
// that needs a tool draws once for success and nothing more.
func (g *GameState) GatherDeedNode(direction references.Direction) bool {
	messages := g.SystemCallbacks.Message
	if !g.CraftingCommandApplies(CraftingGather) {
		messages.AddRowStr("Not here!")

		return false
	}
	// Catch up on any phase change before gathering, so a stale marker can never restore the
	// node this harvest is about to use up.
	g.processDeedRespawn()
	deed := g.crafting.active
	target := direction.GetNewPositionInDirection(&g.MapState.PlayerLocation.Position)
	if target == nil || !isDeedMapSquare(*target) || g.deedSquareOccupied(*target) {
		messages.AddRowStr("Nothing to gather!")

		return false
	}
	position := *target
	nodeDef, isNode := g.crafting.activeLayout.NodeAt(position)
	if !isNode {
		messages.AddRowStr("Nothing to gather!")

		return false
	}
	node := deed.nodeState(g.crafting.activeLayout, nodeDef)
	switch {
	case node.Covered:
		messages.AddRowStr("Nothing to gather!")

		return false
	case node.Depleted:
		messages.AddRowStr("Nothing left to gather!")

		return false
	}
	rule := references.DeedNodeRuleFor(node.Kind)
	inventory := &g.PartyState.Inventory
	if full, name := gatherStackFull(inventory, node.DeedResourceNode); full {
		messages.AddRowStr(fmt.Sprintf("Thou canst carry no more %s!", name))

		return false
	}
	yield := 0
	switch {
	case !rule.NeedsTool || inventory.Tools.HasSome(rule.Tool):
		yield = g.RandomIntInRange(rule.MinYield, rule.MaxYield)
	case rule.BareHandsOneIn == 0:
		messages.AddRowStr(fmt.Sprintf("Thou dost need a %s!", rule.Tool.Name()))

		return false
	case g.OneInXOdds(rule.BareHandsOneIn):
		yield = rule.BareHandsYield
	}
	if yield == 0 {
		material, _ := node.Kind.Material()
		messages.AddRowStr(fmt.Sprintf("Thy bare hands win no %s.", material.Name()))
		g.spendCraftingLabor(rule.Minutes)

		return true
	}
	name := addGatherYield(inventory, node.DeedResourceNode, yield)
	messages.AddRowStr(fmt.Sprintf("Gathered %d %s.", yield, name))
	g.SystemCallbacks.Audio.Play(SoundPushObject).Start()
	if deed.harvests == nil {
		deed.harvests = make(map[references.Position]int)
	}
	deed.harvests[position]++
	if deed.harvests[position] >= rule.Capacity {
		g.SetDeedTile(deed.ID, position, references.DeedDepletedSprite(deed.Terrain, nodeDef))
		messages.AddRowStr("It is used up!")
	}
	g.spendCraftingLabor(rule.Minutes)

	return true
}

// gatherStackFull reports whether the stack a node fills is already at its cap.
func gatherStackFull(inventory *party_state.Inventory, node references.DeedResourceNode) (bool, string) {
	if node.Kind == references.DeedResourceReagent {
		return inventory.Reagent.Get(node.Reagent) >= party_state.ItemQuantitySmallMax, node.Reagent.Name()
	}
	material, _ := node.Kind.Material()

	return inventory.Materials.Get(material) >= party_state.ItemQuantityLargeMax, material.Name()
}

// addGatherYield adds a harvest to the party's stock, saturating at the stack cap, and returns
// the name of what was gathered. Reagents go into the original reagent inventory.
func addGatherYield(inventory *party_state.Inventory, node references.DeedResourceNode, yield int) string {
	amount := uint16(yield) //nolint:gosec // Yields are small positive tunables.
	if node.Kind == references.DeedResourceReagent {
		inventory.Reagent.IncrementBy(node.Reagent, amount)

		return node.Reagent.Name()
	}
	material, _ := node.Kind.Material()
	inventory.Materials.IncrementBy(material, amount)

	return material.Name()
}

// spendCraftingLabor spends the minutes a crafting action takes beyond the ordinary turn of
// the current map: one minute on a deed map, two on the overworld. Crafting actions finish from
// a direction prompt or a menu, and the scene then completes the ordinary command cycle, which
// pays that turn; the total is exactly the minutes the Game Fact states.
func (g *GameState) spendCraftingLabor(minutes int) {
	turn := DefaultLargeMapMinutesPerTurn
	if g.MapState.PlayerLocation.Location.GetMapType() == references.SmallMapType {
		turn = DefaultSmallMapMinutesPerTurn
	}
	if extra := minutes - turn; extra > 0 {
		g.AdvanceGameplayMinutesAndProcessProvisions(extra)
	}
}

// deedNodeLookDescription describes a resource node of the loaded deed map for Look, or reports
// false for any other square, or a node covered by something else (ref:CRAFT-0005).
func (g *GameState) deedNodeLookDescription(position references.Position) (string, bool) {
	if !g.IsOnDeedMap() || !isDeedMapSquare(position) {
		return "", false
	}
	nodeDef, isNode := g.crafting.activeLayout.NodeAt(position)
	if !isNode {
		return "", false
	}
	node := g.crafting.active.nodeState(g.crafting.activeLayout, nodeDef)
	fresh, spent := references.DeedNodeLookName(nodeDef)
	switch {
	case node.Covered:
		return "", false
	case node.Depleted:
		return spent + " (depleted)", true
	default:
		return fresh, true
	}
}
