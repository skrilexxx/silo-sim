package engine

// Phase is a step number from ODD §3. This file is the only place the step
// order and frequency live; changing either changes results, so it needs a
// spec change first.
type Phase int

const (
	PhaseGenerator    Phase = 1
	PhaseLifeSupport  Phase = 2
	PhaseFood         Phase = 3
	PhaseRations      Phase = 4
	PhaseWork         Phase = 5
	PhaseHealth       Phase = 6
	PhaseDeaths       Phase = 7
	PhaseReproduction Phase = 8
	PhaseLifeStages   Phase = 9
	PhasePartnering   Phase = 10
	PhaseLogDaily     Phase = 11
	PhaseLottery      Phase = 12
	PhaseHousing      Phase = 13
	PhaseStaffing     Phase = 14
	PhaseSnapshot     Phase = 15
	PhaseGovernance   Phase = 16
)

// ODD §2 Scales: a year is 365 ticks; monthly steps run every 30 ticks.
const (
	daily       = 1
	everyMonth  = 30
	DaysPerYear = 365
)

// schedule gives each phase its name, how often it runs (on days that are
// multiples of every), and whether it writes output (skipped during
// warm-up unless output.logWarmup is set). Daily logging runs in warm-up
// too, because each row counts what happened since the one before: the
// output module needs the counts at the end of warm-up, and itself writes
// nothing until then.
var schedule = [...]struct {
	name   string
	every  int
	output bool
}{
	PhaseGenerator:    {"generator", daily, false},
	PhaseLifeSupport:  {"life support", daily, false},
	PhaseFood:         {"food production", daily, false},
	PhaseRations:      {"rations", daily, false},
	PhaseWork:         {"work", daily, false},
	PhaseHealth:       {"health", daily, false},
	PhaseDeaths:       {"deaths", daily, false},
	PhaseReproduction: {"reproduction", daily, false},
	PhaseLifeStages:   {"life stages", daily, false},
	PhasePartnering:   {"partnering", daily, false},
	PhaseLogDaily:     {"logging", daily, false},
	PhaseLottery:      {"lottery", everyMonth, false},
	PhaseHousing:      {"housing", everyMonth, false},
	PhaseStaffing:     {"staffing", everyMonth, false},
	PhaseSnapshot:     {"snapshot", DaysPerYear, true},
	PhaseGovernance:   {"governance", DaysPerYear, false},
}

func (p Phase) valid() bool { return p >= PhaseGenerator && p <= PhaseGovernance }

func (p Phase) String() string {
	if !p.valid() {
		return "invalid phase"
	}
	return schedule[p].name
}
