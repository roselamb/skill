#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
runtime_dir=$("$project_root/scripts/fetch-go-toolchain.sh")
go_cache=${PORTABLE_PET_GOCACHE:-"$project_root/work/go-cache"}

mkdir -p \
    "$go_cache" \
    "$project_root/work/go-path/pkg/mod" \
    "$project_root/work/go-bin"

export GOROOT="$runtime_dir"
export GOCACHE="$go_cache"
export GOPATH="$project_root/work/go-path"
export GOMODCACHE="$project_root/work/go-path/pkg/mod"
export GOBIN="$project_root/work/go-bin"
export GOENV=off
export GOTELEMETRY=off
export GOTOOLCHAIN=local

exec "$runtime_dir/bin/go" "$@"
