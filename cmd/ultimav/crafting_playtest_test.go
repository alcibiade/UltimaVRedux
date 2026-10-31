//go:build craftingmanual

package main

// An end-to-end playtest of the optional crafting system from the stock starting save. It
// drives a real GameScene through production input and writes PNG frames to the directory
// named by CRAFTING_MANUAL_DIR:
//
//	CRAFTING_MANUAL_DIR=/abs/dir go test -tags craftingmanual -run TestCraftingPlaytest ./cmd/ultimav/

import (
	"errors"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/files"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/player_save"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

type playtestRig struct {
	t        *testing.T
	game     *Game
	scene    *GameScene
	held     map[ebiten.Key]bool
	outDir   string
	revision string
	width    int
	height   int
	// repeatCaptured records that a repeat-craft frame has been captured.
	repeatCaptured bool
}

func newPlaytestRig(t *testing.T) *playtestRig {
	t.Helper()
	outDir := os.Getenv("CRAFTING_MANUAL_DIR")
	require.NotEmpty(t, outDir)
	require.NoError(t, os.MkdirAll(outDir, 0o755))
	dataPath := os.Getenv("ULTIMA5_DATA_PATH")
	require.NotEmpty(t, dataPath)

	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "playtest.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("existing: true\n"), 0o600))
	configuration, err := config.LoadUltimaVConfiguration(configPath, dataPath)
	require.NoError(t, err)
	configuration.SoundDisabled = true

	scene, err := NewGameSceneFromLegacySaveFile(configuration, files.Resolve(dataPath, "SAVED.GAM"))
	require.NoError(t, err)
	t.Cleanup(scene.Dispose)
	scene.transientCommandInput.focused = func() bool { return true }
	scene.startTime = time.Now()

	rig := &playtestRig{t: t, scene: scene, held: map[ebiten.Key]bool{}, outDir: outDir, width: 1280, height: 720}
	scene.keyboard = input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return rig.held[key] })
	rig.game = &Game{currentScene: scene, playerSaveNow: time.Now, playerSaveGameVersion: "playtest"}
	require.NoError(t, configurePlayerSaveRepository(rig.game, playerSaveStartupOptions{root: t.TempDir()}))

	manager := display.GetManager()
	saved := *manager
	t.Cleanup(func() { *manager = saved })
	manager.PublishLogicalSize(rig.width, rig.height)

	return rig
}

func (r *playtestRig) state() *game_state.GameState { return r.scene.gameState }

func (r *playtestRig) press(keys ...ebiten.Key) {
	r.t.Helper()
	clear(r.held)
	for _, key := range keys {
		r.held[key] = true
	}
	r.pump(1)
	clear(r.held)
	r.pump(2)
}

func (r *playtestRig) pump(frames int) {
	for range frames {
		require.NoError(r.t, r.scene.Update(r.game))
		time.Sleep(16 * time.Millisecond)
	}
}

func (r *playtestRig) topMenu() *widgets.ButtonListModal {
	menu, ok := r.scene.dialogStack.PeekTopModalDialog().(*widgets.ButtonListModal)
	require.True(r.t, ok, "top dialog is not a menu: %T", r.scene.dialogStack.PeekTopModalDialog())

	return menu
}

func (r *playtestRig) menuRow(prefix string) int {
	for index, label := range r.topMenu().ButtonTexts() {
		if strings.HasPrefix(label, prefix) {
			return index
		}
	}
	r.t.Fatalf("menu has no row %q: %v", prefix, r.topMenu().ButtonTexts())

	return -1
}

func (r *playtestRig) menuChoose(rows int) {
	for range rows {
		r.press(ebiten.KeyDown)
	}
	r.press(ebiten.KeyEnter)
}

func (r *playtestRig) capture(name string) {
	r.t.Helper()
	r.pump(30)
	screen := newPresentationRaster(r.width, r.height)
	defer screen.Deallocate()
	r.scene.Draw(screen)
	file, err := os.Create(filepath.Join(r.outDir, name+".png"))
	require.NoError(r.t, err)
	opaque := image.NewRGBA(screen.Bounds())
	draw.Draw(opaque, opaque.Bounds(), image.Black, image.Point{}, draw.Src)
	draw.Draw(opaque, opaque.Bounds(), screen, image.Point{}, draw.Over)
	require.NoError(r.t, png.Encode(file, opaque))
	require.NoError(r.t, file.Close())
}

func (r *playtestRig) logState(label string) {
	s := r.state()
	r.t.Logf("%s: loc=%v floor=%d pos=%v gold=%d date=%v", label, s.MapState.PlayerLocation.Location,
		s.MapState.PlayerLocation.Floor, s.MapState.PlayerLocation.Position, s.PartyState.Inventory.Gold.Get(), s.DateTime)
}

func TestCraftingPlaytest(t *testing.T) {
	rig := newPlaytestRig(t)
	rig.pump(5)
	rig.logState("start")
	rig.capture("p00-start")

	// Esc, Configure, the Crafting and Deeds row, Space, Enter.
	rig.press(ebiten.KeyEscape)
	rig.pump(5)
	rig.capture("p01-escape-menu")
	rig.menuChoose(rig.menuRow("Configure"))
	rig.pump(5)
	for range gameplayCraftingEnabledIndex {
		rig.press(ebiten.KeyDown)
	}
	rig.press(ebiten.KeySpace)
	rig.capture("p02-configure-crafting-on")
	rig.press(ebiten.KeyEnter)
	rig.pump(20)
	rig.t.Logf("top dialog after apply: %T", rig.scene.dialogStack.PeekTopModalDialog())
	if _, ok := rig.scene.dialogStack.PeekTopModalDialog().(*widgets.ButtonListModal); ok {
		rig.press(ebiten.KeyEscape)
		rig.pump(20)
	}
	require.True(t, rig.state().IsCraftingEnabled(), "Configure turned the option on")
	rig.t.Logf("dialogs left: %d", rig.scene.dialogStack.Count())
	rig.capture("p03-after-configure")

	rig.walkOut()
	rig.logState("overworld")
	rig.capture("p04-overworld")
	rig.playDeedLoop()
}

