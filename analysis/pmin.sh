#!/usr/bin/env bash
# Measures the point of no return P_min (ODD §1.2, T12.6).
#
# Runs start from damaged states: the baseline silo (scenarios/baseline.yaml,
# baseline modules only) is warmed up for 50 years, then on the first
# measured day a `cull` event (ODD §6.1) kills a share of everyone, or of
# Mechanical's workers. Each state runs 100 seeds for 50 measured years.
# A seed recovers if the population is above 9,000 on some day.
# P_min is the highest population at which fewer than 5% of seeds recover.
#
# Prints one row per state: the cull, the mean population right after it,
# the share of seeds that recovered, the mean lowest population, the share
# that died out, and the mean final population. For Mechanical cuts the
# population starts above 9,000, so "recover" says nothing there; read the
# share that died out.
# Usage: analysis/pmin.sh [SEEDS] (default 1-100). Needs bin/silosim.
set -euo pipefail
cd "$(dirname "$0")/.."
seeds=${1:-1-100}
work=runs/pmin
mkdir -p "$work"
day=$((50 * 365 + 1)) # the first measured day

state() { # name share [department]
	local name=$1 share=$2 dept=${3:-}
	local scen=$work/$name.yaml
	{
		echo "extends: ../../scenarios/baseline.yaml"
		echo "name: $name"
		echo "years: 50"
		echo "events:"
		echo "  - {day: $day, type: cull, share: $share${dept:+, department: $dept}}"
	} >"$scen"
	bin/silosim batch --scenario "$scen" --seeds "$seeds" --out "$work/$name" >/dev/null 2>"$work/$name.log"
	awk -F, -v name="$name" '
		FNR == 1 { next }
		FNR == 2 { runs++; after += $2; low = $2; up = 0 }
		{ if ($2 < low) low = $2; if ($2 > 9000) up = 1; last = $2 }
		ENDFILE { lows += low; ups += up; dead += (last == 0); final += last }
		END { printf "%-14s %8.0f %8.2f %8.0f %8.2f %8.0f\n", name, after / runs, ups / runs, lows / runs, dead / runs, final / runs }
	' "$work/$name"/*/daily.csv
	rm -rf "${work:?}/$name"
}

printf "%-14s %8s %8s %8s %8s %8s\n" state after recover lowest dead final
for k in 1 2 25 3 35 4 5 6 7 8 9; do
	state "pop-0.$k" "0.$k"
done
for k in 1 2 3 4 5; do
	state "mech-0.$k" "0.$k" mechanical
done
