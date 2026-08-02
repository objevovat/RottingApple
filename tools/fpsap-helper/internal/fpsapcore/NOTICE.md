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

Anyone redistributing this module should honour LGPL-3.0 for this package. The
sibling `fpbridge` and `fairplayhash` packages are independent work under the
Blue Oak Model License 1.0.0 and carry no code from here; every file in all
three carries an SPDX header saying which it is.