// walkOut walks south out of the starting hut, opening doors and answering the leave prompt.
func (r *playtestRig) walkOut() {
	r.t.Helper()
	for range 40 {
		if r.state().MapState.PlayerLocation.Location == references.BritanniaUnderworld && !r.state().EventQueueManager.HasActiveWorkflows() {
			return
		}
		before := r.state().MapState.PlayerLocation.Position
		r.press(ebiten.KeyDown)
		r.pump(10)
		if r.scene.dialogStack.Count() > 0 || r.state().MapState.PlayerLocation.Position == before {
			r.press(ebiten.KeyY)
			r.press(ebiten.KeyEnter)
			r.pump(10)
		}
		if r.state().MapState.PlayerLocation.Position == before && r.state().MapState.PlayerLocation.Location != references.BritanniaUnderworld {
			r.press(ebiten.KeyO)
			r.press(ebiten.KeyDown)
			r.pump(10)
		}
		for range 100 {
			if !r.state().EventQueueManager.HasActiveWorkflows() {
				break
			}
			r.pump(1)
		}
	}
	r.capture("zz-walkout-failure")
	r.t.Fatal("never left the hut")
}

func (r *playtestRig) out() string {
	lines := strings.Split(strings.TrimSpace(r.scene.output.GetOutputStr(false)), "\n")
	if len(lines) > 4 {
		lines = lines[len(lines)-4:]
	}

	return strings.Join(lines, " | ")
}

func (r *playtestRig) settle() {
	for range 2000 {
		if !r.state().EventQueueManager.HasActiveWorkflows() {
			return
		}
		r.pump(1)
	}
	r.capture("zz-settle-failure")
	r.t.Fatal("workflows never settled")
}

func (r *playtestRig) minutes() int {
	d := r.state().DateTime

	return d.GetTotalDays()*1440 + int(d.Hour)*60 + int(d.Minute)
}

func (r *playtestRig) pos() references.Position { return r.state().MapState.PlayerLocation.Position }

var playtestSteps = []struct {
	step references.Position
	key  ebiten.Key
}{{references.Position{Y: -1}, ebiten.KeyUp}, {references.Position{Y: 1}, ebiten.KeyDown},
	{references.Position{X: -1}, ebiten.KeyLeft}, {references.Position{X: 1}, ebiten.KeyRight}}

func playtestKeyFor(from, to references.Position) ebiten.Key {
	for _, s := range playtestSteps {
		if from.X+s.step.X == to.X && from.Y+s.step.Y == to.Y {
			return s.key
		}
	}

	return ebiten.KeySpace
}

// walkTo walks with arrow keys along a shortest passable route to one of the goals (breadth-first
// over IsPassable, within limit squares of the start), and reports whether it arrived.
func (r *playtestRig) walkTo(goals []references.Position, limit int) bool {
	r.t.Helper()
	goal := map[references.Position]bool{}
	for _, g := range goals {
		goal[g] = true
	}
	for attempt := 0; attempt < 3; attempt++ {
		start := r.pos()
		if goal[start] {
			return true
		}
		parent := map[references.Position]references.Position{start: start}
		queue := []references.Position{start}
		var found *references.Position
		for len(queue) > 0 && found == nil {
			current := queue[0]
			queue = queue[1:]
			for _, s := range playtestSteps {
				next := references.Position{X: current.X + s.step.X, Y: current.Y + s.step.Y}
				if _, seen := parent[next]; seen {
					continue
				}
				if ptAbs(int(next.X)-int(start.X)) > limit || ptAbs(int(next.Y)-int(start.Y)) > limit {
					continue
				}
				if !r.state().IsPassable(&next) {
					continue
				}
				parent[next] = current
				if goal[next] {
					found = &next

					break
				}
				queue = append(queue, next)
			}
		}
		if found == nil {
			return false
		}
		var route []references.Position
		for at := *found; at != start; at = parent[at] {
			route = append([]references.Position{at}, route...)
		}
		ok := true
		for _, next := range route {
			hp := r.partyHP()
			r.press(playtestKeyFor(r.pos(), next))
			r.settle()
			if after := r.partyHP(); after < hp {
				r.t.Logf("HP: stepping onto %v cost %d HP (tile %v): %s", next, hp-after,
					r.state().GetCurrentLayeredMap().GetTopTile(&next).Index, r.out())
			}
			if r.pos() != next {
				r.t.Logf("walk blocked at %v -> %v: %s", r.pos(), next, r.out())
				ok = false

				break
			}
		}
		if ok {
			return true
		}
	}

	return false
}

func ptAbs(v int) int {
	if v < 0 {
		return -v
	}

	return v
}

// neighbours lists the four squares around a position.
func ptNeighbours(p references.Position) []references.Position {
	out := make([]references.Position, 0, 4)
	for _, s := range playtestSteps {
		out = append(out, references.Position{X: p.X + s.step.X, Y: p.Y + s.step.Y})
	}

	return out
}

// walkNextTo walks to a passable orthogonal neighbour of the target and returns the key facing it.
func (r *playtestRig) walkNextTo(target references.Position) (ebiten.Key, bool) {
	if !r.walkTo(ptNeighbours(target), 32) {
		return 0, false
	}

	return playtestKeyFor(r.pos(), target), true
}

