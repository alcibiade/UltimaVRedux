//go:build craftingmanual

package main

// Captures screenshots of the optional crafting system for manual review. It drives a real
// GameScene through the production input handlers and writes PNG frames to the
// directory named by CRAFTING_MANUAL_DIR:
//
//	CRAFTING_MANUAL_DIR=/abs/dir go test -tags craftingmanual -run TestCraftingManualCapture ./cmd/ultimav/

import (
	"fmt"
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
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

type craftingManualRig struct {
	t      *testing.T
	scene  *GameScene
	held   map[ebiten.Key]bool
	outDir string
	width  int
	height int
}

func newCraftingManualRig(t *testing.T) *craftingManualRig {
	t.Helper()
	outDir := os.Getenv("CRAFTING_MANUAL_DIR")
	require.NotEmpty(t, outDir, "set CRAFTING_MANUAL_DIR to an absolute output directory")
	require.NoError(t, os.MkdirAll(outDir, 0o755))

	state, _ := game_state.NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(114, 60).WithSystemCallbacks().WithRandomSeed(1988).Build()
	state.DebugOptions.MonsterGen = false
	state.CurrentNPCAIController.GetNpcs().ClearAllMapUnits()
	state.PartyState.Inventory.Gold.Set(450)
	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)

	configuration := snapshotTestGameConfig(state.GameConfig.SavedConfigData.DataFilePath)
	configuration.RawDataOvl, err = os.ReadFile(filepath.Join(configuration.SavedConfigData.DataFilePath, "DATA.OVL"))
	require.NoError(t, err)
	configuration.DisplayConfiguration = &config.DisplayConfiguration{}
	configuration.GameplayConfiguration = &config.GameplayConfiguration{CraftingEnabled: true}
	scene, err := NewGameSceneFromSaveSnapshot(configuration, snapshot)
	require.NoError(t, err)
	t.Cleanup(scene.Dispose)
	scene.transientCommandInput.focused = func() bool { return true }
	scene.startTime = time.Now() // Starts the presentation clock that timed workflows wait on.
	scene.gameState.DebugOptions.MonsterGen = false
	scene.gameState.CurrentNPCAIController.GetNpcs().ClearAllMapUnits()
	// Raids come only where the rig forces one (captureRaid), never at random mid-capture.
	scene.gameState.SuppressDeedRaidsForTest(true)

	rig := &craftingManualRig{t: t, scene: scene, held: map[ebiten.Key]bool{}, outDir: outDir, width: 1280, height: 720}
	scene.keyboard = input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return rig.held[key] })

	manager := display.GetManager()
	saved := *manager
	t.Cleanup(func() { *manager = saved })
	manager.PublishLogicalSize(rig.width, rig.height)

	return rig
}

func (r *craftingManualRig) state() *game_state.GameState { return r.scene.gameState }

func (r *craftingManualRig) noon() {
	r.state().DateTime = datetime.UltimaDate{Year: 139, Month: 5, Day: 12, Hour: 11, Minute: 30}
}

func (r *craftingManualRig) moveTo(x, y int) {
	r.state().MapState.PlayerLocation.Position = references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
}

// press holds keys for one real scene frame, then releases them for two, as a player would.
func (r *craftingManualRig) press(keys ...ebiten.Key) {
	r.t.Helper()
	clear(r.held)
	for _, key := range keys {
		r.held[key] = true
	}
	r.pump(1)
	clear(r.held)
	r.pump(2)
}

// chord presses Alt with a letter.
func (r *craftingManualRig) chord(key ebiten.Key) { r.press(ebiten.KeyAltLeft, key) }

// command presses a plain command letter.
func (r *craftingManualRig) command(key ebiten.Key) { r.press(key) }

// direction answers an open direction prompt.
func (r *craftingManualRig) direction(key ebiten.Key) { r.press(key) }

// menuKey presses a key in the top modal dialog.
func (r *craftingManualRig) menuKey(key ebiten.Key) { r.press(key) }

// menuChoose moves the highlight down n rows and activates that row.
func (r *craftingManualRig) menuChoose(rows int) {
	for range rows {
		r.menuKey(ebiten.KeyDown)
	}
	r.menuKey(ebiten.KeyEnter)
}

func (r *craftingManualRig) topMenu() *widgets.ButtonListModal {
	menu, ok := r.scene.dialogStack.PeekTopModalDialog().(*widgets.ButtonListModal)
	require.True(r.t, ok, "top dialog is not a menu")

	return menu
}

