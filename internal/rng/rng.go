// Package rng provides the seeded random streams. All randomness in the
// simulation goes through a Stream; nothing uses the global math/rand.
package rng

import (
	"fmt"
	"math/rand/v2"
)

type Stream struct {
	r *rand.Rand
}

// New returns the stream for one module under one run seed.
func New(seed uint64, name string) (*Stream, error) {
	id, ok := streamID(name)
	if !ok {
		return nil, fmt.Errorf("rng: no stream for %q (add it to streams.go)", name)
	}
	return &Stream{r: rand.New(rand.NewPCG(seed, id))}, nil
}

// Bernoulli reports true with probability p: never for p ≤ 0, always for p ≥ 1.
func (s *Stream) Bernoulli(p float64) bool {
	return s.r.Float64() < p
}

// IntRange returns a uniform integer in [lo, hi], both ends included.
// It panics if hi < lo.
func (s *Stream) IntRange(lo, hi int) int {
	return lo + s.r.IntN(hi-lo+1)
}

// WeightedIndex returns i with probability w[i] / sum(w). Zero weights are
// never chosen. It returns -1 if there is nothing to choose (no positive
// weight), so callers decide what "no candidate" means.
func (s *Stream) WeightedIndex(w []float64) int {
	total := 0.0
	last := -1
	for i, x := range w {
		if x > 0 {
			total += x
			last = i
		}
	}
	if last < 0 {
		return -1
	}
	u := s.r.Float64() * total
	cum := 0.0
	for i, x := range w {
		if x > 0 {
			cum += x
			if u < cum {
				return i
			}
		}
	}
	// Only reached if rounding makes u == total; the last positive weight
	// owns the top of the range.
	return last
}

// Shuffle puts n elements in random order using swap, like rand.Shuffle.
func (s *Stream) Shuffle(n int, swap func(i, j int)) {
	s.r.Shuffle(n, swap)
}
