package config

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/bradhannah/UltimaVRedux/internal/files"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/pkg/helpers"
)

// Static errors for consistent error handling
var (
	ErrDebugConfigNil          = errors.New("debug configuration is nil")
	ErrGameplayConfigNil       = errors.New("gameplay configuration is nil")
	ErrAudioConfigNil          = errors.New("audio configuration is nil")
	ErrDisplayConfigNil        = errors.New("display configuration is nil")
	ErrConfigFileUnknown       = errors.New("configuration file path is unknown")
	ErrConfigSourceUnavailable = errors.New("configuration source is unavailable")
	ErrResolutionIndex         = errors.New("invalid window resolution index")
	ErrDisplayConfig           = errors.New("invalid display configuration")
)

// always use 16x9 resolutions

// 1280 x 720 (HD)
// 1366 x 768 (HD)
// 1600 x 900
// 1920 x 1080 (Full HD or 1080p)
// 2560 x 1440 (QHD or 1440p)
// 3840 x 2160 (4K UHD)

// var WindowWidth = 1280
// var WindowHeight = 720

// var WindowWidth = 1920
// var WindowHeight = 1080

// var WindowWidth = 3840
// var WindowHeight = 2160
// var WindowWidth = 2560
// var WindowHeight = 1440

type UltimaVConfigurationFlags struct {
	Resolution   int
	FullScreen   bool
	DataFilePath string
}

// AudioConfiguration holds volume settings for different audio channels
// Stores raw slider values (0-10) for simplicity and future tweaking flexibility
type AudioConfiguration struct {
	MusicVolume       int // Music track volume slider (0-10)
	SoundEffectVolume int // Sound effects volume slider (0-10)
	AmbientVolume     int // Ambient sounds volume slider (0-10)
}

// DisplayConfiguration holds desktop display and performance settings.
type DisplayConfiguration struct {
	Mode              Mode
	WindowWidth       int
	WindowHeight      int
	TargetFPS         int  // Target FPS for draw limiting (0 = unlimited, 30, 60, etc.)
	EnableFpsLimiting bool // Enable FPS limiting
}

// GameplayConfiguration holds optional Redux gameplay rules.
type GameplayConfiguration struct {
	// REDUX DEVIATION: Original merchants steal per economic unit; ref:MERCH-0003.
	// BatchMerchantTheft defaults on for one theft per confirmed paid batch,
	// keeping keyboard quantity selection predictable.
	BatchMerchantTheft bool
	// AllowRepeatedMonsterDivision restores original SPLITS behavior when true.
	AllowRepeatedMonsterDivision bool
	// ShowConcealedTurnOrderSlots reserves blank forecast slots for concealed enemies.
	ShowConcealedTurnOrderSlots bool
	// PersistHiddenDungeonDiscoveries retains revealed hidden doors and pits between visits.
	PersistHiddenDungeonDiscoveries bool
	// ShowSubduedUnlitDungeons keeps dungeon geometry faintly visible without real light.
	ShowSubduedUnlitDungeons bool
	// SmoothSurfaceMovement enables optional surface camera scrolling, including boats.
	SmoothSurfaceMovement bool
	// FlexibleSextantUse opts into daytime and settlement sextant navigation.
	FlexibleSextantUse bool
}

// ConfigurationSettings is the complete player-facing Configuration transaction.
type ConfigurationSettings struct {
	Audio    AudioConfiguration
	Display  DisplayConfiguration
	Gameplay GameplayConfiguration
}

const (
	gameplayBatchMerchantTheftKey              = "Gameplay.Merchants.BatchTheft"
	gameplayShowConcealedTurnOrderSlotsKey     = "Gameplay.Combat.ShowConcealedTurnOrderSlots"
	gameplayPersistHiddenDungeonDiscoveriesKey = "Gameplay.Dungeons.PersistHiddenDiscoveries"
	gameplayShowSubduedUnlitDungeonsKey        = "Gameplay.Dungeons.ShowSubduedUnlitDungeons"
	gameplaySmoothSurfaceMovementKey           = "Gameplay.Presentation.SmoothSurfaceMovement"
	gameplayFlexibleSextantUseKey              = "Gameplay.Navigation.FlexibleSextantUse"
)

// LoggingConfiguration holds logging system settings
type LoggingConfiguration struct {
	Enabled        bool            // Global logging enabled/disabled
	StdoutEnabled  bool            // Output to stdout enabled/disabled
	CategoryStates map[string]bool // Per-category enabled/disabled state
	MinLevel       string          // Minimum log level ("DEBUG", "INFO", "WARN", "ERROR", "FATAL")
}

// DebugConfiguration holds debug settings.
type DebugConfiguration struct {
	// Small-map presentation prototype; read from config at startup, not player settings.
	SurfaceMovementDurationMs int
	SurfaceMovementCurve      string
	CombatDetailShown         bool    // Show authentic combat rolls and initiative detail
	CombatDetailSet           bool    // True after loading or explicitly changing the setting
	CombatMovementSpeedFactor float64 // Visual combat movement multiplier; higher is faster
}

// Combat movement speed factors define the supported persisted debug range and default.
const (
	DefaultCombatMovementSpeedFactor = 1.0
	MinCombatMovementSpeedFactor     = 0.25
	MaxCombatMovementSpeedFactor     = 4.0
)

