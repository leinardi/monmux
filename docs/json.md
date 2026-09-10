# JSON output

Five monmux commands take `--json` and print their report as a JSON document instead of as prose: `switch`, `info`,
`catalog list`, `catalog show` and `version`. (`doctor` has no `--json`; its checks are the `checks` array of `info --json`.)
This page is the contract. It says what each document contains, which fields a client may rely
on, what is redacted, and — for `switch`, the only command that writes — exactly how far the guarantee reaches.

Two rules run through all of it.

**Exit code first, JSON second.** The exit code is the authority for what monmux is willing to promise; the document explains
it. A client reads the code, then reads the document, and treats a document that contradicts the code as a protocol error
rather than choosing between them. The codes are unchanged by `--json`:

| Code | Meaning                                                                                        |
| ---- | ---------------------------------------------------------------------------------------------- |
| `0`  | The input-switch command was sent, or a read-only command succeeded.                           |
| `1`  | The tool ran and failed, or the request could not be made. Whether a write happened is stated. |
| `2`  | monmux refused. No DDC write was performed.                                                    |

**Redacted by default.** Serial numbers, serial strings, raw EDID hex and macOS display UUIDs are withheld in JSON exactly as
they are in the text output, unless `--show-serial` is passed. Two shapes: inside an `identity`, `serialNumber` is `0` and
`serialString` is empty, so an absent serial and a withheld one look the same in the document; a private `handle` and the
private arguments of a rendered `command` carry the literal string `<redacted; --show-serial to print>` instead. So a document
is safe to paste into an issue without reading it first, and `--show-serial` remains the only way to see those values.

## What is stable

Stable, and changed only with a deliberate note in the release:

- the field names and the nesting of every document below,
- the closed sets: `outcome`, `writeStatus`, `match`, `status`, `grade`, and the refusal `reason` values,
- the pairing of `outcome` with `writeStatus`, and of both with the exit code.

Not stable, and not to be parsed:

- prose fields — `explanation`, `detail`, `evidence`, `error`, a check's `detail`, and the rendered `command`,
- the order of `displays`, `checks` and `models` beyond what the text output already promises,
- anything printed on stderr.

New fields may be added to any document. A client ignores what it does not know, and treats a `reason` or a `status` it does
not know as unrecognised rather than as an error — a newer monmux may carry one.

## `switch --json`

One document on stdout for every outcome. The exit code is unchanged, and nothing is printed on stderr except the
`--unsafe-model` warning, which belongs to that flag and stays prose.

```json
{
  "outcome": "dry-run",
  "writeStatus": "none",
  "backend": "ddcutil",
  "display": { "label": "card1-DP-1", "handle": "card1-DP-1" },
  "model": "LG 38WR85QC-W",
  "input": "dp",
  "inputLabel": "DisplayPort",
  "operation": { "mechanism": "lg-alt-input", "value": 208, "valueHex": "0xD0" },
  "command": "/usr/bin/ddcutil --edid <redacted; --show-serial to print> setvcp 0xF4 0xD0 --i2c-source-addr=0x50 --noverify",
  "assumed": false
}
```

### The pair

`outcome` and `writeStatus` are always present, and the pair is fixed:

| `outcome` | `writeStatus` | Exit | What monmux knows                                                                     |
| --------- | ------------- | ---- | ------------------------------------------------------------------------------------- |
| `sent`    | `sent`        | `0`  | The command was sent. The switch itself is not independently confirmed.               |
| `dry-run` | `none`        | `0`  | Nothing was executed. This is the same promise the text output's footer makes.        |
| `refused` | `none`        | `2`  | monmux declined before the tool was reached. No DDC write was performed.              |
| `failed`  | `unknown`     | `1`  | The tool ran and failed, or the request could not be made. A write may have happened. |

`writeStatus` is what monmux knows, not what it hopes. `none` is a promise and is made only where the text output makes it
too; `unknown` withholds one.

The exit code is the authority for `unknown`. The document is the authority for telling `sent` from `dry-run`, which the exit
code cannot: both are `0`. **A client validates the pair against the exit code and treats any other combination as a protocol
error** — that is what stops a dry run ever being announced as a completed switch.

### Fields

