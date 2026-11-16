#!/usr/bin/env bash
# Builds magpie-lite for Windows x64 into dist/: one magpie.exe with
# self-update off, the trimmed provider list, and the ConPTY upstream's
# release carries for Windows 10 (internal/conptypair, yetone/magpie#5205).
#
#   build/lite/package-windows.sh
#
# CONPTY_NUPKG names a copy of the package already downloaded; build/conpty
# checks it against the same hash as a download.
set -euo pipefail
cd "$(dirname "$0")/../.."

# the upstream release it is built on, as -lite: a release's version to
# magpie, so a running one follows a newer build put in its place
base=$(git describe --tags --abbrev=0 --match 'v[0-9]*' --exclude '*-*')
VERSION=${VERSION:-$base-lite}
WINVER=$(echo "$base" | sed -E 's/^v//; s/[^0-9.].*//')
out=dist/magpie-lite-windows-x64.exe
trap 'rm -rf rsrc_windows_*.syso internal/update/launcher-*.com' EXIT

# the pair, for -tags conptypair (Makefile's conpty-pair)
go run ./build/conpty ${CONPTY_NUPKG:+"$CONPTY_NUPKG"}

mkdir -p dist
rm -f "$out" "$out.sha256"
go run github.com/tc-hib/go-winres@v0.3.3 make --in build/windows/winres.json --arch amd64 --out rsrc \
	--file-version "${WINVER:-0.0.0}" --product-version "${WINVER:-0.0.0}"
# magpie.com, the console program `magpie` in a terminal runs (Makefile's
# release-windows)
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags="-s -w" \
	-o internal/update/launcher-amd64.com ./internal/update/launcher
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags production,comlauncher,conptypair -trimpath \
	-ldflags="-s -w -X main.version=$VERSION -X github.com/yetone/magpie/internal/update.selfUpdate=off -X github.com/yetone/magpie/internal/provider.trimmed=on -H windowsgui" \
	-o "$out" .
(cd dist && sha256sum "$(basename "$out")" >"$(basename "$out").sha256")
echo "$out"