// GetCombatMovementSpeedFactor returns a safe persisted movement multiplier.
func (d *DebugConfiguration) GetCombatMovementSpeedFactor() float64 {
	if d == nil || !validCombatMovementSpeedFactor(d.CombatMovementSpeedFactor) {
		return DefaultCombatMovementSpeedFactor
	}

	return d.CombatMovementSpeedFactor
}

// SetCombatMovementSpeedFactor updates the multiplier when it is in the supported range.
func (d *DebugConfiguration) SetCombatMovementSpeedFactor(factor float64) bool {
	if d == nil || !validCombatMovementSpeedFactor(factor) {
		return false
	}

	d.CombatMovementSpeedFactor = factor

	return true
}

func validCombatMovementSpeedFactor(factor float64) bool {
	return !math.IsNaN(factor) && !math.IsInf(factor, 0) &&
		factor >= MinCombatMovementSpeedFactor && factor <= MaxCombatMovementSpeedFactor
}

// clampSliderValue ensures slider value is in valid 0-10 range
func clampSliderValue(value int) int {
	if value < 0 {
		return 0
	}
	if value > 10 {
		return 10
	}

	return value
}

// SetMusicVolume sets the music volume slider with validation (0-10)
func (a *AudioConfiguration) SetMusicVolume(sliderValue int) {
	a.MusicVolume = clampSliderValue(sliderValue)
}

// SetSoundEffectVolume sets the sound effect volume slider with validation (0-10)
func (a *AudioConfiguration) SetSoundEffectVolume(sliderValue int) {
	a.SoundEffectVolume = clampSliderValue(sliderValue)
}

// SetAmbientVolume sets the ambient volume slider with validation (0-10)
func (a *AudioConfiguration) SetAmbientVolume(sliderValue int) {
	a.AmbientVolume = clampSliderValue(sliderValue)
}

// Clone creates a deep copy of the AudioConfiguration
func (a *AudioConfiguration) Clone() *AudioConfiguration {
	return &AudioConfiguration{
		MusicVolume:       a.MusicVolume,
		SoundEffectVolume: a.SoundEffectVolume,
		AmbientVolume:     a.AmbientVolume,
	}
}

// SetTargetFPS sets the target FPS with validation
func (d *DisplayConfiguration) SetTargetFPS(fps int) {
	if fps < 0 {
		fps = 0 // 0 = unlimited
	}
	if fps > 144 {
		fps = 144 // Reasonable upper limit
	}
	d.TargetFPS = fps
}

// GetFrameInterval returns the minimum time between frames in milliseconds
func (d *DisplayConfiguration) GetFrameInterval() int64 {
	if !d.EnableFpsLimiting || d.TargetFPS <= 0 {
		return 0 // No limiting
	}

	return 1000 / int64(d.TargetFPS) // Convert FPS to milliseconds per frame
}

// Clone creates a deep copy of the DisplayConfiguration
func (d *DisplayConfiguration) Clone() *DisplayConfiguration {
	return &DisplayConfiguration{
		Mode:              d.Mode,
		WindowWidth:       d.WindowWidth,
		WindowHeight:      d.WindowHeight,
		TargetFPS:         d.TargetFPS,
		EnableFpsLimiting: d.EnableFpsLimiting,
	}
}

type UltimaVConfiguration struct {
	// DataFilePath     string
	RawDataOvl       []byte
	allWindowConfigs []ScreenResolution

	SavedConfigData *UltimaVConfigurationFlags

	// Each configuration owns its selected source, including unknown settings.
	source *viper.Viper

	// Audio configuration for volume settings
	AudioConfiguration *AudioConfiguration

	// SoundDisabled disables runtime audio initialization without changing persisted volume settings.
	SoundDisabled bool

	// Display configuration for performance settings
	DisplayConfiguration *DisplayConfiguration

	// Gameplay configuration for deliberate Redux rules.
	GameplayConfiguration *GameplayConfiguration

	// Logging configuration for debug output
	LoggingConfiguration *LoggingConfiguration

	// Debug configuration for console history
	DebugConfiguration *DebugConfiguration

	// AmbientLightRadius is the visibility radius when avatar has no torch/magic at night.
	// Range: 1-5. Default: 2 tiles. Provides minimal "night vision" in darkness.
	AmbientLightRadius int
}

// NewUltimaVConfiguration loads the user's configuration. Only a missing default
// file is created; reading an existing file never requires write access.
func NewUltimaVConfiguration() (*UltimaVConfiguration, error) {
	source, err := loadUserConfigurationSource()
	if err != nil {
		return nil, err
	}

	return newUltimaVConfiguration(source, "", true)
}

func loadUserConfigurationSource() (*viper.Viper, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("locating configuration home: %w", err)
	}
	source := viper.New()
	source.SetConfigName("config")
	source.SetConfigType("yaml")
	directory := filepath.Join(home, ".ultima_v_redux")
	source.AddConfigPath(directory)
	if err := source.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("reading configuration: %w", err)
		}
		if err := os.MkdirAll(directory, 0o700); err != nil { //nolint:mnd // User configuration is private to its owner.
			return nil, fmt.Errorf("creating configuration directory: %w", err)
		}
		source.Set("DataFilePath", "./OLD")
		if err := source.SafeWriteConfig(); err != nil {
			return nil, fmt.Errorf("writing default configuration: %w", err)
		}
		source.SetConfigFile(filepath.Join(directory, "config.yaml"))
	}

	return source, nil
}

