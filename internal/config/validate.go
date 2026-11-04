package config

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"
	"strings"
)

// sumTolerance absorbs float rounding when shares are added up
// (0.30 + 0.20 + … is not exactly 1 in binary).
const sumTolerance = 1e-9

const daysPerYear = 365

// Validate checks value ranges. Every error names the key, and all
// problems are reported at once rather than one per run.
func (s *Scenario) Validate() error {
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }

	// No parameter in the model is meaningful below zero.
	errs = append(errs, negatives(reflect.ValueOf(*s), "")...)

	// Probabilities, shares, daily rates and 0–1 state (health, skill,
	// condition, air) must lie in [0, 1].
	unit := []struct {
		key string
		v   float64
	}{
		{"init.pairedShareOfAdults", s.Init.PairedShareOfAdults},
		{"init.generatorCondition", s.Init.GeneratorCondition},
		{"health.hungerDamage", s.Health.HungerDamage},
		{"health.airDamage", s.Health.AirDamage},
		{"health.injuryChance", s.Health.InjuryChance},
		{"health.injuryCost", s.Health.InjuryCost},
		{"health.illnessChance", s.Health.IllnessChance},
		{"health.illnessCost", s.Health.IllnessCost},
		{"health.recoveryRate", s.Health.RecoveryRate},
		{"health.maternalDeathBase", s.Health.MaternalDeathBase},
		{"partnering.searchRate", s.Partnering.SearchRate},
		{"lottery.openWindowWeight", s.Lottery.OpenWindowWeight},
		{"lottery.conceptionChance", s.Lottery.ConceptionChance},
		{"lottery.girlShare", s.Lottery.GirlShare},
		{"work.sickLeaveHealth", s.Work.SickLeaveHealth},
		{"work.followParentChance", s.Work.FollowParentChance},
		{"work.crossZoneChance", s.Work.CrossZoneChance},
		{"work.masterMinSkill", s.Work.MasterMinSkill},
		{"work.skillGrowth", s.Work.SkillGrowth},
		{"work.shadowOutputFactor", s.Work.ShadowOutputFactor},
		{"work.retrainBelowShare", s.Work.RetrainBelowShare},
		{"work.retrainStartSkill", s.Work.RetrainStartSkill},
		{"resources.generatorWear", s.Resources.GeneratorWear},
		{"resources.repairRate", s.Resources.RepairRate},
		{"resources.failureBase", s.Resources.FailureBase},
		{"resources.failureConditionDrop", s.Resources.FailureConditionDrop},
		{"resources.backupPower", s.Resources.BackupPower},
		{"resources.airRecoveryShare", s.Resources.AirRecoveryShare},
		{"rations.childNeedShare", s.Rations.ChildNeedShare},
		{"rations.rationingShare", s.Rations.RationingShare},
		{"governance.ownZoneVote", s.Governance.OwnZoneVote},
		{"output.understaffedShare", s.Output.UnderstaffedShare},
		{"network.weakTieStrength", s.Network.WeakTieStrength},
		{"network.strongTieDailyChance", s.Network.StrongTieDailyChance},
		{"beliefs.taughtLegitimacy", s.Beliefs.TaughtLegitimacy},
		{"beliefs.educationWeight", s.Beliefs.EducationWeight},
		{"beliefs.legitimacyCostPerFlag", s.Beliefs.LegitimacyCostPerFlag},
		{"beliefs.passChance", s.Beliefs.PassChance},
		{"beliefs.cueChance", s.Beliefs.CueChance},
		{"beliefs.doseHideChance", s.Beliefs.DoseHideChance},
		{"control.surveillance", s.Control.Surveillance},
		{"control.arrestsPerOfficer", s.Control.ArrestsPerOfficer},
		{"control.findChance", s.Control.FindChance},
		{"control.nameChance", s.Control.NameChance},
		{"control.cleaningFailChance", s.Control.CleaningFailChance},
		{"control.cleaningLift", s.Control.CleaningLift},
		{"control.fearPerTrial", s.Control.FearPerTrial},
		{"rebellion.lossHardship", s.Rebellion.LossHardship},
		{"rebellion.scapegoatHardship", s.Rebellion.ScapegoatHardship},
		{"rebellion.activeThreshold", s.Rebellion.ActiveThreshold},
		{"rebellion.violentThreshold", s.Rebellion.ViolentThreshold},
		{"rebellion.strikeShare", s.Rebellion.StrikeShare},
		{"rebellion.killedShare", s.Rebellion.KilledShare},
		{"rebellion.sabotageChance", s.Rebellion.SabotageChance},
		{"rebellion.collapseLevel", s.Rebellion.CollapseLevel},
		{"rebellion.unrestShare", s.Rebellion.UnrestShare},
		{"rebellion.rebellionShare", s.Rebellion.RebellionShare},
		{"regime.emergencyAir", s.Regime.EmergencyAir},
		{"regime.electionLift", s.Regime.ElectionLift},
	}
	for _, u := range unit {
		if u.v < 0 || u.v > 1 {
			add("%s = %v, must be between 0 and 1", u.key, u.v)
		}
	}

	// Derived daily probabilities must stay ≤ 1, which [0, 1] alone does
	// not guarantee. Maternal risk = base × (2 − coverage) peaks at
	// coverage 0; injury chance is scaled by the department factor.
	if r := 2 * s.Health.MaternalDeathBase; r > 1 {
		add("health.maternalDeathBase = %v gives a maternal death risk of %v at zero Medical coverage, must give at most 1", s.Health.MaternalDeathBase, r)
	}
	f := s.Health.InjuryFactor
	if p := s.Health.InjuryChance * max(f.Mechanical, f.Farms, f.Other); p > 1 {
		add("health.injuryChance × largest health.injuryFactor = %v, must be at most 1", p)
	}

	// Strictly positive: divisors, or quantities that must exist for the
	// model to run at all.
	positive := []struct {
		key string
		v   float64
	}{
		{"mortality.gompertzA", s.Mortality.GompertzA},
		{"mortality.gompertzB", s.Mortality.GompertzB},
		{"init.floorsPerZone", float64(s.Init.FloorsPerZone)},
		{"coverage.needFactor", s.Coverage.NeedFactor},
		{"rations.adultKcal", s.Rations.AdultKcal},
		{"partnering.ageGapScaleYears", s.Partnering.AgeGapScaleYears},
		{"resources.powerDemandPerPerson", s.Resources.PowerDemandPerPerson},
		{"resources.capacityMargin", s.Resources.CapacityMargin},
		{"resources.kcalPerFarmUnit", s.Resources.KcalPerFarmUnit},
		{"lottery.targetPopulation", float64(s.Lottery.TargetPopulation)},
		{"population", float64(s.Population)},
		{"lottery.windowDays", float64(s.Lottery.WindowDays)},
		{"lottery.pregnancyDays", float64(s.Lottery.PregnancyDays)},
		{"resources.outageMinDays", float64(s.Resources.OutageMinDays)},
		{"output.invariantEvery", float64(s.Output.InvariantEvery)},
		{"output.frameEvery", float64(s.Output.FrameEvery)},
		{"space.fullDescentHours", s.Space.FullDescentHours},
		{"space.walkingHoursPerDay", s.Space.WalkingHoursPerDay},
		{"beliefs.halfLifeHeardYears", s.Beliefs.HalfLifeHeardYears},
		{"beliefs.halfLifeLivedYears", s.Beliefs.HalfLifeLivedYears},
		{"beliefs.relicHiddenYears", s.Beliefs.RelicHiddenYears},
		{"beliefs.relicKeptYears", s.Beliefs.RelicKeptYears},
		{"beliefs.waterDosingDays", float64(s.Beliefs.WaterDosingDays)},
		{"control.liftHalfLifeYears", s.Control.LiftHalfLifeYears},
		{"control.fearHalfLifeYears", s.Control.FearHalfLifeYears},
		{"control.deniedTries", float64(s.Control.DeniedTries)},
		{"rebellion.lossHalfLifeYears", s.Rebellion.LossHalfLifeYears},
		{"rebellion.arrestConstant", s.Rebellion.ArrestConstant},
		{"rebellion.unrestDays", float64(s.Rebellion.UnrestDays)},
		{"rebellion.rebellionDays", float64(s.Rebellion.RebellionDays)},
		{"rebellion.quietDays", float64(s.Rebellion.QuietDays)},
		{"rebellion.regimeDays", float64(s.Rebellion.RegimeDays)},
		{"rebellion.splitDays", float64(s.Rebellion.SplitDays)},
		{"rebellion.collapseDays", float64(s.Rebellion.CollapseDays)},
		{"rebellion.stairRepairDays", s.Rebellion.StairRepairDays},
		{"silo1.safeguardEscapes", float64(s.Silo1.SafeguardEscapes)},
		{"silo1.windowYears", s.Silo1.WindowYears},
		{"silo1.safeguardDelayDays", float64(s.Silo1.SafeguardDelayDays)},
	}
	for i, d := range s.Work.Departments {
		positive = append(positive, struct {
			key string
			v   float64
		}{fmt.Sprintf("work.departments[%d].share", i), d.Share})
	}
	for _, p := range positive {
		if p.v <= 0 {
			add("%s = %v, must be greater than 0", p.key, p.v)
		}
	}

	// Ordered pairs: the first must be strictly below the second, except
	// outage days, where a fixed length (min = max) is allowed.
	ordered := []struct {
		lowKey, highKey string
		low, high       int
		strict          bool
	}{
		{"work.shadowAge", "work.workerAge", s.Work.ShadowAge, s.Work.WorkerAge, true},
		{"work.workerAge", "work.retireAge", s.Work.WorkerAge, s.Work.RetireAge, true},
		{"lottery.motherAgeMin", "lottery.motherAgeMax", s.Lottery.MotherAgeMin, s.Lottery.MotherAgeMax, true},
		{"resources.outageMinDays", "resources.outageMaxDays", s.Resources.OutageMinDays, s.Resources.OutageMaxDays, false},
		{"partnering.minAge", "init.pairedMinAge", s.Partnering.MinAge, s.Init.PairedMinAge, false},
		// Each zone must keep at least one residential floor.
		{"init.farmFloors", "init.floorsPerZone", s.Init.FarmFloors, s.Init.FloorsPerZone, true},
		{"init.mechanicalFloors", "init.floorsPerZone", s.Init.MechanicalFloors, s.Init.FloorsPerZone, true},
	}
	for _, o := range ordered {
		if o.low > o.high || (o.strict && o.low == o.high) {
			rel := "below"
			if !o.strict {
				rel = "at most"
			}
			add("%s (%d) must be %s %s (%d)", o.lowKey, o.low, rel, o.highKey, o.high)
		}
	}

	// Department zones name a zone or "all" (works in every zone).
	for i, d := range s.Work.Departments {
		switch d.Zone {
		case "upTop", "mids", "downDeep", "all":
		default:
			add("work.departments[%d].zone = %q, must be upTop, mids, downDeep or all", i, d.Zone)
		}
	}

	// Rules name these departments: retraining draws from General (6.5),
	// injuries weigh Mechanical and Farms, Medical gives coverage (6.2),
	// IT, Judicial and Administration appoint the Mayor (6.8).
	for _, name := range []string{"general", "medical", "mechanical", "farms", "it", "judicial", "administration"} {
		if !slices.ContainsFunc(s.Work.Departments, func(d Department) bool { return d.Name == name }) {
			add("work.departments has no %q, which the rules need", name)
		}
	}

	if s.Governance.Mode != "appointment" && s.Governance.Mode != "election" {
		add("governance.mode = %q, must be appointment or election", s.Governance.Mode)
	}
	if _, err := s.Governance.TermYears(); err != nil {
		errs = append(errs, err)
	}

	// ODD §5: scripted events fall in the measured years; warm-up is
	// thrown away, and a shock there would never be seen.
	warmupDays, lastDay := s.WarmupYears*daysPerYear, (s.WarmupYears+s.Years)*daysPerYear
	for i, e := range s.Events {
		if e.Day <= warmupDays || e.Day > lastDay {
			add("events[%d].day = %d, must be in the measured years, %d to %d", i, e.Day, warmupDays+1, lastDay)
		}
		if e.Type == "" {
			add("events[%d].type is empty", i)
		}
	}

	// ODD §6.10: office floors. That each lies in its department's zone and
	// is not a farm or Mechanical floor is checked when the space module
	// starts, once the floors exist.
	floors := 3 * s.Init.FloorsPerZone
	names := slices.Sorted(maps.Keys(s.Space.OfficeFloors))
	for _, name := range names {
		i := slices.IndexFunc(s.Work.Departments, func(d Department) bool { return d.Name == name })
		if i < 0 || !HasOffices(s.Work.Departments[i]) {
			add("space.officeFloors.%s: not a department that works on office floors", name)
		}
		for _, f := range s.Space.OfficeFloors[name] {
			if f < 1 || f > floors {
				add("space.officeFloors.%s: floor %d, must be 1 to %d", name, f, floors)
			}
		}
	}
	for _, d := range s.Work.Departments {
		if HasOffices(d) && len(s.Space.OfficeFloors[d.Name]) == 0 {
			add("space.officeFloors has no floors for %s", d.Name)
		}
	}

	// ODD §6.13: a raid is a daily chance, R_raid / 365.
	if r := s.Rebellion; r.ViolentThreshold < r.ActiveThreshold {
		add("rebellion.violentThreshold = %v, must be at least rebellion.activeThreshold (%v)", r.ViolentThreshold, r.ActiveThreshold)
	}
	if s.Control.RaidsPerYear > daysPerYear {
		add("control.raidsPerYear = %v, must be at most %d", s.Control.RaidsPerYear, daysPerYear)
	}
	if f := s.Control.MinesFloor; f < 1 || f > floors {
		add("control.minesFloor = %d, must be 1 to %d", f, floors)
	}
	errs = append(errs, s.validatePact()...)

	// Fewer homes than households would leave people homeless on day 1.
	if s.Init.HomeCapacityMargin < 1 {
		add("init.homeCapacityMargin = %v, must be at least 1", s.Init.HomeCapacityMargin)
	}

	// Splits of people across zones and departments must account for everyone.
	deptSum := 0.0
	for _, d := range s.Work.Departments {
		deptSum += d.Share
	}
	sums := []struct {
		key string
		sum float64
	}{
		{"init.zoneSplit", s.Init.ZoneSplit.UpTop + s.Init.ZoneSplit.Mids + s.Init.ZoneSplit.DownDeep},
		{"work.allZoneSplit", s.Work.AllZoneSplit.UpTop + s.Work.AllZoneSplit.Mids + s.Work.AllZoneSplit.DownDeep},
		{"work.departments shares", deptSum},
	}
	for _, x := range sums {
		if math.Abs(x.sum-1) > sumTolerance {
			add("%s sums to %v, must sum to 1", x.key, x.sum)
		}
	}

	// ODD §6.15: each entry is checked against the scenario it would put in
	// force, so only once the scenario itself passes.
	if len(errs) == 0 && !s.inForce {
		errs = append(errs, s.validateRegime()...)
	}
	return errors.Join(errs...)
}

