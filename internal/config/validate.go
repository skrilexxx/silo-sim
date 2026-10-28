package config

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
)

// sumTolerance absorbs float rounding when shares are added up
// (0.30 + 0.20 + … is not exactly 1 in binary).
const sumTolerance = 1e-9

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
		{"work.sickRetireHealth", s.Work.SickRetireHealth},
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
	}
	for _, o := range ordered {
		if o.low > o.high || (o.strict && o.low == o.high) {
			add("%s (%d) must be below %s (%d)", o.lowKey, o.low, o.highKey, o.high)
		}
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

	return errors.Join(errs...)
}

// negatives walks every number in the scenario and reports those below 0.
func negatives(v reflect.Value, path string) []error {
	var errs []error
	switch v.Kind() {
	case reflect.Struct:
		for i := range v.NumField() {
			name, _, _ := strings.Cut(v.Type().Field(i).Tag.Get("yaml"), ",")
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