// menuRow returns the index of the first menu row starting with prefix.
func (r *craftingManualRig) menuRow(prefix string) int {
	for index, label := range r.topMenu().ButtonTexts() {
		if strings.HasPrefix(label, prefix) {
			return index
		}
	}
	r.t.Fatalf("menu has no row %q: %v", prefix, r.topMenu().ButtonTexts())

	return -1
}

// pump runs scene frames at the real frame rate, so timed workflows and the camera advance.
func (r *craftingManualRig) pump(frames int) {
	for range frames {
		require.NoError(r.t, r.scene.Update(nil))
		time.Sleep(16 * time.Millisecond)
	}
}

func (r *craftingManualRig) typeText(text string) {
	for _, character := range strings.ToLower(text) {
		var key ebiten.Key
		switch {
		case character == ' ':
			key = ebiten.KeySpace
		case character >= 'a' && character <= 'z':
			key = ebiten.KeyA + ebiten.Key(character-'a')
		default:
			r.t.Fatalf("cannot type %q", character)
		}
		r.press(key)
	}
}

func (r *craftingManualRig) capture(name string) {
	r.t.Helper()
	r.pump(30)
	r.scene.clearProvisionFeedback()
	r.capturePlain(name)
}

// captureWithFeedback keeps the provision feedback banner, for a frame that is about a cost.
func (r *craftingManualRig) captureWithFeedback(name string) {
	r.t.Helper()
	r.pump(30)
	r.capturePlain(name)
}

func (r *craftingManualRig) capturePlain(name string) {
	r.t.Helper()
	screen := newPresentationRaster(r.width, r.height)
	defer screen.Deallocate()
	r.scene.Draw(screen)
	file, err := os.Create(filepath.Join(r.outDir, name+".png"))
	require.NoError(r.t, err)
	// The window behind the scene is black; transparent panels must stay black in the file.
	opaque := image.NewRGBA(screen.Bounds())
	draw.Draw(opaque, opaque.Bounds(), image.Black, image.Point{}, draw.Src)
	draw.Draw(opaque, opaque.Bounds(), screen, image.Point{}, draw.Over)
	require.NoError(r.t, png.Encode(file, opaque))
	require.NoError(r.t, file.Close())
	r.t.Log("captured ", name)
}

func (r *craftingManualRig) tile(position references.Position) indexes.SpriteIndex {
	return r.state().MapState.LayeredMaps.GetLayeredMap(references.LargeMapType, 0).
		SpriteIndexByLayer(map_state.MapLayer, position)
}

// findDeedSquare returns the overworld square nearest to near whose terrain and four
// neighbours match the sprite, so a deed of that terrain is legal there.
func (r *craftingManualRig) findDeedSquare(near references.Position, sprite indexes.SpriteIndex) references.Position {
	r.t.Helper()
	for radius := 0; radius < 128; radius++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				candidate := *(&references.Position{X: near.X + references.Coordinate(dx), Y: near.Y + references.Coordinate(dy)}).
					GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
				if r.tile(candidate) != sprite {
					continue
				}
				clear := true
				for _, step := range []references.Position{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
					neighbour := *(&references.Position{X: candidate.X + step.X, Y: candidate.Y + step.Y}).
						GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
					clear = clear && r.tile(neighbour) == sprite
				}
				r.moveTo(int(candidate.X), int(candidate.Y))
				if clear && r.state().ProposeDeed().Err == nil {
					return candidate
				}
			}
		}
	}
	r.t.Fatalf("no deed square of sprite %d near %v", sprite, near)

	return references.Position{}
}

// stakeHere stakes a deed on the party's square through Alt+B, the menu and the name prompt.
func (r *craftingManualRig) stakeHere(name string) {
	r.t.Helper()
	r.chord(ebiten.KeyB)
	r.menuChoose(r.menuRow("Stake"))
	r.pump(5)
	r.typeText(name)
	r.press(ebiten.KeyEnter)
	r.pump(10)
	_, owned := r.state().DeedAt(r.state().MapState.PlayerLocation.Position)
	require.True(r.t, owned, "deed %q was not staked", name)
}

// enterHere presses Enter on the hut and waits for the deed map.
func (r *craftingManualRig) enterHere() {
	r.t.Helper()
	r.pump(30)
	r.command(ebiten.KeyE)
	for range 200 {
		if r.state().IsOnDeedMap() && !r.state().EventQueueManager.HasActiveWorkflows() {
			return
		}
		r.pump(1)
	}
	r.capturePlain("zz-enter-failure")
	r.t.Fatal("never reached the deed map")
}

