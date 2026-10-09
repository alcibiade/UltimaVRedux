package audio

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TileTarget is an immutable authoring target with shared animation membership.
type TileTarget struct {
	ID     string
	Name   string
	Frames []int
}

// TileTargets derives canonical targets from the same atlas families as rendering.
func TileTargets() []TileTarget {
	var result []TileTarget
	for sprite, tile := range *references.NewTileReferences() {
		if sprite < 0 || indexes.CanonicalAnimationSprite(sprite) != sprite {
			continue
		}
		item := TileTarget{ID: strconv.Itoa(int(sprite)), Name: tile.Name, Frames: []int{int(sprite)}}
		if family, animated := indexes.AnimationFor(sprite); animated {
			item.Name = strings.TrimRight(tile.Name, "0123456789")
			item.Frames = nil
			for frame := range family.Frames {
				item.Frames = append(item.Frames, int(family.Base)+frame)
			}
		}
		result = append(result, item)
	}
	slices.SortFunc(result, func(a, b TileTarget) int { return a.Frames[0] - b.Frames[0] })

	return result
}

// Routing returns a detached snapshot. SetRouting validates the whole edit before
// installing it, so multi-selection changes are one undoable transaction.
func (e *LibraryEditor) Routing() Routing { return cloneAudioValue(e.library.Routing) }

// SetRouting validates and records one atomic authoring edit.
func (e *LibraryEditor) SetRouting(routing Routing) error {
	next := cloneAudioValue(e.library)
	next.Routing = cloneAudioValue(routing)

	return e.replace(next)
}

// RoutingChanged includes inheritance changes in the editor's dirty indication.
func (e *LibraryEditor) RoutingChanged() bool {
	return string(encodeAudioValue(e.library.Routing)) != string(encodeAudioValue(e.baseline.Routing))
}

// ResolveAssignment reports both effective selection and explicit/inherited state.
func (e *LibraryEditor) ResolveAssignment(target string, dungeon bool) (string, bool) {
	if dungeon {
		return resolveAudioAssignment(e.library.Routing.Dungeon, target)
	}

	return resolveAudioAssignment(e.library.Routing.Music, target)
}

// DefaultAmbientCueGain is the initial authored mix level for newly created ambience.
const DefaultAmbientCueGain = 0.3

// AddAmbient creates a fully authored cue and an empty recording slot atomically.
// An arbitrary new cue needs no enum or compiled identity-table change.
func (e *LibraryEditor) AddAmbient(id, name string, profile AmbientProfile) error {
	return e.AddAmbientWithRecording(id, name, profile, "")
}

// AddAmbientWithRecording creates a sound using an existing recording, or a new
// empty slot when recordingID is blank, as one validated undoable edit.
func (e *LibraryEditor) AddAmbientWithRecording(
	identity, name string, profile AmbientProfile, recordingID string,
) error {
	if identity == "" || strings.Trim(identity, "abcdefghijklmnopqrstuvwxyz0123456789_") != "" {
		return fmt.Errorf("%w: ambient identifier must use lowercase letters, digits and underscores", ErrAudioLibrary)
	}
	cueID := "ambient/" + identity
	if _, exists := e.library.Cues[cueID]; exists {
		return fmt.Errorf("%w: cue %s already exists", ErrAudioLibrary, cueID)
	}
	next := cloneAudioValue(e.library)
	if recordingID == "" {
		recordingID = identity + "_01"
		if _, exists := e.library.Recordings[recordingID]; exists {
			return fmt.Errorf("%w: recording %s already exists", ErrAudioLibrary, recordingID)
		}
		next.Recordings[recordingID] = Recording{ //nolint:exhaustruct // Empty slot: no file, calibration or excerpt.
			Provenance: unknownRecordingProvenance(),
		}
		next.Taxonomy.RecordingFamilies["ambience"] = append(next.Taxonomy.RecordingFamilies["ambience"], recordingID)
	}
	next.Cues[cueID] = Cue{ //nolint:exhaustruct // New ambience has one recording, never a finite SFX cycle.
		Name: name, RecordingID: recordingID, VolumeModifier: DefaultAmbientCueGain,
		HowToHear: "Assign this ambient cue to a tile family or dungeon background; supply its recording to hear it.",
	}
	next.Routing.Ambient[cueID] = profile
	next.Taxonomy.CueProfiles["ambient_bed"] = append(next.Taxonomy.CueProfiles["ambient_bed"], cueID)

	return e.replace(next)
}

// Usage is a navigable configured use, not evidence of a placed map object
// or listening acceptance. Workspace is tiles, music, dungeon, or emitter.
type Usage struct {
	Workspace string
	Target    string
	Label     string
}

// RoutingUsages lists effective assignments, including inherited location music.
func (e *LibraryEditor) RoutingUsages(cueID string) []Usage {
	var usages []Usage
	routing := e.library.Routing
	for _, tile := range TileTargets() {
		assignment := routing.Tiles[tile.ID]
		if assignment.AmbientCue == cueID {
			usages = append(usages, Usage{"tiles", tile.ID, tile.Name + " · ambience"})
		}
		surface := assignment.Surface
		if surface == "" {
			surface = routing.DefaultSurface
		}
		if slices.Contains(routing.Surfaces[surface], cueID) {
			usages = append(usages, Usage{"tiles", tile.ID, tile.Name + " · footsteps: " + surface})
		}
	}
	for target, name := range MusicTargets() {
		if cue, _ := resolveAudioAssignment(routing.Music, target); cue == cueID {
			usages = append(usages, Usage{"music", target, name + " · music"})
		}
	}
	for target, name := range DungeonTargets() {
		if cue, _ := resolveAudioAssignment(routing.Dungeon, target); cue == cueID {
			usages = append(usages, Usage{"dungeon", target, name + " · background"})
		}
	}
	for target, name := range NavalTargets() {
		if assignment := routing.Naval[target]; assignment != nil && *assignment == cueID {
			usages = append(usages, Usage{"naval", target, name + " · state-driven"})
		}
	}
	for emitter, cue := range routing.Emitters {
		if cue == cueID {
			usages = append(usages, Usage{"emitter", emitter, emitter + " · code-owned active source"})
		}
	}
	slices.SortFunc(usages, func(a, b Usage) int { return strings.Compare(a.Label, b.Label) })

	return usages
}
