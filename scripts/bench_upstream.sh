#!/bin/sh
# Compare this fork with github.com/disintegration/imaging on the same input.
#
# Usage: scripts/bench_upstream.sh [rounds] [benchmark regexp]
#
# The two implementations run in turn, one process per round, so that a change
# in the machine's load or clock during the run lands on both sides alike. Set
# CPUS to pin the runs to some cores (for example CPUS=8-11) and GOMAXPROCS to
# the number of them. The results are written to bench-results/ and summarised
# with benchstat, which the script runs through `go run`.
set -eu

rounds=${1:-10}
bench=${2:-.}
root=$(cd "$(dirname "$0")/.." && pwd)
out="$root/bench-results"
mkdir -p "$out"
: "${GOMAXPROCS:=4}"
export GOMAXPROCS

cd "$root/scripts/upstreambench"
go test -c -o "$out/upstreambench.test" .

pin=""
if [ -n "${CPUS:-}" ]; then pin="taskset -c $CPUS"; fi

: > "$out/upstream.txt"
: > "$out/fork.txt"
i=0
while [ "$i" -lt "$rounds" ]; do
  for impl in upstream fork; do
    IMPL=$impl $pin "$out/upstreambench.test" -test.run '^$' -test.bench "^BenchmarkCompare\$/$bench" \
      -test.benchtime 200ms -test.benchmem >> "$out/$impl.txt"
  done
  i=$((i + 1))
done

go run golang.org/x/perf/cmd/benchstat@v0.0.0-20261117185222-4febdb082b2c "$out/upstream.txt" "$out/fork.txt" | tee "$out/benchstat.txt"
