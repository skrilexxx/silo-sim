// Package config loads a scenario file into a typed Scenario.
// The struct mirrors scenarios/baseline.yaml field for field; see ODD §5 and §7.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Scenario struct {
	// Extends names a parent scenario, relative to this file's folder.
	// It is cleared once the parent has been merged in.
	Extends     string     `yaml:"extends,omitempty"`
	Name        string     `yaml:"name"`
	Seed        uint64     `yaml:"seed"`
	WarmupYears int        `yaml:"warmupYears"`
	Years       int        `yaml:"years"`
	Population  int        `yaml:"population"`
	Modules     []string   `yaml:"modules"`
	Output      Output     `yaml:"output"`
	Init        Init       `yaml:"init"`
	Mortality   Mortality  `yaml:"mortality"`
	Health      Health     `yaml:"health"`
	Coverage    Coverage   `yaml:"coverage"`
	Partnering  Partnering `yaml:"partnering"`
	Lottery     Lottery    `yaml:"lottery"`
	Work        Work       `yaml:"work"`
	Resources   Resources  `yaml:"resources"`
	Rations     Rations    `yaml:"rations"`
	Governance  Governance `yaml:"governance"`
	Space       Space      `yaml:"space"`
	Network     Network    `yaml:"network"`
	Beliefs     Beliefs    `yaml:"beliefs"`
	Control     Control    `yaml:"control"`
	Pact        []Rule     `yaml:"pact"`
	Rebellion   Rebellion  `yaml:"rebellion"`
	Regime      Regime     `yaml:"regime"`
	Silo1       Silo1      `yaml:"silo1"`
	Events      []Event    `yaml:"events,omitempty"`

	// inForce marks a scenario built by InForce: its entries were checked
	// when the scenario loaded, so Validate doesn't check them again.
	inForce bool
}

// Event is a scripted shock (ODD §5, Scripted events): on Day, the module
// that defines Type makes it happen. Day counts from the run's first day,
// warm-up included, like the day column of the output files.
type Event struct {
	Day  int    `yaml:"day"`
	Type string `yaml:"type"`

	// Settings some types take (ODD §6.1 cull, §6.14 breakStairs).
	Segment    int     `yaml:"segment,omitempty"`
	Share      float64 `yaml:"share,omitempty"`
	Department string  `yaml:"department,omitempty"`

	// policy (ODD §6.15): an entry run on the event's day, with no trigger.
	Hold        string  `yaml:"hold,omitempty"`
	Set         Levers  `yaml:"set,omitempty"`
	Pact        []Rule  `yaml:"pact,omitempty"`
	Lift        float64 `yaml:"lift,omitempty"`
	WaterDosing bool    `yaml:"waterDosing,omitempty"`
}

// Entry is the policy entry a policy event carries (ODD §6.15).
func (e Event) Entry() Entry {
	return Entry{Hold: e.Hold, Set: e.Set, Pact: e.Pact, Lift: e.Lift, WaterDosing: e.WaterDosing}
}

// Zones holds one value per zone: a share of people or a policy weight.
type Zones struct {
	UpTop    float64 `yaml:"upTop"`
	Mids     float64 `yaml:"mids"`
	DownDeep float64 `yaml:"downDeep"`
}

type Output struct {
	Dir            string `yaml:"dir"`
	LogWarmup      bool   `yaml:"logWarmup"`
	InvariantEvery int    `yaml:"invariantEvery"`
	Frames         bool   `yaml:"frames"`
	FrameEvery     int    `yaml:"frameEvery"`
	// RunAfterCollapse keeps a run going after a collapse, to extinction or
	// the last day (ODD §1.2).
	RunAfterCollapse bool `yaml:"runAfterCollapse"`

	UnderstaffedShare float64 `yaml:"understaffedShare"`
}

type Init struct {
	FloorsPerZone       int     `yaml:"floorsPerZone"`
	ZoneSplit           Zones   `yaml:"zoneSplit"`
	PairedMinAge        int     `yaml:"pairedMinAge"`
	PairedShareOfAdults float64 `yaml:"pairedShareOfAdults"`
	HomeCapacityMargin  float64 `yaml:"homeCapacityMargin"`
	FarmFloors          int     `yaml:"farmFloors"`
	MechanicalFloors    int     `yaml:"mechanicalFloors"`
	FoodStockDays       int     `yaml:"foodStockDays"`
	PartsStockDays      int     `yaml:"partsStockDays"`
	GeneratorCondition  float64 `yaml:"generatorCondition"`
}