// NewUltimaVConfigurationWithPath uses default preferences and explicit game data.
// It never reads HOME and has no persistence destination.
func NewUltimaVConfigurationWithPath(dataFilePath string) (*UltimaVConfiguration, error) {
	return LoadUltimaVConfiguration("", dataFilePath)
}

// PlayableWithDataFilePath loads original data from an explicit directory while
// preserving this configuration's in-memory preferences. It performs no writes.
func (uc *UltimaVConfiguration) PlayableWithDataFilePath(dataFilePath string) (*UltimaVConfiguration, error) {
	if uc == nil || uc.source == nil {
		return nil, fmt.Errorf("loading playable configuration: %w", ErrConfigSourceUnavailable)
	}

	return newUltimaVConfiguration(uc.source, dataFilePath, true)
}

// LoadUltimaVConfiguration reads exactly the selected YAML file, without search
// paths or writes. An empty configFilePath uses in-memory defaults and disables
// persistence. A nonempty dataFilePath overrides the loaded game-data location.
func LoadUltimaVConfiguration(configFilePath, dataFilePath string) (*UltimaVConfiguration, error) {
	source := viper.New()
	source.SetConfigType("yaml")
	if configFilePath != "" {
		absolutePath, err := filepath.Abs(configFilePath)
		if err != nil {
			return nil, fmt.Errorf("resolving configuration %s: %w", configFilePath, err)
		}
		source.SetConfigFile(absolutePath)
		if err := source.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("reading configuration %s: %w", configFilePath, err)
		}
	}

	return newUltimaVConfiguration(source, dataFilePath, true)
}

// newUltimaVConfiguration is the internal constructor used by both public constructors
//

func newUltimaVConfiguration(
	source *viper.Viper,
	explicitDataPath string,
	loadOriginalData bool,
) (*UltimaVConfiguration, error) {
	configuration := UltimaVConfiguration{source: source} //nolint:exhaustruct // Configuration sections are loaded below.
	if err := configuration.loadConfigurationSource(explicitDataPath, loadOriginalData); err != nil {
		return nil, err
	}

	configuration.allWindowConfigs = SupportedWindowResolutions()

	configuration.loadAudioConfiguration()
	configuration.loadDisplayConfiguration()

	configuration.GameplayConfiguration = &GameplayConfiguration{
		BatchMerchantTheft:           configuration.gameplayBooleanDefaultTrue(gameplayBatchMerchantTheftKey),
		SmoothSurfaceMovement:        source.GetBool(gameplaySmoothSurfaceMovementKey),
		FlexibleSextantUse:           source.GetBool(gameplayFlexibleSextantUseKey),
		AllowRepeatedMonsterDivision: source.GetBool("Gameplay.Combat.AllowRepeatedMonsterDivision"),
		ShowConcealedTurnOrderSlots: configuration.gameplayBooleanDefaultTrue(
			gameplayShowConcealedTurnOrderSlotsKey,
		),
		PersistHiddenDungeonDiscoveries: configuration.gameplayBooleanDefaultTrue(
			gameplayPersistHiddenDungeonDiscoveriesKey,
		),
		ShowSubduedUnlitDungeons: configuration.gameplayBooleanDefaultTrue(
			gameplayShowSubduedUnlitDungeonsKey,
		),
	}

	configuration.loadLoggingConfiguration()
	configuration.loadDebugConfiguration()

	// Initialize ambient light radius with fallback to default
	if source.IsSet("Lighting.AmbientLightRadius") {
		configuration.AmbientLightRadius = source.GetInt("Lighting.AmbientLightRadius")
	} else {
		configuration.AmbientLightRadius = 2 // Default: 2 tile radius for night vision
	}
	// Clamp to valid range (1-5)
	if configuration.AmbientLightRadius < 1 || configuration.AmbientLightRadius > 5 {
		configuration.AmbientLightRadius = 2
	}

	return &configuration, nil
}

func (uc *UltimaVConfiguration) loadConfigurationSource(explicitDataPath string, loadOriginalData bool) error {
	uc.SavedConfigData = &UltimaVConfigurationFlags{}
	if err := uc.source.Unmarshal(uc.SavedConfigData); err != nil {
		return fmt.Errorf("decoding configuration: %w", err)
	}

	uc.SavedConfigData.DataFilePath = explicitDataPath
	if explicitDataPath == "" {
		uc.SavedConfigData.DataFilePath = uc.source.GetString("DataFilePath")
		if uc.SavedConfigData.DataFilePath == "" {
			uc.SavedConfigData.DataFilePath = "./OLD"
		}
	}

	if loadOriginalData {
		var err error
		uc.RawDataOvl, err = os.ReadFile(path.Join(uc.SavedConfigData.DataFilePath, files.DATA_OVL))
		if err != nil {
			return fmt.Errorf("reading DATA.OVL from %s: %w", uc.SavedConfigData.DataFilePath, err)
		}
	}

	return nil
}

