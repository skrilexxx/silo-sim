package world

// Offence is a type of act the Pact can forbid (ODD §6.9), in the order of
// config.Offences.
type Offence uint8

const (
	HoldRelic Offence = iota
	TradeRelic
	SpeakForbidden
	AskToGoOut
	UnpermittedConception
	ContactAttempt
	Organise
	Violence
	Sabotage
	NumOffences
)

func (o Offence) String() string {
	return [...]string{"holdRelic", "tradeRelic", "speakForbidden", "askToGoOut", "unpermittedConception",
		"contactAttempt", "organise", "violence", "sabotage"}[o]
}

// Act is one offence by a person, reported by the module where it happens
// (ODD §6.13, Offences). Other is a second offender in the same act (the
// receiver of a handover), or None.
type Act struct {
	Person, Other PersonID
	Offence       Offence
}

// Hold is where Judicial holds a person (ODD §6.13).
type Hold uint8

const (
	Free      Hold = iota
	Custody        // arrested, until the trial
	Prison         // detained on the court floor
	Mines          // detained on the mines floor
	Condemned      // sentenced to cleaning: goes out the next day
)

// Tally counts what the regime did since day 0, for the output files.
type Tally struct {
	Seen          int // acts seen
	Arrests       int
	Warnings      int // trials by the penalty served
	Prisons       int
	Mines         int
	Cleanings     int
	Failed        int // cleanings that failed
	Denied        int // lottery wins denied
	FlaggedBirths int // children born to a flagged mother or father
	Taken         int // relics confiscated
	Raids         int
}

// Control is the control module's state (ODD §6.13, draft); nil when the
// module is off. Per-person slices are indexed by PersonID.
type Control struct {
	Head      PersonID // the Head of IT, or None
	Successor PersonID // the Head's chosen successor, or None

	Acts   []Act // reported since the last control step, in order
	Tally  Tally
	Fear   float64    // F, 0–1: how afraid the silo is of punishment
	Doomed []PersonID // condemned to cleaning: they die at the next deaths step (ODD §6.1)

	// Charges holds, per person, one bit per Pact rule they are charged
	// under: their open case, or the case they are in custody for. Cases
	// lists the people with an open case not yet arrested, in the order
	// the cases were opened.
	Charges []uint32
	Cases   []PersonID

	Hold      []Hold
	Until     []Tick   // the last day of a detention
	Convicted []uint32 // one bit per Pact rule the person was ever convicted under
	Flagged   []bool   // covert lottery denial, for life

	// Covert lottery denial (ODD §6.13): per woman, her denied wins and the
	// first day she may enter again; the wins denied since the last control
	// step; the implant doctors; N_try.
	DeniedWins  []int
	DeniedUntil []Tick
	Denied      []PersonID
	Doctors     []PersonID
	Tries       int

	// Losses for grievance (ODD §6.14), reset at every control step: the
	// cleaners who went out at today's step, and the women whose denied
	// wins it handled.
	Cleaned, DeniedToday []PersonID
}

// Deny records a flagged couple's win (ODD §6.13): no window opens, and
// she may not enter again before the day until.
func (c *Control) Deny(id PersonID, until Tick) {
	c.DeniedWins[id]++
	c.DeniedUntil[id] = until
	c.Denied = append(c.Denied, id)
}

// MayEnter reports whether covert denial lets a woman enter the lottery:
// not within a denied win's year, and not after N_try of them, when she
// believes she is infertile. True when the module is off.
func (c *Control) MayEnter(id PersonID, day Tick) bool {
	return c == nil || int(id) >= len(c.DeniedUntil) || day >= c.DeniedUntil[id] && c.DeniedWins[id] < c.Tries
}

// Flags reports whether a couple is flagged: either partner is (either
// may be None). False when the module is off.
func (c *Control) Flags(a, b PersonID) bool {
	return c != nil && (a != None && c.Flagged[a] || b != None && c.Flagged[b])
}

// Report adds an act for the next control step.
func (c *Control) Report(a Act) { c.Acts = append(c.Acts, a) }

// IsHead reports whether the person is the Head of IT; false when the
// module is off.
func (c *Control) IsHead(id PersonID) bool { return c != nil && c.Head == id }

// Held reports whether Judicial holds the person: in custody or detained.
// Held people have no conversations (ODD §6.11). False when the module is
// off.
func (c *Control) Held(id PersonID) bool {
	return c != nil && int(id) < len(c.Hold) && c.Hold[id] != Free
}