type Mortality struct {
	GompertzA        float64 `yaml:"gompertzA"`
	GompertzB        float64 `yaml:"gompertzB"`
	InfantHazard     float64 `yaml:"infantHazard"`
	HealthMultiplier float64 `yaml:"healthMultiplier"`
}

type Health struct {
	HungerDamage      float64      `yaml:"hungerDamage"`
	AirDamage         float64      `yaml:"airDamage"`
	InjuryChance      float64      `yaml:"injuryChance"`
	InjuryFactor      InjuryFactor `yaml:"injuryFactor"`
	InjuryCost        float64      `yaml:"injuryCost"`
	IllnessChance     float64      `yaml:"illnessChance"`
	IllnessCost       float64      `yaml:"illnessCost"`
	RecoveryRate      float64      `yaml:"recoveryRate"`
	MaternalDeathBase float64      `yaml:"maternalDeathBase"`
}

type InjuryFactor struct {
	Mechanical float64 `yaml:"mechanical"`
	Farms      float64 `yaml:"farms"`
	Other      float64 `yaml:"other"`
}

type Coverage struct {
	NeedFactor float64 `yaml:"needFactor"`
}

type Partnering struct {
	MinAge               int     `yaml:"minAge"`
	SearchRate           float64 `yaml:"searchRate"`
	SameZoneWeight       float64 `yaml:"sameZoneWeight"`
	SameDepartmentWeight float64 `yaml:"sameDepartmentWeight"`
	SameSexWeight        float64 `yaml:"sameSexWeight"`
	AgeGapScaleYears     float64 `yaml:"ageGapScaleYears"`
}

type Lottery struct {
	TargetPopulation    int     `yaml:"targetPopulation"`
	MaxPermitsPerMonth  int     `yaml:"maxPermitsPerMonth"`
	OpenWindowWeight    float64 `yaml:"openWindowWeight"`
	WindowDays          int     `yaml:"windowDays"`
	ConceptionChance    float64 `yaml:"conceptionChance"`
	PregnancyDays       int     `yaml:"pregnancyDays"`
	MotherAgeMin        int     `yaml:"motherAgeMin"`
	MotherAgeMax        int     `yaml:"motherAgeMax"`
	MaxChildren         int     `yaml:"maxChildren"`
	MinYoungestAgeYears int     `yaml:"minYoungestAgeYears"`
	GirlShare           float64 `yaml:"girlShare"`
}

type Work struct {
	ShadowAge           int          `yaml:"shadowAge"`
	WorkerAge           int          `yaml:"workerAge"`
	RetireAge           int          `yaml:"retireAge"`
	SickLeaveDays       int          `yaml:"sickLeaveDays"`
	SickLeaveHealth     float64      `yaml:"sickLeaveHealth"`
	FollowParentChance  float64      `yaml:"followParentChance"`
	CrossZoneChance     float64      `yaml:"crossZoneChance"`
	MaxShadowsPerMaster int          `yaml:"maxShadowsPerMaster"`
	MasterMinSkill      float64      `yaml:"masterMinSkill"`
	SkillGrowth         float64      `yaml:"skillGrowth"`
	ShadowOutputFactor  float64      `yaml:"shadowOutputFactor"`
	RetrainBelowShare   float64      `yaml:"retrainBelowShare"`
	RetrainStartSkill   float64      `yaml:"retrainStartSkill"`
	AllZoneSplit        Zones        `yaml:"allZoneSplit"`
	Departments         []Department `yaml:"departments"`
}

type Department struct {
	Name  string  `yaml:"name"`
	Zone  string  `yaml:"zone"`
	Share float64 `yaml:"share"`
}

