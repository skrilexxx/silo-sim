// Package output writes a run's files (ODD §8). Only the logging steps
// read the world; nothing here changes it.
package output

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/rng"
	"github.com/skrilexxx/silo-sim/internal/world"
)

// Writer is the module that writes the run's files (ODD §8): daily.csv
// at the logging phase, yearly.csv at the snapshot phase, and events.jsonl
// from World.Events plus the department events it detects itself.
//
// Each row counts what happened since the row before. Steps after
// logging (lottery, housing) therefore show up in the next day's row.
// Logging runs during warm-up too, writing nothing, so the first measured
// row starts from the counts at the end of warm-up.
type Writer struct {
	files  []*os.File
	daily  *bufio.Writer
	yearly *bufio.Writer
	events *bufio.Writer
	row    []byte // reused, so logging allocates nothing per tick

	w                 *world.World
	warmupDays        world.Tick // 0 if warm-up is logged too
	understaffedShare float64
	born0             int    // people on day 0; later ones were born
	under             []bool // per department: below understaffedShare of target

	// Counts at the previous row.
	births, deaths, permits int
	zoneMoves               int // at the previous yearly row
}

// ODD §8: the age pyramid uses 5-year bands, the last open-ended.
const (
	bandYears = 5
	topBand   = 100 // the last band is 100+
	bands     = topBand/bandYears + 1
)

// New creates daily.csv, yearly.csv and events.jsonl in dir. Headers are
// written in Init, once the departments are known.
func New(dir string) (*Writer, error) {
	o := &Writer{}
	for _, f := range []struct {
		name string
		buf  **bufio.Writer
	}{{"daily.csv", &o.daily}, {"yearly.csv", &o.yearly}, {"events.jsonl", &o.events}} {
		file, err := os.Create(filepath.Join(dir, f.name))
		if err != nil {
			o.Close()
			return nil, err
		}
		o.files = append(o.files, file)
		*f.buf = bufio.NewWriter(file)
	}
	return o, nil
}

func (o *Writer) Name() string       { return "output" }
func (o *Writer) Requires() []string { return nil }

func (o *Writer) Init(w *world.World, cfg *config.Scenario, _ *rng.Stream) error {
	o.w, o.born0 = w, len(w.People)
	o.understaffedShare = cfg.Output.UnderstaffedShare
	o.under = make([]bool, len(w.Depts))
	if !cfg.Output.LogWarmup {
		o.warmupDays = world.Tick(cfg.WarmupYears * engine.DaysPerYear)
	}

	if _, err := o.daily.WriteString("day,population,popUpTop,popMids,popDownDeep,births,deaths,permits," +
		"openWindows,queued,pregnancies,foodStockDays,powerFactor,generatorCondition,outage,water,air," +
		"healthUpTop,healthMids,healthDownDeep,hardshipUpTop,hardshipMids,hardshipDownDeep," +
		"medicalCoverage,mechanicalCoverage\n"); err != nil {
		return err
	}
	h := []byte("day")
	for b := range bands {
		for _, sex := range "fm" {
			h = append(h, ',', byte(sex))
			h = strconv.AppendInt(h, int64(b*bandYears), 10)
			if b == bands-1 {
				h = append(h, "plus"...)
			} else {
				h = append(h, '_')
				h = strconv.AppendInt(h, int64(b*bandYears+bandYears-1), 10)
			}
		}
	}
	for _, d := range w.Depts {
		h = append(h, ","+d.Name+"Workers,"+d.Name+"Target"...)
	}
	h = append(h, ",zoneMoves,mayor,mayorZone\n"...)
	_, err := o.yearly.Write(h)
	return err
}

func (o *Writer) Steps() []engine.Step {
	return []engine.Step{
		{Phase: engine.PhaseLogDaily, Run: o.logDaily},
		{Phase: engine.PhaseSnapshot, Run: o.logYearly},
	}
}

