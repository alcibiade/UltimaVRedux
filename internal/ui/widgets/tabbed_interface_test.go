package widgets

import (
	stdcolor "image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

// mockWidget is a simple widget for testing tab content
type mockWidget struct {
	updateCalled bool
	drawCalled   bool
}

func (m *mockWidget) Update() {
	m.updateCalled = true
}

func (m *mockWidget) Draw(*ebiten.Image) {
	m.drawCalled = true
}

func TestNewTabbedInterface(t *testing.T) {
	keyboard := newWidgetTestKeyboard(100)

	t.Run("creates tabbed interface with valid config", func(t *testing.T) {
		mockContent1 := &mockWidget{}
		mockContent2 := &mockWidget{}

		config := TabbedInterfaceConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.1,
				EndPercentX:   0.9,
				StartPercentY: 0.1,
				EndPercentY:   0.9,
			},
			Tabs: []Tab{
				{Title: "Stats", Content: mockContent1, IsVisible: true},
				{Title: "Inventory", Content: mockContent2, IsVisible: true},
			},
			InitialActiveTab: 0,
			Keyboard:         keyboard,
			TabHeight:        0.15,
		}

		tabInterface := NewTabbedInterface(config)

		assert.NotNil(t, tabInterface)
		assert.Equal(t, 2, len(tabInterface.tabs))
		assert.Equal(t, 0, tabInterface.activeIndex)
		assert.True(t, tabInterface.tabs[0].IsActive)
		assert.False(t, tabInterface.tabs[1].IsActive)
		assert.Equal(t, 0.15, tabInterface.tabHeight)
		assert.True(t, tabInterface.hasFocus)
		assert.True(t, tabInterface.isVisible)
	})

	t.Run("handles nil keyboard gracefully", func(t *testing.T) {
		config := TabbedInterfaceConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.1,
				EndPercentX:   0.9,
				StartPercentY: 0.1,
				EndPercentY:   0.9,
			},
			Tabs: []Tab{
				{Title: "Test", Content: nil, IsVisible: true},
			},
			Keyboard: nil,
		}

		tabInterface := NewTabbedInterface(config)
		assert.NotNil(t, tabInterface)
		assert.Nil(t, tabInterface.keyboard)
	})

	t.Run("handles empty tabs list", func(t *testing.T) {
		config := TabbedInterfaceConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.1,
				EndPercentX:   0.9,
				StartPercentY: 0.1,
				EndPercentY:   0.9,
			},
			Tabs:     []Tab{}, // Empty
			Keyboard: keyboard,
		}

		tabInterface := NewTabbedInterface(config)
		assert.NotNil(t, tabInterface)
		assert.Equal(t, 1, len(tabInterface.tabs)) // Placeholder added
		assert.Equal(t, "Empty", tabInterface.tabs[0].Title)
	})

	t.Run("handles invalid initial active tab", func(t *testing.T) {
		config := TabbedInterfaceConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.1,
				EndPercentX:   0.9,
				StartPercentY: 0.1,
				EndPercentY:   0.9,
			},
			Tabs: []Tab{
				{Title: "Tab1", Content: nil, IsVisible: true},
				{Title: "Tab2", Content: nil, IsVisible: true},
			},
			InitialActiveTab: 5, // Out of bounds
			Keyboard:         keyboard,
		}

		tabInterface := NewTabbedInterface(config)
		assert.Equal(t, 0, tabInterface.activeIndex) // Reset to valid index
		assert.True(t, tabInterface.tabs[0].IsActive)
	})

	t.Run("handles invalid tab height", func(t *testing.T) {
		config := TabbedInterfaceConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.1,
				EndPercentX:   0.9,
				StartPercentY: 0.1,
				EndPercentY:   0.9,
			},
			Tabs: []Tab{
				{Title: "Test", Content: nil, IsVisible: true},
			},
			TabHeight: 1.5, // Invalid - over 100%
			Keyboard:  keyboard,
		}

		tabInterface := NewTabbedInterface(config)
		assert.Equal(t, 0.1, tabInterface.tabHeight) // Default value
	})

	t.Run("uses custom colors when provided", func(t *testing.T) {
		customActive := stdcolor.RGBA{R: 255, G: 0, B: 0, A: 255}
		customInactive := stdcolor.RGBA{R: 0, G: 255, B: 0, A: 255}
		customText := stdcolor.RGBA{R: 0, G: 0, B: 255, A: 255}
		customBg := stdcolor.RGBA{R: 128, G: 128, B: 128, A: 255}

		config := TabbedInterfaceConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.1,
				EndPercentX:   0.9,
				StartPercentY: 0.1,
				EndPercentY:   0.9,
			},
			Tabs: []Tab{
				{Title: "Test", Content: nil, IsVisible: true},
			},
			ActiveTabColor:   customActive,
			InactiveTabColor: customInactive,
			TabTextColor:     customText,
			BackgroundColor:  customBg,
			Keyboard:         keyboard,
		}

		tabInterface := NewTabbedInterface(config)
		assert.Equal(t, customActive, tabInterface.activeTabColor)
		assert.Equal(t, customInactive, tabInterface.inactiveTabColor)
		assert.Equal(t, customText, tabInterface.tabTextColor)
		assert.Equal(t, customBg, tabInterface.backgroundColor)
	})
}

