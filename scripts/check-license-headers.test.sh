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
# Tests for scripts/check-license-headers.sh, run against scratch repositories.
#
# A shell test rather than a Go one: monmux's Go tests never start a process
# (docs/testing.md, TestNoTestFileCanStartAProcess), and this script is git and bash. It never runs
# monmux, ddcutil or m1ddc.
#
# Usage: bash scripts/check-license-headers.test.sh

set -euo pipefail

SCRIPT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/check-license-headers.sh"

# shellcheck source=scripts/lib/scratch-git.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib/scratch-git.sh"

HEADER_GO='/*
 * Copyright 2026 Roberto Leinardi.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */'

HEADER_HASH="$(printf '%s\n' "$HEADER_GO" | sed -e '1d' -e '$d' -e 's|^ \*|#|' -e 's|^#$|#|')"

# new_repo creates a scratch repository holding one governed file of each kind, every one with the
# header, plus a Makefile naming one make-common fragment that carries none.
new_repo() {
	local dir
	dir="$(scratch_repo)"

	printf '%s\n\npackage main\n' "$HEADER_GO" >"$dir/main.go"
	printf '//go:build linux\n\n%s\n\npackage main\n' "$HEADER_GO" >"$dir/tagged.go"
	printf '#!/usr/bin/env bash\n#\n%s\n#\n\necho hi\n' "$HEADER_HASH" >"$dir/run.sh"
	mkdir -p "$dir/.mk"
	printf '#\n%s\n#\n\nMK_COMMON_FILES := help.mk\n' "$HEADER_HASH" >"$dir/Makefile"
	printf 'help:\n\t@true\n' >"$dir/.mk/help.mk"
	printf '#\n%s\n#\n\nlocal:\n' "$HEADER_HASH" >"$dir/.mk/local.mk"
	printf '# No header, and not governed.\n' >"$dir/README.md"
	printf 'key: value\n' >"$dir/config.yaml"

	scratch_git "$dir" add -A
	scratch_git "$dir" commit -q -m "test(scratch): seed"

	printf '%s\n' "$dir"
}

# check runs the script inside dir and records its exit status and output.
check() {
	local dir="$1"
	shift

	STATUS=0
	OUTPUT="$(cd "$dir" && bash "$SCRIPT" "$@" 2>&1)" || STATUS=$?
}

test_clean_tree_passes() {
	local dir
	dir="$(new_repo)"

	check "$dir"
	expect_status 0
	expect_output "5 files checked"
}

test_missing_header_fails() {
	local dir
	dir="$(new_repo)"

	printf 'package main\n' >"$dir/bare.go"
	scratch_git "$dir" add bare.go

	check "$dir"
	expect_status 1
	expect_output "  bare.go"
}

test_other_licence_fails() {
	local dir
	dir="$(new_repo)"

	printf '// Copyright 2026 Roberto Leinardi.\n// SPDX-License-Identifier: GPL-3.0-or-later\n\npackage main\n' \
		>"$dir/gpl.go"
	scratch_git "$dir" add gpl.go

	check "$dir"
	expect_status 1
	expect_output "  gpl.go"
}

test_header_past_the_window_fails() {
	local dir
	dir="$(new_repo)"

	{
		for _ in $(seq 1 30); do
			printf '// padding\n'
		done
		printf '%s\n\npackage main\n' "$HEADER_GO"
	} >"$dir/late.go"
	scratch_git "$dir" add late.go

	check "$dir"
	expect_status 1
	expect_output "  late.go"
}

test_local_fragment_and_makefile_are_governed() {
	local dir
	dir="$(new_repo)"

	printf 'local:\n' >"$dir/.mk/local.mk"
	printf 'MK_COMMON_FILES := help.mk\n' >"$dir/Makefile"

	check "$dir"
	expect_status 1
	expect_output "  .mk/local.mk"
	expect_output "  Makefile"
}

test_make_common_fragment_is_exempt() {
	local dir
	dir="$(new_repo)"

	# help.mk carries no header and is named in MK_COMMON_FILES; once it is not, it is governed.
	check "$dir"
	expect_status 0

	printf '#\n%s\n#\n\nMK_COMMON_FILES := go.mk\n' "$HEADER_HASH" >"$dir/Makefile"

	check "$dir"
	expect_status 1
	expect_output "  .mk/help.mk"
}

test_untracked_and_deleted_files_are_ignored() {
	local dir
	dir="$(new_repo)"

	printf 'package main\n' >"$dir/untracked.go"
	rm "$dir/main.go"

	check "$dir"
	expect_status 0
	expect_output "4 files checked"
}

test_arguments_are_refused() {
	local dir
	dir="$(new_repo)"

	check "$dir" --fix
	expect_status 2
	expect_output "Usage:"
}

run_tests "$@"
