package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strconv"
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
	if cols := strings.Split(lines[0], ","); len(cols) != 25 || cols[0] != "day" || cols[24] != "mechanicalCoverage" {
		t.Errorf("header = %q", lines[0])
	}
	// Each row's births − deaths is the change in population since the
	// row before, and the first row does not carry the warm-up's births.
	prevPop := -1
	for i, line := range lines[1:] {
		cols := strings.Split(line, ",")
		if len(cols) != 25 {
			t.Fatalf("row %d has %d columns, want 25: %q", i+1, len(cols), line)
		}
		pop, births, deaths := atoi(t, cols[1]), atoi(t, cols[5]), atoi(t, cols[6])
		if i == 0 && births > 5 {
			t.Errorf("first measured row has %d births: warm-up leaked in", births)
		}
		if prevPop >= 0 && pop-prevPop != births-deaths {
			t.Fatalf("day %s: population %d → %d, but %d births and %d deaths", cols[0], prevPop, pop, births, deaths)
		}
		prevPop = pop
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
	if !strings.HasPrefix(yearly[0], "day,f0_4,m0_4,") || !strings.HasSuffix(yearly[0], ",zoneMoves,mayor,mayorZone") ||
		len(yearly)-1 != 10 || !strings.HasPrefix(yearly[1], "730,") {
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

// ODD §8: same seed → byte-identical files; different seeds → different
// files.
func TestSeedsAndIdenticalFiles(t *testing.T) {
	a, b := mustSimulate(t, loadInto(t, goldenScenario)), mustSimulate(t, loadInto(t, goldenScenario))
	other := loadInto(t, goldenScenario)
	other.Seed = 2
	c := mustSimulate(t, other)
	for _, f := range []string{"daily.csv", "yearly.csv", "events.jsonl"} {
		fa, fb, fc := readFile(t, filepath.Join(a, f)), readFile(t, filepath.Join(b, f)), readFile(t, filepath.Join(c, f))
		if !bytes.Equal(fa, fb) {
			t.Errorf("two runs with the same seed wrote different %s", f)
		}
		if bytes.Equal(fa, fc) {
			t.Errorf("seeds 1 and 2 wrote the same %s", f)
		}
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

func atoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	if err != nil {
		t.Fatal(err)
	}
	return n
}