func TestTabbedInterfaceNavigation(t *testing.T) {
	keyboard := newWidgetTestKeyboard(100)

	createTestInterface := func() *TabbedInterface {
		config := TabbedInterfaceConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.1,
				EndPercentX:   0.9,
				StartPercentY: 0.1,
				EndPercentY:   0.9,
			},
			Tabs: []Tab{
				{Title: "Tab1", Content: &mockWidget{}, IsVisible: true},
				{Title: "Tab2", Content: &mockWidget{}, IsVisible: true},
				{Title: "Tab3", Content: &mockWidget{}, IsVisible: true},
			},
			InitialActiveTab: 0,
			TabHeight:        0.15,
			Keyboard:         keyboard,
		}
		return NewTabbedInterface(config)
	}

	t.Run("switches to next tab", func(t *testing.T) {
		tabInterface := createTestInterface()
		assert.Equal(t, 0, tabInterface.GetActiveIndex())

		tabInterface.switchToNextTab()
		assert.Equal(t, 1, tabInterface.GetActiveIndex())
		assert.True(t, tabInterface.tabs[1].IsActive)
		assert.False(t, tabInterface.tabs[0].IsActive)
	})

	t.Run("switches to previous tab", func(t *testing.T) {
		tabInterface := createTestInterface()
		tabInterface.SetActiveTab(2)
		assert.Equal(t, 2, tabInterface.GetActiveIndex())

		tabInterface.switchToPreviousTab()
		assert.Equal(t, 1, tabInterface.GetActiveIndex())
		assert.True(t, tabInterface.tabs[1].IsActive)
		assert.False(t, tabInterface.tabs[2].IsActive)
	})

	t.Run("wraps around at boundaries", func(t *testing.T) {
		tabInterface := createTestInterface()

		// From first tab to last tab (previous)
		tabInterface.switchToPreviousTab()
		assert.Equal(t, 2, tabInterface.GetActiveIndex())

		// From last tab to first tab (next)
		tabInterface.switchToNextTab()
		assert.Equal(t, 0, tabInterface.GetActiveIndex())
	})

	t.Run("switches to specific tab", func(t *testing.T) {
		tabInterface := createTestInterface()

		tabInterface.SetActiveTab(2)
		assert.Equal(t, 2, tabInterface.GetActiveIndex())
		assert.True(t, tabInterface.tabs[2].IsActive)
		assert.False(t, tabInterface.tabs[0].IsActive)
		assert.False(t, tabInterface.tabs[1].IsActive)
	})

	t.Run("ignores invalid tab indices", func(t *testing.T) {
		tabInterface := createTestInterface()
		originalIndex := tabInterface.GetActiveIndex()

		tabInterface.SetActiveTab(-1)
		assert.Equal(t, originalIndex, tabInterface.GetActiveIndex())

		tabInterface.SetActiveTab(10)
		assert.Equal(t, originalIndex, tabInterface.GetActiveIndex())
	})
}

