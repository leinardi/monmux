#!/usr/bin/env bash
#
# Copyright 2026 Roberto Leinardi.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Licence header checker.
#
# monmux is Apache-2.0, and every source file says so in its own header: the notice of the
# IntelliJ copyright profile in .idea/copyright/Apache_2_0.xml. This fails on any tracked source
# file whose first lines do not carry that notice, so a header copied from a project under another
# licence is caught as surely as a missing one.
#
# Governed: Go and shell sources, the Makefile and the repo-local .mk fragments. The generated
# internal/catalog/models_gen.go is governed too: its generator writes the header, and
# TestGeneratedCatalogMatchesTheYAML holds the committed file to the generator's output. Exempt:
# the make-common files the bootstrap fetches (upstream, MIT, named in the Makefile's
# MK_COMMON_FILES) and the bootstrap script itself. Test fixtures under testdata/ are captures, not
# source, and carry no governed extension.
#
# scripts/check-license-headers.test.sh exercises this script against scratch repositories.

set -euo pipefail

usage() {
	cat <<'USAGE'
Usage: scripts/check-license-headers.sh

Checks every tracked source file for the monmux Apache-2.0 header and lists the files that lack
it. Run from anywhere inside the repository.
USAGE
}

if [[ $# -gt 0 ]]; then
	usage >&2
	exit 2
fi

cd "$(git rev-parse --show-toplevel)"

# How many leading lines the header must sit in: a shebang or a build constraint and a blank line
# may precede it.
HEADER_WINDOW=25

REQUIRED_LINES=(
	"Copyright "
	" Roberto Leinardi."
	"Licensed under the Apache License, Version 2.0 (the \"License\");"
	"https://www.apache.org/licenses/LICENSE-2.0"
	"limitations under the License."
)

# Exact paths that are not monmux's own source.
EXEMPT_PATHS=(
	"scripts/bootstrap-mk-common.sh"
)

# The shared make fragments, read from the Makefile so the list cannot drift from what the
# bootstrap actually fetches.
mk_common_files() {
	sed -n 's/^MK_COMMON_FILES[[:space:]]*:=[[:space:]]*//p' Makefile
}

is_governed() {
	local path="$1" base
	base="${path##*/}"

	case "${path}" in
	*.go | *.sh | .mk/*.mk)
		return 0
		;;
	esac

	case "${base}" in
	Makefile)
		return 0
		;;
	esac

	return 1
}

is_exempt() {
	local path="$1" entry

	for entry in "${EXEMPT_PATHS[@]}"; do
		if [[ ${path} == "${entry}" ]]; then
			return 0
		fi
	done

	for entry in ${COMMON_FILES}; do
		if [[ ${path} == ".mk/${entry}" ]]; then
			return 0
		fi
	done

	return 1
}

has_header() {
	local path="$1" head line
	head="$(head -n "${HEADER_WINDOW}" -- "${path}")"

	for line in "${REQUIRED_LINES[@]}"; do
		if [[ ${head} != *"${line}"* ]]; then
			return 1
		fi
	done

	return 0
}

COMMON_FILES="$(mk_common_files)"
missing=()
checked=0

while IFS= read -r -d '' path; do
	if ! is_governed "${path}" || is_exempt "${path}"; then
		continue
	fi

	# A tracked file deleted in the working tree is not this check's concern.
	if [[ ! -f ${path} ]]; then
		continue
	fi

	checked=$((checked + 1))

	if ! has_header "${path}"; then
		missing+=("${path}")
	fi
done < <(git ls-files -z)

if [[ ${#missing[@]} -gt 0 ]]; then
	echo "These files lack the monmux Apache-2.0 header:" >&2
	printf '  %s\n' "${missing[@]}" >&2
	echo "Copy the header from any Go or shell file (IntelliJ: the Apache-2.0 copyright profile)." >&2
	exit 1
fi

echo "License headers: ${checked} files checked, all carry the Apache-2.0 header."
