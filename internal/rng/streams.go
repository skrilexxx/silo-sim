package rng

// streamID gives each module its own fixed random stream, so switching a
// module on or off never shifts another module's draws (common random
// numbers; see docs/architecture.md, Randomness).
//
// The numbers are arbitrary but permanent: changing or reusing one changes
// every run that uses that stream. Add new modules at the end.
func streamID(name string) (uint64, bool) {
	switch name {
	case "engine":
		return 1, true
	case "demography":
		return 2, true
	case "health":
		return 3, true
	case "work":
		return 4, true
	case "resources":
		return 5, true
	case "governance":
		return 6, true
	case "output":
		return 7, true // draws nothing today; every module gets a stream
	case "initpop":
		return 8, true
	case "events":
		return 9, true // scripted events (ODD §5), so their draws shift no module's
	case "space":
		return 10, true
	}
	return 0, false
}