//nolint:mnd // Slider defaults are intentional persisted configuration values.
func (uc *UltimaVConfiguration) loadAudioConfiguration() {
	uc.AudioConfiguration = &AudioConfiguration{MusicVolume: 5, SoundEffectVolume: 6, AmbientVolume: 4}
	if uc.source.IsSet("Audio.MusicVolume") {
		uc.AudioConfiguration.MusicVolume = uc.source.GetInt("Audio.MusicVolume")
	}
	if uc.source.IsSet("Audio.SoundEffectVolume") {
		uc.AudioConfiguration.SoundEffectVolume = uc.source.GetInt("Audio.SoundEffectVolume")
	}
	if uc.source.IsSet("Audio.AmbientVolume") {
		uc.AudioConfiguration.AmbientVolume = uc.source.GetInt("Audio.AmbientVolume")
	}
}

func (uc *UltimaVConfiguration) loadDisplayConfiguration() {
	uc.DisplayConfiguration = defaultDisplayConfiguration()
	if uc.source.IsSet("Display.TargetFPS") {
		uc.DisplayConfiguration.TargetFPS = uc.source.GetInt("Display.TargetFPS")
	}
	if uc.source.IsSet("Display.EnableFpsLimiting") {
		uc.DisplayConfiguration.EnableFpsLimiting = uc.source.GetBool("Display.EnableFpsLimiting")
	}

	modeSet := uc.source.IsSet("Display.Mode")
	widthSet := uc.source.IsSet("Display.WindowWidth")
	heightSet := uc.source.IsSet("Display.WindowHeight")
	if modeSet || widthSet || heightSet {
		if modeSet && widthSet && heightSet {
			modeValue, modeOK := uc.source.Get("Display.Mode").(string)
			mode, modeErr := ParseMode(modeValue)
			width, widthOK := strictConfigurationInt(uc.source.Get("Display.WindowWidth"))
			height, heightOK := strictConfigurationInt(uc.source.Get("Display.WindowHeight"))
			if modeOK && modeErr == nil && widthOK && heightOK &&
				validDisplayDimensions(width, height) {
				uc.DisplayConfiguration.Mode = mode
				uc.DisplayConfiguration.WindowWidth = width
				uc.DisplayConfiguration.WindowHeight = height
			}
		}
		uc.syncLegacyDisplayModel()

		return
	}

	resolution, resolutionOK := strictConfigurationInt(uc.source.Get("Resolution"))
	fullscreen, fullscreenOK := uc.source.Get("FullScreen").(bool)
	resolutions := uc.windowResolutions()
	if uc.source.IsSet("Resolution") && uc.source.IsSet("FullScreen") &&
		resolutionOK && fullscreenOK && resolution >= 0 && resolution < len(resolutions) {
		uc.DisplayConfiguration.WindowWidth = resolutions[resolution].X
		uc.DisplayConfiguration.WindowHeight = resolutions[resolution].Y
		if fullscreen {
			uc.DisplayConfiguration.Mode = ModeFullscreen
		}
	}
	uc.syncLegacyDisplayModel()
}

//nolint:mnd // Thirty FPS is the intentional default frame limit.
func defaultDisplayConfiguration() *DisplayConfiguration {
	return &DisplayConfiguration{
		Mode:              ModeWindowed,
		WindowWidth:       MinimumWindowWidth,
		WindowHeight:      MinimumWindowHeight,
		TargetFPS:         30,
		EnableFpsLimiting: true,
	}
}

func validDisplayDimensions(width, height int) bool {
	return width >= MinimumWindowWidth && height >= MinimumWindowHeight
}

func validDisplayConfiguration(configuration DisplayConfiguration) bool {
	return configuration.Mode.IsValid() &&
		validDisplayDimensions(configuration.WindowWidth, configuration.WindowHeight)
}

func strictConfigurationInt(value any) (int, bool) {
	number, ok := value.(int)

	return number, ok
}

func (uc *UltimaVConfiguration) loadLoggingConfiguration() {
	uc.LoggingConfiguration = &LoggingConfiguration{
		Enabled:        false,
		StdoutEnabled:  false,
		CategoryStates: make(map[string]bool),
		MinLevel:       "DEBUG",
	}
	uc.LoggingConfiguration.Enabled = uc.source.GetBool("Logging.Enabled")
	uc.LoggingConfiguration.StdoutEnabled = uc.source.GetBool("Logging.StdoutEnabled")
	if uc.source.IsSet("Logging.MinLevel") {
		uc.LoggingConfiguration.MinLevel = uc.source.GetString("Logging.MinLevel")
	}
	for category, enabled := range uc.source.GetStringMap("Logging.CategoryStates") {
		if boolValue, ok := enabled.(bool); ok {
			uc.LoggingConfiguration.CategoryStates[category] = boolValue
		}
	}
}

