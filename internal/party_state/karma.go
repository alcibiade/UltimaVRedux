package party_state

const maxKarma = 99

type Karma struct {
	Value int
}

func NewKarma(karma byte) Karma {
	k := Karma{
		Value: int(karma),
	}

	return k
}

func (k *Karma) AddDiff(nDiff int) {
	k.Value = k.Value + nDiff
	if k.Value < 0 {
		k.Value = 0
	} else if k.Value > maxKarma {
		k.Value = maxKarma
	}
}
