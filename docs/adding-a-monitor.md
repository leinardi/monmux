# Adding a monitor

A model does not become write-enabled because it is from the same manufacturer as one that works, or because it belongs to the
same product family, or because a value is documented somewhere. It becomes write-enabled when somebody with the monitor in
front of them switched it with that value and wrote down what happened.

This page is the procedure. It is vendor-neutral: nothing in it is specific to the one manufacturer in the catalog today.

## What you need

- The monitor, connected to a machine you can run monmux on.
- A second working input on it, so that if a switch lands somewhere unexpected you can get the picture back.
- `ddcutil` on Linux, or `m1ddc` on macOS — see [backends.md](backends.md).

## 1. Identify the monitor, read-only

```sh
monmux info --json
```

Run it once per connector you care about, with the monitor on that input. Record for each:

- `identity.manufacturer` and `identity.productCode` — the fingerprint the catalog matches on.
- `identity.modelName` — record it, on **both** operating systems if you can. It is not used for matching unless an identity
  pins it, and pinning is only for the case below; if the two systems report different strings for the same monitor, do not pin.
- `writable` and `status` — a display with `no-ddc-channel`, `edid-unreadable` or `no-uuid` cannot be switched at all, and that
  is a hardware or permissions problem to solve before anything else.

Monitors often report a **different product code depending on which input they are currently displaying**. The model in the
catalog today has two identities for exactly that reason. Check every input you can reach, or the entry you add will work in one
direction and refuse in the other.

A vendor also reuses one product code across products. LG does: `GSM/0x7707` is claimed by the 32UD99, the 27UN880-B and the
32BL95U service manual. If the code you collected is already in the catalog under another model, add `model_name:` to **both**
identities — the pinned EDID descriptor `0xFC` text is what tells them apart. The generator refuses a bare identity beside a
pinned one with the same code, because the bare one would shadow the pinned one, so this is a change to the existing entry as
well as to yours. Say so in the pull request rather than working around it.

Nothing in this step writes. `monmux info` and `monmux doctor` are read-only by construction: monmux never probes by sending a
value and seeing what happens.

## 2. Find the values, from evidence rather than experiment

You need, for each input you want to enable: which mechanism switches it, and which value means that input. A value is up to
16 bits wide, because a SetVCP carries an SH/SL pair; most are one byte.

Do **not** find out by trying values. A manufacturer-specific register that means "select USB-C" on one model can mean something
else entirely on another, and there is at least one report of a monitor left permanently unusable by an unexpected write.
Acceptable sources, in order of strength:

1. The manufacturer's own documentation for that model.
2. A project like `ddcutil`'s wiki, or another tool's source, that names the model.
3. A report from somebody with the same model, ideally with the exact command they ran.

Whatever you find, it is a *candidate* until you have tested it yourself on that model. A value reported for a different model is
not evidence for this one — that is why the catalog carries evidence per input rather than per model.

## 3. Choose a mechanism

`catalog.Mechanism` is a closed enum with two values:

- `lg-alt-input` — the LG side channel: source address `0x50`, VCP `0xF4`, no verification.
- `vcp-input-source` — the standard Input Source feature: the ordinary source address, VCP `0x60`, no verification.

If your monitor switches through neither, that mechanism does not exist yet, and adding it is part of your change. Three rules:

- A mechanism is added **together with the first model that needs it**, never speculatively. A model that only *records* it —
  disabled, with no identity — counts as that model: the point is that a mechanism enters the code with evidence attached, not
  that the first entry using it is trusted.
- A mechanism's backend path is **untested on hardware** until the first model using it is write-enabled. `vcp-input-source` is
  in that state today. Enabling that first model is therefore a first hardware run of the mechanism as well as of the model:
  work through the [testing.md](testing.md) checklist for it, and say in the pull request that it is the first, rather than
  treating it as a routine catalog flip.
