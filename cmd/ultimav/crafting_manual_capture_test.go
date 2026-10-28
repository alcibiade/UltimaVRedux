//go:build craftingmanual

package main

// Captures the screenshots of the crafting and housing user manual. It drives a real
// GameScene through the production input handlers and writes PNG frames to the
// directory named by CRAFTING_MANUAL_DIR:
//
//	CRAFTING_MANUAL_DIR=/abs/dir go test -tags craftingmanual -run TestCraftingManualCapture ./cmd/ultimav/

import (
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

// craftingManualClaimCentre is a meadow with forest to the east and hills and
// mountains to the south-west.
var craftingManualClaimCentre = references.Position{X: 108, Y: 62}

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

func (r *craftingManualRig) tile(x, y int) indexes.SpriteIndex {
	return r.state().MapState.LayeredMaps.GetLayeredMap(references.LargeMapType, 0).
		SpriteIndexByLayer(map_state.MapLayer, references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)})
}

// place builds one piece from the given standing square, through the game rules.
func (r *craftingManualRig) place(pieceID, fromX, fromY int, direction references.Direction) {
	r.t.Helper()
	r.moveTo(fromX, fromY)
	require.True(r.t, r.state().ActionBuildLargeMap(pieceID, direction), "piece %d from %d,%d", pieceID, fromX, fromY)
}

