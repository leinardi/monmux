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
- `identity.modelName` — useful context; **not** used for matching.
- `writable` and `status` — a display with `no-ddc-channel`, `edid-unreadable` or `no-uuid` cannot be switched at all, and that
  is a hardware or permissions problem to solve before anything else.

Monitors often report a **different product code depending on which input they are currently displaying**. The model in the
catalog today has two identities for exactly that reason. Check every input you can reach, or the entry you add will work in one
direction and refuse in the other.

Nothing in this step writes. `monmux info` and `monmux doctor` are read-only by construction: monmux never probes by sending a
value and seeing what happens.

## 2. Find the values, from evidence rather than experiment

You need, for each input you want to enable: which mechanism switches it, and which byte means that input.

Do **not** find out by trying values. A manufacturer-specific register that means "select USB-C" on one model can mean something
else entirely on another, and there is at least one report of a monitor left permanently unusable by an unexpected write.
Acceptable sources, in order of strength:

1. The manufacturer's own documentation for that model.
2. A project like `ddcutil`'s wiki, or another tool's source, that names the model.
3. A report from somebody with the same model, ideally with the exact command they ran.

Whatever you find, it is a *candidate* until you have tested it yourself on that model. A value reported for a different model is
not evidence for this one — that is why the catalog carries evidence per input rather than per model.

## 3. Choose a mechanism

`catalog.Mechanism` is a closed enum. Today it has one value, `lg-alt-input`: the LG side channel, source address `0x50`, VCP
`0xF4`, no verification.

If your monitor switches through a different mechanism — the standard Input Source feature `VCP 0x60`, say — that mechanism does
not exist yet, and adding it is part of your change. Two rules:

- A mechanism is added **together with the first model that needs it**, never speculatively.
- A mechanism is a per-model property and never a fallback. A backend that does not implement a model's mechanism refuses with
  `invalid-operation` rather than trying another one. In particular, monmux does not write `VCP 0x60` because a monitor happens
  to read it.

Implementing a new mechanism means: a value in the enum, the planner code in each backend that supports it, `ValidateOperation`
accepting it there, and a golden test pinning the exact arguments it produces.

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

`internal/catalog/models.go` is Go source, deliberately: the bytes that reach a monitor are reviewable in a diff, and there is no
parser between the file and the write.

```go
{
    Name:   "MODEL-NAME",
    Vendor: "VENDOR",
    Identities: []Identity{
        {Manufacturer: "ABC", ProductCode: 0x1234},
    },
    WriteEnabled: true,
    Inputs: map[Input]inputOp{
        InputDP: {
            mechanism: MechanismLGAltInput,
            value:     0xD0,
            evidence:  "Direct test on the unit, 2026-01-01, Linux (ddcutil): switched from USB-C to DisplayPort",
        },
    },
    Sources: []string{"…"},
},
```

Rules the invariant tests enforce:

- A write-enabled model has at least one identity. Without one it can never match, and an entry that can never match must not
  claim to be enabled.
- An identity belongs to exactly one model, across the whole catalog.
- Every recorded input has non-empty evidence.
- Every recorded input uses a mechanism some backend implements.

## 6. What stays disabled

Leave `WriteEnabled: false` — or leave an input out entirely — when:

- You have values but no unit to test them on. Record them with a source; the entry documents what is known and monmux still
  refuses. The catalog has one entry like this today.
- You tested one input and not another. Record the tested one, leave the other out. Absent is safer than disabled-but-present,
  because it cannot be flipped on by a one-character edit.
- The evidence is "it worked for someone with a similar model". That is not evidence for this model.

A disabled entry is not a lesser contribution. It is the thing that stops the next person guessing.

## 7. Update the documentation and the fixtures

- [compatibility.md](compatibility.md) is checked against the catalog by a test, so add your rows there in the same commit.
- If you add a fixture — a captured EDID, or captured tool output — **sanitize it first**. Replace every serial and UUID with the
  synthetic values in `internal/backend/testdata/README.md`, recompute the EDID checksum, and check that the allowlist test
  passes. It fails the build if any other identifier appears anywhere under a `testdata` directory. Never commit a real serial.

## Pull request checklist

- [ ] `monmux info --json` output for each connector, with serials redacted (the default), included in the description.
- [ ] Evidence recorded per input: date, OS, backend, command, and what the monitor did — in both directions.
- [ ] Every value you tested yourself; nothing enabled on somebody else's say-so.
- [ ] Catalog entry added, with `Sources` naming where the values came from.
- [ ] `compatibility.md` updated to match.
- [ ] Any new fixture sanitized, and `make go-test` passing.
- [ ] `make check` clean.
- [ ] If you added a mechanism: the enum value, both backends' handling of it, and a golden test for the exact arguments.
