package audio

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// Schema-3 assignment groups in Routing.Groups. Missing music/dungeon keys
// inherit the group's default; a present null explicitly selects silence.
const (
	routingGroupMusic   = "music"
	routingGroupDungeon = "dungeon"
	routingGroupNaval   = "naval"
)

const routingDefault = "default"
const maxAmbientRadius = 32
const maxAmbientFadeMs = 60000
const maxTileAmbientGain = 2.0

// Stable authoring identities for the two independently assignable boat beds.
const (
	NavalFrigate = "frigate"
	NavalSkiff   = "skiff"
)

// MusicTargets lists stable authoring targets independently of their assignment.
func MusicTargets() map[string]string {
	targets := map[string]string{
		routingDefault: "Default exploration music", "world/overworld": "Britannia", "world/underworld": "Underworld",
	}
	for location := references.Moonglow; location <= references.CombatRestingShrine; location++ {
		targets[locationMusicKey(location)] = location.String()
	}
	contexts := []string{"intro", "title", "menu", "victory", "win", "reunion", "ending", "underworld", "amiga", "combat"}
	for _, context := range contexts {
		targets["context/"+context] = "Context: " + context
	}

	return targets
}

// DungeonTargets lists background owners. Floors/rooms inherit the visit owner.
func DungeonTargets() map[string]string {
	targets := map[string]string{routingDefault: "Default dungeon background"}
	for location := references.Deceit; location <= references.Doom; location++ {
		targets[locationMusicKey(location)] = location.String()
	}

	return targets
}

// NavalTargets are state-driven sources, not location or vehicle-sprite sounds.
func NavalTargets() map[string]string {
	return map[string]string{
		NavalFrigate: "Aboard frigate", NavalSkiff: "Aboard skiff", "sailing": "Sailing · sails raised",
	}
}

func locationMusicKey(location references.Location) string {
	return "location/" + strconv.Itoa(int(location))
}

func tileAudioKey(sprite indexes.SpriteIndex) string {
	return strconv.Itoa(int(indexes.CanonicalAnimationSprite(sprite)))
}

func resolveAudioAssignment(assignments map[string]*string, target string) string {
	value, explicit := assignments[target]
	if !explicit {
		value = assignments[routingDefault]
	}
	if value == nil {
		return ""
	}

	return *value
}

func musicFor(routing Routing, location references.Location, floor references.FloorNumber) string {
	key := locationMusicKey(location)
	if location == references.BritanniaUnderworld {
		key = "world/overworld"
		if floor == references.FloorNumber(references.UNDERWORLD) {
			key = "world/underworld"
		}
	}
	cue := resolveAudioAssignment(routing.Groups[routingGroupMusic], key)

	return strings.TrimPrefix(cue, "music/track/")
}

func dungeonCue(routing Routing, location references.Location) string {
	if location.GetMapType() != references.DungeonMapType {
		return ""
	}
	cue := resolveAudioAssignment(routing.Groups[routingGroupDungeon], locationMusicKey(location))

	return cue
}

func validateRouting(routing Routing, library *LibraryDocument) error {
	if routing.Ambient == nil || routing.Tiles == nil || routing.Surfaces == nil || routing.Emitters == nil ||
		routing.Groups[routingGroupMusic] == nil || routing.Groups[routingGroupDungeon] == nil ||
		routing.Groups[routingGroupNaval] == nil {
		return fmt.Errorf("%w: routing sections are required", ErrAudioLibrary)
	}
	for cueID, profile := range routing.Ambient {
		if err := validateAmbientProfile(library, cueID, profile); err != nil {
			return err
		}
	}
	if err := validateTilesAndSurfaces(routing, library); err != nil {
		return err
	}
	if err := validateRoutingAssignments(routing.Groups[routingGroupMusic], MusicTargets(), library, false); err != nil {
		return err
	}
	dungeon := routing.Groups[routingGroupDungeon]
	if err := validateRoutingAssignments(dungeon, DungeonTargets(), library, true); err != nil {
		return err
	}
	if err := validateNaval(routing, library); err != nil {
		return err
	}

	return validateEmitters(routing)
}

func validateNaval(routing Routing, library *LibraryDocument) error {
	targets := NavalTargets()
	for target, cue := range routing.Groups[routingGroupNaval] {
		if _, exists := targets[target]; !exists {
			return fmt.Errorf("%w: unknown naval situation %s", ErrAudioLibrary, target)
		}
		if cue == nil {
			continue
		}
		if _, exists := library.Cues[*cue]; !exists {
			return fmt.Errorf("%w: unknown naval cue %s", ErrAudioLibrary, *cue)
		}
		if target == "sailing" {
			if !strings.HasPrefix(*cue, "music/track/") {
				return fmt.Errorf("%w: sailing requires a music cue", ErrAudioLibrary)
			}

			continue
		}
		profile, exists := routing.Ambient[*cue]
		if !exists || profile.RadiusTiles != 0 {
			return fmt.Errorf("%w: %s requires non-spatial ambience", ErrAudioLibrary, target)
		}
	}
	for target := range targets {
		if _, exists := routing.Groups[routingGroupNaval][target]; !exists {
			return fmt.Errorf("%w: missing naval situation %s", ErrAudioLibrary, target)
		}
	}

	return nil
}

