#!/usr/bin/env bash
# Checks on a canon batch beyond C1–C4 (T12.8), read from yearly.csv only:
#  - cleanings per measured year, the plausibility check on surveillance σ
#    (ODD §7: about 0–2 a year, "years can pass between cleanings");
#  - the share of adults who know the history and their mean L, averaged
#    over the runs still alive, at chosen measured years (p_s and π are
#    checked against C4 here; the scripted cleaning failure is in year 150).
# Usage: analysis/canon.sh BATCH_DIR (e.g. runs/t12.8-canon).
set -euo pipefail
batch=${1:?usage: analysis/canon.sh BATCH_DIR}

awk -F, '
FNR == 1 {
	for (i = 1; i <= NF; i++) col[$i] = i
	runs++
	next
}
{
	y = FNR - 1 # measured year of this snapshot
	clean[FILENAME] += $col["cleanings"]
	failed += $col["failedCleanings"]
	years[FILENAME] = y
	know[y] += $col["knowHistory"]
	L[y] += $col["meanL"]
	alive[y]++
}
END {
	lo = 1e9; hi = 0; n = 0; ok = 0; sum = 0
	for (f in clean) {
		rate = clean[f] / years[f]
		sum += rate; n++
		if (rate < lo) lo = rate
		if (rate > hi) hi = rate
		if (rate <= 2) ok++
	}
	printf "runs %d; cleanings per year: mean %.2f, range %.2f–%.2f, %d of %d runs at most 2; failed cleanings %d\n", runs, sum / n, lo, hi, ok, n, failed
	printf "%5s %6s %12s %7s\n", "year", "alive", "knowHistory", "meanL"
	split("1 10 50 100 149 150 151 155 160 175 190 200 250 300", at, " ")
	for (i = 1; i in at; i++) {
		y = at[i]
		if (alive[y]) printf "%5d %6d %12.4f %7.4f\n", y, alive[y], know[y] / alive[y], L[y] / alive[y]
	}
}' "$batch"/*/yearly.csv
