package world

// RebelStage is how far a person has turned against the regime today
// (ODD §6.14, Stages); each stage needs the one before.
type RebelStage uint8

const (
	Calm RebelStage = iota
	Active
	Joined
	Violent
)

// Rebellion is the rebellion module's state (ODD §6.14, draft); nil when
// the module is off. Per-person slices are indexed by PersonID.
type Rebellion struct {
	LossDay []Tick       // day of the person's last loss in the family, or None
	H, G    []float64    // today's hardship and grievance; 0 for children
	Stage   []RebelStage // today's stage

	// Zone counts at the end of yesterday's step, read by today's arrest
	// estimate: adults active or beyond by the zone of their day location,
	// and adults on floors connected to a court (ODD §6.14).
	Actives, Reach [NumZones]int

	// Per department, its workers on strike today (ODD §6.14, Strikes).
	Striking []int

	// Per zone, after today's fights (ODD §6.14, Held zones): its patrol,
	// whether rebels hold it, and for how many days in a row.
	Patrol   [NumZones]float64
	Held     [NumZones]bool
	HeldDays [NumZones]int
	Tally    FightTally

	// Unrest and rebellion (ODD §1.2): days in a row with at least the
	// unrest share of adults active, with at least the rebellion share,
	// and with less than the unrest share; the last day a collective
	// action was under way, or None; the first day of the open unrest
	// spell and of the running rebellion, or None; and the spells ended
	// and rebellions started since day 0.
	Busy, Rising, Quiet         int
	LastAction                  Tick
	UnrestStart, RebellionStart Tick
	UnrestSpells, Rebellions    int
}

// FightTally counts the losses in fights since day 0 (ODD §6.14).
type FightTally struct {
	RebelsKilled, RebelsArrested, OfficersKilled, OfficersWounded int
}

// Holds reports whether rebels hold the zone of a floor today, so Judicial
// arrests nobody there (ODD §6.14, Held zones). False when the module is
// off or the floor is 0 (the stairs).
func (r *Rebellion) Holds(w *World, floor int) bool {
	return r != nil && floor > 0 && r.Held[w.Floors[floor-1].Zone]
}

// ZoneEvent: type "zoneHeld" or "zoneFreed" (ODD §6.14, Held zones).
type ZoneEvent struct {
	EventHeader
	Zone string `json:"zone"`
}

// OnStrike reports whether a worker or shadow is on strike today: they
// were joined or violent at yesterday's rebellion step (ODD §6.14,
// Strikes). A striker gives no output, gains no skill and has no work
// injuries, but still counts as a worker. False when the module is off.
func (r *Rebellion) OnStrike(w *World, id PersonID) bool {
	if r == nil || int(id) >= len(r.Stage) || r.Stage[id] < Joined {
		return false
	}
	s := w.People[id].Stage
	return s == Worker || s == Shadow
}

// UnrestEvent: type "unrest", an unrest spell that ended without becoming
// a rebellion (ODD §1.2), logged at its end.
type UnrestEvent struct {
	EventHeader
	Start Tick `json:"start"`
	End   Tick `json:"end"` // the first of the quiet days that ended it
}

// RebellionEvent: type "rebellionStart" or "rebellionEnd" (ODD §1.2).
type RebellionEvent struct {
	EventHeader
	Start  Tick   `json:"start"`
	End    Tick   `json:"end,omitempty"`    // rebellionEnd only: the first of the quiet days
	Ending string `json:"ending,omitempty"` // rebellionEnd only
}
