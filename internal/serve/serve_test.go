package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/chronicle"
	"github.com/skrilexxx/silo-sim/internal/report"
)

// tree makes a runs folder: a loose run with frames, and a batch of two
// runs, one of which collapsed.
func tree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	put := func(rel, content string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	put("canon-1/run.json", `{"seed":1,"finished":true,"outcomes":{"collapseCause":null},"scenario":{"name":"canon","years":300}}`)
	put("canon-1/daily.csv", "day,population\n1,10000\n")
	put("canon-1/frames.csv.gz", "gzip bytes")
	put("m15/base/exp-base-1/run.json", `{"seed":1,"finished":true,"outcomes":{"collapseCause":"air"},"scenario":{"name":"exp-base","years":150}}`)
	put("m15/base/exp-base-2/run.json", `{"seed":2,"finished":true,"outcomes":{},"scenario":{"name":"exp-base","years":150}}`)
	put("notes.txt", "not a run")
	return root
}

func get(t *testing.T, h http.Handler, url string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", url, nil))
	return rec
}

// The listing finds the loose run and the batch with its two runs and one
// collapse; the run with frames says so.
func TestRuns(t *testing.T) {
	h, err := Handler(tree(t))
	if err != nil {
		t.Fatal(err)
	}
	rec := get(t, h, "/api/runs")
	var got struct {
		Batches []Batch `json:"batches"`
		Runs    []Run   `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err, rec.Body.String())
	}
	if len(got.Batches) != 1 || got.Batches[0].Path != "m15/base" || got.Batches[0].Runs != 2 || got.Batches[0].Collapses != 1 {
		t.Errorf("batches %+v", got.Batches)
	}
	if len(got.Runs) != 3 {
		t.Fatalf("runs %+v", got.Runs)
	}
	for _, r := range got.Runs {
		if r.Path == "canon-1" && (!r.Frames || r.Batch != "" || r.Years != 300) || r.Path == "m15/base/exp-base-1" && (r.Batch != "m15/base" || r.Collapse == nil || *r.Collapse != "air") {
			t.Errorf("run %+v", r)
		}
	}
}

// Files: a run's file is sent with its type, frames gzip-encoded; a path
// outside the root, an absolute one, a name not in the contract, a folder
// that is not a run, or a missing file are refused.
func TestFile(t *testing.T) {
	root := tree(t)
	h, err := Handler(root)
	if err != nil {
		t.Fatal(err)
	}
	rec := get(t, h, "/api/file?run=canon-1&name=daily.csv")
	if rec.Code != 200 || rec.Body.String() != "day,population\n1,10000\n" || rec.Header().Get("Content-Type") != "text/csv" {
		t.Errorf("daily.csv: %d %q %q", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	if err := os.WriteFile(filepath.Join(root, "canon-1", "story.md"), []byte("# A story\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rec := get(t, h, "/api/file?run=canon-1&name=story.md"); rec.Code != 200 || rec.Body.String() != "# A story\n" || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/markdown") {
		t.Errorf("story.md: %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if rec := get(t, h, "/api/file?run=canon-1&name=frames.csv.gz"); rec.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("frames sent without gzip encoding")
	}
	for url, code := range map[string]int{
		"/api/file?run=../" + filepath.Base(root) + "/canon-1&name=daily.csv": 403,
		"/api/file?run=canon-1/..&name=daily.csv":                             403,
		"/api/file?run=" + root + "/canon-1&name=daily.csv":                   403,
		"/api/file?run=canon-1&name=run.go":                                   403,
		"/api/file?run=canon-1&name=../notes.txt":                             403,
		"/api/file?run=m15&name=daily.csv":                                    404,
		"/api/file?run=canon-1&name=events.jsonl":                             404,
	} {
		if rec := get(t, h, url); rec.Code != code {
			t.Errorf("%s: %d, want %d", url, rec.Code, code)
		}
	}
}

// Outcomes: a batch of two one-year runs, one with a rebellion, gives the
// numbers the report package computes for it, as JSON; a path outside the
// root is refused, a folder without runs is not found.
func TestOutcomes(t *testing.T) {
	root := t.TempDir()
	for i, rebellions := range []string{`{"start":100,"end":200,"days":100,"ending":"suppressed"}` + "\n", ""} {
		dir := filepath.Join(root, "b", "run-"+string(rune('1'+i)))
		for name, content := range map[string]string{
			"run.json":         `{"seed":` + string(rune('1'+i)) + `,"finished":true,"outcomes":{},"scenario":{"warmupYears":0}}`,
			"daily.csv":        "day,population,rationing,births,foodStockDays,popUpTop,popMids,popDownDeep,healthUpTop,healthMids,healthDownDeep\n1,3,0,0,100,1,1,1,1,1,1\n365,3,0,0,100,1,1,1,1,1,1\n",
			"yearly.csv":       "day,medianAge,mechanicalInZone,knowHistory\n365,30,1,0\n",
			"events.jsonl":     "",
			"rebellions.jsonl": rebellions,
		} {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	h, err := Handler(root)
	if err != nil {
		t.Fatal(err)
	}
	rec := get(t, h, "/api/outcomes?batch=b")
	seeds, err := report.Load(filepath.Join(root, "b"))
	if err != nil {
		t.Fatal(err)
	}
	o, err := report.OutcomesOf(seeds)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := json.Marshal(o)
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != string(want) {
		t.Errorf("outcomes: %d %s, want %s", rec.Code, rec.Body.String(), want)
	}
	if o.Runs != 2 || o.Rebellions != 1 || o.Endings[0].Count != 1 || len(o.Curves) != 2 {
		t.Errorf("outcomes %+v", o)
	}
	for url, code := range map[string]int{"/api/outcomes?batch=../b": 403, "/api/outcomes?batch=b/run-1": 404} {
		if rec := get(t, h, url); rec.Code != code {
			t.Errorf("%s: %d, want %d", url, rec.Code, code)
		}
	}
}

// Chronicle: a run's chronicle is the bytes chronicle.Write gives; a path
// outside the root is refused, a folder that is not a run is not found.
func TestChronicle(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "r")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"run.json":     `{"seed":3,"scenario":{"name":"tiny","warmupYears":0,"years":1,"modules":["demography"]}}`,
		"events.jsonl": `{"day":100,"type":"failure","outageDays":2}` + "\n",
		"daily.csv":    "day,population\n1,10\n365,11\n",
		"yearly.csv":   "day,medianAge\n365,30\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	h, err := Handler(root)
	if err != nil {
		t.Fatal(err)
	}
	var want strings.Builder
	if err := chronicle.Write(&want, dir); err != nil {
		t.Fatal(err)
	}
	rec := get(t, h, "/api/chronicle?run=r")
	if rec.Code != 200 || rec.Body.String() != want.String() || !strings.Contains(rec.Body.String(), "The generator failed: 2 days") {
		t.Errorf("chronicle: %d\n%s\nwant\n%s", rec.Code, rec.Body.String(), want.String())
	}
	for url, code := range map[string]int{"/api/chronicle?run=../r": 403, "/api/chronicle?run=.": 404} {
		if rec := get(t, h, url); rec.Code != code {
			t.Errorf("%s: %d, want %d", url, rec.Code, code)
		}
	}
}

// The page is served at the root.
func TestPage(t *testing.T) {
	h, err := Handler(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rec := get(t, h, "/")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Silo viewer") {
		t.Errorf("page: %d", rec.Code)
	}
	if rec := get(t, h, "/api/runs"); !strings.Contains(rec.Body.String(), `"batches":[]`) {
		t.Errorf("empty listing: %s", rec.Body.String())
	}
}