func TestCraftingManualCapture(t *testing.T) {
	rig := newCraftingManualRig(t)
	inventory := &rig.state().PartyState.Inventory
	rig.noon()
	rig.pump(2)

	// 1. The option in Configure, Gameplay tab.
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

	// 2. Gathering bare-handed at the forest edge.
	rig.moveTo(114, 60)
	rig.scene.output.Clear()
	rig.chord(ebiten.KeyG)
	rig.capture("02-gather-prompt")
	rig.direction(ebiten.KeyRight)
	for attempt := 0; inventory.Materials.Get(references.Wood) == 0 && attempt < 12; attempt++ {
		rig.chord(ebiten.KeyG)
		rig.direction(ebiten.KeyRight)
	}
	require.Positive(t, inventory.Materials.Get(references.Wood))
	rig.noon()
	rig.capture("03-gather-bare-hands")

	// 3. Crafting an axe after a while of collecting.
	inventory.AddMaterials(references.MaterialCost{references.Wood: 11, references.Stone: 7})
	rig.scene.output.Clear()
	rig.chord(ebiten.KeyC)
	rig.capture("04-craft-menu")
	rig.menuChoose(0)
	rig.noon()
	rig.capture("05-craft-axe")

	// 4. Felling the forest with the axe.
	rig.scene.output.Clear()
	rig.moveTo(114, 61)
	before := rig.tile(115, 61)
	rig.chord(ebiten.KeyG)
	rig.direction(ebiten.KeyRight)
	rig.chord(ebiten.KeyG)
	rig.direction(ebiten.KeyRight)
	t.Logf("forest tile %d -> %d", before, rig.tile(115, 61))
	rig.noon()
	rig.capture("06-gather-axe")

	// 5. A pick, then stone from the hills.
	rig.chord(ebiten.KeyC)
	rig.menuChoose(1)
	rig.moveTo(106, 64)
	rig.scene.output.Clear()
	rig.chord(ebiten.KeyG)
	rig.direction(ebiten.KeyDown)
	rig.chord(ebiten.KeyG)
	rig.direction(ebiten.KeyDown)
	rig.noon()
	rig.capture("07-gather-stone")

	// 6. Staking the claim.
	inventory.AddMaterials(references.MaterialCost{references.Wood: 150, references.Stone: 120})
	rig.moveTo(int(craftingManualClaimCentre.X), int(craftingManualClaimCentre.Y))
	rig.scene.output.Clear()
	rig.chord(ebiten.KeyB)
	rig.capture("08-build-menu-unclaimed")
	rig.menuChoose(0)
	rig.pump(3)
	_, ok := rig.scene.dialogStack.PeekTopModalDialog().(*widgets.InputBox)
	require.True(t, ok, "the claim asks for a name")
	rig.typeText("hearthmere")
	rig.capture("09-claim-name")
	rig.menuKey(ebiten.KeyEnter)
	for frame := 0; frame < 120; frame++ {
		if _, open := rig.scene.dialogStack.PeekTopModalDialog().(*widgets.InputBox); !open {
			break
		}
		rig.pump(1)
	}
	require.Len(t, rig.state().LandClaims(), 1)
	rig.noon()
	rig.captureWithFeedback("10-claim-staked")

	// 7. Building the house: floors, furniture, walls, then the door.
	const (
		left, right, top, bottom = 106, 112, 56, 61
		woodFloor, stoneWall     = 0, 2
		window                   = 4
		tableLeft, tableMiddle   = 8, 9
		tableRight, chairBack    = 10, 13
		bedHead, bedFoot         = 15, 16
		dresser, bookcase        = 17, 18
		barrel, fireplace, chest = 20, 21, 22
	)
	for y := top + 1; y < bottom; y++ {
		for x := left + 1; x < right; x++ {
			rig.place(woodFloor, x, y+1, references.Up)
		}
	}
	furniture := []struct{ piece, x, y int }{
		{bedHead, 107, 57}, {bedFoot, 108, 57}, {bookcase, 110, 57}, {dresser, 111, 57},
		{fireplace, 111, 58},
		{tableLeft, 107, 59}, {tableMiddle, 108, 59}, {tableRight, 109, 59}, {chest, 111, 59},
		{chairBack, 107, 60}, {chairBack, 108, 60}, {barrel, 111, 60},
	}
	for _, item := range furniture {
		rig.place(item.piece, item.x, item.y+1, references.Up)
	}
	for x := left; x <= right; x++ {
		piece := stoneWall
		if x == 108 || x == 110 {
			piece = window
		}
		rig.place(piece, x, top-1, references.Down)
	}
	for y := top + 1; y < bottom; y++ {
		rig.place(stoneWall, left-1, y, references.Right)
		rig.place(stoneWall, right+1, y, references.Left)
	}
	for x := left; x <= right; x++ {
		if x == 109 {
			continue
		}
		rig.place(stoneWall, x, bottom+1, references.Up)
	}
	// The door goes in through the menus, as a player would do it.
	rig.moveTo(109, 62)
	rig.scene.output.Clear()
	rig.chord(ebiten.KeyB)
	rig.capture("11-build-menu-claimed")
	rig.menuChoose(rig.menuRow("Wall"))
	rig.capture("12-build-walls-menu")
	rig.menuChoose(rig.menuRow("Door"))
	rig.capture("13-build-direction")
	rig.direction(ebiten.KeyUp)
	rig.noon()
	rig.capture("14-house-outside")

	// 8. Inside, with the storage chest.
	rig.command(ebiten.KeyO)
	rig.direction(ebiten.KeyUp)
	rig.moveTo(110, 59)
	rig.scene.output.Clear()
	rig.noon()
	rig.capture("15-house-inside")
	rig.command(ebiten.KeyO)
	rig.direction(ebiten.KeyRight)
	rig.capture("16-storage-chest")
	rig.menuChoose(0)
	rig.capture("17-storage-store")
	rig.menuChoose(rig.menuRow("Materials"))
	rig.capture("18-storage-stored")
	rig.menuKey(ebiten.KeyEscape)

	// 9. Demolishing a piece.
	rig.scene.output.Clear()
	rig.moveTo(109, 58)
	rig.chord(ebiten.KeyB)
	rig.menuChoose(rig.menuRow("Demolish"))
	rig.direction(ebiten.KeyDown)
	rig.noon()
	rig.capture("19-demolish")

	// 10. Help lists the chords at the end of the overworld commands.
	rig.scene.showCommandHelp()
	for range 5 {
		rig.press(ebiten.KeyShiftLeft, ebiten.KeyPageDown)
	}
	rig.capture("20-command-help")
	rig.menuKey(ebiten.KeyEscape)

	// 11. The claim is pinned on the Atlas.
	rig.scene.openAtlas(NewGame(rig.scene))
	rig.press(ebiten.KeyShiftLeft, ebiten.KeyTab)
	rig.capture("21-atlas")
	rig.scene.closeAtlasSession(false)

}
