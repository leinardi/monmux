# Safe Cross-Platform LG Monitor Input Switching

## Goal, verified findings, compatibility data, and safety requirements

**Status:** 2026-09-07
**Initial target platforms:** Linux and macOS

## 1. Goal

The goal is to provide a small command-line utility that lets a user switch a supported LG monitor between known video inputs, for example:

- DisplayPort
- USB-C
- HDMI 1
- HDMI 2

The utility should be convenient enough for use from a terminal, keyboard shortcut, launcher, or another automation, while being deliberately conservative about monitor writes.

The primary design objective is **safety over broad compatibility**:

> The utility must not send a monitor-control write unless it can positively identify the attached monitor as a specifically supported model and the requested input is explicitly known to be valid for that model.

The utility should be cross-platform from the user's perspective. The user should request a symbolic input such as `dp` or `usb-c`; platform-specific monitor-control details should remain an internal concern.

---

## 2. Why a model-specific approach is necessary

Recent LG monitors can use a non-standard mechanism for input switching rather than the standard DDC/CI Input Source feature (`VCP 0x60`).

The `ddcutil` project documents recent LG monitors that use:

- DDC/CI source address `0x50`
- manufacturer-specific VCP code `0xF4`
- a model-dependent input value

`ddcutil` calls this an LG service/factory/manufacturer side channel ("DDC2AB"). It documents `0x50` as the input-switching side channel and `0xF4` as the input-switch command. [S1]

This behavior is independently implemented or documented by several open-source projects:

- `m1ddc` on macOS provides `input-alt` specifically for the alternate input mechanism used by some LG displays. [S3]
- `lg-input-switch` for NVIDIA/Windows constructs LG input-switching packets using source address `0x50`, VCP `0xF4`, and a normal DDC/CI destination address. [S5]
- `Jason7536/lg-input-switch` documents recent LG input switching as private VCP `0xF4` over source address `0x50`, instead of standard VCP `0x60`. [S6]
- `LGInputSwitch` for AMD/Windows uses direct DDC/CI/I²C access for LG input switching and explicitly warns that malformed commands can cause display misbehavior. [S7]

### Important clarification about `0x50`

In this LG input-switching mechanism, `0x50` is the **DDC/CI source-address byte used by the command**, not a request to overwrite the monitor's EDID EEPROM.

For example, the NVIDIA implementation defines:

- DDC/CI destination: `0x6E` (the 8-bit representation corresponding to DDC address `0x37`)
- LG input source address: `0x50`
- input VCP: `0xF4`

and constructs the SetVCP packet accordingly. [S5]

This distinction matters because an I²C device address of `0x50` is also commonly associated with EDID storage, but that is not what `ddcutil --i2c-source-addr=0x50` means in this context.

---

## 3. Input values are not universal across LG monitors

Input values must be treated as **model-specific data**.

The `ddcutil` LG compatibility page contains different confirmed input values for different LG monitor models. For example, some models use `0xD1` for USB-C, while others use values such as `0xE0` or `0xC0`. [S1]

Likewise, `m1ddc` documents only *common* alternate-input values:

| Common `m1ddc` value | Hex | Generic label in `m1ddc` |
| ---: | ---: | --- |
| 208 | `0xD0` | DisplayPort 1 |
| 209 | `0xD1` | DisplayPort 2 |
| 144 | `0x90` | HDMI 1 |
| 145 | `0x91` | HDMI 2 |
| 210 | `0xD2` | USB-C / DP 3 |

`m1ddc` describes these as common values, not a guarantee for every LG model. [S3]

This is important because the tested **LG 38WR85QC-W** uses `0xD1` / decimal `209` for USB-C, not the generic `m1ddc` USB-C example `0xD2` / decimal `210`.

Therefore:

> Generic LG input tables must never be used as a fallback for an unknown monitor.

Only values explicitly verified for a particular model may be write-enabled.

---

## 4. Verified compatibility data

### 4.1 LG 38WR85QC-W — directly verified on a real unit

The following data was observed directly during read-only identification and controlled switching tests on an LG 38WR85QC-W.

#### Identification over DisplayPort on Linux

`ddcutil detect` reported:

- EDID manufacturer ID: `GSM` (LG Electronics)
- model string: `LG ULTRAWIDE`
- EDID product code: `30675` / `0x77D3`
- VCP version: `2.1`

