package world

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
	Shadows  [][]PersonID // each master's shadows, rebuilt at every network step
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