// findDeedSite walks the overworld near the party for a square a deed of the terrain may be
// staked on, ignoring the gold check, and walks there.
func (r *playtestRig) findDeedSite(terrain references.DeedTerrain) references.Position {
	r.t.Helper()
	origin := r.pos()
	var best *references.Position
	for radius := 1; radius < 40 && best == nil; radius++ {
		for dy := -radius; dy <= radius && best == nil; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				if ptAbs(dx) != radius && ptAbs(dy) != radius {
					continue
				}
				candidate := references.Position{X: origin.X + references.Coordinate(dx), Y: origin.Y + references.Coordinate(dy)}
				r.state().MapState.PlayerLocation.Position = candidate
				proposal := r.state().ProposeDeed()
				r.state().MapState.PlayerLocation.Position = origin
				if (proposal.Err == nil || errors.Is(proposal.Err, game_state.ErrDeedNotEnoughGold)) && proposal.Terrain == terrain {
					best = &candidate

					break
				}
			}
		}
	}
	require.NotNil(r.t, best, "no %v deed site near %v", terrain, origin)
	r.t.Logf("deed site %v, %d squares from %v", *best, ptAbs(int(best.X-origin.X))+ptAbs(int(best.Y-origin.Y)), origin)
	start := r.minutes()
	require.True(r.t, r.walkTo([]references.Position{*best}, 45), "cannot walk to the deed site %v", *best)
	r.t.Logf("walked to the site in %d minutes", r.minutes()-start)

	return *best
}

func (r *playtestRig) enterDeed() {
	r.t.Helper()
	r.press(ebiten.KeyE)
	for range 400 {
		if r.state().IsOnDeedMap() && !r.state().EventQueueManager.HasActiveWorkflows() {
			r.pump(5)

			return
		}
		r.pump(1)
	}
	r.capture("zz-enter-failure")
	r.t.Fatalf("never reached the deed map: %s", r.out())
}

func (r *playtestRig) leaveDeed() {
	r.t.Helper()
	require.True(r.t, r.walkTo([]references.Position{{X: 15, Y: 31}}, 32), "cannot reach the south edge")
	r.press(ebiten.KeyDown)
	r.pump(10)
	r.press(ebiten.KeyY)
	r.press(ebiten.KeyEnter)
	for range 400 {
		if !r.state().IsOnDeedMap() && !r.state().EventQueueManager.HasActiveWorkflows() {
			r.pump(5)

			return
		}
		r.pump(1)
	}
	r.capture("zz-leave-failure")
	r.t.Fatalf("never left the deed: %s", r.out())
}

func (r *playtestRig) activeDeed() game_state.Deed {
	deed, ok := r.state().ActiveDeed()
	require.True(r.t, ok)

	return deed
}

// freshNode returns the fresh, uncovered node of the kind nearest the party.
func (r *playtestRig) freshNode(kind references.DeedResourceKind) (game_state.DeedResourceNodeState, bool) {
	var best game_state.DeedResourceNodeState
	bestDistance := 1 << 30
	for _, node := range r.state().DeedResourceNodes(r.activeDeed().ID) {
		if node.Kind != kind || node.Depleted || node.Covered {
			continue
		}
		d := ptAbs(int(node.Position.X-r.pos().X)) + ptAbs(int(node.Position.Y-r.pos().Y))
		if d < bestDistance {
			best, bestDistance = node, d
		}
	}

	return best, bestDistance < 1<<30
}

type playtestTally struct {
	attempts, minutes, trammelWaits int
	messages                        map[string]int
}

var tally = map[references.DeedResourceKind]*playtestTally{}

// gatherUntil gathers nodes of a kind until done reports true, waiting for Trammel when every node
// of the kind is used up.
func (r *playtestRig) gatherUntil(kind references.DeedResourceKind, done func() bool) {
	r.t.Helper()
	t := tally[kind]
	if t == nil {
		t = &playtestTally{messages: map[string]int{}}
		tally[kind] = t
	}
	for guard := 0; !done(); guard++ {
		require.Less(r.t, guard, 400, "gathering %s never finished", kind.Name())
		node, ok := r.freshNode(kind)
		if !ok {
			r.waitForTrammel()
			t.trammelWaits++

			continue
		}
		key, ok := r.walkNextTo(node.Position)
		require.True(r.t, ok, "cannot reach %s node at %v", kind.Name(), node.Position)
		before := r.minutes()
		r.scene.output.Clear()
		r.press(ebiten.KeyAltLeft, ebiten.KeyG)
		r.press(key)
		r.settle()
		t.attempts++
		t.minutes += r.minutes() - before
		message := strings.TrimSpace(strings.ReplaceAll(r.out(), "Gather-", ""))
		for _, digit := range "0123456789" {
			message = strings.ReplaceAll(message, string(digit), "#")
		}
		t.messages[message]++
	}
}

// waitForTrammel moves the clock to the eve of the next Trammel phase change and passes turns
// across midnight (a clock shortcut; the respawn rule runs as in play).
func (r *playtestRig) waitForTrammel() {
	r.t.Helper()
	eve := r.state().DateTime
	for datetime.TrammelPhaseChanges(eve.GetTotalDays()+1) == datetime.TrammelPhaseChanges(eve.GetTotalDays()) {
		eve.Day++
		if eve.Day > 28 {
			eve.Day = 1
			eve.Month++
		}
	}
	eve.Hour, eve.Minute = 23, 57
	r.state().DateTime = eve
	r.settle()
	for range 4 {
		r.press(ebiten.KeySpace)
		r.settle()
	}
	r.t.Logf("waited for Trammel: now %v", r.state().DateTime)
}

// craft presses Alt+C and picks the category and recipe rows, returning the message and minutes.
// craft makes a recipe through Alt+C, its category and its row, and returns what the craft
// printed and the minutes it took. A successful craft leaves the menu open on the same row
// (ref:CRAFT-0007); the caller presses Enter to craft again or calls closeCraftMenu.
func (r *playtestRig) craft(category, recipe string) (string, int) {
	r.t.Helper()
	before := r.minutes()
	r.scene.output.Clear()
	r.press(ebiten.KeyAltLeft, ebiten.KeyC)
	r.menuChoose(r.menuRow(category))
	r.menuChoose(r.menuRow(recipe))
	r.settle()

	return r.out(), r.minutes() - before
}

