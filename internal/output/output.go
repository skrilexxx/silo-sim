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

// Writer is the module that writes the run's files: daily.csv at the
// logging phase, yearly.csv at the snapshot phase, and events.jsonl as
// modules append to World.Events. The engine skips both phases during
// warm-up unless output.logWarmup is set; warm-up events are dropped too.
type Writer struct {
	files  []*os.File
	daily  *bufio.Writer
	yearly *bufio.Writer
	events *bufio.Writer
	row    []byte // reused, so logging allocates nothing per tick

	w          *world.World
	warmupDays world.Tick // 0 if warm-up events are kept
}

// New creates daily.csv, yearly.csv and events.jsonl in dir and writes
// the CSV headers.
func New(dir string) (*Writer, error) {
	o := &Writer{}
	for _, f := range []struct {
		name   string
		buf    **bufio.Writer
		header string
	}{
		{"daily.csv", &o.daily, "day,population\n"},
		{"yearly.csv", &o.yearly, "day,mayor,mayorZone\n"},
		{"events.jsonl", &o.events, ""},
	} {
		file, err := os.Create(filepath.Join(dir, f.name))
		if err != nil {
			o.Close()
			return nil, err
		}
		o.files = append(o.files, file)
		*f.buf = bufio.NewWriter(file)
		if _, err := (*f.buf).WriteString(f.header); err != nil {
			o.Close()
			return nil, err
		}
	}
	return o, nil
}

func (o *Writer) Name() string       { return "output" }
func (o *Writer) Requires() []string { return nil }
func (o *Writer) Init(w *world.World, cfg *config.Scenario, _ *rng.Stream) error {
	o.w = w
	if !cfg.Output.LogWarmup {
		o.warmupDays = world.Tick(cfg.WarmupYears * engine.DaysPerYear)
	}
	return nil
}
func (o *Writer) Steps() []engine.Step {
	return []engine.Step{
		{Phase: engine.PhaseLogDaily, Run: o.logDaily},
		{Phase: engine.PhaseSnapshot, Run: o.logYearly},
	}
}

// ODD §8: daily.csv. Only day and population exist so far; the other
// columns arrive with T5.1. Events logged since the last row go out too.
func (o *Writer) logDaily(w *world.World, _ *rng.Stream) error {
	o.row = strconv.AppendInt(o.row[:0], int64(w.Day), 10)
	o.row = append(o.row, ',')
	o.row = strconv.AppendInt(o.row, int64(len(w.Alive)), 10)
	o.row = append(o.row, '\n')
	if _, err := o.daily.Write(o.row); err != nil {
		return err
	}
	return o.writeEvents()
}

// ODD §8: yearly.csv. Only the Mayor so far; the age pyramid, department
// table and zone changes arrive with T5.1. The snapshot (§3 step 15) runs
// before governance (step 16), so a row shows the Mayor before that day's
// change, if any.
func (o *Writer) logYearly(w *world.World, _ *rng.Stream) error {
	o.row = strconv.AppendInt(o.row[:0], int64(w.Day), 10)
	o.row = append(o.row, ',')
	o.row = strconv.AppendInt(o.row, int64(w.Silo.Mayor), 10)
	o.row = append(o.row, ',')
	if w.Silo.Mayor != world.None {
		p := &w.People[w.Silo.Mayor]
		o.row = append(o.row, w.Floors[w.Households[p.Household].Floor-1].Zone.String()...)
	}
	o.row = append(o.row, '\n')
	_, err := o.yearly.Write(o.row)
	return err
}

// writeEvents writes and clears World.Events, dropping warm-up events.
// With no warm-up nothing is dropped, day-0 events included.
func (o *Writer) writeEvents() error {
	for _, e := range o.w.Events {
		if o.warmupDays > 0 && e.Day <= o.warmupDays {
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