func (uc *UltimaVConfiguration) loadDebugConfiguration() {
	uc.DebugConfiguration = &DebugConfiguration{ //nolint:exhaustruct // Remaining fields intentionally use defaults.
		SurfaceMovementDurationMs: uc.source.GetInt("Debug.SurfaceMovementDurationMs"),
		SurfaceMovementCurve:      uc.source.GetString("Debug.SurfaceMovementCurve"),
		CombatMovementSpeedFactor: DefaultCombatMovementSpeedFactor,
	}
	if uc.source.IsSet("Debug.CombatDetailShown") {
		uc.DebugConfiguration.CombatDetailShown = uc.source.GetBool("Debug.CombatDetailShown")
		uc.DebugConfiguration.CombatDetailSet = true
	}
	if uc.source.IsSet("Debug.CombatMovementSpeedFactor") {
		uc.DebugConfiguration.SetCombatMovementSpeedFactor(uc.source.GetFloat64("Debug.CombatMovementSpeedFactor"))
	}
}

func (uc *UltimaVConfiguration) GetLookDataFilePath() string {
	return path.Join(uc.SavedConfigData.DataFilePath, files.LOOK2_DAT)
}

func (uc *UltimaVConfiguration) GetSignsDataFilePath() string {
	return path.Join(uc.SavedConfigData.DataFilePath, files.SIGNS_DAT)
}

func (uc *UltimaVConfiguration) GetCurrentTrackedWindowResolution() ScreenResolution {
	if uc.DisplayConfiguration != nil &&
		validDisplayDimensions(uc.DisplayConfiguration.WindowWidth, uc.DisplayConfiguration.WindowHeight) {
		return ScreenResolution{
			X: uc.DisplayConfiguration.WindowWidth,
			Y: uc.DisplayConfiguration.WindowHeight,
		}
	}
	resolutions := uc.windowResolutions()
	if uc.SavedConfigData != nil &&
		uc.SavedConfigData.Resolution >= 0 && uc.SavedConfigData.Resolution < len(resolutions) {
		return resolutions[uc.SavedConfigData.Resolution]
	}

	return ScreenResolution{X: MinimumWindowWidth, Y: MinimumWindowHeight}
}

func (uc *UltimaVConfiguration) windowResolutions() []ScreenResolution {
	if len(uc.allWindowConfigs) != 0 {
		return uc.allWindowConfigs
	}

	return SupportedWindowResolutions()
}

func (uc *UltimaVConfiguration) syncLegacyDisplayModel() {
	if uc.DisplayConfiguration == nil || uc.SavedConfigData == nil {
		return
	}
	uc.SavedConfigData.FullScreen = uc.DisplayConfiguration.Mode == ModeFullscreen
	uc.SavedConfigData.Resolution = 0
	for index, resolution := range uc.windowResolutions() {
		if resolution.X == uc.DisplayConfiguration.WindowWidth &&
			resolution.Y == uc.DisplayConfiguration.WindowHeight {
			uc.SavedConfigData.Resolution = index

			return
		}
	}
}

// GetWindowResolutionOptions returns a detached copy of supported window resolutions.
func (uc *UltimaVConfiguration) GetWindowResolutionOptions() []ScreenResolution {
	return append([]ScreenResolution(nil), uc.windowResolutions()...)
}

// GetWindowResolutionIndex returns the persisted supported-resolution index.
func (uc *UltimaVConfiguration) GetWindowResolutionIndex() int {
	if uc.SavedConfigData == nil {
		return 0
	}

	return uc.SavedConfigData.Resolution
}

// SetWindowResolutionIndex selects an exact supported resolution without persisting it.
func (uc *UltimaVConfiguration) SetWindowResolutionIndex(index int) bool {
	resolutions := uc.windowResolutions()
	if index < 0 || index >= len(resolutions) ||
		uc.SavedConfigData == nil || uc.DisplayConfiguration == nil {
		return false
	}
	uc.SavedConfigData.Resolution = index
	uc.DisplayConfiguration.WindowWidth = resolutions[index].X
	uc.DisplayConfiguration.WindowHeight = resolutions[index].Y
	if !uc.DisplayConfiguration.Mode.IsValid() {
		uc.DisplayConfiguration.Mode = ModeWindowed
	}

	return true
}

// IncrementHigherResolution selects and persists the next supported resolution.
func (uc *UltimaVConfiguration) IncrementHigherResolution() error {
	if uc.SavedConfigData == nil || uc.DisplayConfiguration == nil {
		return ErrDisplayConfigNil
	}
	beforeDisplay := *uc.DisplayConfiguration
	beforeIndex := uc.SavedConfigData.Resolution
	index := helpers.Min(len(uc.windowResolutions())-1, helpers.Max(beforeIndex, 0)+1)
	if !uc.SetWindowResolutionIndex(index) {
		return fmt.Errorf("%w: %d", ErrResolutionIndex, index)
	}
	if err := uc.UpdateSaveFile(); err != nil {
		*uc.DisplayConfiguration = beforeDisplay
		uc.SavedConfigData.Resolution = beforeIndex

		return err
	}

	return nil
}

// DecrementLowerResolution selects and persists the previous supported resolution.
func (uc *UltimaVConfiguration) DecrementLowerResolution() error {
	if uc.SavedConfigData == nil || uc.DisplayConfiguration == nil {
		return ErrDisplayConfigNil
	}
	beforeDisplay := *uc.DisplayConfiguration
	beforeIndex := uc.SavedConfigData.Resolution
	index := helpers.Max(beforeIndex-1, 0)
	if !uc.SetWindowResolutionIndex(index) {
		return fmt.Errorf("%w: %d", ErrResolutionIndex, index)
	}
	if err := uc.UpdateSaveFile(); err != nil {
		*uc.DisplayConfiguration = beforeDisplay
		uc.SavedConfigData.Resolution = beforeIndex

		return err
	}

	return nil
}

