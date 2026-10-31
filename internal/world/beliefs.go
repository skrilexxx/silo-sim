package world

import "math/bits"

// Flag is one of the five facts a person can know (ODD §6.12).
type Flag uint8

const (
	FlagOutside Flag = iota // the outside may be survivable
	FlagScreen              // the view of the outside on the screen is a lie
	FlagHistory             // the true history
	FlagSilos               // other silos exist
	FlagLottery             // the lottery is rigged
	NumFlags
)

// Flags is a set of facts, one bit per Flag.
type Flags uint8

func (f Flags) Has(x Flag) bool { return f&(1<<x) != 0 }
func (f Flags) Count() int      { return bits.OnesCount8(uint8(f)) }

// Beliefs is the beliefs module's state (ODD §6.12), indexed by PersonID.
type Beliefs struct {
	Base  []float64 // 0–1: legitimacy from upbringing, fixed at worker age
	Known []Flags   // facts known (hidden ones not included)
	Risk  []float64 // risk aversion R, 0–1, fixed at birth (ODD §6.14)
	Cost  float64   // δ: legitimacy lost per known flag
}

// L is the person's legitimacy: their base minus δ per known flag.
func (b *Beliefs) L(id PersonID) float64 {
	return max(0, b.Base[id]-b.Cost*float64(b.Known[id].Count()))
}
