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
	"github.com/skrilexxx/silo-sim/internal/modules/demography"
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
	runErr := run1(cfg, dir, modules)
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

func run1(cfg *config.Scenario, dir string, modules []engine.Module) error {
	r, err := rng.New(cfg.Seed, "initpop")
	if err != nil {
		return err
	}
	w, err := initpop.Build(cfg, r)
	if err != nil {
		return err
	}
	daily, err := output.NewDaily(dir)
	if err != nil {
		return err
	}
	e, err := engine.New(cfg, w, append(modules, daily))
	if err != nil {
		daily.Close()
		return err
	}
	runErr := e.Run()
	if err := daily.Close(); err != nil && runErr == nil {
		runErr = err
	}
	return runErr
}

// newModule builds a model module by its scenario name.
func newModule(name string) (engine.Module, error) {
	switch name {
	case "demography":
		return demography.New(), nil
	case "work":
		return work.New(), nil
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
