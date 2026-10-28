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

// Daily is the module that writes daily.csv at the logging phase. The
// engine skips that phase during warm-up unless output.logWarmup is set.
type Daily struct {
	f   *os.File
	w   *bufio.Writer
	row []byte // reused, so logging allocates nothing per tick
}

// NewDaily creates dir/daily.csv and writes its header.
func NewDaily(dir string) (*Daily, error) {
	f, err := os.Create(filepath.Join(dir, "daily.csv"))
	if err != nil {
		return nil, err
	}
	d := &Daily{f: f, w: bufio.NewWriter(f)}
	if _, err := d.w.WriteString("day,population\n"); err != nil {
		f.Close()
		return nil, err
	}
	return d, nil
}

func (d *Daily) Name() string       { return "output" }
func (d *Daily) Requires() []string { return nil }
func (d *Daily) Init(*world.World, *config.Scenario, *rng.Stream) error {
	return nil
}
func (d *Daily) Steps() []engine.Step {
	return []engine.Step{{Phase: engine.PhaseLogDaily, Run: d.log}}
}

// ODD §8: daily.csv. Only day and population exist so far; the other
// columns arrive with the modules that produce them (T5.1).
func (d *Daily) log(w *world.World, _ *rng.Stream) error {
	d.row = strconv.AppendInt(d.row[:0], int64(w.Day), 10)
	d.row = append(d.row, ',')
	d.row = strconv.AppendInt(d.row, int64(len(w.Alive)), 10)
	d.row = append(d.row, '\n')
	_, err := d.w.Write(d.row)
	return err
}

// Close flushes and closes daily.csv. A failed flush means lost rows, so
// its error must not be ignored.
func (d *Daily) Close() error {
	err := d.w.Flush()
	if cerr := d.f.Close(); err == nil {
		err = cerr
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
