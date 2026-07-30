package fpbridge

import (
	"encoding/hex"
	"fmt"
)

// m3Prefix is a 144-byte m3 header captured once from a post-m1 emulator
// snapshot. Its 16-byte FPLY framing is genuinely constant, but the 128-byte
// body encodes that snapshot's local SAP, which a real sender is supposed to
// generate per session. See FPSAPExchangeM3 for what that costs.
var m3Prefix, _ = hex.DecodeString(
	"46504c590301030000000098038f1a9c991ea22c511e45ba97f1af8dfb0f86f5" +
		"50c54486fe6b3ab233da431ef8e5fc1156dba321fffeabb1b392b09d227e88c7" +
		"12202866eb7bbf310015aa1d19a5df36d5dfd8d3ca1639b376eaece946edfe8b" +
		"7a66cd302d04aac3c1251714019bd5f2d49b543e11eed1646291ec8efd96b691" +
		"01b849fd93a02860d1a0dff5cd4414aa")

// FPSAPExchangeM3 computes the FairPlay SAP m3 response for a given m2 message.
// It returns the full 164-byte m3: a 144-byte prefix followed by the 20-byte
// challenge response.
//
// It builds with a plain `go build`, embeds no Apple snapshot, and runs no ARM64
// interpreter. The 20-byte response is the well-tested part: 142/142 archived
// golden vectors, plus eight vectors from two independent emulator-based senders
// (see external_vectors_test.go).
//
// Known limitation — replays one frozen session. The 144-byte prefix is a
// constant, so every m3 this function emits carries the same local SAP. Real
// senders generate that SAP per session and encrypt it into the m3 body.
// Receivers that validate the body reject the replay: omarroth/doubletake#17
// reports RTSP/1.0 466 Key Management Error from an AppleTV3,2, and doubletake
// removed its own hardcoded prefix in e544a88 (2026-07-20) to fix it.
//
// So: trust FPExchangeBlobless, and treat this framing as a reference that works
// against permissive receivers only. Callers wanting broad device compatibility
// need a session-aware m3 body.
func FPSAPExchangeM3(m2 []byte) ([]byte, error) {
	if len(m2) < 142 {
		return nil, fmt.Errorf("m2 too short: %d bytes (need >= 142)", len(m2))
	}
	var payload [128]byte
	copy(payload[:], m2[14:142])

	hash := FPExchangeBlobless(payload)

	m3 := make([]byte, 164)
	copy(m3[:144], m3Prefix)
	copy(m3[144:], hash[:])
	return m3, nil
}
