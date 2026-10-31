package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
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
var update = flag.Bool("update", false, "rewrite the hashes in testdata/golden")

const (
	goldenDir      = "../../testdata/golden"
	goldenScenario = goldenDir + "/scenario.yaml"
)

// frozenFiles are the outputs that must stay byte-identical to baseline-v1
// (docs/architecture.md, Goals). run.json is not one: it holds the runtime.
var frozenFiles = []string{"daily.csv", "yearly.csv", "events.jsonl"}

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
	if cols := strings.Split(lines[0], ","); len(cols) != 26 || cols[0] != "day" || cols[25] != "mechanicalCoverage" {
		t.Errorf("header = %q", lines[0])
	}
	// Each row's births − deaths is the change in population since the
	// row before, and the first row does not carry the warm-up's births.
	prevPop := -1
	for i, line := range lines[1:] {
		cols := strings.Split(line, ",")
		if len(cols) != 26 {
			t.Fatalf("row %d has %d columns, want 26: %q", i+1, len(cols), line)
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
	if !strings.HasPrefix(yearly[0], "day,medianAge,f0_4,m0_4,") || !strings.HasSuffix(yearly[0], ",mechanicalInZone,zoneMoves,mayor,mayorZone") ||
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

// Each frozen file's SHA-256 is stored as <name>.sha256 (daily.sha256 for
// daily.csv). yearly and events were added in T7.5, checked equal to
// baseline-v1.
func TestGolden(t *testing.T) {
	dir := mustSimulate(t, loadInto(t, goldenScenario))
	for _, f := range frozenFiles {
		sum := sha256.Sum256(readFile(t, filepath.Join(dir, f)))
		got := hex.EncodeToString(sum[:])
		hashFile := filepath.Join(goldenDir, strings.TrimSuffix(f, filepath.Ext(f))+".sha256")

		if *update {
			if err := os.WriteFile(hashFile, []byte(got+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Logf("golden %s hash updated to %s", f, got)
			continue
		}
		want := strings.TrimSpace(string(readFile(t, hashFile)))
		if got != want {
			t.Errorf("golden %s changed:\n got  %s\n want %s\n"+
				"If the change is on purpose, run `make update-golden` and give the reason in the commit message.", f, got, want)
		}
	}
}

// runGoldenSize runs a scenario at the golden test's size, seed and
// invariant interval (every tick), with extra modules switched on.
func runGoldenSize(t *testing.T, path string, extra ...string) string {
	t.Helper()
	golden := loadInto(t, goldenScenario)
	cfg := loadInto(t, path)
	cfg.Seed, cfg.WarmupYears, cfg.Years, cfg.Population = golden.Seed, golden.WarmupYears, golden.Years, golden.Population
	cfg.Output.InvariantEvery = golden.Output.InvariantEvery
	cfg.Modules = append(cfg.Modules, extra...)
	return mustSimulate(t, cfg)
}

func sameFrozenFiles(t *testing.T, a, b, why string) {
	t.Helper()
	for _, f := range frozenFiles {
		if !bytes.Equal(readFile(t, filepath.Join(a, f)), readFile(t, filepath.Join(b, f))) {
			t.Errorf("different %s: %s", f, why)
		}
	}
}

// T7.5: with every part-2 module off, part2-off.yaml writes the same files
// as baseline.yaml. Both run at the golden size, so this is as fast as the
// golden test; TestGolden ties baseline at that size to baseline-v1.
func TestPart2OffEqualsBaseline(t *testing.T) {
	sameFrozenFiles(t, runGoldenSize(t, "../../scenarios/baseline.yaml"), runGoldenSize(t, "../../scenarios/part2-off.yaml"),
		"part2-off.yaml differs from baseline.yaml, so a part-2 setting leaks into the baseline")
}

// ODD §6.10: the space module on, at its null setting (stairs intact, no
// trips), changes nothing in the baseline's files; the 20 years also run
// the space invariant every tick (T8.1).
func TestSpaceNullSettingEqualsBaseline(t *testing.T) {
	sameFrozenFiles(t, runGoldenSize(t, "../../scenarios/baseline.yaml"), runGoldenSize(t, "../../scenarios/baseline.yaml", "space"),
		"the space module at its null setting changed the baseline")
}

func atoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// T7.6: frames.csv.gz only exists when switched on; then, on every frame
// day, population summed over floors equals daily.csv's, and the same seed
// writes the same bytes.
func TestFrames(t *testing.T) {
	if _, err := os.Stat(filepath.Join(mustSimulate(t, loadInto(t, goldenScenario)), "frames.csv.gz")); !os.IsNotExist(err) {
		t.Errorf("frames off, but frames.csv.gz exists (err %v)", err)
	}

	runFrames := func() string {
		cfg := loadInto(t, goldenScenario)
		cfg.Years, cfg.Output.Frames = 3, true
		return mustSimulate(t, cfg)
	}
	dir := runFrames()
	if !bytes.Equal(readFile(t, filepath.Join(dir, "frames.csv.gz")), readFile(t, filepath.Join(runFrames(), "frames.csv.gz"))) {
		t.Error("same seed wrote different frames.csv.gz")
	}

	daily := map[string]string{} // day → population
	for _, line := range strings.Split(strings.TrimSpace(string(readFile(t, filepath.Join(dir, "daily.csv")))), "\n")[1:] {
		cols := strings.SplitN(line, ",", 3)
		daily[cols[0]] = cols[1]
	}
	gz, err := gzip.NewReader(bytes.NewReader(readFile(t, filepath.Join(dir, "frames.csv.gz"))))
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if lines[0] != "day,floor,metric,value" {
		t.Fatalf("header %q", lines[0])
	}
	var days []string
	sums := map[string]int{}
	for _, line := range lines[1:] {
		cols := strings.Split(line, ",")
		if cols[2] != "population" {
			continue
		}
		if len(days) == 0 || days[len(days)-1] != cols[0] {
			days = append(days, cols[0])
		}
		sums[cols[0]] += atoi(t, cols[3])
	}
	// 3 years = 1095 days: frames on days 30, 60, …, 1080; 3 metrics × 144 floors each.
	if len(days) != 36 || days[0] != "30" || days[35] != "1080" || len(lines)-1 != 36*3*144 {
		t.Fatalf("frames on %d days (%v … %v), %d rows; want 36 days from 30 to 1080, %d rows",
			len(days), days[0], days[len(days)-1], len(lines)-1, 36*3*144)
	}
	for _, d := range days {
		if strconv.Itoa(sums[d]) != daily[d] {
			t.Errorf("day %s: floors sum to %d people, daily.csv says %s", d, sums[d], daily[d])
		}
	}
}

// ODD §6.11 / T9.1–T9.2: the network on changes nothing in daily.csv or
// events.jsonl; yearly.csv keeps its baseline columns and adds the degree
// distribution and expected conversations a day by tie kind. The 20 years run the
// network invariant every tick.
func TestNetworkNullSetting(t *testing.T) {
	base := runGoldenSize(t, "../../scenarios/baseline.yaml")
	net := runGoldenSize(t, "../../scenarios/baseline.yaml", "space", "network")
	for _, f := range []string{"daily.csv", "events.jsonl"} {
		if !bytes.Equal(readFile(t, filepath.Join(base, f)), readFile(t, filepath.Join(net, f))) {
			t.Errorf("the network changed %s", f)
		}
	}
	pop := map[string]string{} // day → population, from daily.csv
	for _, line := range strings.Split(strings.TrimSpace(string(readFile(t, filepath.Join(net, "daily.csv")))), "\n")[1:] {
		cols := strings.SplitN(line, ",", 3)
		pop[cols[0]] = cols[1]
	}
	baseRows := strings.Split(strings.TrimSpace(string(readFile(t, filepath.Join(base, "yearly.csv")))), "\n")
	netRows := strings.Split(strings.TrimSpace(string(readFile(t, filepath.Join(net, "yearly.csv")))), "\n")
	if len(baseRows) != len(netRows) || !strings.HasSuffix(netRows[0], ",meanTies,tiesP10,tiesP50,tiesP90,"+
		"talksPartner,talksHousehold,talksKin,talksMasterShadow,talksCoworker,talksNeighbour,talksPorter") {
		t.Fatalf("yearly.csv: %d rows vs %d, header %q", len(netRows), len(baseRows), netRows[0])
	}
	for i := range baseRows {
		rest, ok := strings.CutPrefix(netRows[i], baseRows[i]+",")
		if !ok {
			t.Fatalf("row %d does not start with the baseline row", i)
		}
		if i == 0 {
			continue
		}
		cols := strings.Split(rest, ",")
		mean, _ := strconv.ParseFloat(cols[0], 64)
		p10, p50, p90 := atoi(t, cols[1]), atoi(t, cols[2]), atoi(t, cols[3])
		if mean < 5 || mean > 30 || p10 > p50 || p50 > p90 {
			t.Errorf("row %d: mean ties %v, percentiles %d %d %d", i, mean, p10, p50, p90)
		}
		// T9.2: expected conversations a day, by tie kind. Each one
		// involves two people; at p_s = 0.5 and about 10 ties a head (a
		// third strong) that is roughly 2–5 a person a day.
		talks := 0.0
		for _, c := range cols[4:] {
			x, err := strconv.ParseFloat(c, 64)
			if err != nil {
				t.Fatal(err)
			}
			talks += x
		}
		day := strings.SplitN(baseRows[i], ",", 2)[0]
		perHead := 2 * talks / float64(atoi(t, pop[day]))
		if perHead < 2 || perHead > 5 {
			t.Errorf("row %d: %.3f expected conversations per person per day, want 2–5", i, perHead)
		}
	}
	t.Logf("year 20: %s", netRows[len(netRows)-1][len(baseRows[len(baseRows)-1])+1:])
}

// ODD §6.12 / T10.1: the beliefs module on changes nothing in daily.csv or
// events.jsonl, and with nothing to learn, legitimacy stays at the taught
// level in every zone for 200 years: the rule itself never drifts.
func TestBeliefsSteadyWithNothingToLearn(t *testing.T) {
	golden := loadInto(t, goldenScenario)
	run := func(extra ...string) string {
		cfg := loadInto(t, "../../scenarios/baseline.yaml")
		// A small silo that stays small: the lottery aims at its own size.
		cfg.Seed, cfg.WarmupYears, cfg.Years, cfg.Population = golden.Seed, 0, 200, golden.Population
		cfg.Lottery.TargetPopulation = cfg.Population
		cfg.Output.InvariantEvery = 30
		cfg.Modules = append(cfg.Modules, extra...)
		return mustSimulate(t, cfg)
	}
	base, on := run(), run("space", "network", "beliefs")
	for _, f := range []string{"daily.csv", "events.jsonl"} {
		if !bytes.Equal(readFile(t, filepath.Join(base, f)), readFile(t, filepath.Join(on, f))) {
			t.Errorf("the beliefs module changed %s", f)
		}
	}
	rows := strings.Split(strings.TrimSpace(string(readFile(t, filepath.Join(on, "yearly.csv")))), "\n")
	if !strings.HasSuffix(rows[0], ",meanL,meanLUpTop,meanLMids,meanLDownDeep") || len(rows) != 201 {
		t.Fatalf("%d rows, header %q", len(rows), rows[0])
	}
	want := strconv.FormatFloat(loadInto(t, "../../scenarios/baseline.yaml").Beliefs.TaughtLegitimacy, 'f', 4, 64)
	for i, row := range rows[1:] {
		cols := strings.Split(row, ",")
		for _, c := range cols[len(cols)-4:] {
			if c != want {
				t.Fatalf("year %d: mean L %v, want %s everywhere", i+1, cols[len(cols)-4:], want)
			}
		}
	}
}
