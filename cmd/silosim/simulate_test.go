package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/config"
	"github.com/skrilexxx/silo-sim/internal/output"
)

// The golden test lives only in this package, because `go test ./... -update`
// would fail in every package that does not define the flag. Use
// `make update-golden`.
var update = flag.Bool("update", false, "rewrite testdata/golden/daily.sha256")

const (
	goldenScenario = "../../testdata/golden/scenario.yaml"
	goldenHash     = "../../testdata/golden/daily.sha256"
)

// loadInto loads a scenario and points its output at a fresh temp folder.
func loadInto(t *testing.T, path string) *config.Scenario {
	t.Helper()
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Output.Dir = t.TempDir()
	return cfg
}

func mustSimulate(t *testing.T, cfg *config.Scenario) string {
	t.Helper()
	dir, err := simulate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestTenYearEmptyRunWritesFiles(t *testing.T) {
	cfg := loadInto(t, goldenScenario)
	cfg.WarmupYears, cfg.Years = 1, 10
	dir := mustSimulate(t, cfg)

	lines := strings.Split(strings.TrimSuffix(string(readFile(t, filepath.Join(dir, "daily.csv"))), "\n"), "\n")
	if lines[0] != "day,population" {
		t.Errorf("header = %q", lines[0])
	}
	// One row per measured day; the warm-up year is not logged.
	if rows := len(lines) - 1; rows != 10*365 {
		t.Errorf("daily.csv has %d rows, want %d", rows, 10*365)
	}
	if !strings.HasPrefix(lines[1], "366,") || !strings.HasPrefix(lines[len(lines)-1], "4015,") {
		t.Errorf("rows run from %q to %q, want days 366 to 4015", lines[1], lines[len(lines)-1])
	}

	// One yearly row per measured year; with 5-year terms the Mayor is
	// chosen again on days 1825 and 3650, after the warm-up year.
	yearly := strings.Split(strings.TrimSuffix(string(readFile(t, filepath.Join(dir, "yearly.csv"))), "\n"), "\n")
	if yearly[0] != "day,mayor,mayorZone" || len(yearly)-1 != 10 || !strings.HasPrefix(yearly[1], "730,") {
		t.Errorf("yearly.csv: header %q, %d rows, first %q; want 10 rows from day 730", yearly[0], len(yearly)-1, yearly[1])
	}
	var days []int
	for _, line := range strings.Split(strings.TrimSpace(string(readFile(t, filepath.Join(dir, "events.jsonl")))), "\n") {
		var e struct {
			Day    int    `json:"day"`
			Type   string `json:"type"`
			Reason string `json:"reason"`
		}
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("events.jsonl line %q: %v", line, err)
		}
		if e.Type == "mayor" && e.Reason == "term" {
			days = append(days, e.Day)
		}
	}
	if len(days) != 2 || days[0] != 1825 || days[1] != 3650 {
		t.Errorf("term-end Mayor events on days %v, want [1825 3650]", days)
	}

	var info output.RunInfo
	if err := json.Unmarshal(readFile(t, filepath.Join(dir, "run.json")), &info); err != nil {
		t.Fatal(err)
	}
	if !info.Finished || info.Seed != 1 || info.Scenario["name"] != "golden" || info.GoVersion == "" {
		t.Errorf("run.json incomplete: %+v", info)
	}
	if _, ok := info.Scenario["extends"]; ok {
		t.Error("run.json scenario still has extends; it should be the resolved scenario")
	}
}

func TestSameSeedGivesIdenticalDaily(t *testing.T) {
	a := readFile(t, filepath.Join(mustSimulate(t, loadInto(t, goldenScenario)), "daily.csv"))
	b := readFile(t, filepath.Join(mustSimulate(t, loadInto(t, goldenScenario)), "daily.csv"))
	if !bytes.Equal(a, b) {
		t.Error("two runs with the same seed wrote different daily.csv")
	}
}

func TestUnbuiltModuleFails(t *testing.T) {
	cfg := loadInto(t, "../../scenarios/baseline.yaml")
	cfg.Modules = append(cfg.Modules, "dissent")
	_, err := simulate(cfg)
	if err == nil || !strings.Contains(err.Error(), `module "dissent" is not implemented yet`) {
		t.Errorf("got %v", err)
	}
}

func TestGolden(t *testing.T) {
	dir := mustSimulate(t, loadInto(t, goldenScenario))
	sum := sha256.Sum256(readFile(t, filepath.Join(dir, "daily.csv")))
	got := hex.EncodeToString(sum[:])

	if *update {
		if err := os.WriteFile(goldenHash, []byte(got+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("golden hash updated to %s", got)
		return
	}
	want := strings.TrimSpace(string(readFile(t, goldenHash)))
	if got != want {
		t.Errorf("golden daily.csv changed:\n got  %s\n want %s\n"+
			"If the change is on purpose, run `make update-golden` and give the reason in the commit message.", got, want)
	}
}