func TestCraftingManualCapture(t *testing.T) {
	rig := newCraftingManualRig(t)
	rig.noon()
	rig.pump(2)

	// The option in Configure, Gameplay tab.
	settings, _ := initializeDialogConfigFile(t, "crafting_manual")
	settings.GameplayConfiguration.CraftingEnabled = true
	placement := gameScreenPercents
	dialog := newConfigurationDialogForTab(&placement, newPresentationTestKeyboard(), func() {}, settings, nil,
		audioSettingsGameplayTab)
	for range gameplayCraftingEnabledIndex {
		pressAudioSettingsKey(dialog, ebiten.KeyDown)
	}
	rig.scene.dialogStack.PushModalDialog(dialog)
	rig.capture("01-configure-option")
	rig.scene.dialogStack.MustRemoveWidgetInstance(dialog)

	// Staking a meadow deed: Alt+B offers it, the prompt names it, a hut marks it.
	meadow := rig.findDeedSquare(references.Position{X: 114, Y: 60}, indexes.Grass)
	rig.scene.output.Clear()
	rig.chord(ebiten.KeyB)
	rig.capture("02-build-menu")
	rig.menuChoose(rig.menuRow("Stake"))
	rig.pump(5)
	rig.typeText("Greenhollow")
	rig.capture("03-name-prompt")
	rig.press(ebiten.KeyEnter)
	rig.noon()
	require.Equal(t, indexes.SpriteIndex(indexes.Hut), rig.tile(meadow))
	// One step west shows the hut, and Alt+B there is refused: deeds never touch.
	rig.moveTo(int(meadow.X)-1, int(meadow.Y))
	rig.capture("04-staked-hut")
	rig.scene.output.Clear()
	rig.chord(ebiten.KeyB)
	rig.capture("05-too-close")
	rig.moveTo(int(meadow.X), int(meadow.Y))

	// Entering the deed map, then walking off the south edge back to the hut.
	rig.scene.output.Clear()
	rig.enterHere()
	rig.noon()
	rig.capture("06-meadow-deed-entry")
	rig.state().MapState.PlayerLocation.Position = references.DeedMapCentre
	rig.capture("07-meadow-deed-centre")
	rig.captureGathering()
	rig.captureBuilding()
	rig.captureArmsAndAlchemy()
	rig.captureRaid()
	rig.state().MapState.PlayerLocation.Position = references.Position{X: references.DeedMapEntry.X, Y: 31}
	rig.command(ebiten.KeyDown)
	rig.pump(10)
	rig.capture("09-leave-prompt")
	rig.press(ebiten.KeyY)
	rig.press(ebiten.KeyEnter)
	for range 200 {
		if !rig.state().EventQueueManager.HasActiveWorkflows() {
			break
		}
		rig.pump(1)
	}
	rig.noon()
	rig.capture("10-back-at-hut")
	require.Equal(t, meadow, rig.state().MapState.PlayerLocation.Position)
	require.Equal(t, references.BritanniaUnderworld, rig.state().MapState.PlayerLocation.Location)

	// Every other terrain, staked and entered.
	rig.state().PartyState.Inventory.Gold.Set(2000)
	for index, terrain := range []struct {
		name   string
		sprite indexes.SpriteIndex
		near   references.Position
	}{
		{"Oakwood", indexes.Forest3, references.Position{X: 100, Y: 40}},
		{"Stonerise", indexes.Hills1, references.Position{X: 130, Y: 100}},
		{"Sandreach", indexes.Desert1, references.Position{X: 160, Y: 150}},
	} {
		square := rig.findDeedSquare(terrain.near, terrain.sprite)
		rig.stakeHere(terrain.name)
		rig.scene.output.Clear()
		rig.enterHere()
		rig.noon()
		rig.state().MapState.PlayerLocation.Position = references.DeedMapCentre
		rig.capture(fmt.Sprintf("%02d-%s", 11+index, strings.ToLower(terrain.name)))
		rig.state().TransitionToLocation(references.BritanniaUnderworld, 0, square, game_state.TransitionFullLocation, false)
	}

	// The deeds are pinned on the Atlas.
	rig.moveTo(int(meadow.X), int(meadow.Y))
	rig.scene.openAtlas(NewGame(rig.scene))
	for range 9 { // All, Settlements, ... Vessels, then My Notes.
		rig.press(ebiten.KeyTab)
	}
	rig.capture("14-atlas")
	rig.scene.closeAtlasSession(false)

	// Command help lists the chords.
	rig.scene.showCommandHelp()
	for range 6 {
		rig.press(ebiten.KeyPageDown)
	}
	rig.capture("15-command-help")
}

