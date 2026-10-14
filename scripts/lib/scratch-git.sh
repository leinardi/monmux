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
# Shared helpers for the shell tests of the scripts/check-*.sh hooks: scratch git repositories
# that cannot reach the repository being committed, assertions, and a runner. Sourced, never run.
#
# Isolation is the point. These tests also run from a pre-commit hook, and git exports its own
# location to hooks: GIT_INDEX_FILE always, GIT_DIR and others in some setups, absolute in a linked
# worktree. A git command run in a scratch repository with that environment acts on the real one
# instead -- a `git add` there writes into the index of the commit being made. So sourcing this
# drops every variable that redirects git, and the user's global and system configuration, for the
# whole test process and everything it starts.

for _scratch_var in $(compgen -e); do
	case "${_scratch_var}" in
	GIT_DIR | GIT_INDEX_FILE | GIT_WORK_TREE | GIT_OBJECT_DIRECTORY | \
		GIT_ALTERNATE_OBJECT_DIRECTORIES | GIT_COMMON_DIR | GIT_PREFIX | GIT_NAMESPACE | \
		GIT_CEILING_DIRECTORIES | GIT_CONFIG*)
		unset "${_scratch_var}"
		;;
	esac
done
unset _scratch_var

export GIT_CONFIG_GLOBAL=/dev/null
export GIT_CONFIG_NOSYSTEM=1
export GIT_AUTHOR_NAME="Scratch" GIT_AUTHOR_EMAIL="scratch@example.invalid"
export GIT_COMMITTER_NAME="Scratch" GIT_COMMITTER_EMAIL="scratch@example.invalid"

SCRATCH_ROOT="$(mktemp -d)"
trap 'rm -rf "${SCRATCH_ROOT}"' EXIT

# scratch_repo creates an empty repository on main and prints its path.
scratch_repo() {
	local dir
	dir="$(mktemp -d "${SCRATCH_ROOT}/repo.XXXXXX")"
	git -C "${dir}" init -q -b main
	printf '%s\n' "${dir}"
}

# scratch_git runs git inside a scratch repository.
scratch_git() {
	local dir="$1"
	shift
	git -C "${dir}" "$@"
}

# The assertions read STATUS and OUTPUT, which each test's check helper sets. A failed assertion
# exits: every test runs in its own subshell, so that ends the test and nothing else.

fail() {
	printf '    FAIL: %s\n' "$1" >&2
	printf '%s\n' "${OUTPUT-}" | sed 's/^/      | /' >&2
	exit 1
}

expect_status() {
	if [[ ${STATUS} != "$1" ]]; then
		fail "exit status ${STATUS}, want $1"
	fi
}

expect_output() {
	if [[ ${OUTPUT} != *"$1"* ]]; then
		fail "output lacks \"$1\""
	fi
}

expect_no_output() {
	if [[ ${OUTPUT} == *"$1"* ]]; then
		fail "output has \"$1\""
	fi
}

# run_tests runs every test_* function, or the ones named as arguments, each in its own subshell so
# a failing assertion ends that test alone. Finding no test at all is a failure: a runner that
# checks nothing must not pass.
run_tests() {
	local tests=("$@") name failed=0

	if ((${#tests[@]} == 0)); then
		mapfile -t tests < <(compgen -A function test_)
	fi

	if ((${#tests[@]} == 0)); then
		printf 'no tests found\n' >&2
		return 1
	fi

	local result
	for name in "${tests[@]}"; do
		# Not inside an if: errexit is off in a condition, and a test must stop at its first failing
		# command.
		set +e
		(
			set -euo pipefail
			"${name}"
		)
		result=$?
		set -e

		if ((result == 0)); then
			printf 'ok   %s\n' "${name}"
		else
			printf 'FAIL %s\n' "${name}"
			failed=1
		fi
	done

	return "${failed}"
}
