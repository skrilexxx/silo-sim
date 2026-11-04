// Command silosim runs the Silo 18 simulation.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/report"
)

const usage = `usage:
  silosim run    --scenario FILE [--seed N] [--dry-run]
  silosim batch  --scenario FILE --seeds FIRST-LAST --out DIR [--workers N]
  silosim report --batch DIR [--outcomes]
  silosim serve  [--runs DIR] [--addr HOST:PORT]
  silosim chronicle --run DIR`

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
	case "report":
		err = reportCmd(os.Args[2:])
	case "serve":
		err = serveCmd(os.Args[2:])
	case "chronicle":
		err = chronicleCmd(os.Args[2:])
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

// reportCmd checks a batch folder against the ODD §8 calibration targets,
// or with --outcomes prints the part-2 outcome metrics (ODD §1.2, T15.1).
func reportCmd(args []string) error {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	dir := fs.String("batch", "", "batch folder written by silosim batch")
	outcomes := fs.Bool("outcomes", false, "print the outcome metrics of ODD §1.2 with 95% intervals instead of the calibration targets")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" {
		return errors.New("--batch is required")
	}
	seeds, err := report.Load(*dir)
	if err != nil {
		return err
	}
	if *outcomes {
		return report.WriteOutcomes(os.Stdout, seeds)
	}
	ok, err := report.Write(os.Stdout, seeds)
	if err != nil {
		return err
	}
	if ok {
		fmt.Println("\nevery pattern holds")
	} else {
		fmt.Println("\nsome patterns fail")
	}
	return nil
}