type Resources struct {
	GeneratorWear           float64 `yaml:"generatorWear"`
	RepairRate              float64 `yaml:"repairRate"`
	FailureBase             float64 `yaml:"failureBase"`
	OutageMinDays           int     `yaml:"outageMinDays"`
	OutageMaxDays           int     `yaml:"outageMaxDays"`
	FailureConditionDrop    float64 `yaml:"failureConditionDrop"`
	BackupPower             float64 `yaml:"backupPower"`
	CapacityMargin          float64 `yaml:"capacityMargin"`
	PowerDemandPerPerson    float64 `yaml:"powerDemandPerPerson"`
	PowerDemandPerFarmFloor float64 `yaml:"powerDemandPerFarmFloor"`
	AirRecoveryShare        float64 `yaml:"airRecoveryShare"`
	KcalPerFarmUnit         float64 `yaml:"kcalPerFarmUnit"`
	PartsPerDayFullCoverage float64 `yaml:"partsPerDayFullCoverage"`
	PartsPerRepairDay       float64 `yaml:"partsPerRepairDay"`
	PartsPerFailure         float64 `yaml:"partsPerFailure"`
}

type Rations struct {
	AdultKcal          float64 `yaml:"adultKcal"`
	ChildAgeBelow      int     `yaml:"childAgeBelow"`
	ChildNeedShare     float64 `yaml:"childNeedShare"`
	RationingBelowDays int     `yaml:"rationingBelowDays"`
	RationingShare     float64 `yaml:"rationingShare"`
	Weights            Zones   `yaml:"weights"`
}

type Governance struct {
	Mode        string  `yaml:"mode"`
	Term        string  `yaml:"term"`
	OwnZoneVote float64 `yaml:"ownZoneVote"`
}

// Space is the location and travel module (ODD §6.10).
type Space struct {
	FullDescentHours   float64          `yaml:"fullDescentHours"`
	WalkingHoursPerDay float64          `yaml:"walkingHoursPerDay"`
	OfficeFloors       map[string][]int `yaml:"officeFloors"` // department name → its office floors
}

// Network is the social network module (ODD §6.11).
type Network struct {
	WeakTiesPerKind      int     `yaml:"weakTiesPerKind"`
	StrongTieDailyChance float64 `yaml:"strongTieDailyChance"`
	WeakTieStrength      float64 `yaml:"weakTieStrength"`
}

// Beliefs is the beliefs, knowledge and memory module (ODD §6.12).
type Beliefs struct {
	TaughtLegitimacy      float64 `yaml:"taughtLegitimacy"`
	EducationWeight       float64 `yaml:"educationWeight"`
	LegitimacyCostPerFlag float64 `yaml:"legitimacyCostPerFlag"`
	PassChance            float64 `yaml:"passChance"`
	HalfLifeHeardYears    float64 `yaml:"halfLifeHeardYears"`
	HalfLifeLivedYears    float64 `yaml:"halfLifeLivedYears"`
	Relics                int     `yaml:"relics"`
	RelicHiddenYears      float64 `yaml:"relicHiddenYears"`
	RelicKeptYears        float64 `yaml:"relicKeptYears"`
	CueChance             float64 `yaml:"cueChance"`
	FlamekeeperSeats      int     `yaml:"flamekeeperSeats"`
	DoseHideChance        float64 `yaml:"doseHideChance"`
	WaterDosingDays       int     `yaml:"waterDosingDays"`
}

// Control is the regime control module (ODD §6.13).
type Control struct {
	Surveillance       float64 `yaml:"surveillance"`
	ArrestsPerOfficer  float64 `yaml:"arrestsPerOfficer"`
	RaidsPerYear       float64 `yaml:"raidsPerYear"`
	FindChance         float64 `yaml:"findChance"`
	NameChance         float64 `yaml:"nameChance"`
	MinesFloor         int     `yaml:"minesFloor"`
	PrisonPlaces       int     `yaml:"prisonPlaces"`
	MinesPlaces        int     `yaml:"minesPlaces"`
	CleaningFailChance float64 `yaml:"cleaningFailChance"`
	CleaningLift       float64 `yaml:"cleaningLift"`
	LiftHalfLifeYears  float64 `yaml:"liftHalfLifeYears"`
	FearPerTrial       float64 `yaml:"fearPerTrial"`
	FearHalfLifeYears  float64 `yaml:"fearHalfLifeYears"`
	ImplantDoctors     int     `yaml:"implantDoctors"`
	DeniedTries        int     `yaml:"deniedTries"`
}

