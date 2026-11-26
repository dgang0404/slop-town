#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later
set -eu

cd "$(dirname "$0")/.."
export GOPROXY=off GONOPROXY=none GOTOOLCHAIN=local CGO_ENABLED=0
out=$(mktemp -d)
trap 'rm -rf -- "$out"' EXIT

go version
go vet -mod=readonly ./...
go test -mod=readonly -count=1 ./...
go build -mod=readonly -o "$out/" ./cmd/...
"$out/packcheck" content/smoke

expected='seed=42 sha256=60261db115f851c52ad8e47ec6c6b787992eab8518bf7656591bd22b923e96b5'
for run in 1 2; do
    actual=$("$out/sloptown" -db "$out/save.sqlite" -seed 42)
    printf 'Smoke run %s: %s\n' "$run" "$actual"
    if [ "$actual" != "$expected" ]; then
        printf 'Expected %s\n' "$expected" >&2
        exit 1
    fi
done
