package world

import "github.com/skrilexxx/silo-sim/internal/rng"

// TieKind is the kind of a tie (ODD §6.11, draft), in order of precedence:
// a pair has one tie, of the first kind that applies.
type TieKind int8

const (
	TiePartner TieKind = iota
	TieHousehold
	TieKin
	TieMasterShadow
	TieCoworker // the weak kinds start here
	TieNeighbour
	TiePorter
	NumTieKinds
)

var tieNames = [...]string{"partner", "household", "kin", "masterShadow", "coworker", "neighbour", "porter"}

func (k TieKind) String() string { return tieNames[k] }

// Weak reports whether ties of this kind are drawn (and capped at k)
// rather than read from the records.
func (k TieKind) Weak() bool { return k >= TieCoworker }

type Tie struct {
	Other PersonID
	Kind  TieKind
}

// Network is the state of the network module (ODD §6.11, draft); nil when
// the module is off. Strong ties are not stored: they follow from the
// records (partner, household, Mother and Father, Master). Per-person
// slices are indexed by PersonID.
type Network struct {
	Weak     [][]Tie      // drawn ties, kept symmetric: B is in A's list iff A is in B's, with the same kind
	Children [][]PersonID // everyone's children, living or dead, for kin ties
	Shadows  [][]PersonID // each master's shadows, kept up to date at every network step

	Chance [NumTieKinds]float64 // daily chance a tie of each kind is used (ODD §6.11)
	Talks  *rng.Keyed           // whether a tie is used on a day: the same answer whoever asks
	Part   []int                // per floor: which run of connected floors it is in, as of the network step

	// Each person's ties as Ties gives them, cached between the days they
	// change (T9.2: rebuilding them daily was most of the step). Every
	// living person's list is rebuilt at each network step if invalid, so
	// a list holds the ties as of the last step even once marked invalid.
	cache [][]Tie
	valid []bool
}

// TiesOf returns a person's ties as of the last network step (ODD §6.11),
// from the cache. Changes made later the same day (a move at housing, a
// retrained worker) count from the next step. Callers must not modify it.
func (n *Network) TiesOf(w *World, id PersonID) []Tie {
	for len(n.cache) <= int(id) {
		n.cache, n.valid = append(n.cache, nil), append(n.valid, false)
	}
	if !n.valid[id] {
		n.cache[id] = n.Ties(w, id, n.cache[id][:0])
		n.valid[id] = true
	}
	return n.cache[id]
}

// Forget drops the cached ties of everyone touched since the last network
// step, and of everyone on their ties before and after: ties are
// symmetric, so a change to one person's ties changes the other side's.
// It then rebuilds every living person's list.
func (n *Network) Forget(w *World, touched []PersonID, buf []Tie) {
	for len(n.cache) < len(w.People) {
		n.cache, n.valid = append(n.cache, nil), append(n.valid, false)
	}
	for _, id := range touched {
		for _, t := range n.cache[id] {
			n.valid[t.Other] = false
		}
		n.valid[id] = false
		if w.People[id].Alive {
			for _, t := range n.Ties(w, id, buf[:0]) {
				n.valid[t.Other] = false
			}
		}
	}
	for _, id := range w.Alive {
		n.TiesOf(w, id)
	}
}

// Reachable reports whether two people can talk today (ODD §6.11):
// neither is on the stairs or held by Judicial (ODD §6.13), and the stairs
// connect their day locations.
func (n *Network) Reachable(w *World, a, b PersonID) bool {
	if w.Control.Held(a) || w.Control.Held(b) { // ODD §6.13: held people talk with nobody
		return false
	}
	fa, fb := w.Space.Where(w, a), w.Space.Where(w, b)
	return fa != 0 && fb != 0 && n.Part[fa-1] == n.Part[fb-1]
}

// Talk reports whether a and b, tied by a tie of kind, talk today (ODD
// §6.11): they can reach each other, and the day's draw for the pair is
// below the kind's chance. Either may pass something to the other. The
// draw comes first: it needs no memory, and most draws fail (T10.2).
func (n *Network) Talk(w *World, a, b PersonID, kind TieKind) bool {
	lo, hi := min(a, b), max(a, b)
	return n.Talks.Float64(uint64(w.Day), uint64(lo), uint64(hi)) < n.Chance[kind] && n.Reachable(w, a, b)
}

// Ties appends every tie of a living person to out and returns it: one per
// other living person, of the first kind that applies (ODD §6.11).
func (n *Network) Ties(w *World, id PersonID, out []Tie) []Tie {
	start := len(out)
	add := func(other PersonID, kind TieKind) {
		if other == None || other == id || !w.People[other].Alive {
			return
		}
		for _, t := range out[start:] {
			if t.Other == other {
				return // an earlier, stronger kind already holds this pair
			}
		}
		out = append(out, Tie{other, kind})
	}
	p := &w.People[id]
	add(p.Partner, TiePartner)
	for _, o := range w.Households[p.Household].Members {
		add(o, TieHousehold)
	}
	for _, parent := range [2]PersonID{p.Mother, p.Father} {
		if parent == None {
			continue
		}
		add(parent, TieKin)
		for _, sib := range n.Children[parent] {
			add(sib, TieKin)
		}
	}
	for _, c := range n.Children[id] {
		add(c, TieKin)
	}
	add(p.Master, TieMasterShadow)
	if int(id) < len(n.Shadows) {
		for _, s := range n.Shadows[id] {
			add(s, TieMasterShadow)
		}
	}
	for _, t := range n.Weak[id] {
		add(t.Other, t.Kind)
	}
	return out
}