// craftAgain presses Enter on the reopened Craft menu, crafting the same recipe again.
func (r *playtestRig) craftAgain(recipe string) (string, int) {
	r.t.Helper()
	require.Positive(r.t, r.scene.dialogStack.Count(), "the Craft menu did not stay open for another %s", recipe)
	require.True(r.t, strings.HasPrefix(r.topMenu().ButtonTexts()[r.topMenu().SelectedIndex()], recipe),
		"the reopened menu is not on %s", recipe)
	before := r.minutes()
	r.scene.output.Clear()
	r.press(ebiten.KeyEnter)
	r.settle()

	return r.out(), r.minutes() - before
}

// closeCraftMenu presses Esc on a Craft menu left open after crafting; closing spends nothing.
func (r *playtestRig) closeCraftMenu() {
	r.t.Helper()
	if r.scene.dialogStack.Count() == 0 {
		return
	}
	before := r.minutes()
	r.press(ebiten.KeyEscape)
	r.settle()
	require.Zero(r.t, r.scene.dialogStack.Count(), "Esc did not close the Craft menu")
	require.Equal(r.t, before, r.minutes(), "closing the Craft menu spent time")
}

// mustCraft crafts a recipe times times: once through the menus, then again from the menu the
// craft leaves open, checking each craft costs exactly the recipe's minutes.
func (r *playtestRig) mustCraft(category, recipe string, times int) {
	r.t.Helper()
	for index := range times {
		var message string
		var minutes int
		if index == 0 {
			message, minutes = r.craft(category, recipe)
		} else {
			message, minutes = r.craftAgain(recipe)
			if index == 1 && !r.repeatCaptured {
				r.repeatCaptured = true
				r.capture("p14b-repeat-craft")
			}
		}
		require.Contains(r.t, message, "Crafted", "craft %s: %s", recipe, message)
		r.t.Logf("craft %s: %q in %d min", recipe, message, minutes)
		if want := r.recipeMinutes(recipe); minutes != want {
			r.t.Errorf("TIMING: craft %s took %d minutes, want %d", recipe, minutes, want)
		}
	}
	r.closeCraftMenu()
}

// recipeMinutes returns the Game Fact minutes of the recipe whose name the menu row starts with.
func (r *playtestRig) recipeMinutes(name string) int {
	r.t.Helper()
	for _, recipe := range references.CraftRecipes() {
		if recipe.Name == name {
			return recipe.Minutes
		}
	}
	r.t.Fatalf("no recipe named %q", name)

	return 0
}

// build places a carried piece on the target through Alt+B, standing on a given square.
func (r *playtestRig) build(id references.BuildPieceID, target, stand references.Position) {
	r.t.Helper()
	piece := id.Piece()
	require.True(r.t, r.walkTo([]references.Position{stand}, 32), "cannot stand at %v to build %s", stand, piece.Name)
	before := r.minutes()
	r.scene.output.Clear()
	r.press(ebiten.KeyAltLeft, ebiten.KeyB)
	r.menuChoose(r.menuRow(piece.Category.Name()))
	r.menuChoose(r.menuRow(piece.Name + " ("))
	r.press(playtestKeyFor(stand, target))
	r.settle()
	tile, _ := r.state().DeedTileAt(r.activeDeed().ID, target)
	require.Equal(r.t, piece.Sprite, tile, "%s was not built at %v: %s", piece.Name, target, r.out())
	if minutes := r.minutes() - before; minutes != 5 {
		r.t.Logf("TIMING: building %s took %d minutes", piece.Name, minutes)
	}
}

func (r *playtestRig) material(m references.Material) int {
	return int(r.state().PartyState.Inventory.Materials.Get(m))
}