// Rebellion is the grievance and rebellion module (ODD §6.14).
type Rebellion struct {
	LossHardship      float64 `yaml:"lossHardship"`
	LossHalfLifeYears float64 `yaml:"lossHalfLifeYears"`
	ScapegoatHardship float64 `yaml:"scapegoatHardship"`
	ActiveThreshold   float64 `yaml:"activeThreshold"`
	ArrestConstant    float64 `yaml:"arrestConstant"`
	ViolentThreshold  float64 `yaml:"violentThreshold"`
	StrikeShare       float64 `yaml:"strikeShare"`
	OfficerEffect     float64 `yaml:"officerEffect"`
	RebelEffect       float64 `yaml:"rebelEffect"`
	KilledShare       float64 `yaml:"killedShare"`
	SabotageChance    float64 `yaml:"sabotageChance"`
	StairRepairDays   float64 `yaml:"stairRepairDays"`
	PointOfNoReturn   int     `yaml:"pointOfNoReturn"`

	// Definitions of unrest and rebellion (ODD §1.2, decision D1).
	UnrestShare    float64 `yaml:"unrestShare"`
	UnrestDays     int     `yaml:"unrestDays"`
	RebellionShare float64 `yaml:"rebellionShare"`
	RebellionDays  int     `yaml:"rebellionDays"`
	QuietDays      int     `yaml:"quietDays"`
	RegimeDays     int     `yaml:"regimeDays"`
	MaxDays        int     `yaml:"maxDays"` // a rebellion still running this many days after its start is crushed (D15)
	SplitDays      int     `yaml:"splitDays"`
	SplitPeople    int     `yaml:"splitPeople"`
	CollapseLevel  float64 `yaml:"collapseLevel"`
	CollapseDays   int     `yaml:"collapseDays"`
}

// Regime is the regime policy and response module (ODD §6.15). Its own
// settings are the peacetime values of the levers no other section has.
type Regime struct {
	EmergencyAir float64 `yaml:"emergencyAir"`
	ElectionLift float64 `yaml:"electionLift"`
	Denial       bool    `yaml:"denial"`
	DoseConvicts int     `yaml:"doseConvicts"`
	Playbook     []Entry `yaml:"playbook"`
}

// Silo1 is Silo 1, lean (ODD §6.16): the escapes within a window that
// set off the Safeguard, its delay, and the order sent after each escape.
type Silo1 struct {
	SafeguardEscapes   int     `yaml:"safeguardEscapes"`
	WindowYears        float64 `yaml:"windowYears"`
	SafeguardDelayDays int     `yaml:"safeguardDelayDays"`
	Orders             Entry   `yaml:"orders"`
}

// Entry is a playbook entry (ODD §6.15, Entries): on its trigger, it sets
// levers and Pact rows while it runs, adds its lift once, and may start a
// water dosing. A policy event carries the same parts with no trigger.
type Entry struct {
	On          string  `yaml:"on"`
	Hold        string  `yaml:"hold"` // quiet, forever, or a number of days
	Set         Levers  `yaml:"set,omitempty"`
	Pact        []Rule  `yaml:"pact,omitempty"`
	Lift        float64 `yaml:"lift,omitempty"`
	WaterDosing bool    `yaml:"waterDosing,omitempty"`
}

// Levers are the policy levers (ODD §6.15, The policy vector). Each
// overrides the setting it names; nil leaves it alone.
type Levers struct {
	Surveillance      *float64 `yaml:"surveillance,omitempty"`
	ArrestsPerOfficer *float64 `yaml:"arrestsPerOfficer,omitempty"`
	RaidsPerYear      *float64 `yaml:"raidsPerYear,omitempty"`
	FindChance        *float64 `yaml:"findChance,omitempty"`
	PrisonPlaces      *int     `yaml:"prisonPlaces,omitempty"`
	MinesPlaces       *int     `yaml:"minesPlaces,omitempty"`
	TaughtLegitimacy  *float64 `yaml:"taughtLegitimacy,omitempty"`
	RationWeights     *Zones   `yaml:"rationWeights,omitempty"`
	ScapegoatHardship *float64 `yaml:"scapegoatHardship,omitempty"`
	Denial            *bool    `yaml:"denial,omitempty"`
	DoseConvicts      *int     `yaml:"doseConvicts,omitempty"`
	Governance        *string  `yaml:"governance,omitempty"`
	MayorTerm         *string  `yaml:"mayorTerm,omitempty"`
	EmergencyAir      *float64 `yaml:"emergencyAir,omitempty"`
}

// Triggers are the rebellion events that start playbook entries (ODD
// §6.15), in the order their entries start on a day.
var Triggers = []string{"unrestStart", "rebellionStart", "suppressed", "split", "regimeChange"}