// validateRegime checks the playbook and the policy events (ODD §6.15):
// trigger, hold and lift, and the scenario in force with the entry alone
// running, so every lever passes its setting's checks and the Pact the
// checks of §6.9.
func (s *Scenario) validateRegime() []error {
	var errs []error
	check := func(key string, e Entry, event bool) {
		days, err := ParseHold(e.Hold)
		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("%s.%w", key, err))
		case days == HoldQuiet && (event || e.On != "unrestStart" && e.On != "rebellionStart"):
			errs = append(errs, fmt.Errorf("%s.hold = quiet, allowed only on unrestStart and rebellionStart", key))
		}
		if !event && !slices.Contains(Triggers, e.On) {
			errs = append(errs, fmt.Errorf("%s.on = %q, must be one of %s", key, e.On, strings.Join(Triggers, ", ")))
		}
		if e.Lift < -1 || e.Lift > 1 {
			errs = append(errs, fmt.Errorf("%s.lift = %v, must be between -1 and 1", key, e.Lift))
		}
		if err := s.InForce([]Entry{e}).Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%s puts in force a scenario that fails: %w", key, err))
		}
	}
	for i, e := range s.Regime.Playbook {
		check(fmt.Sprintf("regime.playbook[%d]", i), e, false)
	}
	for i, ev := range s.Events {
		if ev.Type == "policy" {
			check(fmt.Sprintf("events[%d]", i), ev.Entry(), true)
		}
	}
	// ODD §6.16: Silo 1's order is an entry started by an escape.
	o := s.Silo1.Orders
	if o.On != "escape" {
		errs = append(errs, fmt.Errorf("silo1.orders.on = %q, must be escape", o.On))
	}
	o.On = "suppressed" // any trigger that takes a hold of days or forever, for the shared checks
	check("silo1.orders", o, false)
	return errs
}

