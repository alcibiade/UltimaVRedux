package audio

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// AmbientProfile configures a persistent loop. Radius zero is a non-spatial
// background; positive radii are in map tiles. Tile sources can override radius
// and multiply the cue/recording mix by their own relative gain.
type AmbientProfile struct {
	RadiusTiles float64 `json:"radiusTiles"`
	FadeInMs    int64   `json:"fadeInMs"`
	FadeOutMs   int64   `json:"fadeOutMs"`
}

// TileAudio attaches audio to a canonical animation family or a static tile.
// Empty ambience means none; an empty surface inherits DefaultSurface. Nil
// radius/gain inherit the sound's range and 100% mix; explicit zero gain is mute.
type TileAudio struct {
	AmbientCue  string   `json:"ambientCue,omitempty"`
	Surface     string   `json:"surface,omitempty"`
	RadiusTiles *float64 `json:"radiusTiles,omitempty"`
	Gain        *float64 `json:"gain,omitempty"`
}

// WithAmbient returns an assignment with a new sound, retiring source overrides
// when the sound is removed while preserving the independent footstep surface.
func (assignment TileAudio) WithAmbient(cue string) TileAudio {
	assignment.AmbientCue = cue
	if cue == "" {
		assignment.RadiusTiles, assignment.Gain = nil, nil
	}

	return assignment
}

// TileAmbientSettings resolves per-family overrides without changing the shared
// cue, recording calibration, or fades. A missing gain means 100% of the cue mix.
func (routing Routing) TileAmbientSettings(tileID string) (AmbientProfile, float64) {
	assignment := routing.Tiles[tileID]
	profile := routing.Ambient[assignment.AmbientCue]
	level := 1.0
	if assignment.RadiusTiles != nil {
		profile.RadiusTiles = *assignment.RadiusTiles
	}
	if assignment.Gain != nil {
		level = *assignment.Gain
	}

	return profile, level
}

// Routing is shared by the authoring document and immutable game runtime.
// Missing music/dungeon keys inherit; a present null explicitly selects silence.
type Routing struct {
	Ambient        map[string]AmbientProfile `json:"ambient"`
	Tiles          map[string]TileAudio      `json:"tiles"`
	Surfaces       map[string][]string       `json:"surfaces"`
	DefaultSurface string                    `json:"defaultSurface"`
	Music          map[string]*string        `json:"music"`
	Dungeon        map[string]*string        `json:"dungeon"`
	Naval          map[string]*string        `json:"naval"`
	Emitters       map[string]string         `json:"emitters"`
}

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

func resolveAudioAssignment(assignments map[string]*string, target string) (string, bool) {
	value, explicit := assignments[target]
	if !explicit {
		value = assignments[routingDefault]
	}
	if value == nil {
		return "", explicit
	}

	return *value, explicit
}

func (routing Routing) musicFor(location references.Location, floor references.FloorNumber) string {
	key := locationMusicKey(location)
	if location == references.BritanniaUnderworld {
		key = "world/overworld"
		if floor == references.FloorNumber(references.UNDERWORLD) {
			key = "world/underworld"
		}
	}
	cue, _ := resolveAudioAssignment(routing.Music, key)

	return strings.TrimPrefix(cue, "music/track/")
}

func (routing Routing) dungeonCue(location references.Location) string {
	if location.GetMapType() != references.DungeonMapType {
		return ""
	}
	cue, _ := resolveAudioAssignment(routing.Dungeon, locationMusicKey(location))

	return cue
}

func (routing Routing) validate(library *audioLibrary) error {
	if routing.Ambient == nil || routing.Tiles == nil || routing.Surfaces == nil ||
		routing.Music == nil || routing.Dungeon == nil || routing.Naval == nil || routing.Emitters == nil {
		return fmt.Errorf("%w: routing sections are required", ErrAudioLibrary)
	}
	for cueID, profile := range routing.Ambient {
		if err := validateAmbientProfile(library, cueID, profile); err != nil {
			return err
		}
	}
	if err := routing.validateTilesAndSurfaces(library); err != nil {
		return err
	}
	if err := validateRoutingAssignments(routing.Music, MusicTargets(), library, false); err != nil {
		return err
	}
	if err := validateRoutingAssignments(routing.Dungeon, DungeonTargets(), library, true); err != nil {
		return err
	}
	if err := routing.validateNaval(library); err != nil {
		return err
	}

	return routing.validateEmitters()
}

func (routing Routing) validateNaval(library *audioLibrary) error {
	targets := NavalTargets()
	for target, cue := range routing.Naval {
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
		if _, exists := routing.Naval[target]; !exists {
			return fmt.Errorf("%w: missing naval situation %s", ErrAudioLibrary, target)
		}
	}

	return nil
}

func (routing Routing) validateEmitters() error {
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

func validateAmbientProfile(library *audioLibrary, cueID string, profile AmbientProfile) error {
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

func (routing Routing) validateTilesAndSurfaces(library *audioLibrary) error {
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

	return routing.validateTileAssignments()
}

func (routing Routing) validateTileAssignments() error {
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
	assignments map[string]*string, targets map[string]string, library *audioLibrary, background bool,
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

// AmbientDistanceGain is shared by game playback and distance auditions.
func AmbientDistanceGain(distance, radius float64) float64 {
	if radius == 0 {
		return 1
	}
	if distance >= radius {
		return 0
	}
	if distance <= 1 {
		return 1
	}

	return max(0, 1-distance*distance/(radius*radius))
}