// ODD §8: daily.csv. Births, deaths and permits count since the previous
// row; openWindows are women whose window is running, queued those who won
// a permit and wait for Medical (together the lottery's E term, §6.4).
// Hardship is 1 − intake/need; a zone with nobody has health 0.
func (o *Writer) logDaily(w *world.World, _ *rng.Stream) error {
	births, deaths := len(w.People)-o.born0, len(w.People)-len(w.Alive)
	permits := w.Counts.Permits
	if o.warmupDays == 0 || w.Day > o.warmupDays {
		if err := o.writeDaily(w, births-o.births, deaths-o.deaths, permits-o.permits); err != nil {
			return err
		}
	} else {
		o.zoneMoves = w.Counts.ZoneMoves
	}
	o.births, o.deaths, o.permits = births, deaths, permits
	o.checkDepartments(w)
	return o.writeEvents()
}

func (o *Writer) writeDaily(w *world.World, births, deaths, permits int) error {
	var pop [world.NumZones]int
	var health [world.NumZones]float64
	open, queued, pregnant := 0, 0, 0
	for _, id := range w.Alive {
		p := &w.People[id]
		z := zoneOf(w, id)
		pop[z]++
		health[z] += p.Health
		switch p.Implant {
		case world.WindowOpen:
			open++
		case world.Queued:
			queued++
		case world.Pregnant:
			pregnant++
		}
	}
	s := &w.Silo
	foodDays := 0.0
	if s.FoodNeed > 0 {
		foodDays = s.FoodStock / s.FoodNeed
	}
	outage := 0
	if s.GeneratorDown > 0 {
		outage = 1
	}

	b := strconv.AppendInt(o.row[:0], int64(w.Day), 10)
	for _, n := range []int{len(w.Alive), pop[world.UpTop], pop[world.Mids], pop[world.DownDeep],
		births, deaths, permits, open, queued, pregnant} {
		b = append(b, ',')
		b = strconv.AppendInt(b, int64(n), 10)
	}
	b = appendF(b, foodDays, 1)
	b = appendF(b, s.PowerFactor(), 4)
	b = appendF(b, s.GeneratorCondition, 4)
	b = append(b, ',')
	b = strconv.AppendInt(b, int64(outage), 10)
	b = appendF(b, s.WaterSupply, 4)
	b = appendF(b, s.AirQuality, 4)
	for z := range world.NumZones {
		avg := 0.0
		if pop[z] > 0 {
			avg = health[z] / float64(pop[z])
		}
		b = appendF(b, avg, 4)
	}
	for z := range world.NumZones {
		b = appendF(b, 1-s.Intake[z], 4)
	}
	b = appendF(b, coverage(w, "medical"), 4)
	b = appendF(b, coverage(w, "mechanical"), 4)
	o.row = append(b, '\n')
	_, err := o.daily.Write(o.row)
	return err
}

// ODD §8: yearly.csv: the age pyramid (women and men per 5-year band),
// workers and target per department, people who moved to another zone
// since the previous yearly row, and the Mayor. The snapshot (§3 step 15)
// runs before governance (step 16), so a row shows the Mayor before that
// day's change; a Mayor who died since the last governance step is still
// shown, with no zone.
func (o *Writer) logYearly(w *world.World, _ *rng.Stream) error {
	var pyramid [bands][2]int
	for _, id := range w.Alive {
		p := &w.People[id]
		band := min(int(w.Day-p.BirthDay)/engine.DaysPerYear/bandYears, bands-1)
		pyramid[band][p.Sex]++
	}
	b := strconv.AppendInt(o.row[:0], int64(w.Day), 10)
	for _, band := range pyramid {
		for _, n := range band {
			b = append(b, ',')
			b = strconv.AppendInt(b, int64(n), 10)
		}
	}
	for _, d := range w.Depts {
		b = append(b, ',')
		b = strconv.AppendInt(b, int64(d.Headcount), 10)
		b = appendF(b, d.Target, 1)
	}
	b = append(b, ',')
	b = strconv.AppendInt(b, int64(w.Counts.ZoneMoves-o.zoneMoves), 10)
	o.zoneMoves = w.Counts.ZoneMoves
	b = append(b, ',')
	b = strconv.AppendInt(b, int64(w.Silo.Mayor), 10)
	b = append(b, ',')
	if m := w.Silo.Mayor; m != world.None && w.People[m].Alive {
		b = append(b, zoneOf(w, m).String()...)
	}
	o.row = append(b, '\n')
	_, err := o.yearly.Write(o.row)
	return err
}

