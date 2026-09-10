---
name: Bug report
about: Report a bug or unexpected behavior
title: "[Bug]: "
labels: ["bug"]
assignees: []
---
<!--
monmux redacts serial numbers, serial strings, raw EDID hex and macOS display
UUIDs in every output by default, so `monmux info` and `monmux doctor` are safe
to paste. Do not paste `--show-serial` output: it prints exactly what the
redaction exists to keep out of a public issue.
-->

## Step 1: Are you in the right place?

* [ ] I have checked that there are no duplicate active or recent issues describing this problem.
* [ ] I am using the latest release, or have reproduced this on `main`.
* [ ] If this is about a monitor that is not in the catalog, I am filing a **Monitor report** instead.

## Step 2: Describe your environment

* `monmux version`: `?`
* Operating system and version: `?`
* Linux: `ddcutil --version`: `?`
* macOS: `brew info m1ddc` (the installed version): `?`
* Installed from (Homebrew cask, apt, rpm, tarball, `go install`, source): `?`

### `monmux doctor`

```text
<paste here; it redacts by default>
```

### `monmux info`

```text
<paste here; it redacts by default>
```

## Step 3: Describe the problem

### The exact command

<!-- The whole command line, including every flag. -->

```sh
$
```

* Exit code: `?`
* Was `--unsafe-model` involved? If so, which catalog entry did you name? `?`
* Did you run it with `--dry-run` first, and what did it print?

### Observed results

<!-- What happened? Paste the output, including the refusal message if there was one. -->

```text
```

### Expected results

<!-- What did you expect to happen? -->

*

### Anything else

<!-- Whether the monitor changed input, whether it came back, what the OSD showed, and anything else that helps. -->

*
