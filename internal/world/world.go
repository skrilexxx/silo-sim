// Package world holds the simulation state described in ODD §2. It has no
// behaviour: modules change it, the engine and outputs read it.
package world

// Tick is one simulated day.
type Tick int32

// PersonID indexes World.People. IDs are never reused.
type PersonID int32

// HouseholdID indexes World.Households.
type HouseholdID int32

// None marks an empty optional reference (partner, mother, master…).
// IDs are slice indexes, so -1 can never be a real one.
const None = -1

type Zone int8

const (
	UpTop Zone = iota
	Mids
	DownDeep
	AllZones // a department that works in every zone (General services, Porters)
)

// NumZones counts the residential zones: UpTop, Mids, DownDeep.
const NumZones = int(DownDeep) + 1

// String gives the zone's scenario-file name.
func (z Zone) String() string {
	switch z {
	case UpTop:
		return "upTop"
	case Mids:
		return "mids"
	case DownDeep:
		return "downDeep"
	}
	return "all"
}

type Sex int8

const (
	Female Sex = iota
	Male
)

type Stage int8

const (
	Child Stage = iota
	Shadow
	Worker
	Retired
	SickLeave // a worker off sick: no work, keeps the department, returns when health recovers (ODD §6.5)
)

// Implant is the state of a woman's contraceptive implant (ODD §6.4).
type Implant int8

const (
	ImplantIn  Implant = iota
	WindowOpen         // implant removed, conception window running
	Pregnant
	Queued // permit won, waiting for Medical to remove the implant (ODD §6.2)
)

// Clamp01 keeps a 0–1 variable in range (docs/architecture.md, Core types).
func Clamp01(x float64) float64 { return max(0, min(1, x)) }

// ODD §2: Person. Age is never stored; it follows from BirthDay and the day.
type Person struct {
	// Read by most daily steps, so they come first and share a cache line:
	// every step visits people in shuffled order, and memory, not
	// arithmetic, sets the pace (T5.4).
	Health     float64 // 0–1
	Skill      float64 // 0–1
	Department int     // index into World.Depts, or None; kept after retirement and death as a record
	BirthDay   Tick
	Household  HouseholdID
	Partner    PersonID // or None
	Master     PersonID // or None
	Alive      bool     // dead people stay in People, but leave World.Alive
	Sex        Sex
	Stage      Stage
	Implant    Implant

	ID         PersonID
	Mother     PersonID // or None
	Father     PersonID // or None
	PermitDay  Tick     // day she won the lottery (meaningful while queued, open or pregnant)
	WindowEnds Tick     // last tick of the conception window
	DueDay     Tick     // birth day if pregnant
	BabyFather PersonID // partner at conception, recorded for the birth; or None
	LastBirth  Tick     // day of her most recent birth (meaningful if Children > 0)
	LowHealth  int      // consecutive days a worker's health has been below sickLeaveHealth
	Children   int      // children born, alive or not
}

// ODD §2: Household. Its zone follows from its floor. A household whose
// last member leaves is dissolved: no members and Floor = None, which
// frees its home. IDs are never reused.
type Household struct {
	ID      HouseholdID
	Floor   int // floor number, 1-based: World.Floors[Floor-1]; None if dissolved
	Members []PersonID
}

type FloorFunction int8

const (
	Residential FloorFunction = iota
	Farm                      // pure work floor, no homes
	Mechanical                // pure work floor, no homes
)

// ODD §2: Floor.
type Floor struct {
	Number   int
	Zone     Zone
	Homes    int // housing capacity in households; 0 on pure work floors
	Function FloorFunction
}

// ODD §2: Department. Depts keep the ODD §6.5 table order.
type Department struct {
	Name        string
	Zone        Zone // or AllZones
	TargetShare float64
	Target      float64 // target headcount from the monthly staffing step (ODD §6.5)
	Headcount   int
	Shadows     int
	Output      float64 // this tick: worker skill × health, plus shadowOutputFactor × shadow skill × health
	Coverage    float64 // 0–1: min(1, Output / need), ODD §6.6
}