// deedNode returns the first node of the active deed matching kind and sprite (0: any sprite).
func (r *craftingManualRig) deedNode(kind references.DeedResourceKind, sprite indexes.SpriteIndex) game_state.DeedResourceNodeState {
	r.t.Helper()
	deed, ok := r.state().ActiveDeed()
	require.True(r.t, ok)
	for _, node := range r.state().DeedResourceNodes(deed.ID) {
		if node.Kind == kind && (sprite == 0 || node.Sprite == sprite) && !node.Depleted {
			return node
		}
	}
	r.t.Fatalf("no fresh %s node", kind.Name())

	return game_state.DeedResourceNodeState{}
}

// standSouthOf puts the party just south of a node, facing it with Up.
func (r *craftingManualRig) standSouthOf(node game_state.DeedResourceNodeState) {
	r.moveTo(int(node.Position.X), int(node.Position.Y)+1)
	r.pump(5)
}

// gather presses Alt+G and Up.
func (r *craftingManualRig) gather() {
	r.chord(ebiten.KeyG)
	r.direction(ebiten.KeyUp)
}

// look presses L and Up.
func (r *craftingManualRig) look() {
	r.command(ebiten.KeyL)
	r.direction(ebiten.KeyUp)
}

// captureGathering covers ref:CRAFT-0005, ref:CRAFT-0006 and ref:CRAFT-0007 on the meadow deed: bare-handed gathering, the
// recipe menu, crafting tools, felling a tree, mining ore, picking forage and a reagent, Look
// on nodes, and a Trammel respawn.
func (r *craftingManualRig) captureGathering() {
	r.t.Helper()
	inventory := &r.state().PartyState.Inventory
	materials := func(material references.Material) uint16 { return inventory.Materials.Get(material) }

	// Bare hands on trees and rocks until the Axe is affordable.
	tree := r.deedNode(references.DeedResourceWood, indexes.Forest3)
	r.standSouthOf(tree)
	r.scene.output.Clear()
	r.chord(ebiten.KeyG)
	r.capture("08a-gather-prompt")
	r.direction(ebiten.KeyUp)
	for materials(references.Wood) < 2 {
		node := r.deedNode(references.DeedResourceWood, indexes.Forest3)
		r.standSouthOf(node)
		r.gather()
	}
	r.noon()
	r.capture("08b-gather-wood-bare-hands")
	for materials(references.Stone) < 1 {
		r.standSouthOf(r.deedNode(references.DeedResourceStone, 0))
		r.gather()
	}

	// The recipe menu, then an Axe.
	r.scene.output.Clear()
	r.chord(ebiten.KeyC)
	r.capture("08c-craft-menu")
	r.menuChoose(r.menuRow("Tools"))
	r.capture("08c2-craft-tools")
	r.menuChoose(r.menuRow("Axe"))
	r.noon()
	r.captureWithFeedback("08d-crafted-axe")
	require.Equal(r.t, uint16(1), inventory.Tools.Get(references.Axe))

	// Chop a tree until it is felled, then Look at the stump of brush it leaves.
	tree = r.deedNode(references.DeedResourceWood, indexes.Forest3)
	r.standSouthOf(tree)
	r.scene.output.Clear()
	r.look()
	r.capture("08e-look-tree")
	r.scene.output.Clear()
	for range references.DeedNodeRuleFor(references.DeedResourceWood).Capacity {
		r.gather()
	}
	r.noon()
	r.capture("08f-tree-felled")
	r.scene.output.Clear()
	r.look()
	r.capture("08g-look-felled-tree")

	// A Pick from gathered materials, then ore.
	inventory.Materials.Set(references.Wood, 2)
	inventory.Materials.Set(references.Stone, 2)
	r.scene.output.Clear()
	r.craft("Tools", "Pick")
	ore := r.deedNode(references.DeedResourceOre, 0)
	r.standSouthOf(ore)
	r.scene.output.Clear()
	r.look()
	r.gather()
	r.noon()
	r.capture("08h-mine-ore")

	// Forage and a reagent, by hand.
	r.standSouthOf(r.deedNode(references.DeedResourceForage, 0))
	r.scene.output.Clear()
	r.look()
	r.gather()
	r.noon()
	r.capture("08i-pick-forage")
	patch := r.deedNode(references.DeedResourceReagent, 0)
	r.standSouthOf(patch)
	r.scene.output.Clear()
	r.look()
	r.gather()
	r.noon()
	r.capture("08j-pick-reagent")

	// Respawn: the eve of the next Trammel phase change, then Pass across midnight.
	r.standSouthOf(game_state.DeedResourceNodeState{DeedResourceNode: references.DeedResourceNode{Position: tree.Position}})
	eve := r.state().DateTime
	for datetime.TrammelPhaseChanges(eve.GetTotalDays()+1) == datetime.TrammelPhaseChanges(eve.GetTotalDays()) {
		eve.Day++
	}
	eve.Hour, eve.Minute = 23, 58
	r.state().DateTime = eve
	r.settle() // Night raises the moongates; their stage workflow owns input until it ends.
	r.scene.output.Clear()
	r.capture("08k-before-phase-change")
	for range 3 {
		r.command(ebiten.KeySpace)
	}
	r.scene.output.Clear()
	r.look()
	r.capture("08l-after-phase-change")
	r.noon()
	r.settle()
}