// Hold values besides a number of days ≥ 1 (ODD §6.15, Entries).
const (
	HoldForever = 0
	HoldQuiet   = -1 // until the unrest spell or rebellion that started it ends
)

// ParseHold reads an entry's hold: quiet, forever, or days ≥ 1.
func ParseHold(h string) (int, error) {
	switch h {
	case "quiet":
		return HoldQuiet, nil
	case "forever":
		return HoldForever, nil
	}
	n, err := strconv.Atoi(h)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("hold = %q, must be quiet, forever or a number of days of at least 1", h)
	}
	return n, nil
}

// InForce is the scenario with the given entries running, in the order
// they started (ODD §6.15): each entry's levers and Pact rows override the
// scenario's, a later entry over an earlier one. Its Pact holds every rule
// the scenario, a playbook entry or a policy event names, in a fixed
// order (the scenario's rows, then new ids as first named), so a rule
// keeps its index, which convictions are kept by (ODD §6.13); a rule only
// entries add is inactive while none of them runs. The playbook and the
// events are left out: they are not in force, they change it.
func (s *Scenario) InForce(running []Entry) *Scenario {
	p := *s
	p.Regime.Playbook, p.Events, p.inForce = nil, nil, true
	p.Pact = slices.Clone(s.Pact)
	add := func(rows []Rule) {
		for _, r := range rows {
			if !slices.ContainsFunc(p.Pact, func(x Rule) bool { return x.ID == r.ID }) {
				r.Active = false
				p.Pact = append(p.Pact, r)
			}
		}
	}
	for _, e := range s.Regime.Playbook {
		add(e.Pact)
	}
	for _, ev := range s.Events {
		if ev.Type == "policy" {
			add(ev.Pact)
		}
	}
	add(s.Silo1.Orders.Pact)
	for _, e := range running {
		p.apply(e)
	}
	return &p
}

func (p *Scenario) apply(e Entry) {
	l := e.Set
	set(&p.Control.Surveillance, l.Surveillance)
	set(&p.Control.ArrestsPerOfficer, l.ArrestsPerOfficer)
	set(&p.Control.RaidsPerYear, l.RaidsPerYear)
	set(&p.Control.FindChance, l.FindChance)
	set(&p.Control.PrisonPlaces, l.PrisonPlaces)
	set(&p.Control.MinesPlaces, l.MinesPlaces)
	set(&p.Beliefs.TaughtLegitimacy, l.TaughtLegitimacy)
	set(&p.Rations.Weights, l.RationWeights)
	set(&p.Rebellion.ScapegoatHardship, l.ScapegoatHardship)
	set(&p.Regime.Denial, l.Denial)
	set(&p.Regime.DoseConvicts, l.DoseConvicts)
	set(&p.Governance.Mode, l.Governance)
	set(&p.Governance.Term, l.MayorTerm)
	set(&p.Regime.EmergencyAir, l.EmergencyAir)
	for _, r := range e.Pact {
		p.Pact[slices.IndexFunc(p.Pact, func(x Rule) bool { return x.ID == r.ID })] = r
	}
}

func set[T any](dst, v *T) {
	if v != nil {
		*dst = *v
	}
}

// Rule is one row of the Pact (ODD §6.9).
type Rule struct {
	ID         string   `yaml:"id"`
	Active     bool     `yaml:"active"`
	Offences   []string `yaml:"offences"`
	Detection  float64  `yaml:"detection"`
	First      Penalty  `yaml:"first"`
	Repeat     *Penalty `yaml:"repeat,omitempty"` // nil: the same as First
	Enforcer   string   `yaml:"enforcer"`
	Visibility float64  `yaml:"visibility"`
}

// Penalty is what a conviction brings (ODD §6.9); the zero value is a
// warning. Prison and Mines are days.
type Penalty struct {
	Prison     int  `yaml:"prison,omitempty"`
	Mines      int  `yaml:"mines,omitempty"`
	Cleaning   bool `yaml:"cleaning,omitempty"`
	Confiscate bool `yaml:"confiscate,omitempty"`
	FlagFamily bool `yaml:"flagFamily,omitempty"`
}

// Offences are the offence types modules can report (ODD §6.9), in the
// order of world.Offence.
var Offences = []string{"holdRelic", "tradeRelic", "speakForbidden", "askToGoOut", "unpermittedConception",
	"contactAttempt", "organise", "violence", "sabotage"}

