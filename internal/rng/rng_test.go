package rng

import (
	"math"
	"slices"
	"testing"
)

const draws = 100_000

func mustNew(t *testing.T, seed uint64, name string) *Stream {
	t.Helper()
	s, err := New(seed, name)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func sequence(s *Stream, n int) []uint64 {
	out := make([]uint64, n)
	for i := range out {
		out[i] = s.r.Uint64()
	}
	return out
}

func TestSameSeedSameSequence(t *testing.T) {
	a := sequence(mustNew(t, 42, "health"), 1000)
	b := sequence(mustNew(t, 42, "health"), 1000)
	if !slices.Equal(a, b) {
		t.Error("same seed and stream gave different sequences")
	}
}

func TestDifferentStreamOrSeedDiffers(t *testing.T) {
	base := sequence(mustNew(t, 42, "health"), 1000)
	for _, c := range []struct {
		seed uint64
		name string
	}{{42, "work"}, {43, "health"}} {
		other := sequence(mustNew(t, c.seed, c.name), 1000)
		same := 0
		for i := range base {
			if base[i] == other[i] {
				same++
			}
		}
		if same > 0 {
			t.Errorf("seed %d stream %s: %d of 1000 draws equal the base sequence", c.seed, c.name, same)
		}
	}
}

// The reason streams exist: extra draws on one module's stream must not
// change what another module sees.
func TestDrawsOnOneStreamDoNotShiftAnother(t *testing.T) {
	want := sequence(mustNew(t, 42, "demography"), 100)

	a := mustNew(t, 42, "resources")
	b := mustNew(t, 42, "demography")
	got := make([]uint64, 100)
	for i := range got {
		for range i % 3 { // a varying number of extra draws on stream A
			a.Bernoulli(0.5)
		}
		got[i] = b.r.Uint64()
	}
	if !slices.Equal(got, want) {
		t.Error("drawing on the resources stream changed the demography stream")
	}
}

func TestUnknownStreamFails(t *testing.T) {
	if _, err := New(42, "dissent"); err == nil {
		t.Error("unknown module name got a stream")
	}
}

func TestStreamIDsAreDistinct(t *testing.T) {
	var seen []uint64
	for _, name := range []string{"engine", "demography", "health", "work", "resources", "governance", "output", "initpop"} {
		id, ok := streamID(name)
		if !ok || slices.Contains(seen, id) {
			t.Errorf("stream %q: id %d missing or reused", name, id)
		}
		seen = append(seen, id)
	}
}

func TestBernoulli(t *testing.T) {
	s := mustNew(t, 1, "engine")
	for _, p := range []float64{0, 1} {
		for range 1000 {
			if s.Bernoulli(p) != (p == 1) {
				t.Fatalf("Bernoulli(%v) gave the wrong constant answer", p)
			}
		}
	}
	// 0.3 ± 0.01 is about 7 standard errors at 100,000 draws.
	hits := 0
	for range draws {
		if s.Bernoulli(0.3) {
			hits++
		}
	}
	got := float64(hits) / draws
	t.Logf("Bernoulli(0.3) rate over %d draws: %.4f", draws, got)
	if math.Abs(got-0.3) > 0.01 {
		t.Errorf("Bernoulli(0.3) rate = %.4f", got)
	}
}

func TestIntRange(t *testing.T) {
	s := mustNew(t, 1, "engine")
	counts := make([]int, 12) // values 3..14
	for range draws {
		v := s.IntRange(3, 14)
		if v < 3 || v > 14 {
			t.Fatalf("IntRange(3, 14) = %d", v)
		}
		counts[v-3]++
	}
	// Each value expects 1/12 of draws; allow ±10% of that.
	expect := float64(draws) / 12
	t.Logf("IntRange(3, 14) counts over %d draws (expect %.0f each): %v", draws, expect, counts)
	for i, c := range counts {
		if math.Abs(float64(c)-expect) > 0.1*expect {
			t.Errorf("value %d drawn %d times, expected about %.0f", i+3, c, expect)
		}
	}
	if s.IntRange(5, 5) != 5 {
		t.Error("IntRange(5, 5) != 5")
	}
}

func TestWeightedIndex(t *testing.T) {
	s := mustNew(t, 1, "engine")
	counts := make([]int, 3)
	for range draws {
		counts[s.WeightedIndex([]float64{1, 0, 3})]++
	}
	if counts[1] != 0 {
		t.Errorf("zero weight chosen %d times", counts[1])
	}
	ratio := float64(counts[2]) / float64(counts[0])
	t.Logf("WeightedIndex({1, 0, 3}) counts over %d draws: %v, ratio %.3f", draws, counts, ratio)
	if math.Abs(ratio-3) > 0.15 {
		t.Errorf("weight 3 vs 1 chosen %.2f× as often, want about 3×", ratio)
	}
	for _, w := range [][]float64{nil, {}, {0, 0}} {
		if got := s.WeightedIndex(w); got != -1 {
			t.Errorf("WeightedIndex(%v) = %d, want -1", w, got)
		}
	}
}

func TestShuffle(t *testing.T) {
	shuffled := func() []int {
		s := mustNew(t, 1, "engine")
		x := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		s.Shuffle(len(x), func(i, j int) { x[i], x[j] = x[j], x[i] })
		return x
	}
	a, b := shuffled(), shuffled()
	if !slices.Equal(a, b) {
		t.Error("same seed gave different shuffles")
	}
	sorted := slices.Clone(a)
	slices.Sort(sorted)
	if !slices.Equal(sorted, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}) {
		t.Errorf("shuffle lost or duplicated elements: %v", a)
	}
	if slices.IsSorted(a) {
		t.Error("shuffle left the order unchanged")
	}
}