// settle runs frames until no gameplay workflow is active.
func (r *craftingManualRig) settle() {
	for range 600 {
		if !r.state().EventQueueManager.HasActiveWorkflows() {
			return
		}
		r.pump(1)
	}
	r.t.Fatal("workflows never settled")
}

// craft presses Alt+C and picks a category row, then a recipe row, by their labels.
func (r *craftingManualRig) craft(category, recipe string) {
	r.t.Helper()
	r.chord(ebiten.KeyC)
	r.menuChoose(r.menuRow(category))
	r.menuChoose(r.menuRow(recipe))
}

// buildPieceAt places a carried piece on a square through Alt+B, its category, the piece and a
// direction, standing on the first free neighbour of the square.
func (r *craftingManualRig) buildPieceAt(id references.BuildPieceID, x, y int) {
	r.t.Helper()
	piece := id.Piece()
	target := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
	if r.state().PartyState.Inventory.Pieces.Get(id) == 0 {
		r.state().PartyState.Inventory.Pieces.Set(id, 1)
	}
	r.standNextTo(target)
	r.chord(ebiten.KeyB)
	r.menuChoose(r.menuRow(piece.Category.Name()))
	r.menuChoose(r.menuRow(piece.Name + " ("))
	r.direction(r.directionKeyTo(target))
	deed, _ := r.state().ActiveDeed()
	tile, _ := r.state().DeedTileAt(deed.ID, target)
	require.Equal(r.t, piece.Sprite, tile, "%s was not built at %v", piece.Name, target)
}

// standNextTo moves the party to a walkable neighbour of a square: south, north, west, then east.
func (r *craftingManualRig) standNextTo(target references.Position) {
	r.t.Helper()
	for _, step := range []references.Position{{Y: 1}, {Y: -1}, {X: -1}, {X: 1}} {
		stand := references.Position{X: target.X + step.X, Y: target.Y + step.Y}
		if r.state().IsPassable(&stand) {
			r.moveTo(int(stand.X), int(stand.Y))
			r.pump(2)

			return
		}
	}
	r.t.Fatalf("no free square next to %v", target)
}

func (r *craftingManualRig) directionKeyTo(target references.Position) ebiten.Key {
	party := r.state().MapState.PlayerLocation.Position
	switch {
	case target.Y < party.Y:
		return ebiten.KeyUp
	case target.Y > party.Y:
		return ebiten.KeyDown
	case target.X < party.X:
		return ebiten.KeyLeft
	default:
		return ebiten.KeyRight
	}
}

