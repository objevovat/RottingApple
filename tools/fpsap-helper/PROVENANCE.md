# Provenance of the FairPlay SAP implementation

## What is in this directory

`fpemu` exposes the same two functions it always did. Everything under
`internal/` is a native computation of the FairPlay SAP exchange:

| stage | package | what it does |
|---|---|---|
| Phase 1 | `internal/fpbridge` | white-box AES over the 128-byte payload → a 128-byte GP buffer |
| Bridge | `internal/fpsapcore` | nine MD5-family blocks over that buffer → a 20-byte digest |
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

**No Apple addresses either.** None survive as folded constants, as data spilled
into scratch memory, or as pointers inside the tables.

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
- the Phase-2 network's constant tables in `internal/fairplayhash`

The bridge no longer carries any table data at all — that was the part reduced
to a closed form.

So: **blob-free, not snapshot-free.** No Apple binary, no interpreter, no build
tag, no addresses — but constant tables and a small constant memory image.

## Size

**300 KB**, against the ~1.07 MB of interpreter and snapshot it replaces. The
built helper binary also shrinks, 2.96 MB to 2.55 MB.

An earlier revision of this PR was 8.1 MB and argued the trade was "provenance
for bulk" — you carry more code, but none of it is Apple's. That framing is
obsolete. The bridge was machine-generated straight-line code then, produced by
partial evaluation of an execution trace; it has since been reduced to a closed
form. `internal/fpsapcore` is now **697 lines of ordinary byte arithmetic**:

| file | lines | what it is |
|---|---|---|
| `fairplay_sap.go` | 235 | ring diffusion, nonlinear circuit, fold, final scramble |
| `fairplay_md5.go` | 120 | the MD5-family compression and its mutations |
| `scramble.go` | 85 | the scramble collapsed to a GF(2) matrix |
| `descriptor.go` | 61 | the 5-block descriptor |
| `fast.go`, `ring.go` | 94 | optional prefix folding and index tabulation |
| `bridge.go` | 38 | `gp ^ 0x0f` and the word swap |

So there is no longer a size argument against this change. It is smaller than
what it replaces, and the parts that are not constant tables can be read.

## Verification

- 142/142 archived golden vectors (`testdata/golden_vectors.csv`)
- 8/8 vectors published by two unrelated senders — `nored/airfry` and
  `omarroth/doubletake` — that compute this exchange by emulating Apple's
  binary, so agreement is independent of the reverse engineering behind this
  code
- 347/347 differential comparisons against the interpreter this replaces,
  including 300 random payloads and 40 full 142-byte m2 messages, driven end to
  end through both helper binaries

The reverse engineering behind this was done in a separate codebase that is not
currently published. Everything needed to check this code is in this directory:
the vectors, the tests, and the differential harness against the interpreter it
replaces.
