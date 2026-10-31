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

func (f Flag) String() string {
	return [...]string{"outside", "screen", "history", "silos", "lottery"}[f]
}

// Flags is a set of facts, one bit per Flag.
type Flags uint8

func (f Flags) Has(x Flag) bool { return f&(1<<x) != 0 }
func (f Flags) Count() int      { return bits.OnesCount8(uint8(f)) }

// Beliefs is the beliefs module's state (ODD §6.12), indexed by PersonID.
type Beliefs struct {
	Base   []float64 // 0–1: legitimacy from upbringing, fixed at worker age
	Known  []Flags   // facts known (hidden ones not included)
	Lived  []Flags   // of the known and hidden, the ones from the person's own experience; the rest were heard
	Hidden []Flags   // facts hidden by Vitamin D+: not known until recalled
	Risk   []float64 // risk aversion R, 0–1, fixed at birth (ODD §6.14)
	Cost   float64   // δ: legitimacy lost per known flag

	Relics []Relic

	// Vitamin D+ (ODD §6.12): the last day of water dosing, and of each
	// person's targeted dose; −1 for none.
	Water      Tick
	DosedUntil []Tick

	// The Flamekeepers (ODD §6.12): the members, in the order they took
	// their seats, and whether each person is one.
	Members  []PersonID
	IsMember []bool

	// Everyone who knows a flag, and possibly some who have died or
	// forgotten since; the beliefs step drops those. Learn keeps it.
	Knowers []PersonID
}

// Relic is an object from before the silo (ODD §6.12, Relics).
type Relic struct {
	Floor  int      // the floor it lies hidden on: on its own, or in its holder's flat
	Holder PersonID // or None while nobody has it
}

// Learn makes the person know the flag (ODD §6.12), as lived or as heard.
// Every module that teaches a flag uses it, so the beliefs step finds the
// new knower. Living through what one had only heard makes it lived. It
// reports whether the flag is new to them.
func (b *Beliefs) Learn(id PersonID, f Flag, lived bool) bool {
	b.Hidden[id] &^= 1 << f // what one learns again is no longer hidden
	if lived {
		b.Lived[id] |= 1 << f
	}
	k := b.Known[id]
	if k.Has(f) {
		return false
	}
	if k == 0 {
		b.Knowers = append(b.Knowers, id)
	}
	b.Known[id] = k | 1<<f
	return true
}

// Dosed reports whether the person is on Vitamin D+ on day.
func (b *Beliefs) Dosed(id PersonID, day Tick) bool {
	return b.Water >= day || b.DosedUntil[id] >= day
}

// Dose puts the person on Vitamin D+ until the day given (ODD §6.12,
// targeted dosing), or longer if they already are.
func (b *Beliefs) Dose(id PersonID, until Tick) {
	b.DosedUntil[id] = max(b.DosedUntil[id], until)
}

// Hide makes a known flag hidden (ODD §6.12, Vitamin D+). It reports
// whether it is the person's first hidden flag.
func (b *Beliefs) Hide(id PersonID, f Flag) bool {
	first := b.Hidden[id] == 0
	b.Known[id] &^= 1 << f
	b.Hidden[id] |= 1 << f
	return first
}

// Forget makes the person lose the flag for good (ODD §6.12, Forgetting).
func (b *Beliefs) Forget(id PersonID, f Flag) {
	b.Known[id] &^= 1 << f
	b.Lived[id] &^= 1 << f
}

// L is the person's legitimacy: their base minus δ per known flag.
func (b *Beliefs) L(id PersonID) float64 {
	return max(0, b.Base[id]-b.Cost*float64(b.Known[id].Count()))
}