func TestTabbedInterfaceVisibilityHandling(t *testing.T) {
	keyboard := newWidgetTestKeyboard(100)

	t.Run("skips invisible tabs during navigation", func(t *testing.T) {
		config := TabbedInterfaceConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.1,
				EndPercentX:   0.9,
				StartPercentY: 0.1,
				EndPercentY:   0.9,
			},
			Tabs: []Tab{
				{Title: "Tab1", Content: &mockWidget{}, IsVisible: true},
				{Title: "Tab2", Content: &mockWidget{}, IsVisible: false}, // Hidden
				{Title: "Tab3", Content: &mockWidget{}, IsVisible: true},
			},
			InitialActiveTab: 0,
			TabHeight:        0.15,
			Keyboard:         keyboard,
		}

		tabInterface := NewTabbedInterface(config)
		assert.Equal(t, 0, tabInterface.GetActiveIndex())

		// Should skip hidden tab 1 and go to tab 2
		tabInterface.switchToNextTab()
		assert.Equal(t, 2, tabInterface.GetActiveIndex())

		// Should skip hidden tab 1 and go to tab 0
		tabInterface.switchToPreviousTab()
		assert.Equal(t, 0, tabInterface.GetActiveIndex())
	})

	t.Run("switches tab when active tab becomes invisible", func(t *testing.T) {
		config := TabbedInterfaceConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.1,
				EndPercentX:   0.9,
				StartPercentY: 0.1,
				EndPercentY:   0.9,
			},
			Tabs: []Tab{
				{Title: "Tab1", Content: &mockWidget{}, IsVisible: true},
				{Title: "Tab2", Content: &mockWidget{}, IsVisible: true},
			},
			InitialActiveTab: 0,
			TabHeight:        0.15,
			Keyboard:         keyboard,
		}

		tabInterface := NewTabbedInterface(config)
		assert.Equal(t, 0, tabInterface.GetActiveIndex())

		// Hide the active tab - should switch to next visible tab
		tabInterface.SetTabVisible(0, false)
		assert.Equal(t, 1, tabInterface.GetActiveIndex())
	})

	t.Run("ignores switch to invisible tab", func(t *testing.T) {
		config := TabbedInterfaceConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.1,
				EndPercentX:   0.9,
				StartPercentY: 0.1,
				EndPercentY:   0.9,
			},
			Tabs: []Tab{
				{Title: "Tab1", Content: &mockWidget{}, IsVisible: true},
				{Title: "Tab2", Content: &mockWidget{}, IsVisible: false},
			},
			InitialActiveTab: 0,
			TabHeight:        0.15,
			Keyboard:         keyboard,
		}

		tabInterface := NewTabbedInterface(config)
		originalIndex := tabInterface.GetActiveIndex()

		tabInterface.SetActiveTab(1)                                  // Try to switch to invisible tab
		assert.Equal(t, originalIndex, tabInterface.GetActiveIndex()) // Should remain unchanged
	})
}

