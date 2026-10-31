package world

// Rebellion is the rebellion module's state (ODD §6.14, draft); nil when
// the module is off. Per-person slices are indexed by PersonID.
type Rebellion struct {
	LossDay []Tick    // day of the person's last loss in the family, or None
	H, G    []float64 // today's hardship and grievance; 0 for children
}
