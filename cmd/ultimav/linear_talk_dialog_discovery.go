package main

import "github.com/bradhannah/UltimaVRedux/internal/conversation"

func (d *LinearTalkDialog) isMounted() bool {
	for _, dialog := range d.gameScene.dialogStack.Dialogs() {
		if dialog == d {
			return true
		}
	}

	return false
}

// mount commits only the prefix already inserted during dormant construction.
func (d *LinearTalkDialog) mount() {
	if d.closed || !d.isMounted() {
		panic("conversation mount requires a live owned dialog")
	}
	d.mounted = true
	d.publishPresentedExchange()
	d.commitPresentedKnowledge()
}

// Close cancels any unseen response tail before releasing the owned dialog.
func (d *LinearTalkDialog) Close() {
	d.discardPendingKnowledge()
	d.gameScene.dialogStack.RemoveWidgetInstance(d)
}

func (d *LinearTalkDialog) discardPendingKnowledge() {
	d.closed = true
	d.pendingOutput = ""
	d.pendingSegments = nil
	d.isPausing = false
	if d.engine != nil {
		d.engine.Stop()
	}
	if d.callbacks != nil {
		d.callbacks.pendingKnowledge = nil
	}
}

func (d *LinearTalkDialog) commitPresentedKnowledge() {
	if d.closed || !d.mounted || d.currentResponse == nil || d.callbacks == nil {
		return
	}
	if !d.isMounted() {
		d.discardPendingKnowledge()

		return
	}
	d.callbacks.commitPresentedKnowledge(conversation.DiscoveryBoundary{
		Response: d.currentResponse.Serial, SourceOffset: d.presentedSourceBytes,
	})
}
