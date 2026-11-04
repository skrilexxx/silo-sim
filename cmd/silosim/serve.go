package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/skrilexxx/silo-sim/internal/serve"
)

// serveCmd runs the results viewer (docs/viewer.md): read-only, over the
// run folders under --runs.
func serveCmd(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	runs := fs.String("runs", "runs", "folder holding run and batch folders")
	addr := fs.String("addr", "127.0.0.1:8080", "address to listen on")
	if err := fs.Parse(args); err != nil {
		return err
	}
	h, err := serve.Handler(*runs)
	if err != nil {
		return err
	}
	fmt.Printf("viewer on http://%s/ reading %s\n", *addr, *runs)
	return http.ListenAndServe(*addr, h)
}