func TestTabbedInterfaceTabManagement(t *testing.T) {
	keyboard := newWidgetTestKeyboard(100)

	createTestInterface := func() *TabbedInterface {
		config := TabbedInterfaceConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.1,
				EndPercentX:   0.9,
				StartPercentY: 0.1,
				EndPercentY:   0.9,
			},
			Tabs: []Tab{
				{Title: "Tab1", Content: &mockWidget{}, IsVisible: true},
				{Title: "Tab2", Content: &mockWidget{}, IsVisible: true},
			},
			InitialActiveTab: 0,
			TabHeight:        0.15,
			Keyboard:         keyboard,
		}
		return NewTabbedInterface(config)
	}

	t.Run("gets tab information correctly", func(t *testing.T) {
		tabInterface := createTestInterface()

		assert.Equal(t, 2, tabInterface.GetTabCount())

		activeTab := tabInterface.GetActiveTab()
		assert.NotNil(t, activeTab)
		assert.Equal(t, "Tab1", activeTab.Title)
		assert.True(t, activeTab.IsActive)

		tab := tabInterface.GetTab(1)
		assert.NotNil(t, tab)
		assert.Equal(t, "Tab2", tab.Title)
		assert.False(t, tab.IsActive)

		invalidTab := tabInterface.GetTab(10)
		assert.Nil(t, invalidTab)
	})

	t.Run("adds new tab", func(t *testing.T) {
		tabInterface := createTestInterface()
		originalCount := tabInterface.GetTabCount()

		newTab := Tab{Title: "NewTab", Content: &mockWidget{}}
		tabInterface.AddTab(newTab)

		assert.Equal(t, originalCount+1, tabInterface.GetTabCount())

		addedTab := tabInterface.GetTab(originalCount)
		assert.NotNil(t, addedTab)
		assert.Equal(t, "NewTab", addedTab.Title)
		assert.True(t, addedTab.IsVisible)
		assert.False(t, addedTab.IsActive)
	})

	t.Run("removes tab correctly", func(t *testing.T) {
		tabInterface := createTestInterface()
		originalCount := tabInterface.GetTabCount()

		// Remove inactive tab
		tabInterface.RemoveTab(1)
		assert.Equal(t, originalCount-1, tabInterface.GetTabCount())
		assert.Equal(t, 0, tabInterface.GetActiveIndex()) // Active index unchanged
	})

	t.Run("removes active tab and switches to another", func(t *testing.T) {
		config := TabbedInterfaceConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.1,
				EndPercentX:   0.9,
				StartPercentY: 0.1,
				EndPercentY:   0.9,
			},
			Tabs: []Tab{
				{Title: "Tab1", Content: &mockWidget{}, IsVisible: true},
				{Title: "Tab2", Content: &mockWidget{}, IsVisible: true},
				{Title: "Tab3", Content: &mockWidget{}, IsVisible: true},
			},
			InitialActiveTab: 1, // Start with middle tab active
			TabHeight:        0.15,
			Keyboard:         keyboard,
		}

		tabInterface := NewTabbedInterface(config)
		assert.Equal(t, 1, tabInterface.GetActiveIndex())

		// Remove active tab - should switch to next tab
		tabInterface.RemoveTab(1)
		assert.Equal(t, 2, tabInterface.GetTabCount())
		assert.Equal(t, 1, tabInterface.GetActiveIndex()) // New tab at index 1 (was index 2)
		assert.Equal(t, "Tab3", tabInterface.GetActiveTab().Title)
	})

	t.Run("handles removing last tab", func(t *testing.T) {
		config := TabbedInterfaceConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.1,
				EndPercentX:   0.9,
				StartPercentY: 0.1,
				EndPercentY:   0.9,
			},
			Tabs: []Tab{
				{Title: "OnlyTab", Content: &mockWidget{}, IsVisible: true},
			},
			InitialActiveTab: 0,
			TabHeight:        0.15,
			Keyboard:         keyboard,
		}

		tabInterface := NewTabbedInterface(config)
		tabInterface.RemoveTab(0)

		assert.Equal(t, 0, tabInterface.GetTabCount())
		assert.Equal(t, -1, tabInterface.GetActiveIndex())
		assert.Nil(t, tabInterface.GetActiveTab())
	})

	t.Run("ignores invalid remove operations", func(t *testing.T) {
		tabInterface := createTestInterface()
		originalCount := tabInterface.GetTabCount()

		tabInterface.RemoveTab(-1)
		assert.Equal(t, originalCount, tabInterface.GetTabCount())

		tabInterface.RemoveTab(10)
		assert.Equal(t, originalCount, tabInterface.GetTabCount())
	})
}

