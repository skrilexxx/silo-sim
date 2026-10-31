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

// Control is the control module's state (ODD §6.13, draft); nil when the
// module is off. Per-person slices are indexed by PersonID.
type Control struct {
	Head      PersonID // the Head of IT, or None
	Successor PersonID // the Head's chosen successor, or None

	Acts []Act // reported since the last control step, in order
	Seen int   // acts seen since day 0

	// Charges holds, per person, one bit per Pact rule they are charged
	// under in their open case; Cases lists the people with an open case,
	// in the order the cases were opened.
	Charges []uint32
	Cases   []PersonID
}

// Report adds an act for the next control step.
func (c *Control) Report(a Act) { c.Acts = append(c.Acts, a) }

// IsHead reports whether the person is the Head of IT; false when the
// module is off.
func (c *Control) IsHead(id PersonID) bool { return c != nil && c.Head == id }