Read-only checks succeeded:

- brightness (`VCP 0x10`) returned a valid value
- standard Input Source (`VCP 0x60`) returned `DisplayPort-1`, value `0x0F`

#### Identification over USB-C on macOS

`m1ddc display list detailed` reported:

- product name: `LG ULTRAWIDE`
- manufacturer: `GSM`
- model/product value: `30676` / `0x77D4`

The alphanumeric and binary serial values matched those seen from Linux, confirming that these two EDID identities belonged to the same physical monitor.

Read-only brightness also returned the same value seen on Linux.

#### Important EDID observation

The same physical 38WR85QC-W exposed different EDID product IDs depending on the connection used during the test:

| Connection | Manufacturer | Product ID |
| --- | --- | ---: |
| DisplayPort | `GSM` | `0x77D3` |
| USB-C | `GSM` | `0x77D4` |

This is direct observation from the tested unit.

Therefore, a supported-model definition must be able to contain **multiple accepted EDID identities for one monitor model**.

The generic model string `LG ULTRAWIDE` is not specific enough to identify the model safely by itself.

#### Confirmed input-switch values on the tested 38WR85QC-W

Both directions were successfully tested on the same physical monitor:

| Requested input | Hex value | Decimal | Result |
| --- | ---: | ---: | --- |
| DisplayPort | `0xD0` | `208` | Successfully switched from USB-C to DisplayPort |
| USB-C | `0xD1` | `209` | Successfully switched from DisplayPort to USB-C |

These two mappings are therefore **directly verified for the tested 38WR85QC-W**.

HDMI input values have **not** been tested on this unit and should not currently be enabled for this model solely by analogy with another LG monitor.

---

### 4.2 LG 38BR85QC — externally confirmed mapping

The `ddcutil` LG compatibility page lists the **38BR85QC** as confirmed by a tester with these values: [S1]

| Input | Hex value | Decimal |
| --- | ---: | ---: |
| HDMI 1 | `0x90` | `144` |
| HDMI 2 | `0x91` | `145` |
| DisplayPort | `0xD0` | `208` |
| USB-C | `0xD1` | `209` |

This establishes a confirmed input mapping for the 38BR85QC.

However, no trusted EDID manufacturer/product-code fingerprint for the 38BR85QC has yet been collected as part of this work.

Therefore the model can be present in the **compatibility knowledge base**, but it should **not yet be automatically write-enabled** by a strict implementation. It needs an exact, read-only identification fingerprint before a program can safely distinguish it from other monitors.

---

## 5. Linux validation

The Linux reference tool used during validation was `ddcutil`.

Read-only operations successfully performed on the tested 38WR85QC-W included:

- display detection
- reading brightness (`VCP 0x10`)
- reading standard Input Source (`VCP 0x60`)

The successful LG alternate input-switch mechanism used:

- VCP: `0xF4`
- DDC/CI source address: `0x50`
- `0xD1` to switch to USB-C

The switch from Linux/DisplayPort to macOS/USB-C succeeded.

The `ddcutil` LG documentation provides the same mechanism and an example using:

- `setvcp 0xF4`
- `--i2c-source-addr=0x50`
- `--noverify` [S1]

### Meaning of `--noverify`

`ddcutil` normally verifies a SetVCP operation by reading the value back. Its documentation explains that a SetVCP command receives an I²C-level acknowledgement that the request packet was received, but not an acknowledgement that the display actually executed it. It also notes that verification does not occur for write-only features and can fail for input switching. [S2]

The LG compatibility page specifically states that `--noverify` appears necessary on this side channel because the switch command does not report back whether the channel change succeeded. [S1]

Therefore, a successful backend return means that the write operation was sent without an I²C-level failure; it should **not** be described as cryptographic or protocol-level proof that the display visibly switched.

---

## 6. macOS validation

The macOS reference tool used during validation was `m1ddc`.

`m1ddc` is an open-source DDC/CI utility for **Apple Silicon Macs**. It supports external displays over supported USB-C/DisplayPort Alt Mode and HDMI paths and provides an `input-alt` operation for the alternate VCP mechanism used by some LG displays. [S3]

Read-only tests on the tested 38WR85QC-W succeeded for:

- display enumeration
- detailed display identification
- brightness

### Current-input reads are not reliable enough for switching logic

While the tested monitor was visibly using USB-C:

