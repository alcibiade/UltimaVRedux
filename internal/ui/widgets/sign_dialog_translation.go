package widgets

// IsTranslated reports only this reading dialog's transient presentation mode.
func (d *SignDialog) IsTranslated() bool { return d.translated }

// SetTranslated changes the local view without modifying the encoded inscription.
func (d *SignDialog) SetTranslated(translated bool) {
	if d.translated == translated {
		return
	}
	// REDUX DEVIATION: Original sign output renders the encoded inscription; ref:SIGN-0008.
	// Redux offers an optional, per-reading English view for readability.
	d.translated = translated
	if d.buffer != nil {
		d.buffer.Deallocate()
		d.buffer = nil
	}
	d.ensureLayout()
}
