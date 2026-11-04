package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"time"

	"github.com/skrilexxx/silo-sim/internal/narrate"
)

// narrateCmd has Claude write a run's story from its chronicle, into
// story.md in the run folder (docs/narrator.md). The only command that
// uses the network.
func narrateCmd(args []string) error {
	fs := flag.NewFlagSet("narrate", flag.ContinueOnError)
	dir := fs.String("run", "", "run folder written by silosim run or batch")
	model := fs.String("model", narrate.Model, "Claude model")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" {
		return errors.New("--run is required")
	}
	fmt.Printf("narrating %s with %s (one request, streamed; a minute or two)…\n", *dir, *model)
	if err := narrate.Write(context.Background(), *dir, *model, time.Now()); err != nil {
		return err
	}
	fmt.Println("wrote", filepath.Join(*dir, "story.md"))
	return nil
}
