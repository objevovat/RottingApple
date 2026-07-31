# Provenance of the FairPlay SAP implementation

## What is in this directory

`fpemu` exposes the same two functions it always did. Everything under
`internal/` is a native computation of the FairPlay SAP exchange:

| stage | package | what it does |
|---|---|---|
| Phase 1 | `internal/fpbridge` | white-box AES over the 128-byte payload → a 128-byte GP buffer |
| Bridge | `internal/layera`, `layerb`, `layerc` | nine MD5-family blocks over that buffer → a 20-byte digest |
| Phase 2 | `internal/fairplayhash` | the white-box MD5 network → the 20-byte response |

## What was removed

- `fpemu/fpexchange_interp.go` — a 2,711-line ARM64 interpreter
- `fpemu/fpexchange_data.go` — 1,008,975 bytes of Go holding a 165,575-byte
  snapshot of Apple's signed FairPlay binary, which the interpreter executed

## Licensing, stated precisely

The replacement was produced by independent reverse engineering. It is **not**
derived from `omarroth/doubletake`, and it is offered under the
**Blue Oak Model License 1.0.0**, which is permissive and flows into GPL-3.0
without friction.

This directory's `LICENSE` is left as GPL-3.0, unchanged. That is deliberate:
the directory's licence is the maintainer's call, and GPL-3.0 remains valid for
a permissively-licensed contribution. But the rationale recorded there — that
the code is "derived from doubletake" — no longer describes what is here, so it
is worth revisiting. If no GPL-derived code remains in the tree, the
subprocess-isolation arrangement that keeps the main application MIT may no
longer be necessary. That is a larger change than this one and is not attempted
here.

## What is and is not claimed

No Apple instruction is executed or reproduced. There is no interpreter, no
instruction dispatch, and no image of Apple's binary.

**No Apple addresses either.** The generated code carries none — not as folded
constants, not as data spilled into scratch memory, not as pointers inside the
baked tables. `internal/layerguard` fails the build if one returns.

That is worth spelling out because an earlier draft of this file claimed the
opposite: that "79 code-segment addresses survive as constants because the
hashed buffer really does contain stack memory holding them." **That was wrong.**
The dyld shared cache is slid on every boot, so if any part of the response
depended on the value of a code address, a device would return a different `m3`
for the same `m2` after a reboot and the handshake would fail. Apple's does not.
The addresses were inherited from the execution trace, not required by the
algorithm — confirmed by sliding the entire cache window and observing identical
output, against a control that inverts baked bytes and does change it.

What genuinely remains is **data**, as it must for white-box cryptography, where
the key is dissolved into lookup tables so the tables *are* the cipher:

- the white-box AES T-boxes in `internal/fpbridge`
- ~119 KB of constant table pages across the layer packages, four 4 KB pages per
  generated file

Whole pages are carried rather than the entries one trace happened to touch,
because some read indices are payload-dependent. A byte-level sweep puts 21 of
each 16 KB image in demonstrable use; the rest is a deliberate hedge, not proven
necessary and not proven unnecessary.

So: **blob-free, not snapshot-free.** No Apple binary, no interpreter, no build
tag, no addresses — but constant tables and a small constant memory image.

## Size

The generated layer packages are large: roughly **8.1 MB** of Go, against the
~1.07 MB of interpreter and snapshot they replace. They are machine-generated
straight-line code produced by partial evaluation of an execution trace, not a
compact algorithm — `layerc.EncodeX9` alone is ~81,000 statements, and nobody can
currently read it and explain what the encoding does.

That figure was 17 MB when this PR was opened. Three passes in the generator have
since taken it down: removing Apple's address space, re-encoding the emitted
shapes (byte-at-a-time memory access became word operations; the 32-bit
arithmetic, which arrived as nested casts at ~200,000 sites, is now named), and
eliminating register writes whose results are never read (25% of statements).
None changed a single output byte.

Reducing this to closed form remains unfinished work. The trade being offered is
still **provenance for bulk**, and it should be evaluated on those terms — the
bulk is simply a good deal smaller than it was.

## Verification

- 142/142 archived golden vectors (`testdata/golden_vectors.csv`)
- 8/8 vectors published by two unrelated senders — `nored/airfry` and
  `omarroth/doubletake` — that compute this exchange by emulating Apple's
  binary, so agreement is independent of the reverse engineering behind this
  code
- 287/287 differential comparisons against the interpreter this replaces,
  including 250 random payloads and 30 full 142-byte m2 messages, run before the
  interpreter was deleted and re-run after every size pass
- `internal/layerguard`: a digest over all eight generated functions across 512
  payloads, pinned to the value recorded before any of the size work, plus the
  no-Apple-addresses check

The reverse engineering behind this was done in a separate codebase that is not
currently published. Everything needed to check this code is in this directory:
the vectors, the tests, and the differential harness against the interpreter it
replaces.
