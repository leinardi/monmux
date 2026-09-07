# Security policy

## Reporting a vulnerability

Please report security issues privately through GitHub's
[security advisories](https://github.com/leinardi/monmux/security/advisories/new) rather than in a public issue.

Include what you did, what happened, and the output of `monmux info` and `monmux doctor` if it is relevant. Both redact serial
numbers and display UUIDs by default, so their output is safe to paste.

## Security model

monmux writes to hardware, and the whole of its threat model, its mitigations, its trust boundaries and the things it
deliberately never does are documented in [docs/security.md](docs/security.md). Two points are worth repeating here:

- monmux refuses by default. It writes only to a monitor it has positively identified as a model in its built-in catalog, and
  only a value that was recorded there with evidence from a real unit.
- monmux cannot tell a genuine `ddcutil` or `m1ddc` from a maliciously replaced binary with correct ownership and permissions.
  The tool it runs is a trust boundary, not a mitigated threat.

## Supported versions

monmux has not had a release yet. Until it does, only the `main` branch is supported.
