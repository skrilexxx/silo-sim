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
	"slices"
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

// runSmallCentury runs 200 years of a 1,000-person silo, with extra
// modules on; null switches the beliefs module's sources of knowledge off.
func runSmallCentury(t *testing.T, null bool, extra ...string) string {
	t.Helper()
	golden := loadInto(t, goldenScenario)
	cfg := loadInto(t, "../../scenarios/baseline.yaml")
	// A small silo that stays small: the lottery aims at its own size.
	cfg.Seed, cfg.WarmupYears, cfg.Years, cfg.Population = golden.Seed, 0, 200, golden.Population
	cfg.Lottery.TargetPopulation = cfg.Population
	if null {
		cfg.Beliefs.Relics, cfg.Beliefs.FlamekeeperSeats = 0, 0
	}
	cfg.Output.InvariantEvery = 30
	cfg.Modules = append(cfg.Modules, extra...)
	return mustSimulate(t, cfg)
}

// T10.5 done-when: with no regime, the Flamekeepers survive generations:
// 200 years in a 1,000-person silo, with relics, and the group never
// empties. Its size is logged.
func TestFlamekeepersSurviveWithoutRegime(t *testing.T) {
	rows := strings.Split(strings.TrimSpace(string(readFile(t, filepath.Join(runSmallCentury(t, false, "space", "network", "beliefs"), "yearly.csv")))), "\n")
	var sizes []string
	for i, row := range rows[1:] {
		cols := strings.Split(row, ",")
		n := atoi(t, cols[len(cols)-1])
		if n == 0 {
			t.Fatalf("year %d: no Flamekeepers left", i+1)
		}
		if i%25 == 24 {
			sizes = append(sizes, cols[len(cols)-1])
		}
	}
	t.Logf("Flamekeepers every 25 years: %s", strings.Join(sizes, " "))
}

// ODD §6.12 / T10.1: the beliefs module on changes nothing in daily.csv or
// events.jsonl, and with nothing to learn (the null setting), legitimacy stays at the taught
// level in every zone for 200 years: the rule itself never drifts.
func TestBeliefsSteadyWithNothingToLearn(t *testing.T) {
	base, on := runSmallCentury(t, true), runSmallCentury(t, true, "space", "network", "beliefs")
	for _, f := range []string{"daily.csv", "events.jsonl"} {
		if !bytes.Equal(readFile(t, filepath.Join(base, f)), readFile(t, filepath.Join(on, f))) {
			t.Errorf("the beliefs module changed %s", f)
		}
	}
	rows := strings.Split(strings.TrimSpace(string(readFile(t, filepath.Join(on, "yearly.csv")))), "\n")
	const tail = ",meanL,meanLUpTop,meanLMids,meanLDownDeep,knowOutside,knowScreen,knowHistory,knowSilos,knowLottery,hiddenAny,dosed,relicsHidden,relicsHeld,flamekeepers"
	if !strings.HasSuffix(rows[0], tail) || len(rows) != 201 {
		t.Fatalf("%d rows, header %q", len(rows), rows[0])
	}
	l := strconv.FormatFloat(loadInto(t, "../../scenarios/baseline.yaml").Beliefs.TaughtLegitimacy, 'f', 4, 64)
	want := strings.Join([]string{l, l, l, l, "0.0000", "0.0000", "0.0000", "0.0000", "0.0000", "0.0000", "0", "0", "0", "0"}, ",")
	for i, row := range rows[1:] {
		if !strings.HasSuffix(row, ","+want) {
			t.Fatalf("year %d: %q, want it to end with mean L %s everywhere and nobody knowing anything", i+1, row, l)
		}
	}
}

// column returns the named column of yearly.csv, one value per year.
func column(t *testing.T, dir, name string) []int {
	t.Helper()
	rows := strings.Split(strings.TrimSpace(string(readFile(t, filepath.Join(dir, "yearly.csv")))), "\n")
	i := slices.Index(strings.Split(rows[0], ","), name)
	if i < 0 {
		t.Fatalf("no column %s in %q", name, rows[0])
	}
	var out []int
	for _, row := range rows[1:] {
		out = append(out, atoi(t, strings.Split(row, ",")[i]))
	}
	return out
}