func validateEmitters(routing Routing) error {
	emitters := []string{"bard", "moongate"}
	if len(routing.Emitters) != len(emitters) {
		return fmt.Errorf("%w: routing requires bard and moongate emitters", ErrAudioLibrary)
	}
	for _, emitter := range emitters {
		profile, exists := routing.Ambient[routing.Emitters[emitter]]
		if !exists || profile.RadiusTiles <= 0 {
			return fmt.Errorf("%w: %s requires a spatial ambient cue", ErrAudioLibrary, emitter)
		}
	}

	return nil
}

func validateAmbientProfile(library *LibraryDocument, cueID string, profile AmbientProfile) error {
	if !strings.HasPrefix(cueID, "ambient/") || strings.TrimPrefix(cueID, "ambient/") == "" {
		return fmt.Errorf("%w: invalid ambient cue %q", ErrAudioLibrary, cueID)
	}
	if _, exists := library.Cues[cueID]; !exists {
		return fmt.Errorf("%w: ambient profile %s has no cue", ErrAudioLibrary, cueID)
	}
	if math.IsNaN(profile.RadiusTiles) || math.IsInf(profile.RadiusTiles, 0) ||
		profile.RadiusTiles < 0 || profile.RadiusTiles > maxAmbientRadius ||
		profile.FadeInMs < 0 || profile.FadeOutMs < 0 ||
		profile.FadeInMs > maxAmbientFadeMs || profile.FadeOutMs > maxAmbientFadeMs {
		return fmt.Errorf("%w: %s requires radius 0–32 and fades 0–60000 ms", ErrAudioLibrary, cueID)
	}

	return nil
}

func validateTilesAndSurfaces(routing Routing, library *LibraryDocument) error {
	if _, exists := routing.Surfaces[routing.DefaultSurface]; !exists {
		return fmt.Errorf("%w: unknown default footstep surface", ErrAudioLibrary)
	}
	for surface, cues := range routing.Surfaces {
		if surface == "" {
			return fmt.Errorf("%w: empty surface identity", ErrAudioLibrary)
		}
		for _, cue := range cues {
			if _, exists := library.Cues[cue]; !exists || !strings.HasPrefix(cue, "sfx/") {
				return fmt.Errorf("%w: surface %s has invalid cue %s", ErrAudioLibrary, surface, cue)
			}
		}
	}

	return validateTileAssignments(routing)
}

func validateTileAssignments(routing Routing) error {
	tiles := references.NewTileReferences()
	for key, assignment := range routing.Tiles {
		number, err := strconv.Atoi(key)
		if err != nil || number < 0 || tiles.GetTile(indexes.SpriteIndex(number)) == nil ||
			key != tileAudioKey(indexes.SpriteIndex(number)) {
			return fmt.Errorf("%w: tile %s must be a known canonical animation/static tile", ErrAudioLibrary, key)
		}
		if assignment.AmbientCue != "" {
			profile, exists := routing.Ambient[assignment.AmbientCue]
			if !exists || profile.RadiusTiles <= 0 {
				return fmt.Errorf("%w: tile %s requires a spatial ambient cue", ErrAudioLibrary, key)
			}
		}
		if _, exists := routing.Surfaces[assignment.Surface]; assignment.Surface != "" && !exists {
			return fmt.Errorf("%w: tile %s has unknown surface %s", ErrAudioLibrary, key, assignment.Surface)
		}
		if err := validateTileAmbientOverrides(key, assignment); err != nil {
			return err
		}
	}

	return nil
}

func validateTileAmbientOverrides(tileID string, assignment TileAudio) error {
	if assignment.AmbientCue == "" && (assignment.RadiusTiles != nil || assignment.Gain != nil) {
		return fmt.Errorf("%w: tile %s needs an ambient sound before setting source overrides", ErrAudioLibrary, tileID)
	}
	if radius := assignment.RadiusTiles; radius != nil {
		if math.IsNaN(*radius) || math.IsInf(*radius, 0) || *radius <= 0 || *radius > maxAmbientRadius {
			return fmt.Errorf("%w: tile %s hearing distance must be greater than 0 and at most 32", ErrAudioLibrary, tileID)
		}
	}
	if gain := assignment.Gain; gain != nil {
		if math.IsNaN(*gain) || math.IsInf(*gain, 0) || *gain < 0 || *gain > maxTileAmbientGain {
			return fmt.Errorf("%w: tile %s relative level must be between 0 and 200%%", ErrAudioLibrary, tileID)
		}
	}

	return nil
}

func validateRoutingAssignments(
	assignments map[string]*string, targets map[string]string, library *LibraryDocument, background bool,
) error {
	if _, exists := assignments[routingDefault]; !exists {
		return fmt.Errorf("%w: audio assignments require an explicit default", ErrAudioLibrary)
	}
	for target, cue := range assignments {
		if _, exists := targets[target]; !exists {
			return fmt.Errorf("%w: unknown audio target %s", ErrAudioLibrary, target)
		}
		if cue == nil {
			continue
		}
		if _, exists := library.Cues[*cue]; !exists {
			return fmt.Errorf("%w: unknown cue for %s", ErrAudioLibrary, target)
		}
		if background {
			profile, exists := library.Routing.Ambient[*cue]
			if !exists || profile.RadiusTiles != 0 {
				return fmt.Errorf("%w: %s requires a non-spatial background", ErrAudioLibrary, target)
			}
		} else if !strings.HasPrefix(*cue, "music/track/") {
			return fmt.Errorf("%w: %s requires a music cue", ErrAudioLibrary, target)
		}
	}

	return nil
}