// SaveWindowConfiguration persists resolution and fullscreen state.
func (uc *UltimaVConfiguration) SaveWindowConfiguration() error {
	if uc.DisplayConfiguration == nil {
		return ErrDisplayConfigNil
	}

	return uc.UpdateSaveFile()
}

// UpdateSaveFile persists all initialized configuration sections.
func (uc *UltimaVConfiguration) UpdateSaveFile() error {
	if err := uc.saveConfiguration(func(candidate *viper.Viper) {
		setInitializedConfiguration(candidate, uc)
	}); err != nil {
		return fmt.Errorf("saving configuration: %w", err)
	}

	return nil
}

// Stage changes independently so a failed write cannot leak Viper overrides.
func (uc *UltimaVConfiguration) saveConfiguration(update func(*viper.Viper)) error {
	if uc.source == nil || uc.source.ConfigFileUsed() == "" {
		return ErrConfigFileUnknown
	}
	configurationPath := uc.source.ConfigFileUsed()
	candidate := viper.New()
	candidate.SetConfigType("yaml")
	candidate.SetConfigFile(configurationPath)
	if err := candidate.ReadInConfig(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("rereading configuration %s: %w", configurationPath, err)
		}
		if err := candidate.MergeConfigMap(uc.source.AllSettings()); err != nil {
			return fmt.Errorf("recreating configuration %s: %w", configurationPath, err)
		}
	}
	update(candidate)
	if err := writeViperConfigAtomically(candidate, configurationPath); err != nil {
		return err
	}
	uc.source = candidate

	return nil
}

func setInitializedConfiguration(configuration *viper.Viper, model *UltimaVConfiguration) {
	// DataFilePath is intentionally omitted. Ordinary preference saves must not
	// persist an explicit runtime data override or replace a pending next-launch selection.
	if model.SavedConfigData != nil {
		configuration.Set("Resolution", model.SavedConfigData.Resolution)
		configuration.Set("FullScreen", model.SavedConfigData.FullScreen)
	}
	if model.AudioConfiguration != nil {
		setAudioConfiguration(configuration, *model.AudioConfiguration)
	}
	if model.DisplayConfiguration != nil {
		setDisplayConfiguration(configuration, *model.DisplayConfiguration)
	}
	if model.GameplayConfiguration != nil {
		setGameplayConfiguration(configuration, *model.GameplayConfiguration)
	}
	if model.LoggingConfiguration != nil {
		setLoggingConfiguration(configuration, *model.LoggingConfiguration)
	}
	if model.DebugConfiguration != nil {
		setDebugConfiguration(configuration, *model.DebugConfiguration)
	}
}

// SaveAudioConfiguration immediately persists current audio settings to config file
func (uc *UltimaVConfiguration) SaveAudioConfiguration() error {
	if uc.AudioConfiguration == nil {
		return ErrAudioConfigNil
	}

	if err := uc.saveConfiguration(func(candidate *viper.Viper) {
		setAudioConfiguration(candidate, *uc.AudioConfiguration)
	}); err != nil {
		return fmt.Errorf("saving audio configuration: %w", err)
	}

	return nil
}

// SaveConfigurationSettings atomically persists the player-facing settings.
// Existing configuration pointers remain stable for runtime consumers.
func (uc *UltimaVConfiguration) SaveConfigurationSettings(settings ConfigurationSettings) error {
	if uc.AudioConfiguration == nil {
		return ErrAudioConfigNil
	}
	if uc.DisplayConfiguration == nil {
		return ErrDisplayConfigNil
	}
	if uc.GameplayConfiguration == nil {
		return ErrGameplayConfigNil
	}
	if isZeroDisplayConfiguration(settings.Display) {
		settings.Display = *uc.DisplayConfiguration
	}
	if !validDisplayConfiguration(settings.Display) {
		return fmt.Errorf(
			"%w: %s %dx%d",
			ErrDisplayConfig,
			settings.Display.Mode,
			settings.Display.WindowWidth,
			settings.Display.WindowHeight,
		)
	}

	if err := uc.saveConfiguration(func(candidate *viper.Viper) {
		setConfigurationSettings(candidate, settings)
	}); err != nil {
		return fmt.Errorf("saving configuration settings: %w", err)
	}

	*uc.AudioConfiguration = settings.Audio
	*uc.DisplayConfiguration = settings.Display
	*uc.GameplayConfiguration = settings.Gameplay
	uc.syncLegacyDisplayModel()

	return nil
}

func isZeroDisplayConfiguration(configuration DisplayConfiguration) bool {
	return configuration.Mode == "" && configuration.WindowWidth == 0 &&
		configuration.WindowHeight == 0 && configuration.TargetFPS == 0 &&
		!configuration.EnableFpsLimiting
}

