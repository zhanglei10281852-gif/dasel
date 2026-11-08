#!/bin/sh
set -eu
repo="$1"
task_dir=$(cat "$repo/evidence/review/validation-directory.txt")
out="$repo/evidence/review"
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOMODCACHE="$task_dir/modules" GOCACHE="$task_dir/cache" GOFLAGS=-mod=vendor
cd "$task_dir/project"
cp "$out/probe.go" evidence/review/probe.go
go run ./evidence/review/probe.go > "$out/independent-probe.log"
cat "$out/independent-probe.log"
go build -o "$task_dir/review-probe" ./evidence/review/probe.go
: > "$out/repeated-probe.log"
for n in 1 2 3 4 5 6 7 8 9 10; do
 "$task_dir/review-probe" > "$task_dir/repeated-$n.log"
 tail -n 1 "$task_dir/repeated-$n.log" >> "$out/repeated-probe.log"
done
cat "$out/repeated-probe.log"
python3 "$out/cli-probe.py" "$task_dir/dasel" > "$out/real-cli-probe.jsonl"
cat "$out/real-cli-probe.jsonl"
