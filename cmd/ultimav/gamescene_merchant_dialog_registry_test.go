package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

type merchantRegistryTestDialog struct {
	name         string
	dismissed    bool
	dismissCount int
	dismissOrder *[]string
}

func (*merchantRegistryTestDialog) Draw(*ebiten.Image) {}

func (*merchantRegistryTestDialog) Update() {}

func (d *merchantRegistryTestDialog) Dismiss() {
	d.dismissed = true
	d.dismissCount++
	if d.dismissOrder != nil {
		*d.dismissOrder = append(*d.dismissOrder, d.name)
	}
}

func (d *merchantRegistryTestDialog) IsDismissed() bool { return d.dismissed }

func TestMerchantDialogRegistryCleansInReverseRegistrationOrderOnce(t *testing.T) {
	var registry merchantDialogRegistry
	var stack widgets.DialogStack
	order := make([]string, 0, 3)
	first := &merchantRegistryTestDialog{name: "first", dismissOrder: &order}
	second := &merchantRegistryTestDialog{name: "second", dismissOrder: &order}
	third := &merchantRegistryTestDialog{name: "third", dismissOrder: &order}

	registry.registerAndPush(&stack, first)
	registry.registerAndPush(&stack, second)
	registry.registerAndPush(&stack, third)
	registry.dismissAndRemoveAll(&stack)

	require.Equal(t, []string{"third", "second", "first"}, order)
	require.False(t, stack.HasOpenDialog())

	registry.dismissAndRemoveAll(&stack)
	require.Equal(t, []string{"third", "second", "first"}, order)
	require.Equal(t, 1, first.dismissCount)
	require.Equal(t, 1, second.dismissCount)
	require.Equal(t, 1, third.dismissCount)
}

func TestMerchantDialogRegistryRejectsInvalidRegistrations(t *testing.T) {
	tests := []struct {
		name     string
		register func(*merchantDialogRegistry, *widgets.DialogStack)
	}{
		{
			name: "nil stack",
			register: func(registry *merchantDialogRegistry, _ *widgets.DialogStack) {
				registry.registerAndPush(nil, &merchantRegistryTestDialog{})
			},
		},
		{
			name: "nil interface",
			register: func(registry *merchantDialogRegistry, stack *widgets.DialogStack) {
				var dialog merchantWorkflowOwnedDialog
				registry.registerAndPush(stack, dialog)
			},
		},
		{
			name: "typed nil",
			register: func(registry *merchantDialogRegistry, stack *widgets.DialogStack) {
				var dialog *merchantRegistryTestDialog
				registry.registerAndPush(stack, dialog)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var registry merchantDialogRegistry
			var stack widgets.DialogStack
			require.Panics(t, func() { test.register(&registry, &stack) })
		})
	}
}

func TestMerchantDialogRegistryRejectsDuplicateActiveInstance(t *testing.T) {
	var registry merchantDialogRegistry
	var stack widgets.DialogStack
	dialog := &merchantRegistryTestDialog{}

	registry.registerAndPush(&stack, dialog)

	require.Panics(t, func() { registry.registerAndPush(&stack, dialog) })
	registry.dismissAndRemoveAll(&stack)
}

func TestMerchantDialogRegistryRejectsRegistrationAfterCleanupAndReleasesReferences(t *testing.T) {
	var registry merchantDialogRegistry
	var stack widgets.DialogStack
	dialog := &merchantRegistryTestDialog{}

	registry.registerAndPush(&stack, dialog)
	registry.dismissAndRemoveAll(&stack)

	require.Nil(t, registry.owned)
	require.Nil(t, registry.seen)
	require.Panics(t, func() { registry.registerAndPush(&stack, dialog) })
}

func TestMerchantDialogRegistryToleratesRemovedChildrenAndPreservesDecoys(t *testing.T) {
	var registry merchantDialogRegistry
	var stack widgets.DialogStack
	bottomDecoy := &merchantRegistryTestDialog{name: "bottom decoy"}
	parent := &merchantRegistryTestDialog{name: "parent"}
	child := &merchantRegistryTestDialog{name: "child"}
	topDecoy := &merchantRegistryTestDialog{name: "top decoy"}

	stack.PushModalDialog(bottomDecoy)
	registry.registerAndPush(&stack, parent)
	registry.registerAndPush(&stack, child)
	stack.PushModalDialog(topDecoy)
	child.Dismiss()
	require.True(t, stack.RemoveWidgetInstance(child))

	registry.dismissAndRemoveAll(&stack)

	require.Equal(t, 1, child.dismissCount)
	require.Equal(t, 1, parent.dismissCount)
	require.Equal(t, 0, bottomDecoy.dismissCount)
	require.Equal(t, 0, topDecoy.dismissCount)
	require.True(t, stack.RemoveWidgetInstance(topDecoy))
	require.True(t, stack.RemoveWidgetInstance(bottomDecoy))
	require.False(t, stack.HasOpenDialog())
}
