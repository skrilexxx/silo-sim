package chronicle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func folder(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// daily writes day,population rows from the first measured day to last.
func daily(first, last int) string {
	var b strings.Builder
	b.WriteString("day,population\n")
	for d := first; d <= last; d++ {
		fmt.Fprintf(&b, "%d,%d\n", d, 1000+d%7)
	}
	return b.String()
}

// One of each event a part-2 run writes, warm-up one year (365 days),
// the silo collapsing by the Safeguard on day 1230.
func partTwo(t *testing.T) string {
	return folder(t, map[string]string{
		"run.json": `{"seed":7,"outcomes":{"rebellions":1,"unrestSpells":1,"collapseDay":1230,"collapseCause":"safeguard"},
			"scenario":{"name":"tiny","warmupYears":1,"years":20,"modules":["demography","rebellion"],
			"events":[{"day":730,"type":"cleaningFailure"},{"day":1000,"type":"breakStairs","segment":5}],
			"regime":{"playbook":[{"on":"unrestStart","hold":"30","set":{"raidsPerYear":8,"surveillance":0.5}}]},
			"silo1":{"orders":{"on":"escape","hold":"365","waterDosing":true}}}}`,
		"events.jsonl": `{"day":400,"type":"mayor","zone":"mids","reason":"died","mode":"election"}
{"day":500,"type":"failure","outageDays":3}
{"day":500,"type":"emergency"}
{"day":503,"type":"repair"}
{"day":503,"type":"emergencyEnd"}
{"day":600,"type":"emergency"}
{"day":620,"type":"emergencyEnd"}
{"day":650,"type":"policy","trigger":"unrestStart","entry":0,"start":650}
{"day":680,"type":"policyEnd","trigger":"unrestStart","entry":0,"start":650}
{"day":700,"type":"unrest","start":640,"end":690}
{"day":730,"type":"scripted","event":"cleaningFailure"}
{"day":731,"type":"zoneHeld","zone":"downDeep"}
{"day":735,"type":"cleaning","rule":"noViolence","failed":true}
{"day":736,"type":"cleaning","rule":"noViolence","failed":false}
{"day":736,"type":"cleaning","rule":"relics","failed":false}
{"day":740,"type":"zoneFreed","zone":"downDeep"}
{"day":741,"type":"policy","trigger":"silo1","start":741}
{"day":800,"type":"headOfIT","successor":true}
{"day":1000,"type":"scripted","event":"breakStairs"}
{"day":1200,"type":"safeguard","goesOff":1230}
{"day":1230,"type":"collapse","cause":"safeguard"}
`,
		"rebellions.jsonl": `{"start":730,"days":9,"trigger":"cleaningFailure","triggerDay":730,"zone":"downDeep","peakActive":0.1234,"struck":["mechanical"],"fightDays":3,"held":["downDeep"],"rebelsKilled":2,"officersKilled":1,"fightArrests":4,"arrests":1234,"segmentsBroken":1,"ending":"collapse"}
`,
		"daily.csv":  daily(366, 1230),
		"yearly.csv": "day,meanL,knowHistory,flamekeepers,held\n730,0.8123,0.0250,12,3\n1095,0.7000,0.5000,9,40\n",
	})
}

// Every entry appears once, in day order, with its numbers; the
// emergency inside the outage is folded into the failure; the policy ends
// are matched; the successful cleanings are counted by rule in the
// decade line, the failed one listed.
func TestEntries(t *testing.T) {
	var out strings.Builder
	if err := Write(&out, partTwo(t)); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	want := []string{
		"# Silo 18: tiny, seed 7",
		"2.4 measured years of 20, after 1 years of warm-up. Modules: demography, rebellion.",
		"The silo collapsed in year 2.4 (safeguard). Rebellions: 1. Spells of unrest: 1.",
		"## Years 0–2.4",
		"- **Year 0.1.** A new Mayor from the Mids, by election, after the last one died.",
		"- **Year 0.4.** The generator failed: 3 days without power. An emergency was declared, lifted in year 0.4.",
		"- **Year 0.6.** An emergency was declared, with air below the line, lifted in year 0.7.",
		"- **Year 0.8.** A spell of unrest began, lasting until year 0.9.",
		"- **Year 0.8.** The regime answered unrest: raidsPerYear 8, surveillance 0.5, until year 0.9.",
		"- **Year 1.0.** A cleaning failure (scripted): a cleaner walked out of sight.",
		"- **Year 1.0.** A rebellion began in the Down Deep, after the cleaning failure of year 1.0. It lasted 9 days, with at most 12.3% of adults active. Struck: mechanical. Rebels held the Down Deep. 3 days of fighting. 2 rebels and 1 officer killed; 1,234 arrested (4 in fights). Stair segments broken: 1. Ending: the silo collapsed.",
		"- **Year 1.0.** Rebels took the Down Deep.",
		"- **Year 1.0.** A cleaner, sent out under the Pact rule noViolence, walked out of sight.",
		"- **Year 1.0.** Rebels lost their hold on the Down Deep.",
		"- **Year 1.0.** The regime answered Silo 1's order: Vitamin D+ in the water, for the rest of the run.",
		"- **Year 1.2.** A new Head of IT took office: the last Head's shadow.",
		"- **Year 1.7.** The stairs were broken at segment 5 (scripted).",
		"- **Year 2.3.** Silo 1 set off the Safeguard; it goes off in year 2.4.",
		"- **Year 2.4.** The silo collapsed: safeguard.",
		"*Year 2.0:* population 1,003; mean legitimacy 0.70; 50.0% of adults know the true history; 9 Flamekeepers; 40 people held by Judicial; cleanings this decade: noViolence 1, relics 1.",
	}
	at := 0
	for _, w := range want {
		i := strings.Index(got[at:], w)
		if i < 0 {
			t.Fatalf("missing, or out of order: %q\n\n%s", w, got)
		}
		at += i + len(w)
	}
	if n := strings.Count(got, "An emergency was declared"); n != 2 {
		t.Errorf("%d emergency entries, want 2 (one folded into the failure)", n)
	}
}

// A baseline run: no outcomes, no rebellions.jsonl, no part-2 columns;
// decade lines hold the population only, and a 25-year run has three
// sections, the last a half decade.
func TestBaseline(t *testing.T) {
	dir := folder(t, map[string]string{
		"run.json":     `{"seed":1,"scenario":{"name":"baseline","warmupYears":0,"years":25,"modules":["demography"]}}`,
		"events.jsonl": "",
		"daily.csv":    daily(1, 25*365),
		"yearly.csv":   "day,medianAge\n3650,33\n7300,34\n9125,35\n",
	})
	var out strings.Builder
	if err := Write(&out, dir); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, w := range []string{"The silo was still standing at the end.\n", "## Years 0–10", "## Years 10–20", "## Years 20–25",
		"## Years 0–10\n\n*Year 10.0:* population 1,00", "*Year 25.0:* population "} {
		if !strings.Contains(got, w) {
			t.Errorf("missing %q\n\n%s", w, got)
		}
	}
	for _, no := range []string{"legitimacy", "Flamekeepers", "Rebellions:", "cleanings"} {
		if strings.Contains(got, no) {
			t.Errorf("baseline chronicle has %q\n\n%s", no, got)
		}
	}
}

// The same run gives the same bytes.
func TestDeterministic(t *testing.T) {
	dir := partTwo(t)
	var a, b strings.Builder
	if err := Write(&a, dir); err != nil {
		t.Fatal(err)
	}
	if err := Write(&b, dir); err != nil {
		t.Fatal(err)
	}
	if a.String() != b.String() {
		t.Error("two chronicles of one run differ")
	}
}
