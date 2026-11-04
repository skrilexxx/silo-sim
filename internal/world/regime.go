package world

import "github.com/skrilexxx/silo-sim/internal/config"

// Regime is the regime module's state (ODD §6.15, draft); nil when the
// module is off.
type Regime struct {
	// Running are the entries in force, in the order they started.
	Running []Running

	// Changed: rebels took the regime (ODD §1.2); since then only the
	// regimeChange entries and policy events apply.
	Changed bool

	// Emergency: air was below the emergency line at the last regime step,
	// so control and work act on it at their next steps (ODD §6.15,
	// Emergency staffing). EmergencyDays counts such days since day 0.
	Emergency     bool
	EmergencyDays int
}

// Running is a playbook entry or a policy event in force.
type Running struct {
	Entry   *config.Entry
	Trigger string // one of config.Triggers, or "scripted" for a policy event
	Index   int    // the playbook entry's index, or None for a policy event
	Start   Tick
	Until   Tick // it lapses at the regime step of this day; None while quiet or forever
	Quiet   bool // it lapses once no unrest spell or rebellion is open
}

// PolicyEvent: type "policy" when an entry starts, "policyEnd" when it
// lapses (ODD §6.15, Outputs).
type PolicyEvent struct {
	EventHeader
	Trigger string `json:"trigger"`
	Entry   *int   `json:"entry,omitempty"` // the playbook entry's index; absent for a policy event
	Start   Tick   `json:"start"`
}

// Silo1 is the Silo 1 module's state (ODD §6.16, draft, lean); nil when
// the module is off.
type Silo1 struct {
	Escapes   []Tick // the day of each escape seen
	Order     bool   // an escape today: the regime step starts Silo 1's order
	Safeguard Tick   // the day the Safeguard goes off, or None
}

// SafeguardEvent: type "safeguard", the day Silo 1 sets it off and the
// day it goes off (ODD §6.16).
type SafeguardEvent struct {
	EventHeader
	GoesOff Tick `json:"goesOff"`
}