| Field         | Present when           | Notes                                                                    |
| ------------- | ---------------------- | ------------------------------------------------------------------------ |
| `outcome`     | always                 | One of `sent`, `dry-run`, `refused`, `failed`.                           |
| `writeStatus` | always                 | One of `sent`, `none`, `unknown`, paired as above.                       |
| `assumed`     | always                 | `true` when identification was bypassed with `--unsafe-model`.           |
| `backend`     | a backend was opened   | `ddcutil` or `m1ddc`.                                                    |
| `display`     | a decision was reached | `label`, and `handle` — masked when the handle is private data.          |
| `model`       | a decision was reached | The catalog entry's full name, e.g. `LG 38WR85QC-W`.                     |
| `input`       | a decision was reached | The name as typed, e.g. `usb-c`.                                         |
| `inputLabel`  | a decision was reached | The human name monmux prints, e.g. `USB-C`.                              |
| `operation`   | a decision was reached | `mechanism`, `value`, `valueHex` — from the compiled-in catalog.         |
| `command`     | a decision was reached | The invocation, redacted unless `--show-serial`. Prose: do not parse it. |
| `refusal`     | `outcome` is `refused` | See below.                                                               |
| `error`       | `outcome` is `failed`  | The failure text. Prose: do not parse it.                                |

"A decision was reached" means the run got as far as reporting one, which is `sent` and `dry-run` and nothing else. A refusal
never carries `display`, `model`, `input` or `operation`, even the three raised after a target had been chosen —
`target-not-ready`, `invalid-operation` and `identity-changed` — because monmux discards the outcome on every failing path
rather than reporting half of one. What it saw is in `refusal.detected` and `refusal.detail` instead.

`assumed` reports what happened, not what was asked for. A run that named `--unsafe-model` but failed before the policy
assumed anything reports `false`; a bypassed write whose tool then failed reports `true`, because that is the one outcome
where an unverified value may have reached a monitor.

### `refusal`

```json
{
  "outcome": "refused",
  "writeStatus": "none",
  "backend": "ddcutil",
  "assumed": false,
  "refusal": {
    "reason": "input-not-enabled",
    "explanation": "The requested input is not enabled for this model.",
    "detail": "Requested hdmi9 on LG 38WR85QC-W; enabled inputs: dp, usb-c.",
    "detected": [
      {
        "manufacturer": "GSM",
        "productCode": 30675,
        "serialNumber": 0,
        "serialString": "",
        "modelName": "LG ULTRAWIDE"
      }
    ]
  }
}
```

`reason` is the machine-readable code, and it is the same closed set
[docs/troubleshooting.md](troubleshooting.md) documents, one section per reason. `explanation` is the sentence a human would
have been shown; `detail` adds the case-specific part. `detected` is the identities the decision was made about, redacted
unless `--show-serial`, and is an empty array when the refusal happened before anything was read.

### Scope of the guarantee

**One document on stdout for every run that reaches the `switch` handler.** That includes the argument errors the handler
itself raises — an input name that is not a connector, an `--unsafe-model` that names no catalog entry — which are rendered as
`failed` with the message in `error`.

Errors cobra raises *before* the handler — the wrong number of arguments, an unknown flag, a bad flag value — stay prose on
stderr with exit `1`, as they do for every command. A client therefore treats **exit 1 with nothing parsable on stdout** as a
failure whose error text is stderr, and never as a refusal. Rendering those at the root instead was considered and rejected:
`--json` has not been parsed when they occur, so nothing there can know it was asked for.

## `info --json`

The same report `monmux info` prints, and the same one `monmux doctor` prints the `checks` half of. It writes nothing.

```json
{
  "backend": "ddcutil",
  "displays": [
    {
      "label": "card1-DP-1",
      "handle": "card1-DP-1",
      "handlePrivate": false,
      "writable": true,
      "status": "ok",
      "identity": {
        "manufacturer": "GSM",
        "productCode": 30675,
        "serialNumber": 0,
        "serialString": "",
        "modelName": "LG ULTRAWIDE"
      },
      "match": "exact",
      "model": "LG 38WR85QC-W",
      "enabledInputs": ["dp", "usb-c"]
    }
  ],
  "checks": [{ "name": "ddcutil binary", "ok": true, "detail": "/usr/bin/ddcutil" }]
}
```

