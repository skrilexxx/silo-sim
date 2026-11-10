#!/usr/bin/env bash
# M15 (lean plan): every batch of docs/experiments.md, seeds 1–30, and both reports of
# each saved as OUT/<name>.txt; a summary of the key lines in OUT/summary.txt.
# About 13–14 hours on the laptop (measured 2026-11-04/05 and 2026-11-10): a run that
# takes 2 minutes alone takes 4–12 in a batch, as the CPU is held near 2.4 GHz under
# sustained load. WORKERS defaults to 4, one per physical core: on 2026-11-10 eight seeds
# took 8.6 min with 4 workers and 11.3–11.6 min with 8 (hyperthreads share a core's cache).
# A batch whose folder already holds a report is skipped, so the script can be restarted.
# Usage: analysis/m15.sh [OUT]   (default runs/m15); SEEDS=1-30 and WORKERS=4 by default.
set -euo pipefail
seeds=${SEEDS:-1-30}
workers=${WORKERS:-4}
out=${1:-runs/m15}
make build
mkdir -p "$out"

run() { # name scenario
	local name=$1 scen=$2
	if [[ -s "$out/$name.txt" ]]; then
		echo "skip $name (done)"
		return
	fi
	echo "== $name ($scen), $(date +%H:%M)"
	bin/silosim batch --scenario "$scen" --seeds "$seeds" --workers "$workers" --out "$out/$name"
	{
		bin/silosim report --batch "$out/$name" || true
		echo
		bin/silosim report --batch "$out/$name" --outcomes
	} >"$out/$name.tmp"
	mv "$out/$name.tmp" "$out/$name.txt"
}

run canon scenarios/canon.yaml                       # T15.3
for f in scenarios/experiments/*.yaml; do            # E1–E5
	run "$(basename "$f" .yaml)" "$f"
done

# T15.2: one parameter at a time, −50% and +50% of base (docs/experiments.md).
rel=$(realpath --relative-to="$out" scenarios/experiments/base.yaml)
while read -r section key low high; do
	for v in "$low" "$high"; do
		name="sens-$key-$v"
		printf 'extends: %s\nname: %s\n%s: { %s: %s }\n' "$rel" "$name" "$section" "$key" "$v" >"$out/$name.yaml"
		run "$name" "$out/$name.yaml"
	done
done <<'LIST'
beliefs taughtLegitimacy 0.9 0.7
beliefs legitimacyCostPerFlag 0.075 0.225
beliefs passChance 0.001 0.003
rebellion activeThreshold 0.05 0.15
rebellion scapegoatHardship 0.1 0.3
control cleaningFailChance 0.025 0.075
LIST

for f in "$out"/*.txt; do
	[[ "$f" == "$out/summary.txt" ]] && continue
	echo "== $(basename "$f" .txt)"
	grep -E '^C[1-5] |^rebellions per century|^share of measured time|^endings|^collapses|^survival to' "$f" || true
done >"$out/summary.txt"
echo "done: $out/summary.txt"
