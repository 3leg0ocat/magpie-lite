#!/usr/bin/env bash
# Moves the lite commits onto upstream's newest release (or the tag given),
# checks them and builds dist/magpie-lite-windows-x64.exe.
#
#   build/lite/upgrade.sh [tag]
#
# A conflict stops the rebase: resolve it, `git rebase --continue`, and run
# this again; it goes on from the checks.
set -euo pipefail
export TZ=UTC # the rebased commits' dates
cd "$(dirname "$0")/../.."

git diff --quiet && git diff --cached --quiet || { echo "uncommitted changes" >&2; exit 1; }
git fetch upstream --tags
old=$(git describe --tags --abbrev=0 --match 'v[0-9]*' --exclude '*-*')
new=${1:-$(git tag -l 'v[0-9]*' --sort=-v:refname | grep -v -- - | head -1)}
if [ "$old" != "$new" ]; then
	echo "rebasing from $old onto $new"
	git rebase --onto "$new" "$old"
fi

go vet ./internal/claudetui ./internal/update ./internal/provider ./internal/library ./internal/gui ./internal/cli
GOOS=linux go vet ./internal/claudetui
go test ./internal/update ./internal/provider -run '^(TestOff|TestTrimmed)' -count=1
build/lite/package-windows.sh
