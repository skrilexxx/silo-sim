// Package config loads a scenario file into a typed Scenario.
// The struct mirrors scenarios/baseline.yaml field for field; see ODD §5 and §7.
package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Scenario struct {
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
}

type Init struct {
	ZoneSplit           Zones   `yaml:"zoneSplit"`
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
}

type Work struct {
	ShadowAge           int          `yaml:"shadowAge"`
	WorkerAge           int          `yaml:"workerAge"`
	RetireAge           int          `yaml:"retireAge"`
	SickRetireDays      int          `yaml:"sickRetireDays"`
	SickRetireHealth    float64      `yaml:"sickRetireHealth"`
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

// Load reads and strictly parses a scenario file.
func Load(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

// Parse decodes YAML into a Scenario. Unknown keys are errors, so a typo
// fails loudly instead of silently leaving a parameter at zero.
func Parse(data []byte) (*Scenario, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var s Scenario
	if err := dec.Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}
