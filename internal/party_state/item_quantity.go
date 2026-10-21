package party_state

const (
	// ItemQuantityLargeMax is the authoritative maximum for large inventory stacks.
	ItemQuantityLargeMax uint16 = 9999
	// ItemQuantitySmallMax is the authoritative maximum for small inventory stacks.
	ItemQuantitySmallMax uint16 = 99
)

// ItemQuantity is a capped inventory stack; decrements report whether enough was held.
type ItemQuantity interface {
	IncrementByOne()
	DecrementByOne() bool
	IncrementBy(incBy uint16)
	DecrementBy(incBy uint16) bool
	Set(quantity uint16)
	Get() uint16
	HasSome() bool
	SetHasOne()
}

// ItemQuantityLarge is a stack capped at ItemQuantityLargeMax (gold, food, scrolls).
type ItemQuantityLarge struct {
	quantity uint16
	activity QuantityActivity
}

// SetHasOne sets the stack to exactly one.
func (l *ItemQuantityLarge) SetHasOne() {
	l.Set(1)
}

// HasSome reports whether the stack is non-empty.
func (l *ItemQuantityLarge) HasSome() bool {
	return l.Get() > 0
}

// ItemQuantitySmall is a stack capped at ItemQuantitySmallMax (items, reagents, keys).
type ItemQuantitySmall struct {
	quantity uint16
	activity QuantityActivity
}

// SetHasOne sets the stack to exactly one.
func (s *ItemQuantitySmall) SetHasOne() {
	s.activity.record(s.quantity, 1)
	s.quantity = 1
}

// HasSome reports whether the stack is non-empty; a nil stack is empty.
func (s *ItemQuantitySmall) HasSome() bool {
	if s == nil {
		return false
	}

	return s.Get() > 0
}

// Get returns the held quantity.
func (l *ItemQuantityLarge) Get() uint16 {
	return l.quantity
}

// Set stores the quantity, clamped to ItemQuantityLargeMax.
func (l *ItemQuantityLarge) Set(quantity uint16) {
	next := min(quantity, ItemQuantityLargeMax)
	l.activity.record(l.quantity, next)
	l.quantity = next
}

// Get returns the held quantity; a nil stack holds zero.
func (s *ItemQuantitySmall) Get() uint16 {
	if s == nil {
		return 0
	}

	return s.quantity
}

// Set stores the quantity, clamped to ItemQuantitySmallMax; a nil stack ignores it.
func (s *ItemQuantitySmall) Set(quantity uint16) {
	if s == nil {
		return
	}
	next := min(quantity, ItemQuantitySmallMax)
	s.activity.record(s.quantity, next)
	s.quantity = next
}

func incBy(currentQuantity uint16, incBy uint16, maxQuantity uint16) uint16 {
	if incBy >= maxQuantity-currentQuantity {
		return maxQuantity
	}

	return currentQuantity + incBy
}

func decBy(currentQuantity uint16, decBy uint16) (uint16, bool) {
	if decBy > currentQuantity {
		return 0, false
	}

	return currentQuantity - decBy, true
}

// IncrementByOne adds one, saturating at ItemQuantityLargeMax.
func (l *ItemQuantityLarge) IncrementByOne() {
	l.Set(incBy(l.quantity, 1, ItemQuantityLargeMax))
}

// DecrementByOne removes one and reports false, changing nothing, when the stack is empty.
func (l *ItemQuantityLarge) DecrementByOne() bool {
	newQuantity, bWasEnough := decBy(l.quantity, 1)
	if !bWasEnough {
		return false
	}
	l.Set(newQuantity)

	return true
}

// IncrementBy adds the amount, saturating at ItemQuantityLargeMax.
func (l *ItemQuantityLarge) IncrementBy(incByQuantity uint16) {
	l.Set(incBy(l.quantity, incByQuantity, ItemQuantityLargeMax))
}

// DecrementBy removes the amount and reports false, changing nothing, when too few are held.
func (l *ItemQuantityLarge) DecrementBy(decByQuantity uint16) bool {
	newQuantity, bWasEnough := decBy(l.quantity, decByQuantity)
	if !bWasEnough {
		return false
	}
	l.Set(newQuantity)

	return true
}

// IncrementByOne adds one, saturating at ItemQuantitySmallMax.
func (s *ItemQuantitySmall) IncrementByOne() {
	s.Set(incBy(s.quantity, 1, ItemQuantitySmallMax))
}

// DecrementByOne removes one and reports false, changing nothing, when the stack is empty.
func (s *ItemQuantitySmall) DecrementByOne() bool {
	newQuantity, bWasEnough := decBy(s.quantity, 1)
	if !bWasEnough {
		return false
	}
	s.Set(newQuantity)

	return true
}

// IncrementBy adds the amount, saturating at ItemQuantitySmallMax.
func (s *ItemQuantitySmall) IncrementBy(incByQuantity uint16) {
	s.Set(incBy(s.quantity, incByQuantity, ItemQuantitySmallMax))
}

// DecrementBy removes the amount and reports false, changing nothing, when too few are held.
func (s *ItemQuantitySmall) DecrementBy(decByQuantity uint16) bool {
	newQuantity, bWasEnough := decBy(s.quantity, decByQuantity)
	if !bWasEnough {
		return false
	}
	s.Set(newQuantity)

	return true
}