func setConfigurationSettings(configuration *viper.Viper, settings ConfigurationSettings) {
	setAudioConfiguration(configuration, settings.Audio)
	setDisplayConfiguration(configuration, settings.Display)
	setGameplayConfiguration(configuration, settings.Gameplay)
}

func setAudioConfiguration(configuration *viper.Viper, audio AudioConfiguration) {
	configuration.Set("Audio.MusicVolume", audio.MusicVolume)
	configuration.Set("Audio.SoundEffectVolume", audio.SoundEffectVolume)
	configuration.Set("Audio.AmbientVolume", audio.AmbientVolume)
}

func setDisplayConfiguration(configuration *viper.Viper, displayConfiguration DisplayConfiguration) {
	configuration.Set("Display.Mode", displayConfiguration.Mode.String())
	configuration.Set("Display.WindowWidth", displayConfiguration.WindowWidth)
	configuration.Set("Display.WindowHeight", displayConfiguration.WindowHeight)
	configuration.Set("Display.TargetFPS", displayConfiguration.TargetFPS)
	configuration.Set("Display.EnableFpsLimiting", displayConfiguration.EnableFpsLimiting)
}

func setGameplayConfiguration(configuration *viper.Viper, gameplay GameplayConfiguration) {
	configuration.Set(gameplayBatchMerchantTheftKey, gameplay.BatchMerchantTheft)
	configuration.Set(
		"Gameplay.Combat.AllowRepeatedMonsterDivision",
		gameplay.AllowRepeatedMonsterDivision,
	)
	configuration.Set(gameplayShowConcealedTurnOrderSlotsKey, gameplay.ShowConcealedTurnOrderSlots)
	configuration.Set(
		gameplayPersistHiddenDungeonDiscoveriesKey,
		gameplay.PersistHiddenDungeonDiscoveries,
	)
	configuration.Set(gameplayShowSubduedUnlitDungeonsKey, gameplay.ShowSubduedUnlitDungeons)
	configuration.Set(gameplaySmoothSurfaceMovementKey, gameplay.SmoothSurfaceMovement)
	configuration.Set(gameplayFlexibleSextantUseKey, gameplay.FlexibleSextantUse)
}

func writeViperConfigAtomically(configuration *viper.Viper, configPath string) error {
	if configPath == "" {
		return ErrConfigFileUnknown
	}

	directory := filepath.Dir(configPath)
	temporary, err := os.CreateTemp(
		directory,
		"."+filepath.Base(configPath)+"-*.yaml",
	)
	if err != nil {
		return fmt.Errorf("creating temporary configuration: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		if cleanupErr := os.Remove(temporaryPath); cleanupErr != nil && !errors.Is(cleanupErr, os.ErrNotExist) {
			logging.Warn(logging.CategoryConfig, "Removing temporary configuration %s: %v", temporaryPath, cleanupErr)
		}
	}()
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("closing temporary configuration: %w", err)
	}
	if err := configuration.WriteConfigAs(temporaryPath); err != nil {
		return fmt.Errorf("writing temporary configuration: %w", err)
	}
	if fileInfo, statErr := os.Stat(configPath); statErr == nil {
		if err := os.Chmod(temporaryPath, fileInfo.Mode().Perm()); err != nil {
			return fmt.Errorf("preserving configuration permissions: %w", err)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("reading configuration permissions: %w", statErr)
	}
	if err := os.Rename(temporaryPath, configPath); err != nil {
		return fmt.Errorf("replacing configuration: %w", err)
	}

	return nil
}

// SaveLoggingConfiguration immediately persists current logging settings to config file
func (uc *UltimaVConfiguration) SaveLoggingConfiguration() error {
	if uc.LoggingConfiguration == nil {
		return errors.New("logging configuration is nil")
	}

	if err := uc.saveConfiguration(func(candidate *viper.Viper) {
		setLoggingConfiguration(candidate, *uc.LoggingConfiguration)
	}); err != nil {
		return fmt.Errorf("saving logging configuration: %w", err)
	}

	return nil
}

// SaveDebugConfiguration immediately persists current debug settings to config file
func (uc *UltimaVConfiguration) SaveDebugConfiguration() error {
	if uc.DebugConfiguration == nil {
		return ErrDebugConfigNil
	}

	if err := uc.saveConfiguration(func(candidate *viper.Viper) {
		setDebugConfiguration(candidate, *uc.DebugConfiguration)
	}); err != nil {
		return fmt.Errorf("saving debug configuration: %w", err)
	}

	return nil
}

// SaveGameplayConfiguration immediately persists current gameplay settings.
func (uc *UltimaVConfiguration) SaveGameplayConfiguration() error {
	if uc.GameplayConfiguration == nil {
		return ErrGameplayConfigNil
	}
	if err := uc.saveConfiguration(func(candidate *viper.Viper) {
		setGameplayConfiguration(candidate, *uc.GameplayConfiguration)
	}); err != nil {
		return fmt.Errorf("saving gameplay configuration: %w", err)
	}

	return nil
}

func setLoggingConfiguration(configuration *viper.Viper, logging LoggingConfiguration) {
	configuration.Set("Logging.Enabled", logging.Enabled)
	configuration.Set("Logging.StdoutEnabled", logging.StdoutEnabled)
	configuration.Set("Logging.MinLevel", logging.MinLevel)
	configuration.Set("Logging.CategoryStates", maps.Clone(logging.CategoryStates))
}