func (r *playtestRig) playDeedLoop() {
	t := r.t
	t.Helper()
	inventory := &r.state().PartyState.Inventory
	r.state().DebugOptions.MonsterGen = false // Deterministic walk; noted in the report.
	r.state().SuppressDeedRaidsForTest(true)

	// Find grassland and try to stake it with the starting gold.
	site := r.findDeedSite(references.DeedTerrainGrassland)
	calls := r.state().GetRNG().Calls()
	r.pump(60)
	t.Logf("idle: 60 frames on the overworld draw %d RNG", r.state().GetRNG().Calls()-calls)
	calls = r.state().GetRNG().Calls()
	r.press(ebiten.KeyAltLeft, ebiten.KeyG)
	r.settle()
	t.Logf("Alt+G on the overworld: %q, %d RNG", r.out(), r.state().GetRNG().Calls()-calls)
	r.scene.output.Clear()
	r.press(ebiten.KeyAltLeft, ebiten.KeyB)
	r.pump(5)
	t.Logf("Alt+B with %d gold: %q, dialogs %d", inventory.Gold.Get(), r.out(), r.scene.dialogStack.Count())
	r.capture("p05-stake-offer")
	if _, menu := r.scene.dialogStack.PeekTopModalDialog().(*widgets.ButtonListModal); menu {
		r.press(ebiten.KeyEscape)
		r.settle()
	}
	if inventory.Gold.Get() < references.FirstDeedGoldCost {
		t.Logf("BALANCE: the stock party has %d gold, short of the %d-gold deed; topping up", inventory.Gold.Get(), references.FirstDeedGoldCost)
		inventory.Gold.Set(references.FirstDeedGoldCost)
	}
	r.scene.output.Clear()
	r.press(ebiten.KeyAltLeft, ebiten.KeyB)
	r.pump(5)
	r.capture("p06-stake-menu")
	r.menuChoose(r.menuRow("Stake"))
	r.pump(5)
	for _, c := range "meadowmere" {
		r.press(ebiten.KeyA + ebiten.Key(c-'a'))
	}
	r.capture("p07-name-prompt")
	r.press(ebiten.KeyEnter)
	r.settle()
	_, owned := r.state().DeedAt(site)
	require.True(t, owned, "staking failed: %s", r.out())
	t.Logf("staked: %q gold now %d", r.out(), inventory.Gold.Get())
	r.capture("p08-staked")

	r.scene.output.Clear()
	r.enterDeed()
	t.Logf("entered: %q at %v", r.out(), r.pos())
	r.capture("p09-deed-entry")
	nodeCount := map[references.DeedResourceKind]int{}
	for _, node := range r.state().DeedResourceNodes(r.activeDeed().ID) {
		nodeCount[node.Kind]++
	}
	t.Logf("nodes: %v", nodeCount)

	r.cancelPaths()

	// Bare hands: the Axe and the Pick need 4 Wood and 3 Stone.
	start := r.minutes()
	r.gatherUntil(references.DeedResourceWood, func() bool { return r.material(references.Wood) >= 4 })
	r.gatherUntil(references.DeedResourceStone, func() bool { return r.material(references.Stone) >= 3 })
	t.Logf("bare hands: %d minutes for 4 Wood and 3 Stone", r.minutes()-start)
	r.capture("p10-bare-hands")
	r.scene.output.Clear()
	r.press(ebiten.KeyAltLeft, ebiten.KeyC)
	r.pump(5)
	r.capture("p11-craft-menu")
	r.press(ebiten.KeyEscape)
	r.settle()
	r.mustCraft("Tools", "Axe", 1)
	r.mustCraft("Tools", "Pick", 1)
	r.capture("p12-tools")

	// Tools: everything the house, the stations and a dagger need.
	start = r.minutes()
	r.gatherUntil(references.DeedResourceOre, func() bool { return r.material(references.Ore) >= 5 })
	t.Logf("ore: 5 Ore in %d minutes, %d Trammel waits", r.minutes()-start, tally[references.DeedResourceOre].trammelWaits)
	r.capture("p13-ore")
	start = r.minutes()
	r.gatherUntil(references.DeedResourceWood, func() bool { return r.material(references.Wood) >= 24 })
	r.gatherUntil(references.DeedResourceStone, func() bool { return r.material(references.Stone) >= 45 })
	r.gatherUntil(references.DeedResourceForage, func() bool { return r.material(references.Forage) >= 2 })
	reagentsBefore := 0
	for _, reagent := range references.ReagentCatalog() {
		reagentsBefore += int(inventory.Reagent.Get(reagent))
	}
	r.gatherUntil(references.DeedResourceReagent, func() bool {
		total := 0
		for _, reagent := range references.ReagentCatalog() {
			total += int(inventory.Reagent.Get(reagent))
		}

		return total > reagentsBefore
	})
	t.Logf("gathered: wood %d stone %d forage %d in %d minutes", r.material(references.Wood),
		r.material(references.Stone), r.material(references.Forage), r.minutes()-start)
	r.capture("p14-gathered")
	for kind, tl := range tally {
		t.Logf("tally %s: attempts %d minutes %d waits %d messages %v", kind.Name(), tl.attempts, tl.minutes, tl.trammelWaits, tl.messages)
	}

	// Stations east of the yard, reached from 18,17.
	r.mustCraft("Stations", "Workbench", 1)
	r.mustCraft("Stations", "Sawmill", 1)
	workbench, sawmill, forge := references.Position{X: 17, Y: 18}, references.Position{X: 18, Y: 18}, references.Position{X: 19, Y: 18}
	r.build(references.PieceWorkbench, workbench, references.Position{X: 17, Y: 17})
	r.build(references.PieceSawmill, sawmill, references.Position{X: 18, Y: 17})
	require.True(t, r.walkTo([]references.Position{{X: 18, Y: 17}}, 32))
	r.mustCraft("Refined materials", "Cut stone", 13)
	r.mustCraft("Refined materials", "Planks", 3)
	r.mustCraft("Stations", "Forge", 1)
	r.build(references.PieceForge, forge, references.Position{X: 19, Y: 17})
	require.True(t, r.walkTo([]references.Position{{X: 18, Y: 17}}, 32))
	r.mustCraft("Refined materials", "Ingots", 1)
	r.mustCraft("Stations", "Fireplace", 1)
	r.capture("p15-stations")

	// Forge a dagger and ready it.
	r.mustCraft("Blades", "Dagger", 1)
	r.readyOnAvatar(references.Dagger)

	// The house: outer walls 12..16 x 11..14, the fireplace at 14,11 and the door at 14,14.
	for _, piece := range []struct {
		id            references.BuildPieceID
		target, stand references.Position
	}{
		{references.PieceStoneWall, references.Position{X: 12, Y: 11}, references.Position{X: 12, Y: 10}},
		{references.PieceStoneWall, references.Position{X: 13, Y: 11}, references.Position{X: 13, Y: 10}},
		{references.PieceFireplace, references.Position{X: 14, Y: 11}, references.Position{X: 14, Y: 10}},
		{references.PieceStoneWall, references.Position{X: 15, Y: 11}, references.Position{X: 15, Y: 10}},
		{references.PieceStoneWall, references.Position{X: 16, Y: 11}, references.Position{X: 16, Y: 10}},
		{references.PieceStoneWall, references.Position{X: 12, Y: 12}, references.Position{X: 11, Y: 12}},
		{references.PieceStoneWall, references.Position{X: 12, Y: 13}, references.Position{X: 11, Y: 13}},
		{references.PieceStoneWall, references.Position{X: 12, Y: 14}, references.Position{X: 11, Y: 14}},
		{references.PieceStoneWall, references.Position{X: 16, Y: 12}, references.Position{X: 17, Y: 12}},
		{references.PieceStoneWall, references.Position{X: 16, Y: 13}, references.Position{X: 17, Y: 13}},
		{references.PieceStoneWall, references.Position{X: 16, Y: 14}, references.Position{X: 17, Y: 14}},
		{references.PieceStoneWall, references.Position{X: 13, Y: 14}, references.Position{X: 13, Y: 15}},
		{references.PieceStoneWall, references.Position{X: 15, Y: 14}, references.Position{X: 15, Y: 15}},
	} {
		if inventory.Pieces.Get(piece.id) == 0 {
			r.walkTo([]references.Position{{X: 18, Y: 17}}, 32)
			piece := piece.id.Piece()
			r.mustCraft(piece.Category.Name(), piece.Name, 1)
		}
		r.build(piece.id, piece.target, piece.stand)
	}
	r.walkTo([]references.Position{{X: 18, Y: 17}}, 32)
	r.mustCraft("Walls and doors", "Door", 1)
	r.mustCraft("Furnishings", "Bed", 1)
	r.mustCraft("Furnishings", "Storage chest", 1)
	r.build(references.PieceBedHead, references.Position{X: 13, Y: 12}, references.Position{X: 13, Y: 13})
	r.build(references.PieceBedFoot, references.Position{X: 14, Y: 12}, references.Position{X: 14, Y: 13})
	r.build(references.PieceStorageChest, references.Position{X: 15, Y: 12}, references.Position{X: 15, Y: 13})
	r.build(references.PieceDoor, references.Position{X: 14, Y: 14}, references.Position{X: 14, Y: 15})
	r.capture("p16-house-outside")

	// In through the door.
	r.press(ebiten.KeyO)
	r.press(ebiten.KeyUp)
	r.settle()
	r.press(ebiten.KeyUp)
	r.settle()
	r.press(ebiten.KeyUp)
	r.settle()
	t.Logf("inside at %v: %q", r.pos(), r.out())
	r.capture("p17-house-inside")

	// Store the remaining materials in the chest.
	require.True(t, r.walkTo([]references.Position{{X: 15, Y: 13}}, 32))
	r.scene.output.Clear()
	r.press(ebiten.KeyO)
	r.press(ebiten.KeyUp)
	r.pump(10)
	r.capture("p18-chest")
	r.menuChoose(r.menuRow("Store"))
	r.capture("p19-chest-store")
	r.menuChoose(r.menuRow("Materials"))
	r.capture("p20-chest-stored")
	r.press(ebiten.KeyEscape)
	r.pump(5)
	if r.scene.dialogStack.Count() > 0 {
		r.press(ebiten.KeyEscape)
	}
	r.settle()
	contents, _ := r.state().StorageChestContents(references.Position{X: 15, Y: 12})
	t.Logf("chest after store: %+v; party wood %d", contents.Materials, r.material(references.Wood))

	// Take Wood and Forage back to cook, standing on the bed foot next to the hearth.
	r.press(ebiten.KeyO)
	r.press(ebiten.KeyUp)
	r.pump(10)
	r.menuChoose(r.menuRow("Take"))
	r.menuChoose(r.menuRow("Materials"))
	r.press(ebiten.KeyEscape)
	r.pump(5)
	if r.scene.dialogStack.Count() > 0 {
		r.press(ebiten.KeyEscape)
	}
	r.settle()
	require.True(t, r.walkTo([]references.Position{{X: 14, Y: 12}}, 32), "cannot step onto the bed foot")
	food := inventory.Provisions.Food.Get()
	r.mustCraft("Provisions", "Cook food", 1)
	t.Logf("food %d -> %d", food, inventory.Provisions.Food.Get())
	r.capture("p21-cooked")

	// Rest in the bed, wounded, in the evening.
	require.True(t, r.walkTo([]references.Position{{X: 13, Y: 12}}, 32), "cannot step onto the bed head")
	avatar := r.state().PartyState.GetCharacter(0)
	avatar.SetHp(10)
	before := r.state().DateTime
	r.scene.output.Clear()
	r.press(ebiten.KeyH)
	r.pump(60)
	r.capture("p22-sleeping")
	r.settle()
	r.pump(10)
	for r.scene.dialogStack.Count() > 0 {
		r.press(ebiten.KeyEnter)
		r.settle()
	}
	t.Logf("rest: %v -> %v, hp %d, at %v: %q", before, r.state().DateTime, avatar.GetCurrentHp(), r.pos(), r.out())
	r.capture("p23-rested")

	// Store the leftovers and the spare daggers for the save check, then demolish and rebuild a wall.
	require.True(t, r.walkTo([]references.Position{{X: 15, Y: 13}}, 32))
	r.press(ebiten.KeyO)
	r.press(ebiten.KeyUp)
	r.pump(10)
	r.menuChoose(r.menuRow("Store"))
	r.capture("p23b-chest-store-menu")
	food, torches, keys := inventory.Provisions.Food.Get(), inventory.Provisions.Torches.Get(), inventory.Provisions.Keys.Get()
	axes, picks := inventory.Tools.Get(references.Axe), inventory.Tools.Get(references.Pick)
	r.menuChoose(r.menuRow("Everything"))
	r.capture("p23c-chest-everything")
	// Storing everything leaves food, torches, keys and tools with the party (ref:CRAFT-0011).
	require.Equal(t, food, inventory.Provisions.Food.Get(), "the party keeps its food")
	require.Equal(t, torches, inventory.Provisions.Torches.Get())
	require.Equal(t, keys, inventory.Provisions.Keys.Get())
	require.Equal(t, axes, inventory.Tools.Get(references.Axe))
	require.Equal(t, picks, inventory.Tools.Get(references.Pick))
	t.Logf("after Everything the party keeps food %d, torches %d, keys %d, axes %d, picks %d",
		food, torches, keys, axes, picks)
	for range 3 {
		if r.scene.dialogStack.Count() == 0 {
			break
		}
		r.press(ebiten.KeyEscape)
		r.pump(5)
	}
	r.settle()
	contents, _ = r.state().StorageChestContents(references.Position{X: 15, Y: 12})
	t.Logf("chest after Everything: materials %v equipment dagger %d; avatar weapon %v; party daggers %d",
		contents.Materials, contents.Equipment[references.Dagger], avatar.GetEquippedItem(references.SlotWeapon),
		inventory.Equipment.Get(references.Dagger))
	r.demolishAndRebuild()
	r.sellPrices()

	r.saveAndReload()
	r.leaveAndRegrow(site)
}

