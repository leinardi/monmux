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
# Tests for scripts/check-suppressions.sh, run against scratch repositories.
#
# A shell test rather than a Go one: monmux's Go tests never start a process
# (docs/testing.md, TestNoTestFileCanStartAProcess), and this script is git, bash and awk. It never
# runs monmux, ddcutil or m1ddc.
#
# Usage: bash scripts/check-suppressions.test.sh [test_name ...]

set -euo pipefail

SCRIPT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/check-suppressions.sh"
BASELINE=".suppressions-baseline"

# shellcheck source=scripts/lib/scratch-git.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib/scratch-git.sh"

# Fixtures, one per function so each test writes exactly the source it is about.

# One //nolint:gosec directive.
go_with_gosec() {
	cat <<'GO'
package scratch

// Run does nothing in particular.
func Run() string {
	return "ok" //nolint:gosec // scratch fixture
}
GO
}

# The same single directive on another line: the count holds, which is the case the per-rule
# counting alone cannot see.
go_with_gosec_moved() {
	cat <<'GO'
package scratch

//nolint:gosec // scratch fixture, moved
func Run() string {
	return "ok"
}
GO
}

# Two directives for the same linter.
go_with_two_gosec() {
	cat <<'GO'
package scratch

// Run does nothing in particular.
func Run() string {
	value := "ok" //nolint:gosec // scratch fixture
	return value  //nolint:gosec // scratch fixture too
}
GO
}

# A string holding "//" before the directive. golangci-lint reads the directive; a detector that
# stops at the first "//" on the line does not.
go_with_directive_after_url() {
	cat <<'GO'
package scratch

// Run does nothing in particular.
func Run() string {
	return "https://example.com/x" //nolint:gosec // scratch fixture behind a URL
}
GO
}

# A directive mentioned in a sentence: not a suppression.
go_with_prose_mention() {
	cat <<'GO'
package scratch

// Run is written so that it needs no //nolint:gosec directive at all.
func Run() string {
	return "ok"
}
GO
}

# A directive on a continuation line of a block comment, where golangci-lint honours none.
go_with_block_comment() {
	cat <<'GO'
package scratch

/*
 * Run needs no //nolint:gosec directive, and this line is not one.
 */
func Run() string {
	return "ok"
}
GO
}

# Two linters with a space after the comma, which golangci-lint accepts.
go_with_spaced_list() {
	cat <<'GO'
package scratch

// Run does nothing in particular.
func Run() error {
	return nil //nolint:gosec, ErrCheck // scratch fixture
}
GO
}

# The broadest Go suppression: no linter named.
go_with_bare_nolint() {
	cat <<'GO'
package scratch

//nolint // deliberately unspecific
func Run() {}
GO
}

# A shellcheck directive naming two codes. The directive is assembled at run time: written out
# here, it would be a suppression in this file, which the real ratchet counts.
shell_with_disable() {
	printf '#!/usr/bin/env bash\n# shellcheck %s=SC2086,SC2034\necho %sNAMES\n' disable '$'
}

# Directive text held inside a raw string, as a Go test fixture would hold it. None of it is a
# suppression.
go_with_raw_string_fixture() {
	cat <<'GO'
package scratch

// fixture is Go source held as data.
const fixture = `package other

func Run() string {
	return "ok" //nolint:gosec // inside a raw string
}
`
GO
}

# One more line inside that raw string: the ordinary way a new case is written, and a line the
# staged-diff rule must not read as a directive.
go_with_raw_string_fixture_extended() {
	cat <<'GO'
package scratch

// fixture is Go source held as data.
const fixture = `package other

func Run() string {
	value := "ok" //nolint:gosec // inside a raw string too
	return value //nolint:gosec // inside a raw string
}
`
GO
}

# A real directive after the raw string closes, which the same rule must still catch.
go_with_raw_string_fixture_and_directive() {
	go_with_raw_string_fixture
	cat <<'GO'

func Run() string {
	return "ok" //nolint:gosec // a directive in code this time
}
GO
}

# write_baseline writes a baseline holding the given rows, plus a comment as a real one has.
write_baseline() {
	local dir="$1" rows="$2"

	{
		printf '# scratch baseline\n'
		if [[ -n ${rows} ]]; then
			printf '%b\n' "${rows}"
		fi
	} >"${dir}/${BASELINE}"
}

