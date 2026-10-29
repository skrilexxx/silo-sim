package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/skrilexxx/silo-sim/internal/config"
)

func batch(args []string) error {
	fs := flag.NewFlagSet("batch", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "scenario YAML file")
	seedList := fs.String("seeds", "", "seeds to run: N or FIRST-LAST, e.g. 1-100")
	workers := fs.Int("workers", runtime.NumCPU(), "runs at the same time")
	out := fs.String("out", "", "folder for the batch; each seed gets <name>-<seed>/ inside")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *scenario == "" || *seedList == "" || *out == "" {
		return errors.New("--scenario, --seeds and --out are required")
	}
	if *workers < 1 {
		return fmt.Errorf("--workers = %d, must be at least 1", *workers)
	}
	seeds, err := parseSeeds(*seedList)
	if err != nil {
		return err
	}
	cfg, err := config.Load(*scenario)
	if err != nil {
		return err
	}
	cfg.Output.Dir = *out
	if err := runBatch(cfg, seeds, *workers, os.Stderr); err != nil {
		return err
	}
	fmt.Println("wrote", len(seeds), "runs to", *out)
	return nil
}

// parseSeeds reads "N" or "FIRST-LAST" (both included).
func parseSeeds(s string) ([]uint64, error) {
	lo, hi, isRange := strings.Cut(s, "-")
	first, err1 := strconv.ParseUint(lo, 10, 64)
	last, err2 := first, error(nil)
	if isRange {
		last, err2 = strconv.ParseUint(hi, 10, 64)
	}
	if err1 != nil || err2 != nil || last < first {
		return nil, fmt.Errorf("--seeds = %q, want N or FIRST-LAST with FIRST ≤ LAST", s)
	}
	var seeds []uint64
	for seed := first; seed <= last; seed++ {
		seeds = append(seeds, seed)
	}
	return seeds, nil
}

// runBatch runs one simulation per seed, up to workers at a time. Each run
// is a whole simulation in its own goroutine, with its own world and
// random streams, so a batch run of seed 7 writes the same files as `run
// --seed 7` (CLAUDE.md, Determinism). Runs only read the shared config;
// each gets a copy with its own seed. A failed seed doesn't stop the
// others; all failures are returned together.
func runBatch(cfg *config.Scenario, seeds []uint64, workers int, progress io.Writer) error {
	jobs := make(chan uint64)
	type result struct {
		seed uint64
		err  error
	}
	results := make(chan result)
	var wg sync.WaitGroup
	for range min(workers, len(seeds)) {
		wg.Go(func() {
			for seed := range jobs {
				c := *cfg
				c.Seed = seed
				_, err := simulate(&c)
				results <- result{seed, err}
			}
		})
	}
	go func() {
		for _, s := range seeds {
			jobs <- s
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	var errs []error
	done := 0
	for r := range results {
		done++
		status := "ok"
		if r.err != nil {
			status = "FAILED: " + r.err.Error()
			errs = append(errs, fmt.Errorf("seed %d: %w", r.seed, r.err))
		}
		fmt.Fprintf(progress, "[%d/%d] seed %d %s\n", done, len(seeds), r.seed, status)
	}
	return errors.Join(errs...)
}