// checkDepartments logs a department whose workers fall below
// understaffedShare of its target, and again when it is back above (ODD
// §8). Headcounts are recounted daily (life stages), targets monthly.
func (o *Writer) checkDepartments(w *world.World) {
	for d, dep := range w.Depts {
		below := float64(dep.Headcount) < o.understaffedShare*dep.Target
		if below == o.under[d] {
			continue
		}
		o.under[d] = below
		kind := "restaffed"
		if below {
			kind = "understaffed"
		}
		w.Events = append(w.Events, world.DepartmentEvent{EventHeader: world.EventHeader{Day: w.Day, Type: kind},
			Department: dep.Name, Workers: dep.Headcount, Target: dep.Target})
	}
}

// writeEvents writes and clears World.Events, dropping warm-up events.
// With no warm-up nothing is dropped, day-0 events included.
func (o *Writer) writeEvents() error {
	for _, e := range o.w.Events {
		if o.warmupDays > 0 && e.When() <= o.warmupDays {
			continue
		}
		line, err := json.Marshal(e)
		if err != nil {
			return err
		}
		if _, err := o.events.Write(append(line, '\n')); err != nil {
			return err
		}
	}
	o.w.Events = o.w.Events[:0]
	return nil
}

// Close writes the last events, flushes and closes every file. A failed
// flush means lost rows, so its error must not be ignored.
func (o *Writer) Close() error {
	var err error
	if o.w != nil {
		err = o.writeEvents()
	}
	for i, b := range []*bufio.Writer{o.daily, o.yearly, o.events} {
		if i >= len(o.files) {
			break
		}
		if ferr := b.Flush(); err == nil {
			err = ferr
		}
		if cerr := o.files[i].Close(); err == nil {
			err = cerr
		}
	}
	return err
}

func appendF(b []byte, x float64, prec int) []byte {
	return strconv.AppendFloat(append(b, ','), x, 'f', prec, 64)
}

// coverage is a department's coverage, or 1 if there is no such
// department (the work module is off), as demography and health assume.
func coverage(w *world.World, name string) float64 {
	for _, d := range w.Depts {
		if d.Name == name {
			return d.Coverage
		}
	}
	return 1
}

// zoneOf is the zone of a living person's household.
func zoneOf(w *world.World, id world.PersonID) world.Zone {
	return w.Floors[w.Households[w.People[id].Household].Floor-1].Zone
}

// RunInfo is the content of run.json.
type RunInfo struct {
	Scenario       map[string]any `json:"scenario"` // resolved, after extends
	Seed           uint64         `json:"seed"`
	GitCommit      string         `json:"gitCommit"`
	GoVersion      string         `json:"goVersion"`
	Finished       bool           `json:"finished"`
	RuntimeSeconds float64        `json:"runtimeSeconds"`
	Error          string         `json:"error,omitempty"`
}

// ScenarioMap turns the scenario into a map keyed like the YAML file, so
// run.json uses the same names as the scenario (warmupYears, not WarmupYears).
func ScenarioMap(cfg *config.Scenario) (map[string]any, error) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	err = yaml.Unmarshal(data, &m)
	return m, err
}

// WriteRunJSON writes dir/run.json.
func WriteRunJSON(dir string, info RunInfo) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "run.json"), append(data, '\n'), 0o644)
}
