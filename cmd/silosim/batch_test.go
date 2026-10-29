package main

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// T5.2 done-when: a batch's output for seed 7 is identical to a single
// run with seed 7. Seeds 5–9 on 3 workers, so runs really overlap.
func TestBatchMatchesSingleRun(t *testing.T) {
	cfg := loadInto(t, goldenScenario)
	cfg.Years = 5
	var progress bytes.Buffer
	if err := runBatch(cfg, []uint64{5, 6, 7, 8, 9}, 3, &progress); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(progress.String(), " ok\n"); n != 5 {
		t.Errorf("progress shows %d finished runs, want 5:\n%s", n, progress.String())
	}

	single := loadInto(t, goldenScenario)
	single.Years, single.Seed = 5, 7
	one := mustSimulate(t, single)
	seven := filepath.Join(cfg.Output.Dir, "golden-7")
	for _, f := range []string{"daily.csv", "yearly.csv", "events.jsonl"} {
		if !bytes.Equal(readFile(t, filepath.Join(seven, f)), readFile(t, filepath.Join(one, f))) {
			t.Errorf("batch seed 7 %s differs from a single run with seed 7", f)
		}
	}
}

func TestParseSeeds(t *testing.T) {
	for in, want := range map[string]string{"7": "[7]", "1-3": "[1 2 3]", "4-4": "[4]"} {
		got, err := parseSeeds(in)
		if err != nil || fmt.Sprint(got) != want {
			t.Errorf("parseSeeds(%q) = %v, %v; want %s", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "x", "5-2", "1-", "-3", "1-x"} {
		if _, err := parseSeeds(bad); err == nil {
			t.Errorf("parseSeeds(%q) accepted", bad)
		}
	}
}
