package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