- A mechanism is a per-model property and never a fallback. A backend that does not implement a model's mechanism refuses with
  `invalid-operation` rather than trying another one. In particular, monmux does not write `VCP 0x60` because a monitor happens
  to read it — only because a catalog entry names that mechanism.

Implementing a new mechanism means: a value in the enum, the name table in `internal/catalog/internal/generate` that lets the
catalog file spell it, the planner code in each backend that supports it, `ValidateOperation` accepting it there, and a golden
test pinning the exact arguments it produces.

A new connector **kind** works the same way. The kinds are a closed list in `internal/catalog/internal/input`, and a monitor
whose input is not one of them needs a Go change there — added together with the first model that needs it, never
speculatively. A new *port* of a kind that already exists is not a Go change at all: `hdmi3` or `usb-c2` is a `models.yaml`
edit and a `make go-generate`.

## 4. Test it, by hand

**You run these commands. No AI agent may run a command that writes to a monitor** — see [security.md](security.md).

For each input, with the monitor displaying something else:

```sh
monmux switch <input> --dry-run   # read the command; make sure it is what you meant
monmux switch <input>
```

Then switch back, from the other direction, and check that too. An input is evidence only for the direction you tested; test
both, and record both.

Write down: the date, the operating system, the backend, the exact command, and what the monitor did.

## 5. Add the catalog entry

The catalog is `internal/catalog/models.yaml`. Add your entry to the `models:` list:

```yaml
- name: MODEL-NAME
  vendor: VENDOR
  identities:
      - manufacturer: ABC
        product_code: 0x1234
        # model_name: ABC 4K   # only when two models share this product code
  write_enabled: true
  inputs:
      dp:
          mechanism: lg-alt-input
          value: 0xD0
          evidence:
              grade: verified
              date: "2026-01-01"
              tool: Linux (ddcutil)
              note: switched from USB-C to DisplayPort
  notes:
      - "Anything a reader needs that the table has no column for: a negative report, a conflict, an alias, a quirk."
  sources:
      - "…"
```

The evidence is a record, not a sentence. The generator composes the sentence that reaches `models_gen.go` and
[compatibility.md](compatibility.md), so every row of one grade reads the same way and nobody can talk a weak report up in
prose. The grades, and what each one needs:

| Grade        | Means                                                                         | Required fields    |
| ------------ | ----------------------------------------------------------------------------- | ------------------ |
| `verified`   | You ran it on the unit. The only grade that may be enabled.                   | `date, tool, note` |
| `documented` | The manufacturer documents it, and no field report was found.                 | `by, url`          |
| `reported`   | Somebody reports switching that named input with that value.                  | `by, tool, url`    |
| `quoted`     | A report quotes the values and says they work, without saying which it tried. | `by, tool, url`    |

`note` is optional free text for every grade but `verified`, where it says what was switched. `notes` is per-model prose and is
rendered under the model in [compatibility.md](compatibility.md); it is documentation and never an operation, so nothing written
there can reach a monitor.

Then re-render the Go the binary actually compiles, and commit both files:

```sh
make go-generate
```

`internal/catalog/models_gen.go` is generated and must never be edited by hand; `TestGeneratedCatalogMatchesTheYAML` fails the
build if it does not match the YAML. Both files land in the same diff, so a reviewer still reads the literal bytes, and the
rendering happens at development time — the binary has no catalog parser and reads no catalog file at run time.

Rules the generator refuses and the invariant tests re-check:

- A write-enabled model has at least one identity. Without one it can never match, and an entry that can never match must not
  claim to be enabled.
- A model that is not write-enabled records no identity at all. Matching does not consult the flag, so an entry with a
  fingerprint would match a real display and then refuse late instead of never matching.
- An identity belongs to exactly one model, across the whole catalog, and a manufacturer is three uppercase letters. Two
  identities collide when the manufacturer and the product code are equal and either pins no `model_name`, or both pin the
  same one — so only two identities that both pin, with different names, may share a product code.