func (r *playtestRig) readyOnAvatar(item references.Equipment) {
	r.t.Helper()
	r.scene.output.Clear()
	r.press(ebiten.KeyR)
	r.pump(10)
	r.capture("p15b-ready-open")
	ready, ok := r.scene.dialogStack.PeekTopModalDialog().(*widgets.ReadyEquipmentDialog)
	if !ok {
		r.t.Logf("R opened %T first; choosing the Avatar", r.scene.dialogStack.PeekTopModalDialog())
		r.press(ebiten.KeyEnter)
		r.pump(10)
		ready, ok = r.scene.dialogStack.PeekTopModalDialog().(*widgets.ReadyEquipmentDialog)
	}
	require.True(r.t, ok, "R opens the Ready dialog")
	for range 80 {
		if selected, _ := ready.GetSelectedEquipment(); selected == item {
			break
		}
		r.press(ebiten.KeyDown)
	}
	r.press(ebiten.KeyEnter)
	r.pump(10)
	r.capture("p15c-replace")
	r.press(ebiten.KeyEnter)
	r.pump(10)
	r.capture("p15d-readied")
	for range 5 {
		if r.scene.dialogStack.Count() == 0 {
			break
		}
		r.press(ebiten.KeyEscape)
		r.pump(5)
	}
	r.settle()
	avatar := r.state().PartyState.GetCharacter(0)
	r.t.Logf("avatar weapon slot: %v (%q)", avatar.GetEquippedItem(references.SlotWeapon), r.out())
}

