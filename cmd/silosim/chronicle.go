package main

import (
	"errors"
	"flag"
	"os"

	"github.com/skrilexxx/silo-sim/internal/chronicle"
)

// chronicleCmd prints a run's chronicle (docs/narrator.md): the dated
// facts of its files, as Markdown.
func chronicleCmd(args []string) error {
	fs := flag.NewFlagSet("chronicle", flag.ContinueOnError)
	dir := fs.String("run", "", "run folder written by silosim run or batch")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" {
		return errors.New("--run is required")
	}
	return chronicle.Write(os.Stdout, *dir)
}