# commit_initial creates a repository with scratch.go from the fixture function and the baseline,
# commits both and prints the repository's path.
commit_initial() {
	local fixture="$1" rows="$2" dir
	dir="$(scratch_repo)"

	"${fixture}" >"${dir}/scratch.go"
	write_baseline "${dir}" "${rows}"
	scratch_git "${dir}" add -A
	scratch_git "${dir}" commit -q -m "test(scratch): fixture"

	printf '%s\n' "${dir}"
}

# check runs the script inside dir and records its exit status and output.
check() {
	local dir="$1"
	shift

	STATUS=0
	OUTPUT="$(cd "${dir}" && bash "${SCRIPT}" "$@" 2>&1)" || STATUS=$?
}

test_a_tree_that_matches_its_baseline_passes() {
	local dir
	dir="$(commit_initial go_with_gosec 'nolint/gosec\t1')"

	check "${dir}"
	expect_status 0
}

test_an_added_suppression_with_an_untouched_baseline_is_refused() {
	local dir
	dir="$(commit_initial go_with_gosec 'nolint/gosec\t1')"

	go_with_two_gosec >"${dir}/scratch.go"
	scratch_git "${dir}" add scratch.go

	check "${dir}"
	expect_status 1
	expect_output "without touching ${BASELINE}"
}

test_the_same_addition_passes_with_the_baseline_staged() {
	local dir
	dir="$(commit_initial go_with_gosec 'nolint/gosec\t1')"

	go_with_two_gosec >"${dir}/scratch.go"
	write_baseline "${dir}" 'nolint/gosec\t2'
	scratch_git "${dir}" add scratch.go "${BASELINE}"

	check "${dir}"
	expect_status 0
}

test_a_swap_is_refused_although_the_count_holds() {
	local dir
	dir="$(commit_initial go_with_gosec 'nolint/gosec\t1')"

	go_with_gosec_moved >"${dir}/scratch.go"
	scratch_git "${dir}" add scratch.go

	check "${dir}"
	expect_status 1
	expect_output "without touching ${BASELINE}"
}

test_a_count_above_the_baseline_is_refused() {
	local dir
	dir="$(commit_initial go_with_two_gosec 'nolint/gosec\t1')"

	check "${dir}"
	expect_status 1
	expect_output "baseline allows 1"
}

test_a_count_below_the_baseline_is_refused() {
	local dir
	dir="$(commit_initial go_with_gosec 'nolint/gosec\t3')"

	check "${dir}"
	expect_status 1
	expect_output "baseline still claims 3"
}

test_a_rule_the_baseline_does_not_know_is_refused() {
	local dir
	dir="$(commit_initial go_with_gosec 'nolint/dupl\t1')"

	check "${dir}"
	expect_status 1
	expect_output "does not list this rule"
}

test_a_directive_behind_a_url_still_counts() {
	local dir
	dir="$(commit_initial go_with_directive_after_url 'nolint/dupl\t1')"

	check "${dir}"
	expect_status 1
	expect_output "nolint/gosec"
}

test_a_directive_in_prose_is_not_a_suppression() {
	local dir
	dir="$(commit_initial go_with_prose_mention '')"

	check "${dir}"
	expect_status 0
}

test_a_directive_in_a_block_comment_is_not_a_suppression() {
	local dir
	dir="$(commit_initial go_with_block_comment '')"

	check "${dir}"
	expect_status 0
}

test_a_deleted_baseline_is_refused() {
	local dir
	dir="$(commit_initial go_with_gosec 'nolint/gosec\t1')"

	scratch_git "${dir}" rm -q "${BASELINE}"

	check "${dir}"
	expect_status 1
	expect_output "is missing"
}

test_a_deleted_baseline_is_refused_with_no_suppressions_at_all() {
	local dir
	dir="$(commit_initial go_with_prose_mention '')"

	scratch_git "${dir}" rm -q "${BASELINE}"

	check "${dir}"
	expect_status 1
	expect_output "is missing"
}

test_update_writes_a_baseline_the_check_accepts() {
	local dir
	dir="$(commit_initial go_with_two_gosec 'nolint/gosec\t1')"

	check "${dir}"
	expect_status 1

	check "${dir}" --update
	expect_status 0

	OUTPUT="$(cat "${dir}/${BASELINE}")"
	expect_output "$(printf 'nolint/gosec\t2')"

	check "${dir}"
	expect_status 0
}

