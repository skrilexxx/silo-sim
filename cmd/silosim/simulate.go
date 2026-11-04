package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/engine"
	"github.com/skrilexxx/silo-sim/internal/initpop"
	"github.com/skrilexxx/silo-sim/internal/modules/beliefs"
	"github.com/skrilexxx/silo-sim/internal/modules/control"
	"github.com/skrilexxx/silo-sim/internal/modules/demography"
	"github.com/skrilexxx/silo-sim/internal/modules/governance"
	"github.com/skrilexxx/silo-sim/internal/modules/health"
	"github.com/skrilexxx/silo-sim/internal/modules/network"
	"github.com/skrilexxx/silo-sim/internal/modules/rebellion"
	"github.com/skrilexxx/silo-sim/internal/modules/regime"
	"github.com/skrilexxx/silo-sim/internal/modules/resources"
	"github.com/skrilexxx/silo-sim/internal/modules/space"
	"github.com/skrilexxx/silo-sim/internal/modules/work"
	"github.com/skrilexxx/silo-sim/internal/output"
	"github.com/skrilexxx/silo-sim/internal/rng"
)

// simulate runs one scenario and writes its files to
// <output.dir>/<name>-<seed>/, returning that folder.
func simulate(cfg *config.Scenario) (string, error) {
	var modules []engine.Module
	for _, name := range cfg.Modules {
		m, err := newModule(name)
		if err != nil {
			return "", err
		}
		modules = append(modules, m)
	}

	dir := filepath.Join(cfg.Output.Dir, fmt.Sprintf("%s-%d", cfg.Name, cfg.Seed))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	scenario, err := output.ScenarioMap(cfg)
	if err != nil {
		return "", err
	}
	info := output.RunInfo{Scenario: scenario, Seed: cfg.Seed, GitCommit: gitCommit(), GoVersion: runtime.Version()}
	// Written now as well as at the end, so a crashed run still says what it was.
	if err := output.WriteRunJSON(dir, info); err != nil {
		return "", err
	}

	// Wall-clock time is measured here, outside the simulation, and only
	// goes into run.json; it never reaches the model or daily.csv.
	start := time.Now()
	outcomes, runErr := run1(cfg, dir, modules)
	info.Outcomes = outcomes
	info.RuntimeSeconds = time.Since(start).Seconds()
	info.Finished = runErr == nil
	if runErr != nil {
		info.Error = runErr.Error()
	}
	if err := output.WriteRunJSON(dir, info); err != nil && runErr == nil {
		runErr = err
	}
	return dir, runErr
}

// run1 plays the run and returns its outcomes for run.json (nil with the
// rebellion module off).
func run1(cfg *config.Scenario, dir string, modules []engine.Module) (*output.Outcomes, error) {
	r, err := rng.New(cfg.Seed, "initpop")
	if err != nil {
		return nil, err
	}
	w, err := initpop.Build(cfg, r)
	if err != nil {
		return nil, err
	}
	out, err := output.New(dir)
	if err != nil {
		return nil, err
	}
	for _, m := range modules {
		if fs, ok := m.(output.FrameSource); ok {
			out.AddFrameMetrics(fs.FrameMetrics()...)
		}
	}
	e, err := engine.New(cfg, w, append(modules, out))
	if err != nil {
		out.Close()
		return nil, err
	}
	runErr := e.Run()
	if err := out.Close(); err != nil && runErr == nil {
		runErr = err
	}
	return out.Outcomes(), runErr
}

// newModule builds a model module by its scenario name.
func newModule(name string) (engine.Module, error) {
	switch name {
	case "demography":
		return demography.New(), nil
	case "work":
		return work.New(), nil
	case "health":
		return health.New(), nil
	case "resources":
		return resources.New(), nil
	case "governance":
		return governance.New(), nil
	case "space":
		return space.New(), nil
	case "network":
		return network.New(), nil
	case "beliefs":
		return beliefs.New(), nil
	case "control":
		return control.New(), nil
	case "rebellion":
		return rebellion.New(), nil
	case "regime":
		return regime.New(), nil
	}
	return nil, fmt.Errorf("module %q is not implemented yet", name)
}

// gitCommit reads the commit the binary was built from. Binaries built by
// `go build` inside the repo carry it; test binaries do not.
func gitCommit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	rev, dirty := "unknown", false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if dirty {
		rev += "-dirty"
	}
	return rev
}
