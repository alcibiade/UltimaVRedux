package widgets

import (
	"image"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/profiling"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/hajimehoshi/ebiten/v2"
)

// readyContentKey captures the current presentation, not transaction authority.
// Candidate reviews have an explicit revision; the remaining fields are the
// small live values read by the header, slots, ammunition and controls.
type readyContentKey struct {
	revision                    uint64
	character, partyCount       int
	selection, start, end, page int
	warning                     string
	hand                        references.EquipmentSlot
	locked, present, alive      bool
	name                        [party_state.NMaxPlayerNameSize]byte
	class                       party_state.CharacterClass
	strength                    byte
	slots                       [references.SlotAmulet + 1]references.Equipment
	ammunition                  [2]uint16
	item                        availableEquipment
	spriteSheet                 *sprites.SpriteSheet
}

func (d *ReadyEquipmentDialog) currentContentKey() readyContentKey {
	var key readyContentKey
	key.revision, key.character = d.contentRevision, d.selectedCharIdx
	key.partyCount = d.partyState.GetActivePartyCount()
	key.selection = d.itemListNav.GetCurrentIndex()
	key.start, key.end = d.itemListNav.GetVisibleRange()
	key.page, key.warning = d.detailPage, d.warningMessage
	key.hand, key.locked = d.targetHand, d.request.LockCharacter
	key.ammunition = [2]uint16{
		d.inventory.Equipment.Get(references.Arrows), d.inventory.Equipment.Get(references.Quarrels),
	}
	key.spriteSheet = d.spriteSheet
	if character := d.getSelectedCharacter(); character != nil {
		key.present, key.alive = true, character.IsAlive()
		key.name, key.class, key.strength = character.Name, character.Class, character.Strength
		for slot := references.SlotHelm; slot <= references.SlotAmulet; slot++ {
			key.slots[slot] = character.GetEquippedItem(slot)
		}
	}
	if item := d.getSelectedItem(); item != nil {
		key.item = *item
	}

	return key
}

func (d *ReadyEquipmentDialog) drawCachedParent(screen *ebiten.Image) {
	key := d.currentContentKey()
	if d.content == nil {
		var bounds image.Rectangle
		for _, pane := range d.frameRects {
			bounds = bounds.Union(pane)
		}
		// Non-zero bounds preserve existing panel coordinates and clipping.
		d.content = ebiten.NewImageWithOptions(bounds, nil)
	}
	if !d.contentSet || d.contentKey != key {
		d.content.Clear()
		d.drawParentContent(d.content)
		// Drawing clamps detail pages; retain the resulting view, not the stale
		// pre-layout page, so unchanged frames do not repaint a second time.
		d.contentKey = d.currentContentKey()
		d.contentSet = true
		d.contentBuilds++
		profiling.Count(profiling.ReadyRebuilds)
	}
	var options ebiten.DrawImageOptions
	options.GeoM.Translate(float64(d.content.Bounds().Min.X), float64(d.content.Bounds().Min.Y))
	screen.DrawImage(d.content, &options)
}

func (d *ReadyEquipmentDialog) releaseContent() {
	if d.content != nil {
		d.content.Deallocate()
		d.content = nil
	}
	d.contentSet = false
}
