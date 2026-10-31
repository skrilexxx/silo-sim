// Package output writes a run's files (ODD §8). Only the logging steps
// read the world; nothing here changes it.
package output

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

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
	dir    string

	// frames.csv.gz, only if output.frames is on (nil otherwise).
	frames     *bufio.Writer
	gz         *gzip.Writer
	framesFile *os.File
	frameEvery world.Tick
	metrics    []FrameMetric
	vals       []float64 // one per floor, reused

	w                 *world.World
	warmupDays        world.Tick // 0 if warm-up is logged too
	understaffedShare float64
	born0             int        // people on day 0; later ones were born
	under             []bool     // per department: below understaffedShare of target
	ages              []int      // reused, in days
	degrees           []int      // reused: ties per living person (network)
	adultAge          world.Tick // worker age in days (beliefs columns)
	ties              []world.Tie

	// Counts at the previous row.
	births, deaths, permits  int
	zoneMoves                int              // at the previous yearly row
	tally                    world.Tally      // control, at the previous yearly row
	unrestSpells, rebellions int              // rebellion, at the previous yearly row
	fights                   world.FightTally // rebellion, at the previous yearly row
}

// ODD §8: the age pyramid uses 5-year bands, the last open-ended.
const (
	bandYears = 5
	topBand   = 100 // the last band is 100+
	bands     = topBand/bandYears + 1
)

// FrameMetric is one per-floor value in frames.csv.gz: Fill writes one
// value per floor into v (v[i] is floor i+1). It only reads the world.
type FrameMetric struct {
	Name string
	Fill func(w *world.World, v []float64)
}

// FrameSource is a module with per-floor metrics for frames.csv.gz.
type FrameSource interface {
	FrameMetrics() []FrameMetric
}

// New creates daily.csv, yearly.csv and events.jsonl in dir. Headers are
// written in Init, once the departments are known.
func New(dir string) (*Writer, error) {
	o := &Writer{dir: dir, metrics: baselineMetrics}
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
	o.adultAge = world.Tick(cfg.Work.WorkerAge * engine.DaysPerYear)
	o.under = make([]bool, len(w.Depts))
	if !cfg.Output.LogWarmup {
		o.warmupDays = world.Tick(cfg.WarmupYears * engine.DaysPerYear)
	}

	if _, err := o.daily.WriteString("day,population,popUpTop,popMids,popDownDeep,births,deaths,permits," +
		"openWindows,queued,pregnancies,foodStockDays,rationing,powerFactor,generatorCondition,outage,water,air," +
		"healthUpTop,healthMids,healthDownDeep,hardshipUpTop,hardshipMids,hardshipDownDeep," +
		"medicalCoverage,mechanicalCoverage"); err != nil {
		return err
	}
	if w.Rebellion != nil {
		if _, err := o.daily.WriteString(",active,joined,violent,onStrike,heldUpTop,heldMids,heldDownDeep"); err != nil {
			return err
		}
	}
	if err := o.daily.WriteByte('\n'); err != nil {
		return err
	}
	h := []byte("day,medianAge")
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
	h = append(h, ",mechanicalInZone,zoneMoves,mayor,mayorZone"...)
	if w.Network != nil {
		h = append(h, ",meanTies,tiesP10,tiesP50,tiesP90"...)
		for k := range world.NumTieKinds {
			name := k.String()
			h = append(h, ",talks"+strings.ToUpper(name[:1])+name[1:]...)
		}
	}
	if w.Beliefs != nil {
		h = append(h, ",meanL,meanLUpTop,meanLMids,meanLDownDeep"...)
		for f := range world.NumFlags {
			name := f.String()
			h = append(h, ",know"+strings.ToUpper(name[:1])+name[1:]...)
		}
		h = append(h, ",hiddenAny,dosed,relicsHidden,relicsHeld,flamekeepers"...)
	}
	if w.Control != nil {
		h = append(h, ",actsSeen,arrests,warnings,prison,mines,cleanings,failedCleanings,relicsTaken,raids,deniedWins,flaggedBirths,held,flagged,openCases,lift,fear"...)
	}
	if w.Rebellion != nil {
		h = append(h, ",meanH,meanHUpTop,meanHMids,meanHDownDeep,meanG,meanGUpTop,meanGMids,meanGDownDeep,unrestSpells,rebellions,rebelsKilled,rebelsArrested,officersKilled,officersWounded"...)
	}
	h = append(h, '\n')
	if _, err := o.yearly.Write(h); err != nil {
		return err
	}
	if cfg.Output.Frames {
		return o.openFrames(w, cfg.Output.FrameEvery)
	}
	return nil
}

