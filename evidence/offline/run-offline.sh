#!/bin/sh
set -eu
repo=$(cd "$1" && pwd)
out="$repo/evidence/review"
task_dir=$(mktemp -d /tmp/dasel-stream-963a36fb-XXXXXXXX)
mkdir -p "$task_dir/project" "$task_dir/cache" "$task_dir/modules"
git -C "$repo" archive 521d3c39fe06b276f65695d3ae27bd46b76c4106 > "$task_dir/baseline.tar"
tar xf "$task_dir/baseline.tar" -C "$task_dir/project"
cd "$task_dir/project"
git init -q
git apply --check --binary "$out/model-source.diff"
git apply --binary "$out/model-source.diff"
tar xzf "$repo/evidence/offline/vendor.tar.gz"
mkdir -p evidence/review
cp "$out/probe.go" evidence/review/probe.go
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOMODCACHE="$task_dir/modules" GOCACHE="$task_dir/cache" GOFLAGS=-mod=vendor
echo "REGISTERED BASELINE AND ORIGINAL MODEL DIFF APPLIED $task_dir/project"
echo "ISOLATED CACHE $GOCACHE MODULE CACHE $GOMODCACHE"
go version
echo "NETWORK NAMESPACE $(readlink /proc/self/ns/net)"
cat /proc/net/dev
if ip route get 1.1.1.1; then echo 'Unexpected public route'; exit 1; fi
echo 'go test -count=1 -json ./...'
go test -count=1 -json ./... > "$out/linux-tests.jsonl"
echo 'ORIGINAL FULL SUITE PASS'
go test -count=10 -run 'TestStream|TestNonStreamBehaviourUnchanged' ./internal/cli ./parsing/json > "$out/repeated-stream-tests.log"
cat "$out/repeated-stream-tests.log"
go run ./evidence/review/probe.go > "$out/independent-probe.log"
cat "$out/independent-probe.log"
go build -o "$task_dir/dasel" ./cmd/dasel
echo "$task_dir" > "$out/validation-directory.txt"
go build -o "$task_dir/review-probe" ./evidence/review/probe.go
: > "$out/repeated-probe.log"
for n in 1 2 3 4 5 6 7 8 9 10; do
 "$task_dir/review-probe" > "$task_dir/repeated-$n.log"
 tail -n 1 "$task_dir/repeated-$n.log" >> "$out/repeated-probe.log"
done
cat "$out/repeated-probe.log"
python3 "$out/cli-probe.py" "$task_dir/dasel" > "$out/real-cli-probe.jsonl"
cat "$out/real-cli-probe.jsonl"
echo 'OFFLINE REVIEW VALIDATION COMPLETE'