// MaxRules bounds the Pact: a person's charges are one bit per rule.
const MaxRules = 32

// HasOffices reports whether a department works on office floors: every
// department but Farms and Mechanical (pure work floors) and the
// all-zone ones (General services, Porters).
func HasOffices(d Department) bool {
	return d.Zone != "all" && d.Name != "farms" && d.Name != "mechanical"
}

// TermYears reads governance.term: "<n>y" gives n ≥ 1, "life" gives 0.
func (g Governance) TermYears() (int, error) {
	if g.Term == "life" {
		return 0, nil
	}
	n, err := strconv.Atoi(strings.TrimSuffix(g.Term, "y"))
	if err != nil || n < 1 || !strings.HasSuffix(g.Term, "y") {
		return 0, fmt.Errorf("governance.term = %q, must be life or a number of years like 5y", g.Term)
	}
	return n, nil
}

// Load reads a scenario file, merges in its `extends` parents, and checks
// that no key is unknown, none is missing, and every value is in range.
func Load(path string) (*Scenario, error) {
	root, err := loadNode(path, nil)
	if err != nil {
		return nil, err
	}
	if errs := missingKeys(root, reflect.TypeFor[Scenario](), ""); len(errs) > 0 {
		return nil, fmt.Errorf("%s: %w", path, errors.Join(errs...))
	}
	var s Scenario
	if err := root.Decode(&s); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	s.Extends = ""
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &s, nil
}

// loadNode returns the file's top-level mapping with all parents merged
// underneath it. chain holds the files already visited, to catch loops.
func loadNode(path string, chain []string) (*yaml.Node, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if slices.Contains(chain, abs) {
		return nil, fmt.Errorf("extends loop: %s", strings.Join(append(chain, abs), " -> "))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Decoding each file on its own with KnownFields catches unknown keys
	// and wrong types, with the right file name and line in the message.
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var own Scenario
	if err := dec.Decode(&own); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: top level must be a mapping", path)
	}
	if own.Extends == "" {
		return root, nil
	}
	parent, err := loadNode(filepath.Join(filepath.Dir(path), own.Extends), append(chain, abs))
	if err != nil {
		return nil, err
	}
	merge(parent, root)
	return parent, nil
}

// merge writes src over dst. Mappings merge key by key, so a child can
// override one value deep inside a section. Anything else, lists included,
// replaces the parent's value whole.
func merge(dst, src *yaml.Node) {
	for i := 0; i < len(src.Content); i += 2 {
		key, val := src.Content[i], src.Content[i+1]
		if key.Value == "extends" {
			continue
		}
		old := lookup(dst, key.Value)
		switch {
		case old == nil:
			dst.Content = append(dst.Content, key, val)
		case old.Kind == yaml.MappingNode && val.Kind == yaml.MappingNode:
			merge(old, val)
		default:
			*old = *val
		}
	}
}

// lookup returns the value under key in a mapping node, or nil.
func lookup(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// missingKeys reports every struct field with no key in the YAML. A field
// tagged omitempty may be absent, but what it holds is checked when it is
// there (each scripted event needs all its keys). Without this check a
// forgotten key would silently become zero.
func missingKeys(n *yaml.Node, t reflect.Type, path string) []error {
	var errs []error
	switch t.Kind() {
	case reflect.Struct:
		for i := range t.NumField() {
			if !t.Field(i).IsExported() {
				continue
			}
			name, opts, _ := strings.Cut(t.Field(i).Tag.Get("yaml"), ",")
			key := joinPath(path, name)
			child := lookup(n, name)
			if child == nil {
				if opts != "omitempty" {
					errs = append(errs, fmt.Errorf("missing key %s", key))
				}
				continue
			}
			errs = append(errs, missingKeys(child, t.Field(i).Type, key)...)
		}
	case reflect.Pointer: // an optional section: when it is there, it is whole
		errs = append(errs, missingKeys(n, t.Elem(), path)...)
	case reflect.Slice:
		if n.Kind == yaml.SequenceNode {
			for i, item := range n.Content {
				errs = append(errs, missingKeys(item, t.Elem(), fmt.Sprintf("%s[%d]", path, i))...)
			}
		}
	}
	return errs
}

func joinPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}
