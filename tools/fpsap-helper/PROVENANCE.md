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

## Licensing

- `internal/fpbridge`, `internal/fairplayhash` — independent reverse
  engineering, **Blue Oak Model License 1.0.0** (permissive).
- `internal/fpsapcore` — derived from
  [omarroth/doubletake](https://github.com/omarroth/doubletake) at `8ccea5f`,
  **LGPL-3.0**. See `internal/fpsapcore/NOTICE.md`.

Both flow into this directory's existing GPL-3.0 without friction, so keep the
`LICENSE` and the subprocess isolation as they are.

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

Against the interpreter this replaces, both binaries driven end to end:

- **4,923 inputs, 0 mismatches** — every single-byte position across the payload,
  one bit set in each 16-byte block, all-ones with a byte cleared, solid fills,
  counter ramps, 4,500 random payloads and 200 full 142-byte m2 messages. The
  concatenated outputs hash identically.
- **13/13 error and edge cases behave identically** — empty input, odd-length
  hex, non-hex, off-by-one lengths, trailing newline, uppercase hex, oversized
  input. Same exit codes, same stderr behaviour.

Independently of the interpreter:

- 142/142 archived golden vectors, both payload→hash and m2→m3
- 8/8 vectors published by `nored/airfry` and `omarroth/doubletake`, which
  compute this exchange by emulating Apple's binary
- **15.2 million fuzz executions** across `FPExchangeBlobless` and
  `FPSAPExchangeM3`, no crashes
- builds for linux/amd64, linux/arm64, linux/386, windows/amd64, windows/386 and
  darwin/arm64; `go vet` clean on 32- and 64-bit

Internal consistency, in the repository's existing habit of keeping the slow path
and testing the fast one against it:

- the fast descriptor against the reference, 3,000 random bodies
- the collapsed scramble against the reference, 50,000 random inputs
- the ring loop's tabulated and counter-driven indices against the naive
  derivation, with a control asserting the `uint32` wrap really is irregular
  (it differs from the obvious form on all 155 affected steps)

The differential harness was checked against a deliberate fault: perturbing one
mask in the bridge makes 52 of 52 sampled inputs disagree.
