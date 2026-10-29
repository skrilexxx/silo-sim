// Command silosim runs the Silo 18 simulation.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/skrilexxx/silo-sim/internal/config"
)

const usage = `usage:
  silosim run   --scenario FILE [--seed N] [--dry-run]
  silosim batch --scenario FILE --seeds FIRST-LAST --out DIR [--workers N]`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "run":
		err = run(os.Args[2:])
	case "batch":
		err = batch(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q\n%s", os.Args[1], usage)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "silosim:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	scenario := fs.String("scenario", "", "scenario YAML file")
	seed := fs.Uint64("seed", 0, "override the scenario seed")
	dryRun := fs.Bool("dry-run", false, "print the resolved config and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *scenario == "" {
		return errors.New("--scenario is required")
	}
	cfg, err := config.Load(*scenario)
	if err != nil {
		return err
	}
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "seed" {
			cfg.Seed = *seed
		}
	})

	if *dryRun {
		out, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(out)
		return err
	}
	dir, err := simulate(cfg)
	if err != nil {
		return err
	}
	fmt.Println("wrote", dir)
	return nil
}