func sum(xs []int) int {
	n := 0
	for _, x := range xs {
		n += x
	}
	return n
}

// T11.1–T11.2: 200 years of a 1,000-person silo at the canon settings,
// with the control module on and invariants checked every 30 days. There
// is always a Head of IT; acts are seen, people arrested and tried.
func TestControlRuns(t *testing.T) {
	on := runSmallCentury(t, false, "space", "network", "beliefs", "control")
	heads := strings.Count(string(readFile(t, filepath.Join(on, "events.jsonl"))), `"type":"headOfIT"`)
	keepers := column(t, on, "flamekeepers")
	t.Logf("200 years: %d acts seen, %d arrests, %d warnings, %d prison, %d mines, %d relics taken, %d raids, %d Heads of IT",
		sum(column(t, on, "actsSeen")), sum(column(t, on, "arrests")), sum(column(t, on, "warnings")), sum(column(t, on, "prison")),
		sum(column(t, on, "mines")), sum(column(t, on, "relicsTaken")), sum(column(t, on, "raids")), heads)
	t.Logf("Flamekeepers every 20 years: %v", everyNth(keepers, 20))
	if sum(column(t, on, "arrests")) == 0 || heads < 2 {
		t.Errorf("no arrests or fewer than 2 Heads of IT")
	}
}

// T11.2 done-when: with arrests on, the Flamekeepers can die out. With
// every telling seen (σ = 1) and every arrested member naming another
// (p_name = 1), they are rolled up within 30 years.
func TestFlamekeepersCanDieOut(t *testing.T) {
	golden := loadInto(t, goldenScenario)
	cfg := loadInto(t, "../../scenarios/baseline.yaml")
	cfg.Seed, cfg.WarmupYears, cfg.Years, cfg.Population = golden.Seed, 0, 30, golden.Population
	cfg.Lottery.TargetPopulation = cfg.Population
	cfg.Control.Surveillance, cfg.Control.NameChance = 1, 1
	cfg.Output.InvariantEvery = 30
	cfg.Modules = append(cfg.Modules, "space", "network", "beliefs", "control")
	keepers := column(t, mustSimulate(t, cfg), "flamekeepers")
	gone := slices.Index(keepers, 0)
	t.Logf("Flamekeepers each year: %v; gone in year %d", keepers, gone+1)
	if gone < 0 {
		t.Error("the Flamekeepers survived")
	}
}

func everyNth(xs []int, n int) []int {
	var out []int
	for i := n - 1; i < len(xs); i += n {
		out = append(out, xs[i])
	}
	return out
}

// ODD §6.13, Off and null: with σ = 0 and no raids nobody is charged or
// arrested. The files are not byte-identical to a run without the
// module: the Head of IT taking a shadow shifts the work module's draws.
func TestControlNull(t *testing.T) {
	golden := loadInto(t, goldenScenario)
	cfg := loadInto(t, "../../scenarios/baseline.yaml")
	cfg.Seed, cfg.WarmupYears, cfg.Years, cfg.Population = golden.Seed, 0, 100, golden.Population
	cfg.Lottery.TargetPopulation = cfg.Population
	cfg.Control.Surveillance, cfg.Control.RaidsPerYear = 0, 0
	cfg.Output.InvariantEvery = 30
	cfg.Modules = append(cfg.Modules, "space", "network", "beliefs", "control")
	dir := mustSimulate(t, cfg)
	for _, name := range []string{"actsSeen", "arrests", "relicsTaken", "raids", "held", "flagged", "openCases"} {
		if n := sum(column(t, dir, name)); n != 0 {
			t.Errorf("%s: %d, want 0", name, n)
		}
	}
}
