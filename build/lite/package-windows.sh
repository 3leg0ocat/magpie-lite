#!/usr/bin/env bash
# Builds magpie-lite for Windows x64 into dist/: one magpie.exe with
# self-update off, the trimmed provider list, and Microsoft's ConPTY in it
# (Windows 10's own drops pasted text, yetone/magpie#5205), which it puts in
# conpty\<version> beside itself when Claude Code first runs.
#
#   build/lite/package-windows.sh
#
# CONPTY_NUPKG names a copy of the package already downloaded; it is checked
# against the same hash as a download is.
set -euo pipefail
cd "$(dirname "$0")/../.."

# Microsoft.Windows.Console.ConPTY from NuGet; the hash is the file's, and
# its SHA-512 is the one NuGet's catalog lists for this version
CONPTY_VERSION=1.25.260930003
CONPTY_SHA256=02b07b349af66d801159bdf9e440d4a1ce78bb951f37fc8609731665afdae7ee

# the upstream release it is built on, as -lite: a release's version to
# magpie, so a running one follows a newer build put in its place
base=$(git describe --tags --abbrev=0 --match 'v[0-9]*' --exclude '*-*')
VERSION=${VERSION:-$base-lite}
WINVER=$(echo "$base" | sed -E 's/^v//; s/[^0-9.].*//')
out=dist/magpie-lite-windows-x64.exe
bundle=internal/claudetui/conpty-bundle
work=$(mktemp -d)
trap 'rm -rf "$work" rsrc_windows_*.syso internal/update/launcher-*.com; find "$bundle" -type f ! -name .gitignore -delete' EXIT

nupkg=${CONPTY_NUPKG:-$work/conpty.nupkg}
if [ ! -f "$nupkg" ]; then
	curl -fsSL -o "$nupkg" "https://api.nuget.org/v3-flatcontainer/microsoft.windows.console.conpty/$CONPTY_VERSION/microsoft.windows.console.conpty.$CONPTY_VERSION.nupkg"
fi
echo "$CONPTY_SHA256  $nupkg" | sha256sum -c - >/dev/null
python -c 'import sys, zipfile; zipfile.ZipFile(sys.argv[1]).extractall(sys.argv[2], sys.argv[3:])' "$nupkg" "$work/conpty" \
	runtimes/win-x64/native/conpty.dll build/native/runtimes/x64/OpenConsole.exe
cp "$work/conpty/runtimes/win-x64/native/conpty.dll" "$work/conpty/build/native/runtimes/x64/OpenConsole.exe" "$bundle/"
cp build/lite/ConPTY-LICENSE.txt "$bundle/LICENSE.txt"
echo "$CONPTY_VERSION" >"$bundle/version"

mkdir -p dist
rm -f "$out" "$out.sha256"
go run github.com/tc-hib/go-winres@v0.3.3 make --in build/windows/winres.json --arch amd64 --out rsrc \
	--file-version "${WINVER:-0.0.0}" --product-version "${WINVER:-0.0.0}"
# magpie.com, the console program `magpie` in a terminal runs (Makefile's
# release-windows)
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags="-s -w" \
	-o internal/update/launcher-amd64.com ./internal/update/launcher
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags production,comlauncher,liteconpty -trimpath \
	-ldflags="-s -w -X main.version=$VERSION -X github.com/yetone/magpie/internal/update.selfUpdate=off -X github.com/yetone/magpie/internal/provider.trimmed=on -H windowsgui" \
	-o "$out" .
(cd dist && sha256sum "$(basename "$out")" >"$(basename "$out").sha256")
echo "$out"