| Field                      | Notes                                                                                            |
| -------------------------- | ------------------------------------------------------------------------------------------------ |
| `displays[].label`         | How the display is named in every message.                                                       |
| `displays[].handle`        | The backend's address. Masked unless `--show-serial` when `handlePrivate` is `true`.             |
| `displays[].writable`      | Whether the backend could write to it at all — before any catalog question.                      |
| `displays[].status`        | `ok`, `no-ddc-channel`, `edid-unreadable`, `no-uuid`.                                            |
| `displays[].identity`      | The parsed EDID identity. `serialNumber` and `serialString` are redacted unless `--show-serial`. |
| `displays[].match`         | `exact`, `none` or `ambiguous`.                                                                  |
| `displays[].model`         | The matched entry's full name, empty when `match` is not `exact`.                                |
| `displays[].enabledInputs` | Input **names**, and empty for any display monmux will not write to.                             |
| `checks[]`                 | The backend's diagnostics: `name`, `ok`, `detail`.                                               |

`enabledInputs` is a list of names, not of labels, and stays that way. A client that wants the human label joins on the
catalog entry named by `model`, where every enabled input is recorded by construction — see below. `switch --json` carries
`inputLabel` for the same reason.

An `info` run that produced a report and then hit a failure still prints the report, and still exits `1`: the report is the
diagnostic for exactly those failures. Read the exit code before trusting the document to be complete.

## `catalog list --json` and `catalog show --json`

The catalog compiled into this binary. No backend, no configuration file, no monitor.

```json
{
  "models": [
    {
      "vendor": "LG",
      "name": "38WR85QC-W",
      "fullName": "LG 38WR85QC-W",
      "writeEnabled": true,
      "identities": ["GSM/0x77D3", "GSM/0x77D4"],
      "mechanisms": ["lg-alt-input"],
      "inputs": [
        {
          "name": "dp",
          "label": "DisplayPort",
          "mechanism": "lg-alt-input",
          "value": 208,
          "valueHex": "0xD0",
          "grade": "verified",
          "evidence": "Direct test on the unit, …"
        }
      ],
      "notes": [],
      "sources": []
    }
  ],
  "count": 71,
  "writeEnabled": 1
}
```

`catalog show --json` prints one of those `models` entries as the whole document. `count` and `writeEnabled` are the number of
entries listed, after any filter, and how many of them monmux may write to.

`--verbose` changes nothing here: the JSON always carries `value`, `valueHex`, `grade` and `evidence`, which is what that flag
adds to the text listing.

`inputs[]` is what the entry **records**, which is not the same as what monmux will write: only a `writeEnabled` entry can be
switched, and only to an input it records. `label` is the human name monmux itself prints for that input, so a client can show
what monmux shows without inventing a spelling of its own. `grade` is the evidence grade, and `evidence` is the prose behind
it.

## `version --json`

```json
{ "version": "v0.6.0", "commit": "9f1c2d3e4b5a6c7d8e9f0a1b2c3d4e5f60718293", "date": "2026-09-10T12:34:56Z" }
```

Every field is whatever the build put there, and there are three cases:

| Build                      | `version`                                                                   | `commit`      | `date`                      |
| -------------------------- | --------------------------------------------------------------------------- | ------------- | --------------------------- |
| A release                  | the tag verbatim, **leading `v` kept**, e.g. `v0.6.0`                       | 40 hex digits | RFC 3339, the commit's date |
| `make go-build`            | `git describe --tags --dirty --always`, e.g. `v0.5.0-4-gabc123def456-dirty` | 12 hex digits | RFC 3339, the build's time  |
| `go build` with no ldflags | `dev`                                                                       | `none`        | `unknown`                   |

A client that gates on the version parses it accordingly: the leading `v` is part of the released string, and neither a
`git describe` string nor `dev` is a plain semantic version. Treating an unparsable `version` as acceptable is the right
default — a binary that answers `version --json` at all has this contract by construction — while a parsable one below the
minimum is not.

A monmux old enough not to know `--json` answers this with cobra's `unknown flag` on stderr and exit `1`, which is how a
client detects a version below the one it needs without having to parse anything.

## See also

- [docs/troubleshooting.md](troubleshooting.md) — every refusal reason in prose, one section per `refusal.reason`.
- [docs/configuration.md](configuration.md) — the flags, and which wins.
- [README.md](../README.md) — the exit codes, and what a switch claims.