func TestTabbedInterfaceCallbacks(t *testing.T) {
	keyboard := newWidgetTestKeyboard(100)

	t.Run("calls tab changed callback", func(t *testing.T) {
		var callbackOldIndex, callbackNewIndex int
		var callbackTab *Tab
		callbackCalled := false

		config := TabbedInterfaceConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.1,
				EndPercentX:   0.9,
				StartPercentY: 0.1,
				EndPercentY:   0.9,
			},
			Tabs: []Tab{
				{Title: "Tab1", Content: &mockWidget{}, IsVisible: true},
				{Title: "Tab2", Content: &mockWidget{}, IsVisible: true},
			},
			InitialActiveTab: 0,
			TabHeight:        0.15,
			Keyboard:         keyboard,
			OnTabChanged: func(oldIndex, newIndex int, tab *Tab) {
				callbackCalled = true
				callbackOldIndex = oldIndex
				callbackNewIndex = newIndex
				callbackTab = tab
			},
		}

		tabInterface := NewTabbedInterface(config)
		tabInterface.SetActiveTab(1)

		assert.True(t, callbackCalled)
		assert.Equal(t, 0, callbackOldIndex)
		assert.Equal(t, 1, callbackNewIndex)
		assert.NotNil(t, callbackTab)
		assert.Equal(t, "Tab2", callbackTab.Title)
	})

	t.Run("doesn't call callback when switching to same tab", func(t *testing.T) {
		callbackCalled := false

		config := TabbedInterfaceConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.1,
				EndPercentX:   0.9,
				StartPercentY: 0.1,
				EndPercentY:   0.9,
			},
			Tabs: []Tab{
				{Title: "Tab1", Content: &mockWidget{}, IsVisible: true},
			},
			InitialActiveTab: 0,
			TabHeight:        0.15,
			Keyboard:         keyboard,
			OnTabChanged: func(oldIndex, newIndex int, tab *Tab) {
				callbackCalled = true
			},
		}

		tabInterface := NewTabbedInterface(config)
		tabInterface.SetActiveTab(0) // Switch to same tab

		assert.False(t, callbackCalled)
	})

	t.Run("handles nil callback gracefully", func(t *testing.T) {
		config := TabbedInterfaceConfig{
			Placement: sprites.PercentBasedPlacement{
				StartPercentX: 0.1,
				EndPercentX:   0.9,
				StartPercentY: 0.1,
				EndPercentY:   0.9,
			},
			Tabs: []Tab{
				{Title: "Tab1", Content: &mockWidget{}, IsVisible: true},
				{Title: "Tab2", Content: &mockWidget{}, IsVisible: true},
			},
			InitialActiveTab: 0,
			TabHeight:        0.15,
			Keyboard:         keyboard,
			OnTabChanged:     nil, // No callback
		}

		tabInterface := NewTabbedInterface(config)

		// Should not panic
		tabInterface.SetActiveTab(1)
		assert.Equal(t, 1, tabInterface.GetActiveIndex())
	})
}

func TestTabbedInterfaceDisplay(t *testing.T) {
	keyboard := newWidgetTestKeyboard(100)

	tabInterface := NewTabbedInterface(TabbedInterfaceConfig{
		Placement: sprites.PercentBasedPlacement{
			StartPercentX: 0.2,
			EndPercentX:   0.8,
			StartPercentY: 0.2,
			EndPercentY:   0.8,
		},
		Tabs: []Tab{
			{Title: "Tab1", Content: &mockWidget{}, IsVisible: true},
		},
		Keyboard: keyboard,
	})

	t.Run("manages visibility correctly", func(t *testing.T) {
		assert.True(t, tabInterface.IsVisible())

		tabInterface.SetVisible(false)
		assert.False(t, tabInterface.IsVisible())

		tabInterface.SetVisible(true)
		assert.True(t, tabInterface.IsVisible())
	})

	t.Run("manages focus correctly", func(t *testing.T) {
		assert.True(t, tabInterface.hasFocus) // Default focused

		tabInterface.SetFocus(false)
		assert.False(t, tabInterface.hasFocus)

		tabInterface.SetFocus(true)
		assert.True(t, tabInterface.hasFocus)
	})

	t.Run("updates colors correctly", func(t *testing.T) {
		newActive := stdcolor.RGBA{R: 255, G: 0, B: 0, A: 255}
		newInactive := stdcolor.RGBA{R: 0, G: 255, B: 0, A: 255}
		newText := stdcolor.RGBA{R: 0, G: 0, B: 255, A: 255}
		newBg := stdcolor.RGBA{R: 128, G: 128, B: 128, A: 255}

		tabInterface.SetColors(newActive, newInactive, newText, newBg)

		assert.Equal(t, newActive, tabInterface.activeTabColor)
		assert.Equal(t, newInactive, tabInterface.inactiveTabColor)
		assert.Equal(t, newText, tabInterface.tabTextColor)
		assert.Equal(t, newBg, tabInterface.backgroundColor)
	})

	t.Run("handles nil colors in SetColors", func(t *testing.T) {
		originalActive := tabInterface.activeTabColor
		originalInactive := tabInterface.inactiveTabColor
		originalText := tabInterface.tabTextColor
		originalBg := tabInterface.backgroundColor

		tabInterface.SetColors(nil, nil, nil, nil)

		// Colors should remain unchanged when nil
		assert.Equal(t, originalActive, tabInterface.activeTabColor)
		assert.Equal(t, originalInactive, tabInterface.inactiveTabColor)
		assert.Equal(t, originalText, tabInterface.tabTextColor)
		assert.Equal(t, originalBg, tabInterface.backgroundColor)
	})

	t.Run("updates callback function", func(t *testing.T) {
		callbackCalled := false
		newCallback := func(oldIndex, newIndex int, tab *Tab) {
			callbackCalled = true
		}

		tabInterface.SetOnTabChanged(newCallback)
		assert.NotNil(t, tabInterface.onTabChanged)

		// Test that new callback works
		tabInterface.AddTab(Tab{Title: "Test", Content: &mockWidget{}, IsVisible: true})
		tabInterface.SetActiveTab(1)
		assert.True(t, callbackCalled)
	})
}

