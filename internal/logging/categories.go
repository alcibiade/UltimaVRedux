package logging

// LogCategory constants for consistent categorization across the codebase
// These categories help organize and filter log messages by functional area
const (
	// CategoryMoongate for moongate system logs (rendering, animation, travel)
	CategoryMoongate = "moongate"

	// CategoryConversation for semantic NPC conversation processing
	CategoryConversation = "conversation"

	// CategoryAI for NPC AI, pathfinding, and behavior logs
	CategoryAI = "ai"

	// CategoryAudio for audio system, music, and sound effect logs
	CategoryAudio = "audio"

	// CategoryConfig for configuration loading and validation logs
	CategoryConfig = "config"

	// CategoryUI for user interface, widgets, and dialog logs
	CategoryUI = "ui"

	// CategoryGameState for game state management and save/load operations
	CategoryGameState = "gamestate"

	// CategoryCombat for combat system and battle mechanics
	CategoryCombat = "combat"

	// CategoryMovement for player and NPC movement, collision detection
	CategoryMovement = "movement"

	// CategoryInventory for inventory management and item operations
	CategoryInventory = "inventory"

	// CategorySprites for sprite loading, animation, and rendering
	CategorySprites = "sprites"

	// CategorySystem for low-level system operations and initialization
	CategorySystem = "system"

	// CategoryData for data loading, parsing, and reference management
	CategoryData = "data"

	// CategoryInput for input handling and user commands
	CategoryInput = "input"

	// CategoryRendering for rendering, graphics, and visual effects
	CategoryRendering = "rendering"

	// CategoryValidation for validation, bounds checking, and integrity checks
	CategoryValidation = "validation"

	// CategoryPerformance for performance monitoring and profiling
	CategoryPerformance = "performance"

	// CategoryLighthouse for lighthouse rotation and lighting system
	CategoryLighthouse = "lighthouse"
)

// GetAllCategories returns all available log categories for configuration UI
func GetAllCategories() []string {
	return []string{
		CategoryMoongate,
		CategoryConversation,
		CategoryAI,
		CategoryAudio,
		CategoryConfig,
		CategoryUI,
		CategoryGameState,
		CategoryCombat,
		CategoryMovement,
		CategoryInventory,
		CategorySprites,
		CategorySystem,
		CategoryData,
		CategoryInput,
		CategoryRendering,
		CategoryValidation,
		CategoryPerformance,
		CategoryLighthouse,
	}
}
