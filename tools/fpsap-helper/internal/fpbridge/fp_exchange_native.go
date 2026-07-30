package fpbridge

import "rottingapple/fpsap-helper/internal/fairplayhash"

// FPExchangeNative computes the FairPlay SAP m3 response (20 bytes) from the
// 128-byte m2 challenge with no interpreter and no transliterated code:
//
//	Phase 1 : wbaesFullPhase1        -> the 128-byte GP buffer
//	Bridge  : 9 MD5-family blocks, messages and chaining deltas computed from
//	          GP slices, then the output encoding -> x9Data
//	Phase 2 : fairplayhash.ComputeHashAnalytical
//
// It differs from FPExchangeBlobless only in the middle step. FPExchangeBlobless
// is blobless but not algorithmic: it still replays a transliteration of Apple's
// ARM64 bridge against a baked memory image. This computes the same thing.
//
// Phase 2's real input surface is narrow, and measured rather than assumed
// (TestLayerDPhase2* ): it never reads the 16KB scratch window or initialMD5,
// x9Data[0:16] reaches it only through Vreg0, and Vreg1..3 are constants. So a
// zeroed scratch buffer is correct here, not a shortcut.
func FPExchangeNative(payload [128]byte) [20]byte {
	gp := wbaesFullPhase1(payload)
	x9Data := bridgeX9Data(gp)
	ns := bridgeNeonState(x9Data)

	var state fairplayhash.HashState
	state.Mem = make([]byte, 16384)

	fairplayhash.ComputeM3Setup(&state, [4]uint32{})
	fairplayhash.ComputeHashAnalytical(&state, &ns, x9Data)

	var result [20]byte
	copy(result[:], state.Mem[fairplayhash.Span7Offset:fairplayhash.Span7Offset+20])
	return result
}
