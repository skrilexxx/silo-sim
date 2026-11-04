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
	Own     []float64    // h_own: own hardship, drawn at birth (ODD §6.14, Hardship)
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

	// Sabotage (ODD §6.14): the generator or the water pump wrecked today,
	// for the resources module's next step to act on; the last day the pump
	// is down; acts by target and stair repairs since day 0.
	GeneratorHit, PumpHit bool
	PumpUntil             Tick
	BrokeOn               []Tick // per stair segment, the day it last broke, or None (ODD §1.2, Split)
	Sabotage              [NumTargets]int
	Repairs               int

	// Collapse (ODD §1.2): days in a row with air and with water below the
	// collapse level, and the day of the collapse, or None.
	AirLow, WaterLow int
	Collapsed        Tick

	// Unrest and rebellion (ODD §1.2): days in a row with at least the
	// unrest share of adults active, with at least the rebellion share,
	// and with less than the unrest share; the last day a collective
	// action was under way, or None; the first day of the open unrest
	// spell and of the running rebellion, or None; and the spells ended
	// and rebellions started since day 0.
	Busy, Rising, Quiet         int
	Fired                       []string // the triggers of ODD §6.15 today, in config.Triggers' names
	LastAction                  Tick
	UnrestStart, RebellionStart Tick
	UnrestSpells, Rebellions    int

	// The rebellion log (ODD §6.14, Outputs): one entry per day, the last
	// RebellionDays days while no rebellion runs (a rebellion's start lies
	// that far back when it is detected), every day since its start while
	// one runs; the running rebellion's trigger; and the records of
	// rebellions ended since the output module last took them.
	Days       []RebellionDay
	Trigger    string
	TriggerDay Tick
	Ended      []RebellionRecord
}

// RebellionDay is what one day adds to a rebellion's record (ODD §6.14,
// Outputs). Counts are the day's own, not running totals.
type RebellionDay struct {
	Day      Tick
	Share    float64 // adults active or beyond
	Zone     Zone    // the home zone with the highest share of its adults among them; the first on a tie (ODD §6.14)
	Struck   []int   // critical departments on strike
	Fought   bool
	Held     [NumZones]bool
	Fights   FightTally
	Arrests  int // by Judicial, for any offence (ODD §6.13)
	Sabotage [NumTargets]int
}

// RebellionRecord is one line of rebellions.jsonl (ODD §6.14, Outputs).
type RebellionRecord struct {
	Start             Tick     `json:"start"`
	End               *Tick    `json:"end"` // nil while unfinished
	Days              int      `json:"days"`
	Trigger           string   `json:"trigger"`
	TriggerDay        *Tick    `json:"triggerDay,omitempty"`
	Zone              string   `json:"zone"`
	PeakActive        float64  `json:"peakActive"`
	Struck            []string `json:"struck"`
	FightDays         int      `json:"fightDays"`
	Held              []string `json:"held"`
	RebelsKilled      int      `json:"rebelsKilled"`
	OfficersKilled    int      `json:"officersKilled"`
	FightArrests      int      `json:"fightArrests"`
	Arrests           int      `json:"arrests"`
	SegmentsBroken    int      `json:"segmentsBroken"`
	GeneratorSabotage int      `json:"generatorSabotage"`
	PumpSabotage      int      `json:"pumpSabotage"`
	Ending            string   `json:"ending"`
}

// Record sums the running rebellion's days into its record (ODD §6.14,
// Outputs). It covers the days from its start up to its end day, or the
// day before when suppressed, as that end day is the first quiet one; end
// None is a rebellion still running, its record unfinished, covering every
// day so far. Days is end − start (today + 1 − start while unfinished).
func (r *Rebellion) Record(w *World, end Tick, ending string) RebellionRecord {
	start := r.RebellionStart
	rec := RebellionRecord{Start: start, Trigger: r.Trigger, Struck: []string{}, Held: []string{}, Ending: ending}
	last := end
	switch {
	case end == None:
		last = w.Day
		rec.Days = int(w.Day + 1 - start)
	case ending == "suppressed":
		last = end - 1
		fallthrough
	default:
		rec.End = &end
		rec.Days = int(end - start)
	}
	if r.TriggerDay != None {
		rec.TriggerDay = &r.TriggerDay
	}
	struck := make([]bool, len(w.Depts))
	var held [NumZones]bool
	for i := range r.Days {
		d := &r.Days[i]
		if d.Day < start || d.Day > last {
			continue
		}
		if d.Day == start {
			rec.Zone = d.Zone.String()
		}
		rec.PeakActive = max(rec.PeakActive, d.Share)
		for _, dep := range d.Struck {
			struck[dep] = true
		}
		if d.Fought {
			rec.FightDays++
		}
		for z, h := range d.Held {
			held[z] = held[z] || h
		}
		rec.RebelsKilled += d.Fights.RebelsKilled
		rec.OfficersKilled += d.Fights.OfficersKilled
		rec.FightArrests += d.Fights.RebelsArrested
		rec.Arrests += d.Arrests
		rec.SegmentsBroken += d.Sabotage[TargetStairs]
		rec.GeneratorSabotage += d.Sabotage[TargetGenerator]
		rec.PumpSabotage += d.Sabotage[TargetPump]
	}
	for dep, s := range struck {
		if s {
			rec.Struck = append(rec.Struck, w.Depts[dep].Name)
		}
	}
	for z, h := range held {
		if h {
			rec.Held = append(rec.Held, Zone(z).String())
		}
	}
	return rec
}

// Target is what a sabotage hits (ODD §6.14, Sabotage).
type Target uint8

const (
	TargetGenerator Target = iota
	TargetPump
	TargetStairs
	NumTargets
)

func (t Target) String() string { return [...]string{"generator", "pump", "stairs"}[t] }

// SabotageEvent: type "sabotage" (ODD §6.14).
type SabotageEvent struct {
	EventHeader
	Person  PersonID `json:"person"`
	Target  string   `json:"target"`
	Segment int      `json:"segment,omitempty"` // stairs only
}

// SegmentEvent: type "stairRepaired" (ODD §6.14, Stair repairs).
type SegmentEvent struct {
	EventHeader
	Segment int `json:"segment"`
}

// CollapseEvent: type "collapse" (ODD §1.2).
type CollapseEvent struct {
	EventHeader
	Cause string `json:"cause"` // air, water or population
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