// AddFrameMetrics adds a module's metrics after the baseline's, in the
// order given, so the file's row order is fixed.
func (o *Writer) AddFrameMetrics(ms ...FrameMetric) {
	o.metrics = append(o.metrics, ms...)
}

func (o *Writer) openFrames(w *world.World, every int) error {
	f, err := os.Create(filepath.Join(o.dir, "frames.csv.gz"))
	if err != nil {
		return err
	}
	// gzip's header holds no time or name unless set, so the same seed
	// gives the same bytes.
	o.framesFile, o.gz = f, gzip.NewWriter(f)
	o.frames = bufio.NewWriter(o.gz)
	o.frameEvery = world.Tick(every)
	o.vals = make([]float64, len(w.Floors))
	_, err = o.frames.WriteString("day,floor,metric,value\n")
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
	if o.warmupDays > 0 && w.Day == o.warmupDays+1 {
		// A department still understaffed from warm-up is logged again on
		// the first measured day, since its warm-up event was dropped.
		clear(o.under)
	}
	if o.warmupDays == 0 || w.Day > o.warmupDays {
		if err := o.writeDaily(w, births-o.births, deaths-o.deaths, permits-o.permits); err != nil {
			return err
		}
		if o.frames != nil && w.Day%o.frameEvery == 0 {
			if err := o.writeFrame(w); err != nil {
				return err
			}
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

	b := strconv.AppendInt(o.row[:0], int64(w.Day), 10)
	for _, n := range []int{len(w.Alive), pop[world.UpTop], pop[world.Mids], pop[world.DownDeep],
		births, deaths, permits, open, queued, pregnant} {
		b = append(b, ',')
		b = strconv.AppendInt(b, int64(n), 10)
	}
	b = appendF(b, foodDays, 1)
	b = append(b, ',')
	b = strconv.AppendInt(b, int64(flag(s.Rationing)), 10)
	b = appendF(b, s.PowerFactor(), 4)
	b = appendF(b, s.GeneratorCondition, 4)
	b = append(b, ',')
	b = strconv.AppendInt(b, int64(flag(s.GeneratorDown > 0)), 10)
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
	if r := w.Rebellion; r != nil { // ODD §6.14, Outputs: adults active or beyond, joined or beyond, violent; workers on strike; held zones
		var at [world.Violent + 1]int
		for _, id := range w.Alive {
			at[r.Stage[id]]++
		}
		strikers := 0
		for _, n := range r.Striking {
			strikers += n
		}
		for _, n := range []int{at[world.Active] + at[world.Joined] + at[world.Violent], at[world.Joined] + at[world.Violent], at[world.Violent], strikers} {
			b = append(b, ',')
			b = strconv.AppendInt(b, int64(n), 10)
		}
		for _, held := range r.Held {
			b = append(b, ',')
			b = strconv.AppendInt(b, int64(flag(held)), 10)
		}
	}
	o.row = append(b, '\n')
	_, err := o.daily.Write(o.row)
	return err
}

// writeFrame writes one row per floor and metric, metrics in registration
// order, floors from 1 to 144 (ODD §1.2, Outputs for viewers).
func (o *Writer) writeFrame(w *world.World) error {
	for _, m := range o.metrics {
		clear(o.vals)
		m.Fill(w, o.vals)
		for i, v := range o.vals {
			b := strconv.AppendInt(o.row[:0], int64(w.Day), 10)
			b = append(b, ',')
			b = strconv.AppendInt(b, int64(i+1), 10)
			b = append(b, ',')
			b = append(b, m.Name...)
			b = appendTrimmed(append(b, ','), v)
			o.row = append(b, '\n')
			if _, err := o.frames.Write(o.row); err != nil {
				return err
			}
		}
	}
	return nil
}

// baselineMetrics are the frames every run has: people, households and
// average health per floor (0 on a floor where nobody lives).
var baselineMetrics = []FrameMetric{
	{"population", func(w *world.World, v []float64) {
		for _, h := range w.Households {
			if h.Floor != world.None {
				v[h.Floor-1] += float64(len(h.Members))
			}
		}
	}},
	{"households", func(w *world.World, v []float64) {
		for _, h := range w.Households {
			if h.Floor != world.None {
				v[h.Floor-1]++
			}
		}
	}},
	{"health", func(w *world.World, v []float64) {
		people := make([]int, len(v))
		for _, h := range w.Households {
			if h.Floor == world.None {
				continue
			}
			for _, id := range h.Members {
				v[h.Floor-1] += w.People[id].Health
			}
			people[h.Floor-1] += len(h.Members)
		}
		for i, n := range people {
			if n > 0 {
				v[i] /= float64(n)
			}
		}
	}},
}

// ODD §8: yearly.csv: the median age in years, the age pyramid (women
// and men per 5-year band), workers and target per department, the share
// of Mechanical workers living in Mechanical's zone (Down Deep; the §8
// "persistent zones" target), people who moved to another zone since the
// previous yearly row, and the Mayor. The snapshot (§3 step 15)
// runs before governance (step 16), so a row shows the Mayor before that
// day's change; a Mayor who died since the last governance step is still
// shown, with no zone.
func (o *Writer) logYearly(w *world.World, _ *rng.Stream) error {
	var pyramid [bands][2]int
	o.ages = o.ages[:0]
	mech, mechInZone := 0, 0
	for _, id := range w.Alive {
		p := &w.People[id]
		age := int(w.Day - p.BirthDay)
		o.ages = append(o.ages, age)
		pyramid[min(age/engine.DaysPerYear/bandYears, bands-1)][p.Sex]++
		if p.Stage == world.Worker && w.Depts[p.Department].Name == "mechanical" {
			mech++
			if zoneOf(w, id) == w.Depts[p.Department].Zone {
				mechInZone++
			}
		}
	}
	b := strconv.AppendInt(o.row[:0], int64(w.Day), 10)
	b = appendF(b, median(o.ages)/engine.DaysPerYear, 2)
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
	share := 0.0
	if mech > 0 {
		share = float64(mechInZone) / float64(mech)
	}
	b = appendF(b, share, 4)
	b = append(b, ',')
	b = strconv.AppendInt(b, int64(w.Counts.ZoneMoves-o.zoneMoves), 10)
	o.zoneMoves = w.Counts.ZoneMoves
	b = append(b, ',')
	b = strconv.AppendInt(b, int64(w.Silo.Mayor), 10)
	b = append(b, ',')
	if m := w.Silo.Mayor; m != world.None && w.People[m].Alive {
		b = append(b, zoneOf(w, m).String()...)
	}
	if w.Network != nil {
		b = o.appendTies(w, b)
	}
	if w.Beliefs != nil {
		b = o.appendBeliefs(w, b)
	}
	if w.Control != nil {
		b = o.appendControl(w, b)
	}
	if w.Rebellion != nil {
		b = o.appendRebellion(w, b)
	}
	o.row = append(b, '\n')
	_, err := o.yearly.Write(o.row)
	return err
}

// appendTies writes the network columns (ODD §6.11, Outputs): the mean
// number of ties per living person and its 10th, 50th and 90th
// percentiles, by nearest rank (the smallest value with at least p of
// people at or below it), then expected conversations a day by tie kind.
func (o *Writer) appendTies(w *world.World, b []byte) []byte {
	o.degrees = o.degrees[:0]
	total := 0
	for _, id := range w.Alive {
		o.ties = w.Network.Ties(w, id, o.ties[:0])
		o.degrees = append(o.degrees, len(o.ties))
		total += len(o.ties)
	}
	mean := 0.0
	if n := len(o.degrees); n > 0 {
		mean = float64(total) / float64(n)
	}
	b = appendF(b, mean, 4)
	slices.Sort(o.degrees)
	for _, p := range []float64{0.1, 0.5, 0.9} {
		v := 0
		if n := len(o.degrees); n > 0 {
			v = o.degrees[int(math.Ceil(p*float64(n)))-1]
		}
		b = append(b, ',')
		b = strconv.AppendInt(b, int64(v), 10)
	}
	// Expected conversations today, by tie kind: each tie between two
	// people who can reach each other, counted once, with its kind's daily
	// chance. Conversations are drawn only when a module asks (ODD §6.11),
	// so their number is logged as its expectation.
	var talks [world.NumTieKinds]float64
	n := w.Network
	for _, a := range w.Alive {
		for _, t := range n.TiesOf(w, a) {
			if a < t.Other && n.Reachable(w, a, t.Other) {
				talks[t.Kind] += n.Chance[t.Kind]
			}
		}
	}
	for _, x := range talks {
		b = appendF(b, x, 1)
	}
	return b
}

// appendBeliefs writes the beliefs columns (ODD §6.12, Outputs): the mean
// L of adults, overall and by the zone of their home, then the share of
// adults who know each flag (0 with no adults), the share with at least
// one flag hidden, the people dosed today, then relics hidden and held,
// and Flamekeepers.
func (o *Writer) appendBeliefs(w *world.World, b []byte) []byte {
	var sum [world.NumZones + 1]float64 // the last one is everyone
	var n [world.NumZones + 1]int
	var know [world.NumFlags]float64
	hidden, dosed := 0.0, 0
	for _, id := range w.Alive {
		if w.Beliefs.Dosed(w, id) {
			dosed++
		}
		if w.Day-w.People[id].BirthDay < o.adultAge {
			continue
		}
		if w.Beliefs.Hidden[id] != 0 {
			hidden++
		}
		l, z := w.Beliefs.L(id), zoneOf(w, id)
		sum[z] += l
		n[z]++
		sum[world.NumZones] += l
		n[world.NumZones]++
		for f := range world.NumFlags {
			if w.Beliefs.Known[id].Has(f) {
				know[f]++
			}
		}
	}
	mean := func(x float64, n int) float64 {
		if n == 0 {
			return 0
		}
		return x / float64(n)
	}
	for _, z := range [...]int{world.NumZones, int(world.UpTop), int(world.Mids), int(world.DownDeep)} {
		b = appendF(b, mean(sum[z], n[z]), 4)
	}
	for _, k := range know {
		b = appendF(b, mean(k, n[world.NumZones]), 4)
	}
	b = appendF(b, mean(hidden, n[world.NumZones]), 4)
	b = append(b, ',')
	b = strconv.AppendInt(b, int64(dosed), 10)
	held := 0
	for _, r := range w.Beliefs.Relics {
		if r.Holder != world.None {
			held++
		}
	}
	b = append(b, ',')
	b = strconv.AppendInt(b, int64(len(w.Beliefs.Relics)-held), 10)
	b = append(b, ',')
	b = strconv.AppendInt(b, int64(held), 10)
	b = append(b, ',')
	return strconv.AppendInt(b, int64(len(w.Beliefs.Members)), 10)
}

// appendControl writes the control columns (ODD §6.13, Outputs): over
// the year, acts seen, arrests, trials by the penalty served (warning,
// prison, mines, cleaning), failed cleanings, relics confiscated, raids,
// denied lottery wins and children born to flagged parents; now, people held by Judicial, flagged people alive, open cases
// not yet arrested, the lift S and the fear F.
func (o *Writer) appendControl(w *world.World, b []byte) []byte {
	c := w.Control
	t, p := c.Tally, o.tally
	for _, n := range []int{t.Seen - p.Seen, t.Arrests - p.Arrests, t.Warnings - p.Warnings, t.Prisons - p.Prisons,
		t.Mines - p.Mines, t.Cleanings - p.Cleanings, t.Failed - p.Failed, t.Taken - p.Taken, t.Raids - p.Raids,
		t.Denied - p.Denied, t.FlaggedBirths - p.FlaggedBirths} {
		b = append(b, ',')
		b = strconv.AppendInt(b, int64(n), 10)
	}
	o.tally = t
	held, flagged := 0, 0
	for _, id := range w.Alive {
		if c.Held(id) {
			held++
		}
		if c.Flagged[id] {
			flagged++
		}
	}
	for _, n := range []int{held, flagged, len(c.Cases)} {
		b = append(b, ',')
		b = strconv.AppendInt(b, int64(n), 10)
	}
	b = appendF(b, w.Beliefs.Lift, 4)
	return appendF(b, c.Fear, 4)
}

// appendRebellion writes the rebellion columns (ODD §6.14, Outputs): the
// mean hardship H and grievance G of adults, overall and by home zone; over
// the year, unrest spells ended, rebellions started, and rebels killed and
// arrested and officers killed and wounded in fights.
func (o *Writer) appendRebellion(w *world.World, b []byte) []byte {
	var h, g [world.NumZones + 1]float64 // the last one is everyone
	var n [world.NumZones + 1]int
	for _, id := range w.Alive {
		if w.Day-w.People[id].BirthDay < o.adultAge {
			continue
		}
		for _, z := range [...]int{int(zoneOf(w, id)), world.NumZones} {
			h[z] += w.Rebellion.H[id]
			g[z] += w.Rebellion.G[id]
			n[z]++
		}
	}
	for _, x := range [...]*[world.NumZones + 1]float64{&h, &g} {
		for _, z := range [...]int{world.NumZones, int(world.UpTop), int(world.Mids), int(world.DownDeep)} {
			mean := 0.0
			if n[z] > 0 {
				mean = x[z] / float64(n[z])
			}
			b = appendF(b, mean, 4)
		}
	}
	r := w.Rebellion
	t, p := r.Tally, o.fights
	for _, n := range []int{r.UnrestSpells - o.unrestSpells, r.Rebellions - o.rebellions, t.RebelsKilled - p.RebelsKilled,
		t.RebelsArrested - p.RebelsArrested, t.OfficersKilled - p.OfficersKilled, t.OfficersWounded - p.OfficersWounded} {
		b = append(b, ',')
		b = strconv.AppendInt(b, int64(n), 10)
	}
	o.unrestSpells, o.rebellions, o.fights = r.UnrestSpells, r.Rebellions, t
	return b
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
	if o.frames != nil {
		for _, step := range []func() error{o.frames.Flush, o.gz.Close, o.framesFile.Close} {
			if serr := step(); err == nil {
				err = serr
			}
		}
	}
	return err
}

// median of ages in days (mean of the middle two for an even count); it
// sorts ages. 0 for nobody.
func median(ages []int) float64 {
	if len(ages) == 0 {
		return 0
	}
	slices.Sort(ages)
	n := len(ages)
	if n%2 == 1 {
		return float64(ages[n/2])
	}
	return float64(ages[n/2-1]+ages[n/2]) / 2
}

func flag(on bool) int {
	if on {
		return 1
	}
	return 0
}

func appendF(b []byte, x float64, prec int) []byte {
	return strconv.AppendFloat(append(b, ','), x, 'f', prec, 64)
}

// appendTrimmed writes x with 4 decimals, like the CSV files, minus
// trailing zeros: counts come out as "73", not "73.0000", which keeps a
// 250-year frames file small.
func appendTrimmed(b []byte, x float64) []byte {
	n := len(b)
	b = strconv.AppendFloat(b, x, 'f', 4, 64)
	for len(b) > n && b[len(b)-1] == '0' {
		b = b[:len(b)-1]
	}
	if b[len(b)-1] == '.' {
		b = b[:len(b)-1]
	}
	return b
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