// ODD §2: Silo, the global state.
type Silo struct {
	FoodStock          float64 // kcal
	FoodNeed           float64 // kcal: today's total need (ODD §6.7)
	FoodOpening        float64 // kcal: today's stock before production, for the conservation invariant (ODD §8)
	FoodProduced       float64 // kcal produced today
	FoodHandedOut      float64 // kcal handed out today
	GeneratorCondition float64 // 0–1; 1 = like new
	GeneratorDown      int     // ticks until a failed generator is repaired
	PowerSupply        float64 // abstract units
	PowerDemand        float64
	WaterSupply        float64    // 0–1
	AirQuality         float64    // 0–1
	PartsStock         float64    // abstract units
	RationShares       [3]float64 // per zone, indexed by Zone (policy)
	Intake             [3]float64 // today's intake/need per zone (ODD §6.7); may exceed 1 under unequal weights
	Rationing          bool       // today's hand-out was cut to rationingShare
	GovernanceMode     string     // appointment or election
	Mayor              PersonID   // or None
	MayorSince         Tick       // day the current term started
}

// PowerFactor is the share of today's power demand that is met (ODD
// §6.6). With no demand (the resources module is off) all of it is.
func (s *Silo) PowerFactor() float64 {
	if s.PowerDemand == 0 {
		return 1
	}
	return min(1, s.PowerSupply/s.PowerDemand)
}

// HousingRequest is a couple waiting to move into a home of their own
// (ODD §6.3). Zone is chosen the first month housing handles them.
type HousingRequest struct {
	A, B    PersonID
	Zone    Zone
	ZoneSet bool
}

type World struct {
	Day        Tick // 0 before the first tick; tick t sets Day = t
	People     []Person
	Alive      []PersonID // living people, reshuffled every tick (ODD §3)
	Households []Household
	Floors     []Floor // Floors[i] is floor number i+1
	Depts      []Department
	Silo       Silo

	HousingQueue []HousingRequest // oldest first

	Counts Counts  // running totals for the output files
	Events []Event // not yet written to events.jsonl; the output module drains it
}

// Counts are running totals since day 0 that no state variable holds.
// Births and deaths need none: they follow from People and Alive.
type Counts struct {
	Permits   int // lottery winners (ODD §6.4)
	ZoneMoves int // people whose home moved to another zone (ODD §6.3)
}

// Event is one line of events.jsonl (ODD §8). Each kind embeds
// EventHeader, whose fields JSON puts first on the line.
type Event interface{ When() Tick }

type EventHeader struct {
	Day  Tick   `json:"day"`
	Type string `json:"type"`
}

func (h EventHeader) When() Tick { return h.Day }

// MayorEvent: type "mayor", a Mayor chosen (ODD §6.8).
type MayorEvent struct {
	EventHeader
	Mayor    PersonID `json:"mayor"`           // or None if nobody could be chosen
	Previous PersonID `json:"previous"`        // or None
	Zone     string   `json:"zone"`            // the new Mayor's home zone
	Reason   string   `json:"reason"`          // start, died, term or vacant
	Mode     string   `json:"mode"`            // appointment or election
	Votes    []int    `json:"votes,omitempty"` // elections: votes per zone's candidate, Up Top first; -1 for no candidate
}

// GeneratorEvent: type "failure" (the generator breaks down) or "repair"
// (it runs again) (ODD §6.6).
type GeneratorEvent struct {
	EventHeader
	OutageDays int     `json:"outageDays,omitempty"` // failures only
	Condition  float64 `json:"condition"`
	Parts      float64 `json:"parts"`
}

// RationingEvent: type "rationingOn" or "rationingOff" (ODD §6.7).
type RationingEvent struct {
	EventHeader
	FoodStockDays float64 `json:"foodStockDays"`
}

// ScriptedEvent: type "scripted", a shock from the scenario's events: list
// (ODD §5). What it causes is logged by the module that defines it.
type ScriptedEvent struct {
	EventHeader
	Event string `json:"event"`
}

// DepartmentEvent: type "understaffed" (workers fell below
// output.understaffedShare of target) or "restaffed" (back above).
type DepartmentEvent struct {
	EventHeader
	Department string  `json:"department"`
	Workers    int     `json:"workers"`
	Target     float64 `json:"target"`
}
