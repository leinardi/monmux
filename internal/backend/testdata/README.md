# Backend test fixtures

Everything in this directory is fed to a backend's parser by a test. Nothing here
is ever sent to a monitor, and no test in this repository executes `ddcutil` or
`m1ddc`: the real runner refuses to start a process from a test binary at all.

## Fixtures carry synthetic identifiers only

A fixture is a capture of real hardware output, so it starts life containing the
serial number of somebody's monitor. Before it is committed, every identifier is
replaced with one of the values below, and `TestFixturesCarryOnlySyntheticSerials`
fails if any other value appears.

| Identifier                | Value                                   |
| ------------------------- | --------------------------------------- |
| EDID numeric serial       | `0x01020304`                            |
| EDID serial string (0xFF) | `TESTSERIAL01`                          |
| m1ddc alphanumeric serial | `TESTSERIAL01`                          |
| m1ddc binary serial       | `16909060 (0x01020304)`                 |
| macOS display UUID        | `00000000-0000-4000-8000-00000000000N`  |

m1ddc prints its numeric fields twice, in decimal and in hex, and prints `(null)`
for a value the IORegistry did not supply; both forms are accepted for a serial,
and nothing else is.

The values are format-valid rather than obviously fake: a parser must accept them
exactly as it accepts the real thing, or the fixture would prove nothing.

## Sanitizing a capture

1. Capture the tool's output, or copy the connector's `edid` from sysfs.
2. Replace every serial and UUID with the value from the table above.
3. For an EDID, recompute the block-0 checksum: the 128 bytes must sum to zero
   modulo 256, or `edid.Parse` will reject the fixture.
4. Run `make go-test`. The allowlist test reads every fixture in every
   `testdata` directory in the repository, at any depth.

A fixture is committed exactly as captured. The whitespace hooks in
`.pre-commit-config.yaml` skip `testdata/`, because a capture that a formatter
has tidied no longer proves the parser handles what the tool actually prints -
`m1ddc` pads an empty field with trailing spaces and ends its no-display message
without a newline, and both of those are things the parser has to survive.

If a fixture needs an identifier the table does not cover, add it to the table
and to the allowlist in the test, in the same commit as the fixture.
