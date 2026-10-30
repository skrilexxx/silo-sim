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
	Events      []Event    `yaml:"events,omitempty"`
}

// Event is a scripted shock (ODD §5, Scripted events): on Day, the module
// that defines Type makes it happen. Day counts from the run's first day,
// warm-up included, like the day column of the output files.
type Event struct {
	Day  int    `yaml:"day"`
	Type string `yaml:"type"`
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
