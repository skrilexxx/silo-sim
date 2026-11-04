// Package serve is the results viewer's server (docs/viewer.md, M16): the
// embedded page and a read-only JSON and file API over the run folders
// under one root. It never imports the model and never writes: a run
// viewed is the same as a run not viewed (the viewer rule, T7.2).
package serve

import (
	"cmp"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/skrilexxx/silo-sim/internal/report"
)

//go:embed static
var static embed.FS

// files are the run files the viewer may read (docs/viewer.md, Data contract).
var files = map[string]string{
	"run.json":         "application/json",
	"daily.csv":        "text/csv",
	"yearly.csv":       "text/csv",
	"events.jsonl":     "application/x-ndjson",
	"rebellions.jsonl": "application/x-ndjson",
	"frames.csv.gz":    "text/csv", // sent gzip-encoded, unpacked by the browser
}

// Handler serves the page and the API over the run folders under root.
func Handler(root string) (http.Handler, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	page, err := fs.Sub(static, "static")
	if err != nil {
		return nil, err
	}
	s := &server{root: root}
	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServerFS(page))
	mux.HandleFunc("GET /api/runs", s.runs)
	mux.HandleFunc("GET /api/file", s.file)
	mux.HandleFunc("GET /api/outcomes", s.outcomes)
	return mux, nil
}

type server struct{ root string }

// Run is one run folder in the listing.
type Run struct {
	Path     string  `json:"path"`  // relative to the root, with slashes
	Batch    string  `json:"batch"` // the batch folder it is in, or ""
	Scenario string  `json:"scenario"`
	Seed     uint64  `json:"seed"`
	Finished bool    `json:"finished"`
	Years    int     `json:"years"` // measured years asked for
	Collapse *string `json:"collapse"`
	Frames   bool    `json:"frames"`
	Modified int64   `json:"modified"` // run.json's modification time, Unix seconds
}

// Batch is a folder holding run folders.
type Batch struct {
	Path      string `json:"path"`
	Scenario  string `json:"scenario"`
	Runs      int    `json:"runs"`
	Collapses int    `json:"collapses"`
	Modified  int64  `json:"modified"`
}

// runs lists every run folder (one with a run.json) under the root, and
// every batch (a folder whose own subfolders hold runs), newest first.
func (s *server) runs(w http.ResponseWriter, _ *http.Request) {
	var runs []Run
	err := filepath.WalkDir(s.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "run.json" {
			return err
		}
		r, err := readRun(path)
		if err != nil {
			return nil // a run still being written, or broken: not listed
		}
		dir := filepath.Dir(path)
		rel, _ := filepath.Rel(s.root, dir)
		r.Path = filepath.ToSlash(rel)
		if parent := filepath.Dir(rel); parent != "." {
			r.Batch = filepath.ToSlash(parent)
		}
		_, ferr := os.Stat(filepath.Join(dir, "frames.csv.gz"))
		r.Frames = ferr == nil
		runs = append(runs, r)
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var batches []Batch
	for _, r := range runs {
		if r.Batch == "" {
			continue
		}
		i := slices.IndexFunc(batches, func(b Batch) bool { return b.Path == r.Batch })
		if i < 0 {
			batches = append(batches, Batch{Path: r.Batch, Scenario: r.Scenario})
			i = len(batches) - 1
		}
		b := &batches[i]
		b.Runs++
		b.Modified = max(b.Modified, r.Modified)
		if r.Collapse != nil {
			b.Collapses++
		}
	}
	slices.SortFunc(runs, func(a, b Run) int { return cmp.Or(cmp.Compare(b.Modified, a.Modified), cmp.Compare(a.Path, b.Path)) })
	slices.SortFunc(batches, func(a, b Batch) int { return cmp.Or(cmp.Compare(b.Modified, a.Modified), cmp.Compare(a.Path, b.Path)) })
	if batches == nil { // JSON [] rather than null
		batches = []Batch{}
	}
	if runs == nil {
		runs = []Run{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Batches []Batch `json:"batches"`
		Runs    []Run   `json:"runs"`
	}{batches, runs})
}

// readRun reads what the listing shows from a run.json.
func readRun(path string) (Run, error) {
	var info struct {
		Seed     uint64 `json:"seed"`
		Finished bool   `json:"finished"`
		Outcomes struct {
			CollapseCause *string `json:"collapseCause"`
		} `json:"outcomes"`
		Scenario struct {
			Name  string `json:"name"`
			Years int    `json:"years"`
		} `json:"scenario"`
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Run{}, err
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return Run{}, err
	}
	st, err := os.Stat(path)
	if err != nil {
		return Run{}, err
	}
	return Run{Scenario: info.Scenario.Name, Seed: info.Seed, Finished: info.Finished, Years: info.Scenario.Years,
		Collapse: info.Outcomes.CollapseCause, Modified: st.ModTime().Unix()}, nil
}

// file sends one of a run's files. The run path must stay inside the root
// and name a run folder; the name must be one of the viewer's files.
func (s *server) file(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	kind, ok := files[name]
	dir, inside := s.within(r.URL.Query().Get("run"))
	if !ok || !inside {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if _, err := os.Stat(filepath.Join(dir, "run.json")); err != nil {
		http.Error(w, "not a run", http.StatusNotFound)
		return
	}
	f, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		http.Error(w, "no such file", http.StatusNotFound)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", kind)
	if strings.HasSuffix(name, ".gz") {
		w.Header().Set("Content-Encoding", "gzip")
	}
	_, _ = io.Copy(w, f)
}

// outcomes sends a batch's outcome metrics, the numbers silosim report
// --outcomes prints (docs/viewer.md, T16.4).
func (s *server) outcomes(w http.ResponseWriter, r *http.Request) {
	dir, ok := s.within(r.URL.Query().Get("batch"))
	if !ok {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	seeds, err := report.Load(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	o, err := report.OutcomesOf(seeds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(o)
}

// within resolves a path relative to the root, refusing an absolute one,
// one with "..", or one that resolves outside the root.
func (s *server) within(rel string) (string, bool) {
	if rel == "" || filepath.IsAbs(rel) || slices.Contains(strings.Split(filepath.ToSlash(rel), "/"), "..") {
		return "", false
	}
	dir := filepath.Join(s.root, filepath.FromSlash(rel))
	back, err := filepath.Rel(s.root, dir)
	if err != nil || back == ".." || strings.HasPrefix(back, ".."+string(filepath.Separator)) {
		return "", false
	}
	return dir, true
}
