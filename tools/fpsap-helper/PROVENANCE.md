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

This changed with the closed form, and the change runs against my earlier claim
here, so it is worth being blunt about.

- `internal/fpbridge` and `internal/fairplayhash` are independent reverse
  engineering, offered under the **Blue Oak Model License 1.0.0**.
- `internal/fpsapcore` **is derived from
  [omarroth/doubletake](https://github.com/omarroth/doubletake)** at commit
  `8ccea5f`, which is **LGPL-3.0**. `fairplay_sap.go` and `fairplay_md5.go` come
  from its `internal/airplay` package; `descriptor.go` carries its descriptor
  function and constants. Local modifications (`bridge.go`, `fast.go`,
  `ring.go`) fold away payload-independent prefix blocks and tabulate the
  scramble's index sequences, and are covered by the same licence.

An earlier revision of this file said the contribution derived from doubletake
"not at all". That was true of the generated bridge it described — 8.1 MB of
partial-evaluation output — and stopped being true when that bridge was replaced
by the closed form. Redistributors should honour LGPL-3.0 for `fpsapcore`.

It also reverses an argument made here previously. This directory's `LICENSE` is
GPL-3.0 precisely *because* it derived from doubletake, and I suggested that
rationale no longer described the contents and might be retired. **It describes
them again.** Keep the GPL-3.0 licence and the subprocess isolation; the earlier
suggestion was based on a tree that no longer exists.

The two licences compose without friction: Blue Oak is permissive, LGPL-3.0
flows into GPL-3.0, and this directory is already GPL-3.0.

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
