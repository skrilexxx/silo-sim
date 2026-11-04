#!/usr/bin/env bash
# M15 (lean plan): every batch of docs/experiments.md, seeds 1–30, and both reports of
# each saved as OUT/<name>.txt; a summary of the key lines in OUT/summary.txt.
# About 6–7 hours on 8 cores (canon ~30 min, E1–E5 ~3 h, sensitivity ~3 h).
# A batch whose folder already holds a report is skipped, so the script can be restarted.
# Usage: analysis/m15.sh [OUT]   (default runs/m15); SEEDS=1-30 by default.
set -euo pipefail
seeds=${SEEDS:-1-30}
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
	bin/silosim batch --scenario "$scen" --seeds "$seeds" --out "$out/$name"
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