// captureBuilding covers ref:CRAFT-0008 to ref:CRAFT-0012 on the meadow deed: crafting stations, making
// planks, the Build menu, a small Ultima V house (stone walls, windows, a door, a wood floor, a
// bed, a table and chair, a fireplace and a storage chest), storing goods, resting in the bed,
// cooking at the fireplace and demolishing.
func (r *craftingManualRig) captureBuilding() {
	r.t.Helper()
	r.noon()
	inventory := &r.state().PartyState.Inventory
	inventory.Materials.Set(references.Wood, 120)
	inventory.Materials.Set(references.Stone, 120)
	inventory.Materials.Set(references.Ore, 12)
	inventory.Materials.Set(references.Forage, 6)

	// Two stone-floored workshops south of the yard, as a smithy has in a towne.
	for _, x := range []int{10, 11, 12, 13, 16, 17, 18, 19} {
		r.buildPieceAt(references.PieceStoneFloor, x, 18)
	}

	// Bare hands make the first stations, then the stations make the rest.
	r.moveTo(12, 17)
	r.scene.output.Clear()
	r.chord(ebiten.KeyC)
	r.capture("16a-craft-categories")
	r.menuChoose(r.menuRow("Stations"))
	r.capture("16b-craft-stations")
	r.menuChoose(r.menuRow("Workbench"))
	r.craft("Stations", "Sawmill")
	r.buildPieceAt(references.PieceWorkbench, 11, 18)
	r.buildPieceAt(references.PieceSawmill, 12, 18)
	r.moveTo(12, 17)
	r.scene.output.Clear()
	r.craft("Refined materials", "Planks")
	r.craft("Refined materials", "Planks")
	r.noon()
	r.captureWithFeedback("16c-making-planks")
	for range 12 {
		r.craft("Refined materials", "Cut stone")
	}
	r.craft("Stations", "Forge")
	r.craft("Stations", "Fireplace")
	r.buildPieceAt(references.PieceForge, 17, 18)
	r.moveTo(17, 17)
	r.craft("Refined materials", "Ingots")
	r.moveTo(12, 17)
	r.craft("Stations", "Alchemy table")
	r.buildPieceAt(references.PieceAlchemyTable, 18, 18)
	r.moveTo(15, 17)
	r.scene.output.Clear()
	r.noon()
	r.capture("16d-crafting-stations")
	r.moveTo(17, 19)
	r.scene.output.Clear()
	r.look()
	r.capture("16e-look-forge")

	// The Build menu, a category and the direction prompt.
	r.moveTo(12, 17)
	r.craft("Walls and doors", "Stone wall")
	r.moveTo(13, 16)
	r.scene.output.Clear()
	r.chord(ebiten.KeyB)
	r.capture("17a-build-menu")
	r.menuChoose(r.menuRow("Walls and doors"))
	r.capture("17b-build-walls")
	r.menuChoose(r.menuRow("Stone wall ("))
	r.capture("17c-build-direction")
	r.direction(ebiten.KeyUp)
	r.noon()
	r.capture("17d-first-wall")

	// The rest of the house: walls with a fireplace in the north wall, windows and a door in
	// the south wall, a wood floor, a bed, a table and chair, a bookcase and a storage chest.
	for x := 11; x <= 17; x++ {
		piece := references.PieceStoneWall
		if x == 14 {
			piece = references.PieceFireplace
		}
		r.buildPieceAt(piece, x, 11)
	}
	for y := 12; y <= 14; y++ {
		r.buildPieceAt(references.PieceStoneWall, 11, y)
		if y == 13 {
			r.buildPieceAt(references.PieceWindow, 17, y)
		} else {
			r.buildPieceAt(references.PieceStoneWall, 17, y)
		}
	}
	for y := 12; y <= 14; y++ {
		for x := 12; x <= 16; x++ {
			r.buildPieceAt(references.PieceWoodFloor, x, y)
		}
	}
	r.buildPieceAt(references.PieceBedHead, 12, 12)
	r.buildPieceAt(references.PieceBedFoot, 13, 12)
	r.buildPieceAt(references.PieceStorageChest, 16, 12)
	r.buildPieceAt(references.PieceBookcase, 12, 14)
	r.buildPieceAt(references.PieceTableWestEnd, 15, 14)
	r.buildPieceAt(references.PieceTableEastEnd, 16, 14)
	r.buildPieceAt(references.PieceChairBack, 15, 13)
	for x := 11; x <= 17; x++ {
		switch x {
		case 12, 16:
			r.buildPieceAt(references.PieceWindow, x, 15)
		case 14:
			r.buildPieceAt(references.PieceDoor, x, 15)
		default:
			if x != 13 { // 13,15 holds the first wall already.
				r.buildPieceAt(references.PieceStoneWall, x, 15)
			}
		}
	}
	r.moveTo(14, 17)
	r.scene.output.Clear()
	r.noon()
	r.capture("17e-house-outside")
	r.moveTo(14, 16)
	r.command(ebiten.KeyO)
	r.direction(ebiten.KeyUp)
	r.capture("17f-door-open")
	r.moveTo(14, 13)
	r.scene.output.Clear()
	r.noon()
	r.capture("17g-house-inside")

	// The storage chest: Open shows it; store materials and gold.
	r.moveTo(15, 12)
	r.scene.output.Clear()
	r.command(ebiten.KeyO)
	r.direction(ebiten.KeyRight)
	r.capture("18a-chest")
	r.menuChoose(r.menuRow("Store"))
	r.capture("18b-chest-store")
	r.menuChoose(r.menuRow("Materials"))
	r.menuChoose(r.menuRow("Store"))
	r.menuChoose(r.menuRow("Gold"))
	r.capture("18c-chest-stored")
	r.menuKey(ebiten.KeyEscape)
	r.settle()

	// Cooking at the fireplace: Forage and Wood make Food.
	r.moveTo(14, 12)
	// The chest keeps what was stored; the party brings fresh Forage and Wood.
	inventory.Materials.Set(references.Wood, 2)
	inventory.Materials.Set(references.Forage, 4)
	r.scene.output.Clear()
	r.chord(ebiten.KeyC)
	r.menuChoose(r.menuRow("Provisions"))
	r.capture("19a-cook-menu")
	r.menuChoose(r.menuRow("Cook food"))
	r.noon()
	r.captureWithFeedback("19b-cooked-food")

	// Resting in the bed, wounded, in the evening.
	r.moveTo(12, 12)
	r.state().PartyState.GetCharacter(0).SetHp(5)
	r.state().DateTime.Hour, r.state().DateTime.Minute = 19, 0
	r.scene.output.Clear()
	r.capture("20a-in-bed")
	r.command(ebiten.KeyH)
	r.pump(80)
	r.capturePlain("20b-sleeping")
	r.settle()
	r.capture("20c-rested")

	// Demolishing the bookcase returns it to stock.
	r.moveTo(13, 14)
	r.scene.output.Clear()
	r.chord(ebiten.KeyB)
	r.menuChoose(r.menuRow("Demolish"))
	r.capture("21a-demolish-prompt")
	r.direction(ebiten.KeyLeft)
	r.capture("21b-demolished")
	r.noon()
	r.settle()
}

