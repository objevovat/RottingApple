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
instruction dispatch, and no image of Apple's memory.

Apple-derived **data** does remain, and always will — this is white-box
cryptography, where the key is dissolved into lookup tables, so the tables *are*
the cipher. Concretely: white-box AES tables in `internal/fpbridge`, ~116 KB of
baked table pages across the layer packages, and 79 distinct addresses from
Apple's code segment that survive as constants because the buffer being hashed
genuinely contains stack memory holding those return addresses. The change is
that this data is no longer accompanied by Apple's code.

## Size

The generated layer packages are large: roughly 17 MB of Go, against the ~1.07 MB
of interpreter and snapshot they replace. They are machine-generated
straight-line code produced by partial evaluation of an execution trace, not a
compact algorithm — `layerc.EncodeX9` alone is 131,196 statements, and nobody
can currently read it and explain what the encoding does.

Reducing these to closed form is unfinished work. The trade being offered is
**provenance for bulk**, and it should be evaluated on those terms.

## Verification

- 142/142 archived golden vectors (`testdata/golden_vectors.csv`)
- 8/8 vectors published by two unrelated senders — `nored/airfry` and
  `omarroth/doubletake` — that compute this exchange by emulating Apple's
  binary, so agreement is independent of the reverse engineering behind this
  code
- 227/227 differential comparisons against the interpreter this replaces,
  including 200 random payloads and 20 full 142-byte m2 messages, run before the
  interpreter was deleted

Upstream research repository:
<https://github.com/objevovat/fairplay-sap-airplay2-authentication-handshake-whitebox-aes-md5-reverse-engineering-go-rust>
