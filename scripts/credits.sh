#!/bin/sh
# Writes CREDITS from the licenses of the Go modules compiled into the release binaries,
# followed by _EXTRA_CREDITS.
#
# gocredits is not run on the repository itself because, where a go.mod is present, it
# credits every module go.sum holds a source hash for, which includes modules that only
# the tests of dependencies need. Those are neither in the binary nor downloaded by a
# build, so it warns about each one it cannot find. It is run instead on a copy of go.mod
# with a go.sum narrowed to the modules the binaries are built from.
set -eu

cd "$(dirname "$0")/.."

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

cp go.mod "$tmp/"
# The union over the platforms .goreleaser.yml builds for, since a module can be
# imported behind a build constraint of one of them only.
for goos in linux darwin windows; do
  CGO_ENABLED=0 GOOS=$goos go list -deps \
    -f '{{with .Module}}{{if not .Main}}{{.Path}} {{.Version}}{{end}}{{end}}' .
done | sort -u > "$tmp/modules"
awk 'NR == FNR { m[$1 " " $2] = 1; next } $2 !~ /\/go\.mod$/ && ($1 " " $2) in m' \
  "$tmp/modules" go.sum > "$tmp/go.sum"

(cd "$tmp" && gocredits -skip-missing .) > CREDITS
cat _EXTRA_CREDITS >> CREDITS
