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

ifndef MK_LOCAL_VERIFY_INCLUDED
MK_LOCAL_VERIFY_INCLUDED := 1

# Repo-local umbrella target: one command that runs every Go check a change needs before it is
# pushed. It runs nothing that writes to a monitor: the tests refuse to start a process at all
# (docs/testing.md), and nothing here runs monmux.
#
# go-vet and go-lint see only the host's backend, because each OS backend sits behind its own build
# tag (.mk/cross.mk). go-vet-cross and go-lint-cross add the other one, so a change that breaks
# internal/backend/m1ddc fails here on Linux and not first in CI's lint-macos job. go-lint runs the
# golangci-lint the pre-commit hook pins; go-lint-cross needs one on PATH, for the reason given
# in .mk/cross.mk.
.PHONY: verify-go
verify-go: go-build go-test go-vet go-vet-cross go-lint go-lint-cross ## Verify the Go code (build, race tests, vet and lint for both OS backends)

endif  # MK_LOCAL_VERIFY_INCLUDED
