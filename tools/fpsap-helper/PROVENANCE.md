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

**396 KB**, against the ~1.07 MB of interpreter and snapshot it replaces. The
built helper binary also shrinks, 2.96 MB to 2.49 MB.

An earlier revision of this PR was 8.1 MB and argued the trade was "provenance
for bulk" — you carry more code, but none of it is Apple's. That framing is
obsolete. The bridge was machine-generated straight-line code then, produced by
partial evaluation of an execution trace; it has since been reduced to a closed
form. `internal/fpsapcore` is now **1,000 lines of ordinary byte arithmetic**
across ten files:

| file | lines | what it is |
|---|---|---|
| `fairplay_sap.go` | 243 | ring diffusion, nonlinear circuit, fold, final scramble |
| `message_encrypt.go` | 135 | forward-AES encryption of the m3 message body |
| `fairplay_md5.go` | 125 | the MD5-family compression and its mutations |
| `ring_swar.go` | 109 | the ring loop in SWAR form |
| `scramble.go` | 88 | the scramble collapsed to a GF(2) matrix |
| `fairplay_md5_unrolled.go` | 73 | the round loop unrolled by four |
| `descriptor.go` | 64 | the 5-block descriptor |
| `fast.go`, `ring.go`, `bridge.go` | 163 | prefix folding, index tabulation, `gp ^ 0x0f` and the word swap |

So there is no longer a size argument against this change. It is smaller than
what it replaces, and the parts that are not constant tables can be read.

## Two correctness fixes since this PR was opened

Both were found in the upstream research tree and are folded in here.

**It answered mode 3 to every m2.** An m2 selects a FairPlay message mode in
byte 13, and the mode picks both the CBC IV and the AES round keys for the
message body — so the same 128-byte challenge produces four entirely different
responses under modes 0..3. The helper read bytes 14:142 straight out of the m2
and never looked at byte 13, so a mode-0 challenge got a mode-3 answer: wrong
bytes returned confidently, which is worse than an error. Phase 1's tables bake
mode 3's key schedule and no parameter could select another, so the fix is to
say so. `main.go` now parses instead of slicing, and any other mode exits 1 with
a message naming the mode it got. `TestModeIdentityAgainstDoubletake` pins which
mode we are against omarroth/doubletake, which implements all four.

**The m3 framing replayed one captured session.** `FPSAPExchangeM3`'s 144-byte
prefix is a constant whose body encodes a local SAP captured from a single
emulator snapshot. Real senders generate that per session, and receivers that
check the body reject the replay — doubletake#17 is an AppleTV3,2 answering
`RTSP/1.0 466 Key Management Error`. `fpemu.NewFPSAPSession(rand.Reader)`
generates its own local SAP, encrypts it into the m3 body and folds it into the
response. It is checked two ways: driven with the frozen local SAP it reproduces
the captured 164-byte m3 for all 142 golden vectors, and given a fresh one it
matches doubletake byte for byte across the whole frame. The frozen function
stays, because the golden vectors pin it and `crates/rotten-crypto` does its own
framing.

Neither fix changes the 20-byte response for a well-formed mode-3 m2, which is
what `main.go` emits — the 142 golden vectors and the 8 external vectors are
unchanged.

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
