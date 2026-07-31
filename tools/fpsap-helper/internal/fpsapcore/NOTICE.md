# Third-party code

`fpsapcore` is derived from
[omarroth/doubletake](https://github.com/omarroth/doubletake) at commit
`8ccea5f`, which is licensed **LGPL-3.0**. The files `fairplay_sap.go` and
`fairplay_md5.go` are taken from its `internal/airplay` package; `descriptor.go`
carries its descriptor function and constants.

That code is the closed form of the FairPlay Phase-1 bridge. This project had
independently reached the same 20 bytes through 7.2 MB of generated
straight-line code, and the two were verified byte-for-byte equal over thousands
of payloads before the closed form was adopted — see `HANDOFF-2026-07-31.md`.

Modifications here (`bridge.go`, `fast.go`, `ring.go`) fold away the
payload-independent prefix blocks and tabulate the scramble's index sequences.
They are covered by the same licence.

Anyone redistributing this module should honour LGPL-3.0 for that package.