// captureArmsAndAlchemy covers ref:CRAFT-0013 to ref:CRAFT-0019 on the meadow deed, standing between
// the Forge (17,18) and the Alchemy table (18,18) that captureBuilding placed: the weapon and
// armour menus, forging a Long Sword and readying it, the alchemy menus, mixing a potion,
// writing a scroll and refining a reagent.
func (r *craftingManualRig) captureArmsAndAlchemy() {
	r.t.Helper()
	r.noon()
	inventory := &r.state().PartyState.Inventory
	inventory.Materials.Set(references.Ingots, 12)
	inventory.Materials.Set(references.Planks, 6)

	// The Forge's menus; the Workbench is out of reach from here.
	r.moveTo(18, 17)
	r.scene.output.Clear()
	r.chord(ebiten.KeyC)
	r.capture("22a-craft-categories")
	r.menuChoose(r.menuRow("Blades"))
	r.capture("22b-blades")
	r.menuKey(ebiten.KeyEscape)
	r.menuChoose(r.menuRow("Body armour"))
	r.capture("22c-body-armour")
	r.menuKey(ebiten.KeyEscape)
	r.menuChoose(r.menuRow("Helms and shields"))
	r.capture("22d-helms-shields")
	r.menuKey(ebiten.KeyEscape)
	r.menuKey(ebiten.KeyEscape)
	r.settle()

	// Forge a Long Sword and ready it on the Avatar through the Ready dialog. The Avatar first
	// sets down his gear, which stays carried, so the sword fits his Strength.
	avatar := r.state().PartyState.GetCharacter(0)
	for _, slot := range references.EquipmentSlots() {
		avatar.SetEquippedItem(slot, references.NoEquipment)
	}
	inventory.Equipment.Set(references.LongSword, 0)
	r.scene.output.Clear()
	r.craft("Blades", "Long Sword")
	r.noon()
	r.captureWithFeedback("22e-forged-long-sword")
	require.Equal(r.t, uint16(1), inventory.Equipment.Get(references.LongSword))
	r.scene.output.Clear()
	r.command(ebiten.KeyR)
	r.pump(10)
	ready, ok := r.scene.dialogStack.PeekTopModalDialog().(*widgets.ReadyEquipmentDialog)
	require.True(r.t, ok, "R opens the Ready dialog")
	for range 60 {
		if selected, _ := ready.GetSelectedEquipment(); selected == references.LongSword {
			break
		}
		r.press(ebiten.KeyDown)
	}
	r.capture("22f-ready-dialog")
	r.press(ebiten.KeyEnter)
	r.pump(10)
	r.capture("22g-long-sword-readied")
	require.Equal(r.t, references.LongSword, avatar.GetEquippedItem(references.SlotWeapon), "the forged sword is readied")
	r.press(ebiten.KeyEscape)
	r.settle()

	// Alchemy: the menus, a potion, a scroll and a refined reagent.
	for _, reagent := range references.ReagentCatalog() {
		inventory.Reagent.Set(reagent, 0)
	}
	inventory.Reagent.Set(references.Ginseng, 4)
	inventory.Reagent.Set(references.SpiderSilk, 2)
	inventory.Reagent.Set(references.SulfurAsh, 7)
	inventory.Reagent.Set(references.BloodMoss, 2)
	inventory.Potions.Set(references.Yellow, 0)
	inventory.Scrolls.Set(references.ScrollRelHur, 0)
	r.scene.output.Clear()
	r.chord(ebiten.KeyC)
	r.menuChoose(r.menuRow("Potions"))
	r.capture("23a-potions")
	r.menuChoose(r.menuRow("Yellow Potion"))
	r.noon()
	r.captureWithFeedback("23b-mixed-potion")
	require.Equal(r.t, uint16(1), inventory.Potions.Get(references.Yellow))

	r.scene.output.Clear()
	r.chord(ebiten.KeyC)
	r.menuChoose(r.menuRow("Scrolls"))
	r.capture("23c-scrolls")
	r.menuChoose(r.menuRow("Rel Hur"))
	r.noon()
	r.captureWithFeedback("23d-written-scroll")
	require.Equal(r.t, uint16(1), inventory.Scrolls.Get(references.ScrollRelHur))

	r.scene.output.Clear()
	r.chord(ebiten.KeyC)
	r.menuChoose(r.menuRow("Reagent refining"))
	r.capture("23e-refining")
	r.menuChoose(r.menuRow("Black Pearl"))
	r.noon()
	r.captureWithFeedback("23f-refined-black-pearl")
	require.Equal(r.t, uint16(1), inventory.Reagent.Get(references.BlackPearl))

	// Away from the Alchemy table, long rows name only the station.
	r.moveTo(14, 17)
	r.scene.output.Clear()
	r.chord(ebiten.KeyC)
	r.menuChoose(r.menuRow("Scrolls"))
	r.capture("23g-scrolls-away")
	r.menuKey(ebiten.KeyEscape)
	r.menuKey(ebiten.KeyEscape)
	r.noon()
	r.settle()
}

