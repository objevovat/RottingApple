# Third-party code

`fpsapcore` is derived from
[omarroth/doubletake](https://github.com/omarroth/doubletake) at commit
`8ccea5f`, which is licensed **LGPL-3.0**.

**Treat the whole package as LGPL-3.0.** Every file is either their code or a
modification of it:

| file | relationship |
|---|---|
| `fairplay_sap.go` | taken from their `internal/airplay` |
| `fairplay_md5.go` | taken from their `internal/airplay` |
| `descriptor.go` | their descriptor function and constants |
| `bridge.go` | our `gp ^ 0x0f` and word swap around their function |
| `fast.go` | our precompute of the payload-independent prefix blocks |
| `ring.go`, `ring_swar.go` | our tabulation and SWAR form of their scramble loop |
| `scramble.go` | our GF(2)-matrix collapse of their scramble |
| `fairplay_md5_unrolled.go` | our unrolling of their round loop |
| `message_encrypt.go` | our forward-AES m3 body encryption, on their message layout |

An earlier revision of this file named only `bridge.go`, `fast.go` and `ring.go`
as local work. That was accurate when written and went stale as the package
grew; the table above is the current state.

That code is the closed form of the FairPlay Phase-1 bridge. This project had
independently reached the same 20 bytes through 7.2 MB of generated
straight-line code, and the two were verified byte-for-byte equal over thousands
of payloads before the closed form was adopted. The independent version is not
what ships; theirs is smaller and faster, so it replaced ours.

Anyone redistributing this module should honour LGPL-3.0 for this package.

`fairplayhash` is independent work under the Blue Oak Model License 1.0.0.
`fpbridge` is mostly Blue Oak too, with four exceptions that are **also
LGPL-3.0-or-later**, because they were written while reading doubletake's
`exchangeM3` and `validateFPSAPRecord` and it shows in their shape:

| file | what came from where |
|---|---|
| `fp_sap_session.go` | the session's structure — build the record, encrypt the local SAP into bytes 16..144, fold it into the descriptor — follows their `exchangeM3` |
| `fp_sap_m3.go` | field-by-field record validation, modelled on their `validateFPSAPRecord` |
| `mode_identity_test.go` | four response constants produced by running their code |
| `session_xcheck_test.go` | a local SAP and a 164-byte m3 produced by running their code |

Being precise, because a licence claim that overstates independence is worse
than one that understates it. Every *constant* involved is independently present
in data captured here — `FPLY`, the version bytes, the declared length, the mode
byte, the label `8f 1a 9c` and the local SAP's `00 01` head are all readable
straight out of `m3Prefix`, which came from an emulator snapshot rather than
from them. What was taken is the reading of the layout: that byte 12 is the
mode, that 13..16 is a label, that a sender randomises the SAP from byte 2.
Knowing where to look is the contribution, and it was theirs.

Every file in all three packages carries an SPDX header saying which licence
applies. Note that `fpbridge` imports `fpsapcore`, so any binary built from this
module is a combined work under LGPL-3.0-or-later regardless; the per-package
split matters only if you lift a package out on its own.