// CheckPact checks the Pact against ODD §6.9, as at load; the regime
// module uses it on the Pact several running entries put in force.
func (s *Scenario) CheckPact() error { return errors.Join(s.validatePact()...) }

// negatives walks every number in the scenario and reports those below 0.
func negatives(v reflect.Value, path string) []error {
	var errs []error
	switch v.Kind() {
	case reflect.Struct:
		for i := range v.NumField() {
			name, _, _ := strings.Cut(v.Type().Field(i).Tag.Get("yaml"), ",")
			if name == "lift" { // ODD §6.15: a lift may be a legitimacy cost; checked in validateRegime
				continue
			}
			errs = append(errs, negatives(v.Field(i), joinPath(path, name))...)
		}
	case reflect.Slice:
		for i := range v.Len() {
			errs = append(errs, negatives(v.Index(i), fmt.Sprintf("%s[%d]", path, i))...)
		}
	case reflect.Int, reflect.Int64:
		if v.Int() < 0 {
			errs = append(errs, fmt.Errorf("%s = %d, must not be negative", path, v.Int()))
		}
	case reflect.Float64:
		if v.Float() < 0 {
			errs = append(errs, fmt.Errorf("%s = %v, must not be negative", path, v.Float()))
		}
	}
	return errs
}

// validatePact checks the Pact table (ODD §6.9).
func (s *Scenario) validatePact() []error {
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }
	if len(s.Pact) > MaxRules {
		add("pact has %d rules, at most %d allowed", len(s.Pact), MaxRules)
	}
	ids := map[string]bool{}
	activeFor := map[string]string{} // offence → the active rule listing it
	for i, r := range s.Pact {
		key := fmt.Sprintf("pact[%d]", i)
		if r.ID == "" || ids[r.ID] {
			add("%s.id = %q, must be unique and not empty", key, r.ID)
		}
		ids[r.ID] = true
		if len(r.Offences) == 0 {
			add("%s.offences is empty", key)
		}
		for _, o := range r.Offences {
			if !slices.Contains(Offences, o) {
				add("%s.offences: %q, must be one of %s", key, o, strings.Join(Offences, ", "))
			}
			if other, ok := activeFor[o]; ok && r.Active {
				add("%s.offences: %q is also listed by active rule %s", key, o, other)
			}
			if r.Active {
				activeFor[o] = r.ID
			}
		}
		pens := []struct {
			key string
			p   *Penalty
		}{{key + ".first", &r.First}, {key + ".repeat", r.Repeat}}
		for _, pen := range pens {
			if pen.p == nil {
				continue
			}
			if pen.p.Prison < 0 || pen.p.Mines < 0 {
				add("%s: prison and mines days must not be negative", pen.key)
			}
			n := 0
			for _, on := range []bool{pen.p.Prison > 0, pen.p.Mines > 0, pen.p.Cleaning} {
				if on {
					n++
				}
			}
			if n > 1 {
				add("%s: at most one of prison, mines and cleaning", pen.key)
			}
		}
		if r.Visibility > 1 {
			add("%s.visibility = %v, must be between 0 and 1", key, r.Visibility)
		}
		switch r.Enforcer {
		case "judicial", "sheriff", "silo1":
		default:
			add("%s.enforcer = %q, must be judicial, sheriff or silo1", key, r.Enforcer)
		}
	}
	return errs
}