type playtestCheck struct {
	materials [references.NumberOfMaterials]uint16
	pieces    []references.Position
	chest     game_state.StorageChest
	harvested []references.Position
	food      uint16
	weapon    references.Equipment
	date      datetime.UltimaDate
	position  references.Position
}

func (r *playtestRig) check() playtestCheck {
	deed := r.activeDeed()
	var c playtestCheck
	for index := range c.materials {
		c.materials[index] = r.state().PartyState.Inventory.Materials.Get(references.Material(index))
	}
	c.pieces = deed.PlacementPositions()
	c.chest, _ = r.state().StorageChestContents(references.Position{X: 15, Y: 12})
	c.harvested = deed.HarvestedNodePositions()
	c.food = r.state().PartyState.Inventory.Provisions.Food.Get()
	c.weapon = r.state().PartyState.GetCharacter(0).GetEquippedItem(references.SlotWeapon)
	c.date = r.state().DateTime
	c.position = r.pos()

	return c
}

// saveAndReload saves through the player save repository, loads the slot into a new scene as the
// Load menu does, and compares.
func (r *playtestRig) saveAndReload() {
	r.t.Helper()
	before := r.check()
	for range 600 {
		if r.scene.dialogStack.Count() == 0 {
			break
		}
		r.pump(1)
	}
	if r.scene.dialogStack.Count() > 0 {
		r.t.Logf("dialog still open before save: %T", r.scene.dialogStack.PeekTopModalDialog())
	}
	r.scene.setPlaythroughMetadata(playthroughMetadata{id: testPlaythroughID})
	record, err := r.game.savePlayerSaveSlot(player_save.Slot01, r.revision, "deed playtest")
	require.NoError(r.t, err, "saving on the deed map")
	r.revision = record.Revision
	require.NoError(r.t, r.game.loadPlayerSaveSlot(player_save.Slot01, record.Revision), "loading the deed save")
	scene, ok := r.game.currentScene.(*GameScene)
	require.True(r.t, ok)
	require.NotSame(r.t, r.scene, scene)
	r.scene = scene
	scene.transientCommandInput.focused = func() bool { return true }
	scene.startTime = time.Now()
	scene.keyboard = input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return r.held[key] })
	r.state().DebugOptions.MonsterGen = false
	r.state().SuppressDeedRaidsForTest(true)
	r.pump(20)
	r.settle()
	require.True(r.t, r.state().IsOnDeedMap(), "reloaded on the deed map")
	after := r.check()
	require.Equal(r.t, before, after, "the deed survives save and load")
	r.t.Logf("reload ok: %d pieces, chest %+v", len(after.pieces), after.chest.Materials)
	r.capture("p24-reloaded")
}