- `m1ddc get input` returned `15` (`0x0F`, DisplayPort)
- `m1ddc get input-alt` returned `-7`

An existing `m1ddc` issue reports the same behavior on another setup:

- `get input` always returns `15`
- `get input-alt` returns `-7`
- `set input-alt` still switches inputs successfully [S4]

Therefore, the utility must **not depend on `m1ddc` current-input reads to decide what input is active**.

The macOS write that switched the tested 38WR85QC-W from USB-C back to DisplayPort used:

- alternate input value `208`
- `208 == 0xD0`

The switch succeeded.

Together with the successful Linux switch to `0xD1`, this directly validates `0xD0` / `0xD1` as DisplayPort / USB-C on the tested 38WR85QC-W.

---

## 7. Relevant open-source projects

These projects independently support or corroborate the LG alternate input-switching mechanism.

### `ddcutil`

Linux DDC/CI utility and the primary source documenting LG's `0x50` / `0xF4` side channel and model-specific test results. [S1]

### `m1ddc`

macOS DDC/CI utility for Apple Silicon. It exposes `input-alt` for the alternate input mechanism used by some LG monitors. [S3]

### `meer-cha/lg-input-switch`

Windows/NVIDIA utility. Its source defines:

- `INPUT_SOURCE_ADDR = 0x50`
- `INPUT_VCP_CODE = 0xF4`
- `DDC_DEVICE_ADDR = 0x6E`

and constructs the corresponding raw DDC/CI packet. [S5]

### `Jason7536/lg-input-switch`

Windows/Intel utility. It explicitly describes recent LG input switching as private VCP `0xF4` over source address `0x50` and describes itself as the Intel/Windows equivalent of the relevant `ddcutil --i2c-source-addr=0x50` mechanism. [S6]

### `phillip9933/LGInputSwitch`

Windows/AMD utility for LG input switching over direct DDC/CI/I²C access. Its documentation warns that malformed commands can cause display misbehavior. [S7]

These projects are useful corroboration, but their generic input values must not override model-specific confirmed mappings.

---

## 8. Known risk evidence

A `ddcutil` issue reports an LG **38WN95CP-W** becoming unusable with a black screen and no OSD after the user says they were trying to switch inputs with `ddcutil`. [S8]

Important limits of that report:

- it concerns a different LG model
- the issue does not provide the exact write command that caused the failure
- the issue is labeled around the standard Input Source feature (`VCP 0x60`)
- it therefore does **not** establish that the LG `0x50` / `0xF4` mechanism caused the failure

The report is still valid evidence that monitor-control writes should be treated conservatively.

This, combined with documented model-to-model variation in LG input values, is the main reason for the strict whitelist and fail-closed design.

---

## 9. Required safeguards

### 9.1 Fail closed

The default behavior must be:

> **If there is any uncertainty, perform no write.**

Examples that must result in refusal:

- unknown monitor
- incomplete identification
- ambiguous match
- unsupported platform/backend
- requested input not explicitly enabled for the matched model
- more than one candidate monitor when the intended target cannot be determined safely
- missing required monitor-control capability

A refusal should clearly state that **no DDC write was performed**.

---

### 9.2 Exact supported-model whitelist

Writes must only be allowed for models in a built-in or otherwise trusted whitelist.

A model entry should contain enough read-only identification data to distinguish it safely, such as:

- EDID manufacturer ID
- one or more exact EDID product IDs
- any other stable model-identification fields that have been explicitly verified

A human-readable model string alone must not be accepted when it is generic.

For the tested 38WR85QC-W, the current known identity set is:

- `GSM` + `0x77D3`
- `GSM` + `0x77D4`

Both correspond to the same tested physical 38WR85QC-W over different connections.

---

### 9.3 Multiple identities per model

The whitelist format must support more than one accepted identity for a single model.

This is required because the tested 38WR85QC-W exposed:

- `0x77D3` over DisplayPort
- `0x77D4` over USB-C

The utility must not assume that one physical monitor model always exposes one EDID product ID on every input.

---

### 9.4 Explicit per-model input whitelist

Each supported model must have its own exact list of permitted input names and values.

For example, the currently verified 38WR85QC-W entry should enable only:

- `dp` -> `0xD0`
- `usb-c` -> `0xD1`

until other inputs are independently verified for that model.

The utility must never fall back to:

