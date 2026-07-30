package fpbridge

import (
	"encoding/binary"

	"rottingapple/fpsap-helper/internal/fairplayhash"
	"rottingapple/fpsap-helper/internal/layera"
	"rottingapple/fpsap-helper/internal/layerb"
	"rottingapple/fpsap-helper/internal/layerc"
)

// The Phase-1 bridge hash, computed rather than replayed.
//
// The bridge compresses nine MD5-family blocks (B1..B5, C1..C4) and turns the
// result into x9Data, the only thing Phase 2 consumes. It is NOT plain
// Merkle-Damgard: each block's input state is the previous block's output plus
// a per-block delta, added lane-wise as 32-bit words.
//
// Every payload-dependent input is a function of one contiguous slice of the
// Phase-1 GP buffer, which splits into exactly three: gp[0:47], gp[47:111],
// gp[111:128]. B- and C-blocks consuming the same slice are nevertheless
// different functions, so there are five message derivations, not three; the
// deltas, by contrast, genuinely do alias.

type bridgeBlock struct {
	msg     [16]uint32
	delta   [4]uint32
	offset  uint32
	variant fairplayhash.BridgeMutation
	first   bool // start from the IV rather than the previous block's output
}

func bridgeWords16(b []byte) [16]uint32 {
	var m [16]uint32
	for w := 0; w < 16; w++ {
		m[w] = binary.LittleEndian.Uint32(b[w*4:])
	}
	return m
}

func bridgeWords4(b []byte) [4]uint32 {
	var m [4]uint32
	for w := 0; w < 4; w++ {
		m[w] = binary.LittleEndian.Uint32(b[w*4:])
	}
	return m
}

// bridgeX9Data computes x9Data (64 bytes) from the GP buffer, natively.
func bridgeX9Data(gp [128]byte) []byte {
	mb3, mb4, mb5 := layerb.MsgB3(gp[0:47]), layerb.MsgB4(gp[47:111]), layerb.MsgB5(gp[111:128])
	mc3, mc4 := layerb.MsgC3(gp[0:47]), layerb.MsgC4(gp[47:111])
	db4 := layera.DeltaB4(gp[0:47])   // == delta(C3)
	db5 := layera.DeltaB5(gp[47:111]) // == delta(C4)

	kdf, cyc := fairplayhash.BridgeMutationKDF, fairplayhash.BridgeMutationCycle
	h1, h1f := fairplayhash.BridgeHash1Offset, fairplayhash.BridgeHash1FinalOffset
	h2 := fairplayhash.BridgeHash2Offset

	chain := [9]bridgeBlock{
		{bridgeWords16(bridgeConstMsgB1[:]), [4]uint32{}, h1, kdf, true},
		{bridgeWords16(bridgeConstMsgB2[:]), bridgeConstDeltaB2, h1, kdf, false},
		{bridgeWords16(mb3[:]), bridgeConstDeltaB3, h1, kdf, false},
		{bridgeWords16(mb4[:]), bridgeWords4(db4[:]), h1, kdf, false},
		{bridgeWords16(mb5[:]), bridgeWords4(db5[:]), h1f, kdf, false},
		{bridgeWords16(bridgeConstMsgC1[:]), bridgeConstDeltaC1, h2, cyc, true},
		{bridgeWords16(bridgeConstMsgC2[:]), bridgeConstDeltaC2, h2, cyc, false},
		{bridgeWords16(mc3[:]), bridgeWords4(db4[:]), h2, cyc, false},
		{bridgeWords16(mc4[:]), bridgeWords4(db5[:]), h2, cyc, false},
	}

	var prev [4]uint32
	for _, blk := range chain {
		state := prev
		if blk.first {
			state = bridgeIV
		}
		for k := range state {
			state[k] += blk.delta[k]
		}
		msg := blk.msg
		fairplayhash.BridgeMD5Compress(&state, &msg, blk.offset, blk.variant)
		prev = state
	}

	// The output encoding reads 64 bytes: the 32 around the final digest,
	// followed by the last 32 bytes of the GP buffer. The 8 bytes on either
	// side of the digest are payload-independent.
	in := make([]byte, 0, 64)
	in = append(in, bridgeDigestFlankLo[:]...)
	for _, w := range prev {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], w)
		in = append(in, b[:]...)
	}
	in = append(in, bridgeDigestFlankHi[:]...)
	in = append(in, gp[96:128]...)

	head := layerc.EncodeX9(in)

	x9 := make([]byte, 64)
	copy(x9, head[:])
	copy(x9[20:], bridgeX9Tail[:])
	return x9
}

// bridgeNeonState builds Phase 2's vector inputs. Only Vreg0 is
// payload-dependent: it is x9Data[0:16] put through the NEON prologue
// transform, with the three constant vector registers as its masks.
func bridgeNeonState(x9 []byte) fairplayhash.NeonState {
	w0, w1, w2, w3 := fairplayhash.NeonBlockExport(
		binary.LittleEndian.Uint64(x9[0:8]),
		binary.LittleEndian.Uint64(x9[8:16]),
		bridgeVreg1, bridgeVreg3, bridgeVreg2)
	return fairplayhash.NeonState{
		Vreg0: [2]uint64{
			uint64(w0) | uint64(w1)<<32,
			uint64(w2) | uint64(w3)<<32,
		},
		Vreg1: bridgeVreg1,
		Vreg2: bridgeVreg2,
		Vreg3: bridgeVreg3,
	}
}