// leaveAndRegrow leaves the deed, waits on the overworld across a Trammel phase change and
// comes back to see the nodes regrow.
func (r *playtestRig) leaveAndRegrow(site references.Position) {
	r.t.Helper()
	before := r.state().DeedResourceNodes(r.activeDeed().ID)
	depleted := 0
	for _, node := range before {
		if node.Depleted {
			depleted++
		}
	}
	r.leaveDeed()
	require.Equal(r.t, site, r.pos())
	r.capture("p25-left")
	r.waitForTrammel()
	r.waitForTrammel()
	r.enterDeed()
	after := r.state().DeedResourceNodes(r.activeDeed().ID)
	depletedAfter := 0
	for _, node := range after {
		if node.Depleted {
			depletedAfter++
		}
	}
	r.t.Logf("depleted nodes: %d before leaving, %d after two Trammel changes", depleted, depletedAfter)
	require.Less(r.t, depletedAfter, depleted, "nodes regrow")
	r.state().MapState.PlayerLocation.Position = references.DeedMapCentre
	r.capture("p26-regrown")

	// Save with the built door open, then during a raid.
	require.True(r.t, r.walkTo([]references.Position{{X: 14, Y: 15}}, 32))
	r.press(ebiten.KeyO)
	r.press(ebiten.KeyUp)
	r.settle()
	r.t.Logf("door: %q", r.out())
	r.saveAndReload()
	r.capture("p27-reloaded-door-open")
	r.state().SpawnDeedRaidersForTest(references.MonsterOrc, []references.Position{{X: 15, Y: 29}, {X: 3, Y: 22}})
	r.t.Logf("raiders before save: %d", r.state().DeedRaiderCount())
	r.saveAndReload()
	r.t.Logf("raiders after reload: %d", r.state().DeedRaiderCount())
	r.capture("p28-reloaded-after-raid")

	// Save on the overworld beside the hut.
	r.leaveDeed()
	deeds := len(r.state().Deeds())
	for range 600 {
		if r.scene.dialogStack.Count() == 0 {
			break
		}
		r.pump(1)
	}
	record, err := r.game.savePlayerSaveSlot(player_save.Slot01, r.revision, "overworld")
	require.NoError(r.t, err)
	require.NoError(r.t, r.game.loadPlayerSaveSlot(player_save.Slot01, record.Revision))
	r.scene = r.game.currentScene.(*GameScene)
	r.scene.transientCommandInput.focused = func() bool { return true }
	r.scene.startTime = time.Now()
	r.scene.keyboard = input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return r.held[key] })
	r.pump(10)
	require.Equal(r.t, deeds, len(r.state().Deeds()))
	r.t.Logf("overworld reload ok at %v", r.pos())

	// Trammel phase changes over two months, for the respawn pace.
	changes := []int{}
	start := r.state().DateTime.GetTotalDays()
	for day := start; day < start+56; day++ {
		if datetime.TrammelPhaseChanges(day+1) != datetime.TrammelPhaseChanges(day) {
			changes = append(changes, day+1-start)
		}
	}
	r.t.Logf("TRAMMEL: phase changes on days %v after today", changes)
}

func (r *playtestRig) partyHP() int {
	total := 0
	for index := range r.state().PartyState.GetActivePartyCount() {
		total += int(r.state().PartyState.GetCharacter(index).GetCurrentHp())
	}

	return total
}

// cancelPaths opens each crafting command and backs out with Esc, checking no time passes.
func (r *playtestRig) cancelPaths() {
	r.t.Helper()
	calls := r.state().GetRNG().Calls()
	r.pump(60)
	r.t.Logf("idle: 60 frames on the deed map draw %d RNG", r.state().GetRNG().Calls()-calls)
	for _, sequence := range []struct {
		name string
		keys [][]ebiten.Key
	}{
		{"gather", [][]ebiten.Key{{ebiten.KeyAltLeft, ebiten.KeyG}, {ebiten.KeyEscape}}},
		{"craft", [][]ebiten.Key{{ebiten.KeyAltLeft, ebiten.KeyC}, {ebiten.KeyEscape}}},
		{"craft-category", [][]ebiten.Key{{ebiten.KeyAltLeft, ebiten.KeyC}, {ebiten.KeyEnter}, {ebiten.KeyEscape}, {ebiten.KeyEscape}}},
		{"build", [][]ebiten.Key{{ebiten.KeyAltLeft, ebiten.KeyB}, {ebiten.KeyEscape}}},
		{"demolish", [][]ebiten.Key{{ebiten.KeyAltLeft, ebiten.KeyB}, {ebiten.KeyEnter}, {ebiten.KeyEscape}}},
		{"ctrl-alt-g", [][]ebiten.Key{{ebiten.KeyAltLeft, ebiten.KeyControlLeft, ebiten.KeyG}}},
	} {
		before, calls := r.minutes(), r.state().GetRNG().Calls()
		r.scene.output.Clear()
		for _, keys := range sequence.keys {
			r.press(keys...)
		}
		r.settle()
		r.t.Logf("cancel %s: %d minutes, %d RNG, dialogs %d, %q", sequence.name, r.minutes()-before,
			r.state().GetRNG().Calls()-calls, r.scene.dialogStack.Count(), r.out())
		if r.scene.dialogStack.Count() > 0 {
			r.capture("zz-cancel-" + sequence.name)
			for r.scene.dialogStack.Count() > 0 {
				r.press(ebiten.KeyEscape)
			}
		}
	}
}

// demolishAndRebuild demolishes the wall at 16,13 from outside and builds it again.
func (r *playtestRig) demolishAndRebuild() {
	r.t.Helper()
	inventory := &r.state().PartyState.Inventory
	stand := references.Position{X: 17, Y: 13}
	require.True(r.t, r.walkTo([]references.Position{stand}, 32))
	walls, cut := inventory.Pieces.Get(references.PieceStoneWall), r.material(references.CutStone)
	before := r.minutes()
	r.scene.output.Clear()
	r.press(ebiten.KeyAltLeft, ebiten.KeyB)
	r.menuChoose(r.menuRow("Demolish"))
	r.press(ebiten.KeyLeft)
	r.settle()
	r.t.Logf("demolish: %q, %d min, walls %d -> %d, cut stone %d -> %d", r.out(), r.minutes()-before, walls,
		inventory.Pieces.Get(references.PieceStoneWall), cut, r.material(references.CutStone))
	r.capture("p23d-demolished")
	r.build(references.PieceStoneWall, references.Position{X: 16, Y: 13}, stand)
}

// sellPrices logs what an Arms shop pays the Avatar for crafted goods, against their inputs.
func (r *playtestRig) sellPrices() {
	intelligence := int(r.state().PartyState.GetCharacter(0).Intelligence)
	for _, item := range []references.Equipment{references.Dagger, references.Bow, references.Crossbow, references.LeatherArmour,
		references.SmallShield, references.LargeShield, references.LongSword, references.ChainMail, references.PlateMail, references.Halberd} {
		r.t.Logf("SELL %s: base %d, Avatar (Int %d) gets %d", item.String(), item.GetBasePrice(), intelligence,
			game_state.CalculateArmsSellPrice(item.GetBasePrice(), intelligence))
	}
}
