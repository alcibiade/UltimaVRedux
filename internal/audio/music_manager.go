package audio

import (
	"strings"
	"sync"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// MusicManager handles location-based background music selection and playback
// Maps specific locations to appropriate music tracks for atmospheric gameplay
type MusicManager struct {
	audioManager      musicHost
	currentLocation   references.Location
	policy            *musicPolicy
	pauseDepth        int
	mutex             sync.RWMutex
	loadFailureLogged bool
}

type musicHost interface {
	ensureAssetsLoaded() error
	musicLibrary() *Library
	musicReadLock()
	musicReadUnlock()
}

func newMusicManager(host musicHost, output musicPlayback, now func() time.Time) *MusicManager {
	return &MusicManager{
		audioManager:    host,
		currentLocation: references.EmptyLocation,
		policy:          newMusicPolicy(output, now),
		pauseDepth:      0, mutex: sync.RWMutex{}, loadFailureLogged: false,
	}
}

func musicForLocation(location references.Location) string {
	return musicFor(productionLibrary().Routing, location, references.FloorNumber(references.OVERWORLD))
}

// GetMusicForContext returns appropriate music for game contexts (intro, victory, etc.)
func (mm *MusicManager) GetMusicForContext(context string) string {
	return musicForContext(mm.audioManager.musicLibrary().document.Routing.Groups[routingGroupMusic], context)
}

func musicForContext(assignments map[string]*string, context string) string {
	cue := resolveAudioAssignment(assignments, "context/"+context)

	return strings.TrimPrefix(cue, "music/track/")
}

// musicContextTracks keeps named context selection and catalog coverage in sync.
func musicContextTracks() map[string]string {
	tracks := make(map[string]string)
	for target := range MusicTargets() {
		if strings.HasPrefix(target, "context/") {
			cue := resolveAudioAssignment(productionLibrary().Routing.Groups[routingGroupMusic], target)
			tracks[strings.TrimPrefix(target, "context/")] = strings.TrimPrefix(cue, "music/track/")
		}
	}

	return tracks
}

// PlayMusicForLocation starts playing appropriate music for a specific location
func (mm *MusicManager) PlayMusicForLocation(location references.Location) error {
	return mm.PlayMusicForScene(references.PlayerLocation{
		Location: location, Floor: references.FloorNumber(references.OVERWORLD), Position: references.Position{X: 0, Y: 0},
	})
}

// PlayMusicForScene requests exploration music with world-layer context.
func (mm *MusicManager) PlayMusicForScene(location references.PlayerLocation) error {
	if err := mm.audioManager.ensureAssetsLoaded(); err != nil {
		return err
	}
	mm.audioManager.musicReadLock()
	defer mm.audioManager.musicReadUnlock()
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	mm.policy.cue = "" // An explicit new location request replaces any pending special cue.
	mm.currentLocation = location.Location
	mm.policy.exploration = musicFor(mm.audioManager.musicLibrary().document.Routing, location.Location, location.Floor)

	return mm.policy.update()
}

// SetSceneSoundscape follows authoritative location, encounter, victory and sail state.
func (mm *MusicManager) SetSceneSoundscape(
	location references.PlayerLocation, encounter uint64, sailsRaised, victory bool,
) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	if encounter == 0 {
		if mm.currentLocation != location.Location {
			mm.policy.cue = ""
		}
		mm.currentLocation = location.Location
		mm.policy.exploration = musicFor(mm.audioManager.musicLibrary().document.Routing, location.Location, location.Floor)
	}
	mm.policy.combatTrack = mm.GetMusicForContext("combat")
	track := ""
	if cue := mm.audioManager.musicLibrary().document.Routing.Groups[routingGroupNaval]["sailing"]; cue != nil {
		track = strings.TrimPrefix(*cue, "music/track/")
	}
	mm.policy.setSailing(sailsRaised, track)
	mm.policy.setCombatEncounter(encounter)
	mm.policy.setCombatVictory(victory)
}

// PlayMusicForContext starts playing music for a specific game context
func (mm *MusicManager) PlayMusicForContext(context string) error {
	if err := mm.audioManager.ensureAssetsLoaded(); err != nil {
		return err
	}
	mm.audioManager.musicReadLock()
	defer mm.audioManager.musicReadUnlock()
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	mm.policy.cue = mm.GetMusicForContext(context)

	return mm.policy.update()
}

// PauseBackgroundMusic temporarily pauses background music (for camping, dialogs, etc.)
func (mm *MusicManager) PauseBackgroundMusic() {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mm.pauseDepth++
	mm.policy.setSuppressed(musicWorkflow, true)
}

// ResumeBackgroundMusic resumes previously paused background music
func (mm *MusicManager) ResumeBackgroundMusic() {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if mm.pauseDepth > 0 {
		mm.pauseDepth--
	}
	if mm.pauseDepth == 0 {
		mm.policy.setSuppressed(musicWorkflow, false)
	}
}

// SetEnabled enables or disables the music manager
func (mm *MusicManager) SetEnabled(enabled bool) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mm.policy.history.disabled = !enabled
	mm.policy.last = mm.policy.now()
	if !enabled {
		mm.policy.suspend()
	}
}

// SetSuppressed applies an independent presentation owner without losing history.
func (mm *MusicManager) SetSuppressed(owner MusicSuppression, suppressed bool) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	mm.policy.setSuppressed(owner, suppressed)
}

// IsEnabled returns whether the music manager is enabled
func (mm *MusicManager) IsEnabled() bool {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	return !mm.policy.history.disabled
}

// Update advances presentation-only scheduling and fades.
func (mm *MusicManager) Update() {
	mm.mutex.RLock()
	needed := mm.policy.exploration != "" || mm.policy.cue != "" || mm.policy.combatEncounter != 0 ||
		mm.policy.sailingRaised
	mm.mutex.RUnlock()
	if needed {
		if err := mm.audioManager.ensureAssetsLoaded(); err != nil {
			mm.mutex.Lock()
			if !mm.loadFailureLogged {
				logging.Warnf(logging.CategoryAudio, "Load scheduled music: %v", err)
				mm.loadFailureLogged = true
			}
			mm.mutex.Unlock()

			return
		}
	}
	// Match AudioManager's lock order. Playback reads category preferences,
	// which live gain changes mutate under the manager's exclusive lock.
	mm.audioManager.musicReadLock()
	defer mm.audioManager.musicReadUnlock()
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	if err := mm.policy.update(); err != nil {
		logging.Warnf(logging.CategoryAudio, "Update music policy: %v", err)
	}
}

// LocalMusic gives an audible bard/instrument precedence, with a release delay
// to avoid score chatter at a bard's range boundary or between instrument notes.
func (mm *MusicManager) LocalMusic() {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	mm.policy.localMusic()
}

// History exports detached session memory, never device or interruption state.
func (mm *MusicManager) History() MusicHistory {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	return mm.policy.exportHistory()
}

// RestoreHistory installs application-session history before scene activation.
func (mm *MusicManager) RestoreHistory(history MusicHistory) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	mm.policy.restoreHistory(history)
}

func locationMusicMappings() map[references.Location]string {
	mappings := make(map[references.Location]string)

	// Get all location constants up to the last known location
	for i := references.Location(0); i <= references.CombatRestingShrine; i++ {
		if i != references.EmptyLocation {
			mappings[i] = musicForLocation(i)
		}
	}

	return mappings
}