- A pinned `model_name` is 1 to 13 characters of printable ASCII with no leading or trailing whitespace, because that is what
  an EDID descriptor can hold.
- Every recorded input is a well-formed name. A name is a connector kind — `dp`, `hdmi`, `usb-c`,
  `dvi`, `vga`, `thunderbolt` — optionally followed by a port number: a positive decimal integer with no leading zero and no
  separator, so `hdmi2` and `usb-c2` are names and `hdmi0`, `hdmi01` and `hdmi-1` are not. Write a kind bare when the model has
  one port of it and numbered when it has several; one model may not do both, so `usb-c` next to `usb-c2` is rejected. The
  human-readable label derives from the name — `hdmi3` prints as "HDMI 3" — so there is nothing else to add for a new port.
- Every recorded input uses a mechanism some backend implements.
- Every recorded input carries an evidence record whose `grade` is one of the four above, with every field that grade needs.
- A write-enabled model carries `grade: verified` on **every** one of its inputs. A value nobody ran on that unit never becomes
  writable, and that is a build failure rather than a review note.
- A `url` starts with `https://`.
- No text the documentation renders — `by`, `tool`, `date`, `url`, `note`, a `notes` entry or a source — holds a `|` or a line
  break, because the document test reads Markdown table cells and single-line bullets.
- Every model names at least one source.
- Every identity writes a `product_code`, and every input writes a `value`. Leaving one out is an error, not a zero: a value
  nobody recorded must never reach a monitor.

An unknown key is an error rather than something ignored, so a misspelled field cannot silently drop an entry, and the file
must hold exactly one YAML document, so entries cannot hide after a `---` where the generator would never read them.

## 6. What stays disabled

Leave `WriteEnabled: false` — or leave an input out entirely — when:

- You have values but no unit to test them on. Record them with a source; the entry documents what is known and monmux still
  refuses. Every disabled entry in the catalog is one of these.
- You tested one input and not another. Record the tested one, leave the other out. Absent is safer than disabled-but-present,
  because it cannot be flipped on by a one-character edit.
- The evidence is "it worked for someone with a similar model". That is not evidence for this model.
- The grade is anything but `verified`. `documented`, `reported` and `quoted` all describe somebody else's claim, and the
  generator refuses to enable a model on one. `quoted` is specifically for a report that quotes values and says they work
  without naming which inputs were tried individually — the weakest thing the catalog will record at all.

A disabled entry is not a lesser contribution. It is the thing that stops the next person guessing.

## 7. Update the documentation and the fixtures

- The table and the `### Vendor Model` sections of [compatibility.md](compatibility.md) are generated: `make go-generate`
  writes them from your `notes` and `sources`, so there are no rows to type and nothing to keep byte-identical by hand. Commit
  the regenerated file. Anything you want to say that is not in those two lists belongs outside the markers, or in the YAML.
- If you add a fixture — a captured EDID, or captured tool output — **sanitize it first**. Replace every serial and UUID with the
  synthetic values in `internal/backend/testdata/README.md`, recompute the EDID checksum, and check that the allowlist test
  passes. It fails the build if any other identifier appears anywhere under a `testdata` directory. Never commit a real serial.

## Pull request checklist

- [ ] `monmux info --json` output for each connector, with serials redacted (the default), included in the description.
- [ ] Evidence recorded per input: date, OS, backend, command, and what the monitor did — in both directions.
- [ ] Every value you tested yourself; nothing enabled on somebody else's say-so.
- [ ] Catalog entry added to `models.yaml`, with `sources` naming where the values came from.
- [ ] `make go-generate` run, and everything it rewrote — `models_gen.go` and `compatibility.md` — committed alongside the YAML.
- [ ] Any new fixture sanitized, and `make go-test` passing.
- [ ] `make check` clean.
- [ ] If you added a mechanism: the enum value, the generator's name table, both backends' handling of it, and a golden test for
      the exact arguments.