test_every_family_is_counted_per_rule() {
	local dir
	dir="$(scratch_repo)"

	go_with_bare_nolint >"${dir}/bare.go"
	go_with_spaced_list >"${dir}/spaced.go"
	shell_with_disable >"${dir}/run.sh"
	write_baseline "${dir}" ''
	scratch_git "${dir}" add -A
	scratch_git "${dir}" commit -q -m "test(scratch): one file per family"

	check "${dir}" --update
	expect_status 0

	OUTPUT="$(cat "${dir}/${BASELINE}")"
	expect_output "$(printf 'nolint/*\t1')"
	expect_output "$(printf 'nolint/gosec\t1')"
	expect_output "$(printf 'nolint/errcheck\t1')"
	expect_output "$(printf 'shellcheck-disable/SC2086\t1')"
	expect_output "$(printf 'shellcheck-disable/SC2034\t1')"
}

test_generated_and_fetched_files_are_skipped() {
	local dir
	dir="$(scratch_repo)"

	write_baseline "${dir}" ''
	scratch_git "${dir}" add -A
	scratch_git "${dir}" commit -q -m "test(scratch): empty baseline"

	mkdir -p "${dir}/internal/catalog" "${dir}/scripts"
	go_with_gosec >"${dir}/internal/catalog/models_gen.go"
	shell_with_disable >"${dir}/scripts/bootstrap-mk-common.sh"
	scratch_git "${dir}" add -A

	check "${dir}"
	expect_status 0
}

# The negative control of the test above: a file next to an excluded path, but not it, is counted,
# so an exclusion that grew too broad fails here.
test_neighbours_of_skipped_files_are_counted() {
	local dir name
	for name in internal/catalog/catalog.go scripts/bootstrap.sh; do
		dir="$(scratch_repo)"

		write_baseline "${dir}" ''
		scratch_git "${dir}" add -A
		scratch_git "${dir}" commit -q -m "test(scratch): empty baseline"

		mkdir -p "${dir}/$(dirname "${name}")"
		case "${name}" in
		*.go) go_with_gosec >"${dir}/${name}" ;;
		*.sh) shell_with_disable >"${dir}/${name}" ;;
		esac
		scratch_git "${dir}" add -A

		check "${dir}"
		expect_status 1
		expect_output "does not list this rule"
	done
}

test_a_line_added_inside_a_raw_string_is_not_a_suppression() {
	local dir
	dir="$(commit_initial go_with_raw_string_fixture '')"

	go_with_raw_string_fixture_extended >"${dir}/scratch.go"
	scratch_git "${dir}" add scratch.go

	check "${dir}"
	expect_status 0
}

test_a_directive_added_after_a_raw_string_is_one() {
	local dir
	dir="$(commit_initial go_with_raw_string_fixture '')"

	go_with_raw_string_fixture_and_directive >"${dir}/scratch.go"
	scratch_git "${dir}" add scratch.go

	check "${dir}"
	expect_status 1
	# The staged-diff rule's own message: the count rule refuses this edit too, and asserting on
	# the refusal alone would let a detector that found nothing pass.
	expect_output "without touching ${BASELINE}"
}

test_update_discharges_a_swap() {
	local dir before after
	dir="$(commit_initial go_with_gosec '')"

	check "${dir}" --update
	expect_status 0
	scratch_git "${dir}" add "${BASELINE}"
	scratch_git "${dir}" commit -q -m "test(scratch): record the baseline"
	before="$(cat "${dir}/${BASELINE}")"

	go_with_gosec_moved >"${dir}/scratch.go"
	scratch_git "${dir}" add scratch.go

	check "${dir}"
	expect_status 1

	check "${dir}" --update
	expect_status 0
	after="$(cat "${dir}/${BASELINE}")"
	if [[ ${after} == "${before}" ]]; then
		OUTPUT="${after}"
		fail "--update left the baseline byte-identical, so there is nothing to stage"
	fi

	scratch_git "${dir}" add "${BASELINE}"

	check "${dir}"
	expect_status 0
}

test_a_rename_is_not_an_addition() {
	local dir
	dir="$(commit_initial go_with_two_gosec 'nolint/gosec\t2')"

	scratch_git "${dir}" mv scratch.go moved.go

	check "${dir}"
	expect_status 0
}

test_unknown_arguments_are_refused() {
	local dir
	dir="$(commit_initial go_with_gosec 'nolint/gosec\t1')"

	check "${dir}" --frobnicate
	expect_status 2
	expect_output "Usage:"
}

run_tests "$@"
