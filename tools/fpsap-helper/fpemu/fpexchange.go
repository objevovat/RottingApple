// Package fpemu computes the FairPlay SAP exchange natively.
//
// The name is historical. This package used to be an ARM64 interpreter that
// executed a 165,575-byte snapshot of Apple's signed FairPlay binary, embedded
// in the repository as a Go byte array. Neither the interpreter nor the
// snapshot exists any more: the exchange is now computed from its recovered
// algebra, so no Apple instruction is executed or reproduced here.
//
// The exported API is unchanged, so main.go and the Rust caller in
// crates/rotten-crypto need no modification.
//
// What the computation is, in three stages:
//
//	Phase 1  white-box AES over the 128-byte payload -> a 128-byte GP buffer
//	Bridge   nine MD5-family blocks over that buffer -> a 20-byte digest
//	Phase 2  the white-box MD5 network              -> the 20-byte response
//
// Apple-derived *data* necessarily remains, as it must for white-box crypto:
// the cipher's key is dissolved into lookup tables, so the tables are the
// cipher. What changed is that the data is no longer accompanied by Apple's
// code.
package fpemu

import (
	"fmt"

	"rottingapple/fpsap-helper/internal/fpbridge"
)

// FPSAPExchangeStandalone computes the 20-byte FairPlay SAP response for a
// 128-byte challenge payload. Same signature and same output as the
// interpreter-backed version it replaces.
func FPSAPExchangeStandalone(payload [128]byte) [20]byte {
	return fpbridge.FPExchangeBlobless(payload)
}

// FPSAPExchangeM3 computes the full 164-byte m3 response for a 142-byte m2.
//
// Retained for API compatibility; the helper's main.go does not call it, and
// crates/rotten-crypto does its own framing. Note the caveat carried by the
// underlying implementation: the 144-byte prefix is a constant captured from a
// single session, so this replays one local SAP rather than generating a fresh
// one per session. See fpbridge.FPSAPExchangeM3.
func FPSAPExchangeM3(m2 []byte) ([]byte, error) {
	m3, err := fpbridge.FPSAPExchangeM3(m2)
	if err != nil {
		return nil, fmt.Errorf("fpemu: %w", err)
	}
	return m3, nil
}