- generic LG values
- `m1ddc`'s common values
- values from a similar model
- guessed sequential values
- values discovered by brute force

---

### 9.5 Symbolic user-facing inputs only

Normal usage should accept symbolic targets such as:

- `dp`
- `usb-c`
- `hdmi1`
- `hdmi2`

The user-facing switching interface should **not** expose an arbitrary raw VCP/value mechanism.

For example, normal usage should not allow a user to provide an unchecked value such as `0xD2`, `0xFF`, or an arbitrary VCP feature code.

This prevents accidental use of unsupported manufacturer-specific commands.

---

### 9.6 No probing writes

Monitor discovery and compatibility checks must be read-only.

The utility must never determine compatibility by:

- trying different input values
- scanning manufacturer-specific VCP codes
- brute-forcing source addresses
- sending a write and seeing what happens

Unknown hardware must remain unknown until its compatibility is established through external evidence and controlled testing.

---

### 9.7 Do not use standard VCP `0x60` writes as a fallback

Reading `VCP 0x60` can be useful for diagnostics and worked correctly while the tested monitor was on DisplayPort.

However, the utility should not fall back to writing standard Input Source `VCP 0x60` on these LG models merely because the feature is readable.

The `ddcutil` LG documentation states that recent LG monitors covered by its workaround no longer switch inputs through standard `setvcp` methods and instead use the alternate `0x50` / `0xF4` channel. [S1]

A separate bricking report involving input switching on another LG model is also associated with `VCP 0x60`, although its exact triggering command is unknown. [S8]

---

### 9.8 Do not rely on reading the current input

The tested macOS setup showed that current-input reads can be stale or meaningless while alternate input switching still works.

An upstream `m1ddc` issue reports the same pattern. [S4]

Therefore:

- explicit commands such as "switch to DisplayPort" and "switch to USB-C" are safe design primitives
- switching behavior must not depend on `get input` or `get input-alt` being accurate
- an automatic toggle feature should not infer its next target from these unreliable reads

A future toggle feature would need a different, explicitly safe design.

---

### 9.9 Resolve exactly one target monitor

A write must target one uniquely identified supported display.

Numeric enumeration such as "display 1" may be useful to communicate with a backend, but it must not itself be treated as proof of model identity.

Before writing, the utility should first identify the monitor using read-only data, match it to exactly one whitelist entry, and only then address the corresponding backend display.

If resolution is ambiguous, refuse the write.

---

### 9.10 Optional physical-monitor pinning

For users who want stronger protection, the utility may optionally support pinning operation to one physical monitor serial.

When enabled:

- the model must match the supported whitelist
- the configured serial must also match
- otherwise no write occurs

This should be an optional extra restriction, not the mechanism used to define general model compatibility.

---

### 9.11 No application-level write loops or brute-force retries

The utility should request only the intended, known operation.

It should not repeatedly resend a write because the picture did not change or because the current-input read does not confirm it.

Backend libraries may have their own transport-level retry behavior; the utility should not add uncontrolled higher-level write loops on top.

---

### 9.12 Accurate success reporting

Because the LG side-channel does not provide a normal read-back confirmation, the utility should distinguish between:

- **command sent successfully**
- **input switch independently confirmed**

A successful DDC/I²C call should not be described as proof that the monitor executed the switch unless there is a safe and reliable confirmation mechanism.

---

### 9.13 Read-only diagnostic mode

The utility should provide a read-only diagnostic or information operation that can report, without writing:

- detected external monitors
- identity fields used for matching
- whether each monitor is supported
- which whitelist entry matched
- which symbolic input targets are enabled
- which platform/backend is available

This is useful both for users and for safely collecting data for additional models.

---

### 9.14 Clear refusal messages

When a write is refused, the utility should explain why, for example:

```text
Refusing to switch input.

Detected monitor:
  Manufacturer: GSM
  Product ID:   0x1234

No supported-model whitelist entry matches this identity.
No DDC write was performed.
```

The key safety property is that unsupported hardware fails visibly and harmlessly.

---

## 10. Adding support for another monitor model

A new model should not become write-enabled simply because it is an LG monitor or belongs to a similar product family.

Before enabling writes for a new model, the project should have:

1. **Exact read-only identification data**
   - enough to distinguish the model from other displays
   - including all known connection-specific EDID variants that need support

