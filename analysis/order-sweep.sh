#!/usr/bin/env bash
# The Order's blame (ODD §6.15, T13.2): s_DD while unrest runs, swept at 0.4, 0.6 and
# 0.8 on the canon scenario, everything else as canon.yaml. Keep the lowest value that
# passes C3 (≥ 60% of rebellions start Down Deep, ≥ 50% of those with a Mechanical
# strike) and write it into canon.yaml and the calibration log (Adam, 2026-11-04).
# About 2 hours on 8 cores with the default 20 seeds per value.
# Usage: analysis/order-sweep.sh [SEEDS] [OUT]   (defaults: 1-20, runs/t13.2-order)
set -euo pipefail
seeds=${1:-1-20}
out=${2:-runs/t13.2-order}
make build
mkdir -p "$out"
rel=$(realpath --relative-to="$out" scenarios/baseline.yaml)
for v in 0.4 0.6 0.8; do
	s="$out/order-$v.yaml"
	sed -e "s/scapegoatHardship: 0.6,/scapegoatHardship: $v,/" -e "s/^name: canon/name: order-$v/" \
		-e "s|^extends: baseline.yaml|extends: $rel|" scenarios/canon.yaml >"$s"
	grep -q "scapegoatHardship: $v," "$s" || { echo "canon.yaml has no 'scapegoatHardship: 0.6,' to sweep" >&2; exit 1; }
	bin/silosim batch --scenario "$s" --seeds "$seeds" --out "$out/$v"
done
for v in 0.4 0.6 0.8; do
	echo "== s_DD under The Order: $v"
	bin/silosim report --batch "$out/$v" | grep -E '^C[1-4] |^\(rebellions' || true
done
