package engine

import "slices"

// Phase is 100 × a step number from ODD §3, so a part-2 step fits between
// two baseline steps (750 is ODD step 7.5) without renumbering them. This
// file is the only place the step order and frequency live; changing
// either changes results, so it needs a spec change first.
type Phase int

const (
	PhaseGenerator    Phase = 100
	PhaseLifeSupport  Phase = 200
	PhaseFood         Phase = 300
	PhaseRations      Phase = 400
	PhaseWork         Phase = 500
	PhaseHealth       Phase = 600
	PhaseDeaths       Phase = 700
	PhaseReproduction Phase = 800
	PhaseLifeStages   Phase = 900
	PhasePartnering   Phase = 1000
	PhaseNetwork      Phase = 1050 // part 2: ties and conversations (ODD §6.11)
	PhaseBeliefs      Phase = 1070 // part 2: legitimacy, knowledge and memory (ODD §6.12)
	PhaseControl      Phase = 1080 // part 2: surveillance, Judicial and punishment (ODD §6.13)
	PhaseRebellion    Phase = 1090 // part 2: grievance and rebellion (ODD §6.14)
	PhaseLogDaily     Phase = 1100
	PhaseLottery      Phase = 1200
	PhaseHousing      Phase = 1300
	PhaseStaffing     Phase = 1400
	PhaseSnapshot     Phase = 1500
	PhaseGovernance   Phase = 1600
	PhaseLocation     Phase = 1650 // part 2: where everyone is the next day (ODD §6.10)
)

// ODD §2 Scales: a year is 365 ticks; monthly steps run every 30 ticks.
const (
	daily        = 1
	DaysPerMonth = 30
	DaysPerYear  = 365
)

// schedule lists the phases in the order they run, with each one's name,
// how often it runs (on days that are multiples of every), and whether it
// writes output (skipped during warm-up unless output.logWarmup is set).
// Daily logging runs in warm-up too, because each row counts what
// happened since the one before: the output module needs the counts at
// the end of warm-up, and itself writes nothing until then.
var schedule = []phaseInfo{
	{PhaseGenerator, "generator", daily, false},
	{PhaseLifeSupport, "life support", daily, false},
	{PhaseFood, "food production", daily, false},
	{PhaseRations, "rations", daily, false},
	{PhaseWork, "work", daily, false},
	{PhaseHealth, "health", daily, false},
	{PhaseDeaths, "deaths", daily, false},
	{PhaseReproduction, "reproduction", daily, false},
	{PhaseLifeStages, "life stages", daily, false},
	{PhasePartnering, "partnering", daily, false},
	{PhaseNetwork, "network", daily, false},
	{PhaseBeliefs, "beliefs", daily, false},
	{PhaseControl, "control", daily, false},
	{PhaseRebellion, "rebellion", daily, false},
	{PhaseLogDaily, "logging", daily, false},
	{PhaseLottery, "lottery", DaysPerMonth, false},
	{PhaseHousing, "housing", DaysPerMonth, false},
	{PhaseStaffing, "staffing", DaysPerMonth, false},
	{PhaseSnapshot, "snapshot", DaysPerYear, true},
	{PhaseGovernance, "governance", DaysPerYear, false},
	{PhaseLocation, "location", daily, false},
}

type phaseInfo struct {
	phase  Phase
	name   string
	every  int
	output bool
}

// slot is the phase's position in schedule, or -1 if it is not a phase.
func (p Phase) slot() int {
	return slices.IndexFunc(schedule, func(s phaseInfo) bool { return s.phase == p })
}

func (p Phase) String() string {
	if i := p.slot(); i >= 0 {
		return schedule[i].name
	}
	return "invalid phase"
}