func setDebugConfiguration(configuration *viper.Viper, debug DebugConfiguration) {
	if debug.CombatDetailSet {
		configuration.Set("Debug.CombatDetailShown", debug.CombatDetailShown)
	}
	configuration.Set("Debug.CombatMovementSpeedFactor", debug.GetCombatMovementSpeedFactor())
}

// SetAllowRepeatedMonsterDivision updates and persists the repeated-division rule atomically.
func (uc *UltimaVConfiguration) SetAllowRepeatedMonsterDivision(enabled bool) error {
	if uc.GameplayConfiguration == nil {
		return ErrGameplayConfigNil
	}

	before := uc.GameplayConfiguration.AllowRepeatedMonsterDivision
	uc.GameplayConfiguration.AllowRepeatedMonsterDivision = enabled
	if err := uc.SaveGameplayConfiguration(); err != nil {
		uc.GameplayConfiguration.AllowRepeatedMonsterDivision = before

		return err
	}

	return nil
}

// SetPersistHiddenDungeonDiscoveries updates and persists hidden-discovery retention atomically.
func (uc *UltimaVConfiguration) SetPersistHiddenDungeonDiscoveries(enabled bool) error {
	if uc.GameplayConfiguration == nil {
		return ErrGameplayConfigNil
	}

	before := uc.GameplayConfiguration.PersistHiddenDungeonDiscoveries
	uc.GameplayConfiguration.PersistHiddenDungeonDiscoveries = enabled
	if err := uc.SaveGameplayConfiguration(); err != nil {
		uc.GameplayConfiguration.PersistHiddenDungeonDiscoveries = before

		return err
	}

	return nil
}

// SetShowSubduedUnlitDungeons updates and persists subdued unlit rendering atomically.
func (uc *UltimaVConfiguration) SetShowSubduedUnlitDungeons(enabled bool) error {
	if uc.GameplayConfiguration == nil {
		return ErrGameplayConfigNil
	}

	before := uc.GameplayConfiguration.ShowSubduedUnlitDungeons
	uc.GameplayConfiguration.ShowSubduedUnlitDungeons = enabled
	if err := uc.SaveGameplayConfiguration(); err != nil {
		uc.GameplayConfiguration.ShowSubduedUnlitDungeons = before

		return err
	}

	return nil
}

// SetSmoothSurfaceMovement updates and persists optional surface scrolling atomically.
func (uc *UltimaVConfiguration) SetSmoothSurfaceMovement(enabled bool) error {
	if uc.GameplayConfiguration == nil {
		return ErrGameplayConfigNil
	}
	before := uc.GameplayConfiguration.SmoothSurfaceMovement
	uc.GameplayConfiguration.SmoothSurfaceMovement = enabled
	if err := uc.SaveGameplayConfiguration(); err != nil {
		uc.GameplayConfiguration.SmoothSurfaceMovement = before

		return err
	}

	return nil
}

// SetFlexibleSextantUse updates and persists optional sextant navigation atomically.
func (uc *UltimaVConfiguration) SetFlexibleSextantUse(enabled bool) error {
	if uc.GameplayConfiguration == nil {
		return ErrGameplayConfigNil
	}
	before := uc.GameplayConfiguration.FlexibleSextantUse
	uc.GameplayConfiguration.FlexibleSextantUse = enabled
	if err := uc.SaveGameplayConfiguration(); err != nil {
		uc.GameplayConfiguration.FlexibleSextantUse = before

		return err
	}

	return nil
}

func (uc *UltimaVConfiguration) gameplayBooleanDefaultTrue(key string) bool {
	if !uc.source.IsSet(key) {
		return true
	}

	return uc.source.GetBool(key)
}

func (uc *UltimaVConfiguration) GetFileWithFullPath(fileName string) string {
	return path.Join(uc.SavedConfigData.DataFilePath, fileName)
}

func (uc *UltimaVConfiguration) GetAllNpcFilePaths() []string {
	return []string{
		uc.GetFileWithFullPath(files.TOWNE_NPC),
		uc.GetFileWithFullPath(files.DWELLING_NPC),
		uc.GetFileWithFullPath(files.CASTLE_NPC),
		uc.GetFileWithFullPath(files.KEEP_NPC),
	}
}

// func (uc *UltimaVConfiguration) GetAllTalkFilePaths() []string {
//	return []string{
//		path.Join(uc.SavedConfigData.DataFilePath, files.TOWNE_TLK),
//		path.Join(uc.SavedConfigData.DataFilePath, files.DWELLING_TLK),
//		path.Join(uc.SavedConfigData.DataFilePath, files.CASTLE_TLK),
//		path.Join(uc.SavedConfigData.DataFilePath, files.KEEP_TLK)}
//}

// GetAmbientLightRadius returns the ambient visibility radius with fallback to default.
// This is the radius for PlayerLightAmbient - minimal night vision when no torch/magic.
func (uc *UltimaVConfiguration) GetAmbientLightRadius() int {
	if uc.AmbientLightRadius < 1 {
		return 2 // Fallback default
	}

	return uc.AmbientLightRadius
}
