#!/usr/bin/env bash
# Proves the domain rules of arch-go.yml bite.
#
# Builds a throwaway module that mirrors the layout of internal/domain, then
# adds one forbidden import at a time and checks that arch-go rejects it.
# Nothing is written inside the repository.
#
# Usage: scripts/arch-probes.sh        (from modules/[[.Name]])
set -euo pipefail

cd "$(dirname "$0")/.."
rules=$PWD/arch-go.yml
mod=[[.OrgPrefix]]/[[.Name]]
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

pkg() { # pkg <dir> <package name> [import path suffix]
	mkdir -p "$work/internal/domain/$1"
	{
		echo "package $2"
		[ -z "${3-}" ] || echo "import _ \"$mod/internal/domain$3\""
	} >"$work/internal/domain/$1/$2.go"
}

printf 'module %s\n\ngo 1.27\n' "$mod" >"$work/go.mod"
cp "$rules" "$work/arch-go.yml"
pkg . domain
pkg vo vo
pkg [[.Entity]] [[.Entity]] ""
pkg autre autre /vo
pkg [[.Entity]]/sub sub
echo 'import _ "'"$mod"'/internal/domain"' >>"$work/internal/domain/[[.Entity]]/[[.Entity]].go"

check() { (cd "$work" && arch-go >/dev/null 2>&1); }

failures=0
expect() { # expect <pass|fail> <label>
	local got=pass
	check || got=fail
	if [ "$got" = "$1" ]; then
		echo "ok    $2 ($got)"
	else
		echo "WRONG $2: expected $1, arch-go says $got"
		failures=$((failures + 1))
	fi
}

probe() { # probe <dir> <package name> <import path suffix> <label>
	local f="$work/internal/domain/$1/zz_probe.go"
	printf 'package %s\n\nimport _ "%s/internal/domain%s"\n' "$2" "$mod" "$3" >"$f"
	expect fail "$4"
	rm "$f"
}

expect pass "allowed layout: [[.Entity]] -> root, autre -> vo"
probe vo vo "" "vo imports the root package"
probe vo vo /[[.Entity]] "vo imports a domain"
probe . domain /vo "root imports vo"
probe [[.Entity]] [[.Entity]] /autre "[[.Entity]] imports another domain"
probe autre autre /[[.Entity]] "another domain imports [[.Entity]]"
probe [[.Entity]]/sub sub /autre "a sub-package of a domain imports another domain"

[ "$failures" -eq 0 ] || { echo "$failures expectation(s) not met"; exit 1; }
echo "all domain boundary rules bite"
