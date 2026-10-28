package work

import (
	"math"
	"testing"

	"github.com/skrilexxx/silo-sim/internal/world"
)

// T2.2 done-when: a fully staffed department at the silo's average skill
// × health (0.924, measured over 50 years of the full silo) has an output
// ratio of 1.1, so coverage is capped at 1; at half staff, 0.55.
func TestCoverage(t *testing.T) {
	tw := newTiny(t)
	const target, avg = 100, 0.924
	tw.m.need[medical] = tw.m.cfg.Coverage.NeedFactor * target
	for range target {
		tw.add(world.Worker, 30*year, world.UpTop, medical, avg)
	}
	tw.m.output(tw.w)
	d := tw.w.Depts[medical]
	ratio := d.Output / tw.m.need[medical]
	t.Logf("full staff at skill × health %.3f: output %.1f, need %.1f, ratio %.3f, coverage %.2f", avg, d.Output, tw.m.need[medical], ratio, d.Coverage)
	if math.Abs(ratio-1.1) > 0.005 || d.Coverage != 1 {
		t.Errorf("ratio %.3f, coverage %v; want 1.1 and 1", ratio, d.Coverage)
	}

	for _, id := range tw.w.Alive[target/2:] {
		tw.w.People[id].Alive = false
	}
	tw.w.Alive = tw.w.Alive[:target/2]
	tw.m.output(tw.w)
	if c := tw.w.Depts[medical].Coverage; math.Abs(c-0.55) > 0.005 {
		t.Errorf("half staff: coverage %.3f, want 0.55", c)
	}
}

// Output counts each shadow at shadowOutputFactor of their skill × health,
// and health scales output.
func TestOutput(t *testing.T) {
	tw := newTiny(t)
	a := tw.add(world.Worker, 30*year, world.Mids, farms, 0.8)
	tw.w.People[a].Health = 0.5
	tw.add(world.Shadow, 15*year, world.Mids, farms, 0.4)
	tw.add(world.Retired, 70*year, world.Mids, farms, 0.9) // retired: no output
	tw.m.output(tw.w)
	want := 0.8*0.5 + tw.m.c.ShadowOutputFactor*0.4
	if got := tw.w.Depts[farms].Output; math.Abs(got-want) > 1e-12 {
		t.Errorf("farms output %v, want %v", got, want)
	}
}

// retrainWorld: Mechanical (Down Deep) at 70% of a 20-worker target, with
// General workers in the Down Deep and the Mids.
func retrainWorld(t *testing.T, generalDownDeep, generalMids int) *tiny {
	tw := newTiny(t)
	for range 14 {
		tw.add(world.Worker, 30*year, world.DownDeep, mechanical, 0.9)
	}
	for range generalDownDeep {
		tw.add(world.Worker, 30*year, world.DownDeep, general, 0.9)
	}
	for range generalMids {
		tw.add(world.Worker, 30*year, world.Mids, general, 0.9)
	}
	tw.m.recount(tw.w)
	tw.m.target = make([]float64, len(tw.w.Depts))
	tw.m.target[mechanical] = 20
	tw.m.target[general] = 10
	return tw
}

// T2.2 done-when: retraining lifts a department from 70% to at least 80% of
// target; the movers come from General in its zone and restart at skill 0.2.
func TestRetrainingLiftsDepartment(t *testing.T) {
	tw := retrainWorld(t, 20, 20)
	tw.m.retrain(tw.w, stream(t))
	tw.check()
	if got := tw.w.Depts[mechanical].Headcount; got != 16 {
		t.Errorf("Mechanical has %d workers after retraining, want 16 (80%% of 20)", got)
	}
	moved := 0
	for _, id := range tw.w.Alive {
		p := &tw.w.People[id]
		if p.Department == mechanical && p.Skill == tw.m.c.RetrainStartSkill {
			moved++
			if homeZone(tw.w, id) != world.DownDeep {
				t.Errorf("retrained worker %d came from zone %d, not the Down Deep", id, homeZone(tw.w, id))
			}
		}
	}
	if moved != 2 {
		t.Errorf("%d workers retrained at skill %.1f, want 2", moved, tw.m.c.RetrainStartSkill)
	}
}

// General never gives people below 80% of its own target (8 of 10 here),
// and only its workers in the department's zone count.
func TestRetrainingLimits(t *testing.T) {
	tw := retrainWorld(t, 9, 0) // General has 9; it may give only 1
	tw.m.retrain(tw.w, stream(t))
	if m, g := tw.w.Depts[mechanical].Headcount, tw.w.Depts[general].Headcount; m != 15 || g != 8 {
		t.Errorf("Mechanical %d, General %d; want 15 and 8", m, g)
	}

	tw = retrainWorld(t, 0, 30) // plenty of General, but all in the Mids
	tw.m.retrain(tw.w, stream(t))
	if m := tw.w.Depts[mechanical].Headcount; m != 14 {
		t.Errorf("Mechanical %d after retraining from another zone, want 14", m)
	}
}

// A retrained master's General shadows get a new General master.
func TestRetrainingRemastersShadows(t *testing.T) {
	tw := retrainWorld(t, 0, 0)
	mst := tw.add(world.Worker, 30*year, world.DownDeep, general, 0.9)
	for range 12 {
		tw.add(world.Worker, 30*year, world.Mids, general, 0.9) // keep General above its floor
	}
	sh := tw.add(world.Shadow, 15*year, world.DownDeep, general, 0.1)
	tw.w.People[sh].Master = mst
	tw.m.recount(tw.w)
	tw.m.retrain(tw.w, stream(t))
	tw.check()
	if tw.w.People[mst].Department != mechanical {
		t.Fatal("the Down Deep General worker was not retrained")
	}
	if m := tw.w.People[sh].Master; m == mst || m == world.None || tw.w.People[m].Department != general {
		t.Errorf("shadow's master %d, want a General worker", m)
	}
}