2. **Explicit input mappings**
   - each enabled input value must be supported by direct testing or a trustworthy model-specific source

3. **Evidence recorded**
   - model name
   - identification fingerprint(s)
   - verified input/value pairs
   - platform/path used for testing
   - source URL or reproducible validation notes

4. **No inferred inputs**
   - an untested HDMI value should remain disabled even if it is common on other LG monitors

5. **Fail-closed behavior for incomplete entries**
   - a model whose input mapping is known but whose identity cannot yet be recognized safely should remain non-write-enabled

This is currently the situation for the 38BR85QC: its input mapping is externally confirmed, but exact EDID identification data still needs to be collected before strict automatic support can be enabled.

---

## 11. Current compatibility status

| Monitor | Identification data | DP | USB-C | HDMI 1 | HDMI 2 | Safe for automatic writes now? |
| --- | --- | ---: | ---: | ---: | ---: | --- |
| LG 38WR85QC-W | Directly observed: `GSM/0x77D3`, `GSM/0x77D4` | `0xD0` verified | `0xD1` verified | Not verified for this model | Not verified for this model | **Yes, for DP and USB-C only** |
| LG 38BR85QC | Exact EDID fingerprint not yet collected | `0xD0` externally confirmed | `0xD1` externally confirmed | `0x90` externally confirmed | `0x91` externally confirmed | **No; identification data still required** |

---

## 12. Reference behavior for the initial use case

For a recognized 38WR85QC-W, the intended user experience is conceptually:

```text
switch-input dp
switch-input usb-c
```

The internal mapping for this model is fixed:

```text
dp     -> 0xD0 / 208
usb-c  -> 0xD1 / 209
```

Before either write:

1. detect the monitor using read-only operations
2. resolve exactly one target
3. match the exact monitor identity against the whitelist
4. confirm the requested symbolic input is enabled for that model
5. send only the corresponding known input-switch operation
6. never substitute, guess, scan, or probe another value

If any step fails, perform no write.

---

## 13. Sources

**[S1] ddcutil — Switching input source on LG monitors**
Documents the LG `0x50` side channel, `0xF4` input command, model-specific values, and confirmed 38BR85QC mapping.
<https://github.com/rockowitz/ddcutil/wiki/Switching-input-source-on-LG-monitors>

**[S2] ddcutil — `setvcp` documentation**
Explains SetVCP acknowledgement and verification behavior, including `--noverify`.
<https://www.ddcutil.com/command_setvcp/>

**[S3] m1ddc**
Open-source Apple Silicon macOS DDC/CI utility; documents `input-alt` and common alternate LG values.
<https://github.com/waydabber/m1ddc>

**[S4] m1ddc issue #49 — current input reads**
Reports `get input` returning `15` and `get input-alt` returning `-7` while `set input-alt` still works.
<https://github.com/waydabber/m1ddc/issues/49>

**[S5] meer-cha/lg-input-switch — NVIDIA implementation**
Source shows LG input source address `0x50`, VCP `0xF4`, DDC destination `0x6E`, and raw packet construction.
<https://github.com/meer-cha/lg-input-switch/blob/main/lg_switch.py>

**[S6] Jason7536/lg-input-switch — Intel implementation**
Documents recent LG input switching as private VCP `0xF4` over source address `0x50`.
<https://github.com/Jason7536/lg-input-switch>

**[S7] phillip9933/LGInputSwitch — AMD implementation**
Uses direct DDC/CI/I²C input switching and includes an explicit warning regarding malformed commands.
<https://github.com/phillip9933/LGInputSwitch>

**[S8] ddcutil issue #419 — reported LG monitor failure after input-switch experimentation**
Useful as risk evidence; the exact write that caused the reported failure is not shown.
<https://github.com/rockowitz/ddcutil/issues/419>

---

## Evidence policy for this document

This document intentionally distinguishes:

- **externally documented facts**, with source URLs
- **direct observations from the tested LG 38WR85QC-W unit**
- **safety requirements**, which are conservative design decisions derived from those facts

It intentionally does **not** claim:

- that all LG monitors use the same values
- that 38BR85QC and 38WR85QC have identical firmware
- that untested HDMI mappings apply to the 38WR85QC-W
- that a successful DDC write guarantees the visible input changed
- that the `0x50` / `0xF4` method is risk-free

The compatibility list should grow only from verified model-specific evidence.
