#!/bin/sh
set -eu

scanner=$1
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

db="$work/vulndb-v1"
mkdir -p "$db/ID" "$db/index"
printf '%s\n' '{"schema_version":"1.3.1","id":"GO-9999-0001","modified":"2026-01-01T00:00:00Z","published":"2026-01-01T00:00:00Z","summary":"Synthetic probe vulnerability","affected":[{"package":{"name":"stdlib","ecosystem":"Go"},"ranges":[{"type":"SEMVER","events":[{"introduced":"0"},{"fixed":"99.0.0"}]}],"ecosystem_specific":{"imports":[{"path":"net/http","symbols":["Get"]}]}}]}' > "$db/ID/GO-9999-0001.json"
printf '%s\n' '{"modified":"2026-01-01T00:00:00Z"}' > "$db/index/db.json"
printf '%s\n' '[{"path":"stdlib","vulns":[{"id":"GO-9999-0001","modified":"2026-01-01T00:00:00Z","fixed":"99.0.0"}]}]' > "$db/index/modules.json"
printf '%s\n' '[{"id":"GO-9999-0001","modified":"2026-01-01T00:00:00Z"}]' > "$db/index/vulns.json"

fixture=$work/fixture
mkdir -p "$fixture"
go_version=$(GOTOOLCHAIN=local go env GOVERSION)
printf '%s\n' 'module example.com/govulncheck-fixture' '' "go ${go_version#go}" > "$fixture/go.mod"
printf '%s\n' 'package main' '' 'import "net/http"' '' 'func main() {' '    _, _ = http.Get("https://example.com")' '}' > "$fixture/main.go"

(cd "$fixture" && GOTOOLCHAIN=local "$scanner" -db "file://$db" -scan symbol -json ./... > "$work/scan.json")
grep -Eq '"scanner_name"[[:space:]]*:[[:space:]]*"govulncheck"' "$work/scan.json"
grep -Eq '"scan_level"[[:space:]]*:[[:space:]]*"symbol"' "$work/scan.json"
grep -Eq '"osv"[[:space:]]*:[[:space:]]*"GO-9999-0001"' "$work/scan.json"
grep -Eq '"function"[[:space:]]*:[[:space:]]*"Get"' "$work/scan.json"
grep -Eq '"function"[[:space:]]*:[[:space:]]*"main"' "$work/scan.json"
printf '%s\n' 'govulncheck completed source analysis and reported the synthetic call trace'
