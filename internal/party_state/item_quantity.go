package party_state

const (
	// ItemQuantityLargeMax is the authoritative maximum for large inventory stacks.
	ItemQuantityLargeMax uint16 = 9999
	// ItemQuantitySmallMax is the authoritative maximum for small inventory stacks.
	ItemQuantitySmallMax uint16 = 99
)

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

type ItemQuantityLarge struct {
	quantity uint16
	activity QuantityActivity
}

func (l *ItemQuantityLarge) SetHasOne() {
	l.Set(1)
}

func (l *ItemQuantityLarge) HasSome() bool {
	return l.Get() > 0
}

type ItemQuantitySmall struct {
	quantity uint16
	activity QuantityActivity
}

func (s *ItemQuantitySmall) SetHasOne() {
	s.activity.record(s.quantity, 1)
	s.quantity = 1
}

func (s *ItemQuantitySmall) HasSome() bool {
	if s == nil {
		return false
	}

	return s.Get() > 0
}

func (l *ItemQuantityLarge) Get() uint16 {
	return l.quantity
}

func (l *ItemQuantityLarge) Set(quantity uint16) {
	next := min(quantity, ItemQuantityLargeMax)
	l.activity.record(l.quantity, next)
	l.quantity = next
}

func (s *ItemQuantitySmall) Get() uint16 {
	if s == nil {
		return 0
	}

	return s.quantity
}

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

func (l *ItemQuantityLarge) IncrementByOne() {
	l.Set(incBy(l.quantity, 1, ItemQuantityLargeMax))
}

func (l *ItemQuantityLarge) DecrementByOne() bool {
	newQuantity, bWasEnough := decBy(l.quantity, 1)
	if !bWasEnough {
		return false
	}
	l.Set(newQuantity)

	return true
}

func (l *ItemQuantityLarge) IncrementBy(incByQuantity uint16) {
	l.Set(incBy(l.quantity, incByQuantity, ItemQuantityLargeMax))
}

func (l *ItemQuantityLarge) DecrementBy(decByQuantity uint16) bool {
	newQuantity, bWasEnough := decBy(l.quantity, decByQuantity)
	if !bWasEnough {
		return false
	}
	l.Set(newQuantity)

	return true
}

func (s *ItemQuantitySmall) IncrementByOne() {
	s.Set(incBy(s.quantity, 1, ItemQuantitySmallMax))
}

func (s *ItemQuantitySmall) DecrementByOne() bool {
	newQuantity, bWasEnough := decBy(s.quantity, 1)
	if !bWasEnough {
		return false
	}
	s.Set(newQuantity)

	return true
}

func (s *ItemQuantitySmall) IncrementBy(incByQuantity uint16) {
	s.Set(incBy(s.quantity, incByQuantity, ItemQuantitySmallMax))
}

func (s *ItemQuantitySmall) DecrementBy(decByQuantity uint16) bool {
	newQuantity, bWasEnough := decBy(s.quantity, decByQuantity)
	if !bWasEnough {
		return false
	}
	s.Set(newQuantity)

	return true
}