func TestTabbedInterfaceDraw(t *testing.T) {
	keyboard := newWidgetTestKeyboard(100)

	mockContent := &mockWidget{}
	tabInterface := NewTabbedInterface(TabbedInterfaceConfig{
		Placement: sprites.PercentBasedPlacement{
			StartPercentX: 0.2,
			EndPercentX:   0.8,
			StartPercentY: 0.2,
			EndPercentY:   0.8,
		},
		Tabs: []Tab{
			{Title: "Test", Content: mockContent, IsVisible: true},
		},
		Keyboard: keyboard,
	})

	t.Run("draws when visible and calls content draw", func(t *testing.T) {
		screen := ebiten.NewImage(800, 600)
		tabInterface.SetVisible(true)

		// Reset mock widget state
		mockContent.drawCalled = false

		// Should not panic and should call content draw
		tabInterface.Draw(screen)

		assert.True(t, mockContent.drawCalled)
	})

	t.Run("skips drawing when invisible", func(t *testing.T) {
		screen := ebiten.NewImage(800, 600)
		tabInterface.SetVisible(false)

		// Reset mock widget state
		mockContent.drawCalled = false

		// Should not panic and should skip drawing
		tabInterface.Draw(screen)

		// Content draw should not be called when interface is invisible
		assert.False(t, mockContent.drawCalled)
	})
}

func TestTabbedInterfaceUpdate(t *testing.T) {
	keyboard := newWidgetTestKeyboard(100)

	mockContent := &mockWidget{}
	tabInterface := NewTabbedInterface(TabbedInterfaceConfig{
		Placement: sprites.PercentBasedPlacement{
			StartPercentX: 0.1,
			EndPercentX:   0.9,
			StartPercentY: 0.1,
			EndPercentY:   0.9,
		},
		Tabs: []Tab{
			{Title: "Tab1", Content: mockContent, IsVisible: true},
			{Title: "Tab2", Content: &mockWidget{}, IsVisible: true},
		},
		InitialActiveTab: 0,
		Keyboard:         keyboard,
	})

	t.Run("updates active content when visible and focused", func(t *testing.T) {
		tabInterface.SetVisible(true)
		tabInterface.SetFocus(true)

		// Reset mock widget state
		mockContent.updateCalled = false

		tabInterface.Update()

		assert.True(t, mockContent.updateCalled)
	})

	t.Run("skips update when invisible", func(t *testing.T) {
		tabInterface.SetVisible(false)
		tabInterface.SetFocus(true)

		// Reset mock widget state
		mockContent.updateCalled = false

		tabInterface.Update()

		assert.False(t, mockContent.updateCalled)
	})

	t.Run("skips update when not focused", func(t *testing.T) {
		tabInterface.SetVisible(true)
		tabInterface.SetFocus(false)

		// Reset mock widget state
		mockContent.updateCalled = false

		tabInterface.Update()

		assert.False(t, mockContent.updateCalled)
	})

}
