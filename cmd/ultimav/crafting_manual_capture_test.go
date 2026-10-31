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
	rig.chord(ebiten.KeyG)
	rig.capture("08-gather-placeholder")
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
