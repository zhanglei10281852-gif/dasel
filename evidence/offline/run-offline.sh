#!/bin/sh
set -eu
task_dir=/tmp/dasel-delivery-review-54cf53db
mkdir -p "$task_dir/project" "$task_dir/cache" "$task_dir/modules"
tar xf /mnt/f/swe/work/dasel-delivery-review/source.tar -C "$task_dir/project"
tar xzf /mnt/f/swe/099924/project-01/evidence/offline/vendor.tar.gz -C "$task_dir/project"
cp /mnt/f/swe/work/dasel-delivery-review/probe.go "$task_dir/probe.go"
cd "$task_dir/project"
export PATH=/root/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.2.linux-amd64/bin:$PATH
export GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOMODCACHE="$task_dir/modules" GOCACHE="$task_dir/cache" GOFLAGS=-mod=vendor
go version
echo "NETWORK NAMESPACE $(readlink /proc/self/ns/net)"
cat /proc/net/route
cat /proc/net/dev
if ip route get 1.1.1.1; then echo 'Unexpected public route'; exit 1; fi
echo 'CORE go test -count=1 -json ./...'
go test -count=1 -json ./... > /mnt/f/swe/099924/project-01/evidence/review/linux-tests.jsonl
echo 'CORE ALL PACKAGES PASS'
go run "$task_dir/probe.go" > /mnt/f/swe/099924/project-01/evidence/review/independent-probe.log
cat /mnt/f/swe/099924/project-01/evidence/review/independent-probe.log
go build -o "$task_dir/dasel" ./cmd/dasel
printf '%s' '{"items":[{"name":"a"},{"name":"b"}]}' | "$task_dir/dasel" query -i json -o json 'def label($suffix) { name + $suffix }; items.map(label("!"))'
printf '%s' 'null' | "$task_dir/dasel" query -i json -o json --var base=json:10 'def plus($n) { $n + $base }; plus(5)'
echo 'OFFLINE VALIDATION COMPLETE'
