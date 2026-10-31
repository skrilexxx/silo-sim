package world

// RebelStage is how far a person has turned against the regime today
// (ODD §6.14, Stages); each stage needs the one before.
type RebelStage uint8

const (
	Calm RebelStage = iota
	Active
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
}
