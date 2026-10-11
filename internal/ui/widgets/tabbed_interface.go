package widgets

import (
	"image"
	stdcolor "image/color"

	"github.com/hajimehoshi/ebiten/v2"
	etext "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/bradhannah/UltimaVRedux/internal/text"
	"github.com/bradhannah/UltimaVRedux/pkg/color"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

// TabOrientation determines whether tabs are displayed horizontally (top) or vertically (left)
type TabOrientation int

const (
	// TabOrientationHorizontal displays tabs in a row at the top (default)
	TabOrientationHorizontal TabOrientation = iota
	// TabOrientationVertical displays tabs in a column on the left side
	TabOrientationVertical
)

// Tab represents a single tab in the tabbed interface
type Tab struct {
	// Tab identification
	Title   string // Display text for the tab
	Content Widget // Content widget to display when tab is active

	// Visual state
	IsActive  bool
	IsVisible bool
}

// TabbedInterface provides a tabbed interface for organizing multiple content areas.
// Supports keyboard navigation and tab switching with visual feedback.
type TabbedInterface struct {
	// Core properties
	tabs        []Tab
	activeIndex int

	// Display properties
	border    *Border
	font      *text.UltimaFont
	output    *text.Output
	placement sprites.PercentBasedPlacement

	// Input handling
	keyboard *input.Keyboard

	// Visual customization
	activeTabColor   stdcolor.Color
	inactiveTabColor stdcolor.Color
	tabTextColor     stdcolor.Color
	backgroundColor  stdcolor.Color

	// Layout properties
	tabHeight   float64        // Tab header height as percentage of total (horizontal mode)
	tabWidth    float64        // Tab panel width as percentage of total (vertical mode)
	orientation TabOrientation // Orientation of the tabs (horizontal/vertical)

	// Callback when tab is changed
	onTabChanged func(oldIndex, newIndex int, tab *Tab)

	// Display state
	hasFocus  bool
	isVisible bool

	// Reusable draw options for safety-first performance
	tabDrawOp     *ebiten.DrawImageOptions
	contentDrawOp *ebiten.DrawImageOptions
	textDrawOp    *ebiten.DrawImageOptions
}

// TabbedInterfaceConfig provides configuration for creating a tabbed interface
type TabbedInterfaceConfig struct {
	// Screen placement using percentage-based positioning
	Placement sprites.PercentBasedPlacement

	// Tab definitions
	Tabs []Tab // List of tabs to display

	// Initial state
	InitialActiveTab int // Index of initially active tab (default 0)

	// Visual customization (optional - safe defaults provided)
	ActiveTabColor   stdcolor.Color // Active tab background color
	InactiveTabColor stdcolor.Color // Inactive tab background color
	TabTextColor     stdcolor.Color // Tab text color
	BackgroundColor  stdcolor.Color // Content area background color
	TabHeight        float64        // Tab header height as percentage (default 0.1, horizontal mode)
	TabWidth         float64        // Tab panel width as percentage (default 0.2, vertical mode)
	Orientation      TabOrientation // Horizontal (tabs at top) or Vertical (tabs on left)

	// Input handling
	Keyboard *input.Keyboard // Keyboard handler for tab switching

	// Callback functions (optional)
	OnTabChanged func(oldIndex, newIndex int, tab *Tab) // Called when tab changes

	// Display options
	FontSize float64 // Font size for tab text (default 12)
}

const (
	tabbedInterfaceDefaultTabHeight = 0.1 // 10% of total height (horizontal mode)
	tabbedInterfaceDefaultTabWidth  = 0.2 // 20% of total width (vertical mode)
	tabbedInterfaceBorderWidth      = 301
	tabbedInterfaceMaxCharsPerTab   = 15
)

// tabbedInterfaceDefaultFontSize uses FontSizeXSmall (12) from typography system

// NewTabbedInterface creates a new tabbed interface widget with safe defaults and validation.
//
// The tabbed interface supports multiple content areas organized into tabs,
// keyboard navigation for tab switching, and customizable visual styling.
//
// Parameters:
//   - config: Configuration for the tabbed interface widget
//
// Returns a configured TabbedInterface that implements the Widget interface.
func NewTabbedInterface(config TabbedInterfaceConfig) *TabbedInterface {
	// Error handling: validate required inputs
	if config.Keyboard == nil {
		logging.Warn(logging.CategoryUI, "NewTabbedInterface called with nil keyboard, interface may not respond to input")
	}

	if len(config.Tabs) == 0 {
		logging.Warn(logging.CategoryUI, "NewTabbedInterface called with empty tabs list, adding placeholder tab")
		config.Tabs = []Tab{{Title: "Empty", Content: nil, IsVisible: true}}
	}

	// Validate initial active tab
	if config.InitialActiveTab < 0 || config.InitialActiveTab >= len(config.Tabs) {
		logging.Warn(logging.CategoryUI, "NewTabbedInterface InitialActiveTab %d out of range, using 0", config.InitialActiveTab)
		config.InitialActiveTab = 0
	}

	// Apply safe defaults
	if config.TabHeight <= 0 || config.TabHeight >= 1.0 {
		logging.Warn(logging.CategoryUI, "NewTabbedInterface invalid TabHeight %f, using default %f", config.TabHeight, tabbedInterfaceDefaultTabHeight)
		config.TabHeight = tabbedInterfaceDefaultTabHeight
	}

	if config.TabWidth <= 0 || config.TabWidth >= 1.0 {
		config.TabWidth = tabbedInterfaceDefaultTabWidth
	}

	if config.FontSize <= 0 {
		config.FontSize = float64(FontSizeXSmall) // Default to 12
	}

	// Apply default colors if not specified
	if config.ActiveTabColor == nil {
		config.ActiveTabColor = color.TabActiveBlue.RGBA() // Light blue
	}
	if config.InactiveTabColor == nil {
		config.InactiveTabColor = color.TabInactiveGray.RGBA() // Gray
	}
	if config.TabTextColor == nil {
		config.TabTextColor = color.TabTextWhite.RGBA() // White
	}
	if config.BackgroundColor == nil {
		config.BackgroundColor = color.TabBackgroundDark.RGBA() // Dark gray
	}

	widget := &TabbedInterface{
		tabs:             config.Tabs,
		activeIndex:      config.InitialActiveTab,
		placement:        config.Placement,
		keyboard:         config.Keyboard,
		activeTabColor:   config.ActiveTabColor,
		inactiveTabColor: config.InactiveTabColor,
		tabTextColor:     config.TabTextColor,
		backgroundColor:  config.BackgroundColor,
		tabHeight:        config.TabHeight,
		tabWidth:         config.TabWidth,
		orientation:      config.Orientation,
		onTabChanged:     config.OnTabChanged,
		hasFocus:         true,
		isVisible:        true,
	}

	// Mark initial active tab
	for i := range widget.tabs {
		widget.tabs[i].IsActive = i == widget.activeIndex
	}

	// Initialize display components using centralized typography system
	widget.font, widget.output = GetScaledFontWithOutput(FontConfig{
		Size:     StandardFontSize(config.FontSize),
		MaxLines: 1, // Single line for tab titles
		MaxChars: tabbedInterfaceMaxCharsPerTab,
	})

	// Create border for the tabbed interface
	widget.border = NewBorder(
		widget.placement,
		tabbedInterfaceBorderWidth,
		color.Black,
	)

	// Initialize reusable draw options for safety-first performance
	widget.tabDrawOp = &ebiten.DrawImageOptions{}
	widget.contentDrawOp = &ebiten.DrawImageOptions{}
	widget.textDrawOp = &ebiten.DrawImageOptions{}

	return widget
}

// Update processes keyboard input for tab navigation
func (t *TabbedInterface) Update() {
	if !t.isVisible || !t.hasFocus || t.keyboard == nil {
		return
	}

	// Use keyboard abstraction layer for all input detection
	// Prevents direct ebiten API calls, maintains abstraction layer principle

	// Navigation keys depend on orientation
	// Horizontal mode: Left/Right navigate tabs
	// Vertical mode: Up/Down navigate tabs
	var prevKey, nextKey ebiten.Key
	if t.orientation == TabOrientationVertical {
		prevKey = ebiten.KeyUp
		nextKey = ebiten.KeyDown
	} else {
		prevKey = ebiten.KeyLeft
		nextKey = ebiten.KeyRight
	}

	// Previous tab
	if t.keyboard.IsKeyPressedAndRegister(prevKey) {
		t.switchToPreviousTab()

		return
	}

	// Next tab
	if t.keyboard.IsKeyPressedAndRegister(nextKey) {
		t.switchToNextTab()

		return
	}

	// Tab key - next tab
	if t.keyboard.IsKeyPressedAndRegister(ebiten.KeyTab) {
		t.switchToNextTab()

		return
	}

	// Number keys 1-9 for direct tab selection
	for i := 0; i < 9 && i < len(t.tabs); i++ {
		key := ebiten.Key(int(ebiten.Key1) + i)
		if t.keyboard.IsKeyPressedAndRegister(key) {
			t.switchToTab(i)

			return
		}
	}

	// Update active content widget
	if t.activeIndex >= 0 && t.activeIndex < len(t.tabs) {
		activeTab := &t.tabs[t.activeIndex]
		if activeTab.Content != nil {
			activeTab.Content.Update()
		}
	}
}

// Draw renders the tabbed interface to the screen
func (t *TabbedInterface) Draw(screen *ebiten.Image) {
	// Error handling: validate screen parameter
	if screen == nil {
		logging.Warn(logging.CategoryUI, "TabbedInterface.Draw called with nil screen, skipping draw")

		return
	}

	if !t.isVisible {
		return
	}

	// Draw border and background
	t.border.DrawBackground(screen)

	// Calculate layout areas
	tabHeaderRect := t.getTabHeaderRect()
	contentRect := t.getContentRect()

	// Draw content area background
	t.drawContentBackground(screen, contentRect)

	// Draw tabs
	t.drawTabs(screen, tabHeaderRect)

	// Draw active content
	t.drawActiveContent(screen)

	// Draw border on top
	t.border.DrawBorder(screen)
}

// getTabHeaderRect calculates the rectangle for tab headers
func (t *TabbedInterface) getTabHeaderRect() image.Rectangle {
	totalRect := sprites.GetRectangleFromPercents(t.placement)

	if t.orientation == TabOrientationVertical {
		// Vertical mode: tabs on left side
		tabPanelWidth := int(float64(totalRect.Dx()) * t.tabWidth)

		return image.Rectangle{
			Min: image.Point{X: totalRect.Min.X + 2, Y: totalRect.Min.Y + 2},
			Max: image.Point{X: totalRect.Min.X + tabPanelWidth, Y: totalRect.Max.Y - 2},
		}
	}

	// Horizontal mode: tabs at top
	tabHeaderHeight := int(float64(totalRect.Dy()) * t.tabHeight)

	return image.Rectangle{
		Min: image.Point{X: totalRect.Min.X + 2, Y: totalRect.Min.Y + 2},
		Max: image.Point{X: totalRect.Max.X - 2, Y: totalRect.Min.Y + tabHeaderHeight},
	}
}

// getContentRect calculates the rectangle for tab content
func (t *TabbedInterface) getContentRect() image.Rectangle {
	totalRect := sprites.GetRectangleFromPercents(t.placement)

	if t.orientation == TabOrientationVertical {
		// Vertical mode: content to the right of tabs
		tabPanelWidth := int(float64(totalRect.Dx()) * t.tabWidth)

		return image.Rectangle{
			Min: image.Point{X: totalRect.Min.X + tabPanelWidth, Y: totalRect.Min.Y + 2},
			Max: image.Point{X: totalRect.Max.X - 2, Y: totalRect.Max.Y - 2},
		}
	}

	// Horizontal mode: content below tabs
	tabHeaderHeight := int(float64(totalRect.Dy()) * t.tabHeight)

	return image.Rectangle{
		Min: image.Point{X: totalRect.Min.X + 2, Y: totalRect.Min.Y + tabHeaderHeight},
		Max: image.Point{X: totalRect.Max.X - 2, Y: totalRect.Max.Y - 2},
	}
}

// drawContentBackground draws the content area background
func (t *TabbedInterface) drawContentBackground(screen *ebiten.Image, contentRect image.Rectangle) {
	for y := contentRect.Min.Y; y < contentRect.Max.Y; y++ {
		for x := contentRect.Min.X; x < contentRect.Max.X; x++ {
			screen.Set(x, y, t.backgroundColor)
		}
	}
}

// drawTabs draws all tab headers
func (t *TabbedInterface) drawTabs(screen *ebiten.Image, headerRect image.Rectangle) {
	if len(t.tabs) == 0 {
		return
	}

	// Calculate tab dimensions based on orientation
	var tabRect image.Rectangle
	var tabWidth, tabHeight int

	if t.orientation == TabOrientationVertical {
		// Vertical mode: tabs stacked vertically
		tabHeight = headerRect.Dy() / len(t.tabs)
		tabWidth = headerRect.Dx()
	} else {
		// Horizontal mode: tabs side by side
		tabWidth = headerRect.Dx() / len(t.tabs)
		tabHeight = headerRect.Dy()
	}

	for index, tab := range t.tabs {
		if !tab.IsVisible {
			continue
		}

		// Calculate tab rectangle based on orientation
		if t.orientation == TabOrientationVertical {
			tabRect = image.Rectangle{
				Min: image.Point{X: headerRect.Min.X, Y: headerRect.Min.Y + index*tabHeight},
				Max: image.Point{X: headerRect.Min.X + tabWidth, Y: headerRect.Min.Y + (index+1)*tabHeight},
			}
		} else {
			tabRect = image.Rectangle{
				Min: image.Point{X: headerRect.Min.X + index*tabWidth, Y: headerRect.Min.Y},
				Max: image.Point{X: headerRect.Min.X + (index+1)*tabWidth, Y: headerRect.Max.Y},
			}
		}

		// Draw tab background
		tabColor := t.inactiveTabColor
		if index == t.activeIndex {
			tabColor = t.activeTabColor
		}

		for y := tabRect.Min.Y; y < tabRect.Max.Y; y++ {
			for x := tabRect.Min.X; x < tabRect.Max.X; x++ {
				screen.Set(x, y, tabColor)
			}
		}

		// Draw tab border
		t.drawTabBorder(screen, tabRect, index == t.activeIndex)

		// Draw tab text
		t.drawTabText(screen, tabRect, tab.Title)
	}
}

// drawTabBorder draws a border around a tab
func (t *TabbedInterface) drawTabBorder(screen *ebiten.Image, tabRect image.Rectangle, isActive bool) {
	borderColor := color.TabBorderDefault.RGBA()
	if isActive {
		borderColor = color.TabBorderActive.RGBA()
	}

	// Top border
	for x := tabRect.Min.X; x < tabRect.Max.X; x++ {
		screen.Set(x, tabRect.Min.Y, borderColor)
	}

	// Left border
	for y := tabRect.Min.Y; y < tabRect.Max.Y; y++ {
		screen.Set(tabRect.Min.X, y, borderColor)
	}

	// Right border - skip for active tabs in vertical mode (connects to content)
	if !isActive || t.orientation != TabOrientationVertical {
		for y := tabRect.Min.Y; y < tabRect.Max.Y; y++ {
			screen.Set(tabRect.Max.X-1, y, borderColor)
		}
	}

	// Bottom border - skip for active tabs in horizontal mode (connects to content)
	if !isActive || t.orientation != TabOrientationHorizontal {
		for x := tabRect.Min.X; x < tabRect.Max.X; x++ {
			screen.Set(x, tabRect.Max.Y-1, borderColor)
		}
	}
}

// drawTabText draws the text for a tab
func (t *TabbedInterface) drawTabText(screen *ebiten.Image, tabRect image.Rectangle, title string) {
	// Center the text in the tab using Output pattern
	// Convert center point to top-left for DrawPanelContent
	centerX := tabRect.Min.X + tabRect.Dx()/2
	centerY := tabRect.Min.Y + tabRect.Dy()/2

	// Container width is the tab width
	containerWidth := tabRect.Dx()
	topLeftX := centerX - containerWidth/2

	// Build and render using Output pattern
	t.output.Reset()
	t.output.AppendLine().Text(title, t.tabTextColor).WithAlignment(etext.AlignCenter).Done()
	t.output.DrawPanelContent(screen, image.Pt(topLeftX, centerY), containerWidth, etext.AlignCenter)
}

// drawActiveContent draws the content of the active tab
func (t *TabbedInterface) drawActiveContent(screen *ebiten.Image) {
	if t.activeIndex < 0 || t.activeIndex >= len(t.tabs) {
		return
	}

	activeTab := &t.tabs[t.activeIndex]
	if activeTab.Content == nil {
		return
	}

	// Content widgets handle their own positioning within the content area
	activeTab.Content.Draw(screen)
}

// switchToPreviousTab switches to the previous visible tab
func (t *TabbedInterface) switchToPreviousTab() {
	for i := range len(t.tabs) {
		candidateIndex := (t.activeIndex - 1 - i + len(t.tabs)) % len(t.tabs)
		if t.tabs[candidateIndex].IsVisible {
			t.switchToTab(candidateIndex)

			break
		}
	}

	// If no switch occurred, stay on current tab
}

// switchToNextTab switches to the next visible tab
func (t *TabbedInterface) switchToNextTab() {
	for i := range len(t.tabs) {
		candidateIndex := (t.activeIndex + 1 + i) % len(t.tabs)
		if t.tabs[candidateIndex].IsVisible {
			t.switchToTab(candidateIndex)

			break
		}
	}

	// If no switch occurred, stay on current tab
}

// switchToTab switches to a specific tab by index
func (t *TabbedInterface) switchToTab(index int) {
	if index < 0 || index >= len(t.tabs) {
		logging.Warn(logging.CategoryUI, "TabbedInterface.switchToTab invalid index %d", index)

		return
	}

	if !t.tabs[index].IsVisible {
		logging.Warn(logging.CategoryUI, "TabbedInterface.switchToTab tab %d not visible", index)

		return
	}

	oldIndex := t.activeIndex

	// Mark old tab as inactive
	if oldIndex >= 0 && oldIndex < len(t.tabs) {
		t.tabs[oldIndex].IsActive = false
	}

	// Mark new tab as active
	t.activeIndex = index
	t.tabs[index].IsActive = true

	// Call callback if provided
	if t.onTabChanged != nil && oldIndex != index {
		t.onTabChanged(oldIndex, index, &t.tabs[index])
	}
}

// GetActiveTab returns the currently active tab (test accessor)
func (t *TabbedInterface) GetActiveTab() *Tab {
	if t.activeIndex >= 0 && t.activeIndex < len(t.tabs) {
		return &t.tabs[t.activeIndex]
	}

	return nil
}

// GetActiveIndex returns the index of the currently active tab (test accessor)
func (t *TabbedInterface) GetActiveIndex() int {
	return t.activeIndex
}

// SetActiveTab sets the active tab by index (test accessor)
func (t *TabbedInterface) SetActiveTab(index int) {
	t.switchToTab(index)
}

// GetTabCount returns the number of tabs (test accessor)
func (t *TabbedInterface) GetTabCount() int {
	return len(t.tabs)
}

// GetTab returns a tab by index (test accessor)
func (t *TabbedInterface) GetTab(index int) *Tab {
	if index >= 0 && index < len(t.tabs) {
		return &t.tabs[index]
	}

	return nil
}

// GetContentRect returns the content area rectangle for external layout calculations
func (t *TabbedInterface) GetContentRect() image.Rectangle {
	return t.getContentRect()
}

// GetOrientation returns the current tab orientation
func (t *TabbedInterface) GetOrientation() TabOrientation {
	return t.orientation
}

// AddTab adds a new tab to the interface
func (t *TabbedInterface) AddTab(tab Tab) {
	tab.IsVisible = true
	tab.IsActive = false
	t.tabs = append(t.tabs, tab)
}

// RemoveTab removes a tab by index
func (t *TabbedInterface) RemoveTab(index int) {
	if index < 0 || index >= len(t.tabs) {
		logging.Warn(logging.CategoryUI, "TabbedInterface.RemoveTab invalid index %d", index)

		return
	}

	// Store original active index
	originalActiveIndex := t.activeIndex

	// If removing active tab, select a new active tab
	if index == t.activeIndex {
		if len(t.tabs) > 1 {
			// Try to switch to next tab, or previous if at end
			newActiveIndex := index + 1
			if newActiveIndex >= len(t.tabs) {
				newActiveIndex = index - 1
			}

			// Set new active tab
			t.tabs[originalActiveIndex].IsActive = false
			t.tabs[newActiveIndex].IsActive = true
			t.activeIndex = newActiveIndex
		} else {
			t.activeIndex = -1 // No tabs left
		}
	}

	// Remove the tab from the slice
	copy(t.tabs[index:], t.tabs[index+1:])
	t.tabs = t.tabs[:len(t.tabs)-1]

	// Adjust active index if we removed a tab before the active tab
	if t.activeIndex > index {
		t.activeIndex--
	}
}

// SetTabVisible sets the visibility of a tab
func (t *TabbedInterface) SetTabVisible(index int, visible bool) {
	if index < 0 || index >= len(t.tabs) {
		logging.Warn(logging.CategoryUI, "TabbedInterface.SetTabVisible invalid index %d", index)

		return
	}

	t.tabs[index].IsVisible = visible

	// If hiding active tab, switch to another visible tab
	if !visible && index == t.activeIndex {
		t.switchToNextTab()
	}
}

// SetFocus sets whether the tabbed interface has keyboard focus
func (t *TabbedInterface) SetFocus(focused bool) {
	t.hasFocus = focused
}

// SetVisible sets whether the tabbed interface is visible
func (t *TabbedInterface) SetVisible(visible bool) {
	t.isVisible = visible
}

// IsVisible returns whether the tabbed interface is visible
func (t *TabbedInterface) IsVisible() bool {
	return t.isVisible
}

// SetColors updates the tabbed interface colors
func (t *TabbedInterface) SetColors(active, inactive, text, background stdcolor.Color) {
	if active != nil {
		t.activeTabColor = active
	}
	if inactive != nil {
		t.inactiveTabColor = inactive
	}
	if text != nil {
		t.tabTextColor = text
	}
	if background != nil {
		t.backgroundColor = background
	}
}

// SetOnTabChanged sets the callback function for tab changes
func (t *TabbedInterface) SetOnTabChanged(callback func(oldIndex, newIndex int, tab *Tab)) {
	t.onTabChanged = callback
}
