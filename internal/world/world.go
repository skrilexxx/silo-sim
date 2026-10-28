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
	ID         PersonID
	Sex        Sex
	BirthDay   Tick
	Alive      bool // dead people stay in People, but leave World.Alive
	Household  HouseholdID
	Partner    PersonID // or None
	Mother     PersonID // or None
	Father     PersonID // or None
	Stage      Stage
	Department int      // index into World.Depts, or None; kept after retirement and death as a record
	Master     PersonID // or None
	Skill      float64  // 0–1
	Health     float64  // 0–1
	LowHealth  int      // consecutive days a worker's health has been below sickRetireHealth
	Implant    Implant
	PermitDay  Tick     // day she won the lottery (meaningful while queued, open or pregnant)
	WindowEnds Tick     // last tick of the conception window
	DueDay     Tick     // birth day if pregnant
	BabyFather PersonID // partner at conception, recorded for the birth; or None
	Children   int      // children born, alive or not
	LastBirth  Tick     // day of her most recent birth (meaningful if Children > 0)
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
	Headcount   int
	Shadows     int
	Output      float64 // this tick: worker skill × health, plus shadowOutputFactor × shadow skill × health
	Coverage    float64 // 0–1: min(1, Output / need), ODD §6.6
}

// ODD §2: Silo, the global state.
type Silo struct {
	FoodStock          float64 // kcal
	GeneratorCondition float64 // 0–1; 1 = like new
	GeneratorDown      int     // ticks until a failed generator is repaired
	PowerSupply        float64 // abstract units
	PowerDemand        float64
	WaterSupply        float64    // 0–1
	AirQuality         float64    // 0–1
	PartsStock         float64    // abstract units
	RationShares       [3]float64 // per zone, indexed by Zone (policy)
	GovernanceMode     string     // appointment or election
	Mayor              PersonID   // or None
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
}