// passTurns presses Space n times, one ordinary turn each.
func (r *craftingManualRig) passTurns(n int) {
	for range n {
		r.command(ebiten.KeySpace)
	}
}

// captureRaid covers ref:CRAFT-0020 and ref:CRAFT-0021 on the meadow deed, around the house of
// captureBuilding: a forced raid of orcs from the south, the raiders working their way around
// the house toward the party north of it, the battle, and the house intact afterwards.
func (r *craftingManualRig) captureRaid() {
	r.t.Helper()
	r.noon()
	deed, ok := r.state().ActiveDeed()
	require.True(r.t, ok)
	builtBefore := deed.PlacementPositions()
	house := func(position references.Position) bool { // Beside the house, which spans 11-17 by 11-15.
		return position.X >= 9 && position.X <= 19 && position.Y >= 11 && position.Y <= 15
	}

	r.moveTo(14, 9)
	r.scene.output.Clear()
	r.state().SpawnDeedRaidersForTest(references.MonsterOrc, []references.Position{{X: 15, Y: 29}, {X: 3, Y: 22}, {X: 28, Y: 24}})
	r.capture("24a-raiders-approach")

	// The orcs close in and must go around the house to reach the party.
	for range 20 {
		r.command(ebiten.KeySpace)
		around := 0
		for _, raider := range r.state().DeedRaiderPositionsForTest() {
			if house(raider) {
				around++
			}
		}
		if around >= 2 || r.state().CombatScene != nil {
			break
		}
	}
	require.Nil(r.t, r.state().CombatScene, "the raid reached the party before it was seen")
	r.noon()
	r.capture("24b-raiders-around-the-house")

	// The first orc to reach the party starts the battle, fought by the whole raid.
	for range 20 {
		if r.state().CombatScene != nil {
			break
		}
		r.command(ebiten.KeySpace)
	}
	require.NotNil(r.t, r.state().CombatScene, "the raid never reached the party")
	r.capture("24c-raid-battle")

	// The battle won, the party is back on the deed with the house as it was.
	r.state().EndCombat(game_state.CombatVictory)
	r.pump(30)
	r.capturePlain("24d-battle-complete")
	r.menuKey(ebiten.KeyEnter)
	r.settle()
	r.noon()
	r.scene.output.Clear()
	r.capture("24e-after-the-raid")
	deed, _ = r.state().ActiveDeed()
	require.Equal(r.t, builtBefore, deed.PlacementPositions(), "the raid left every piece in place")
	require.Zero(r.t, r.state().DeedRaiderCount())
}
