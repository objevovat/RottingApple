package fairplayhash

// SPN1 computes the FairPlay Phase-2 first white-box AES SPN pass, fully
// analytically (no interpreter). Its plaintext is a payload-independent
// constant (SPN1CoreInput); the only payload-dependent inputs are the 9 round
// keys, which are BigEndian(roundOutputs[10+r]) for r = 0..8 -- the same mixing
// table that feeds TailSPN, consumed in forward round order.
//
// Structure (standard AES):
//
//	state = SPN1CoreInput
//	for r in 0..8:
//	  stage1[out] = SPN1CoreTables[r][out][ state[ShiftRows[out]] ]   // SubBytes+ShiftRows
//	  state       = ApplyMixColumns(stage1) ^ mix[r]                  // MixColumns + AddRoundKey
//	out = ApplyTrailing(state)                                        // final round + output encode
//
// The 16-byte result is SPN#1's output (mem[SP+496]), which is byte-reversed
// and staged into round 8's WB-MD5 hidden words.
// spn1ShiftRows is the AES ShiftRows source permutation for SPN#1's core rounds:
// output position `out` reads the round input at spn1ShiftRows[out].
var spn1ShiftRows = [16]int{0, 5, 10, 15, 4, 9, 14, 3, 8, 13, 2, 7, 12, 1, 6, 11}

func SPN1(roundOutputs *[20][4]uint32) [16]byte {
	state := SPN1CoreInput
	for r := 0; r < 9; r++ {
		var stage1 [16]byte
		for out := 0; out < 16; out++ {
			stage1[out] = SPN1CoreTables[r][out][state[spn1ShiftRows[out]]]
		}
		mixed := ApplyMixColumns(stage1)
		mix := beRoundOutput(roundOutputs[10+r])
		for i := 0; i < 16; i++ {
			state[i] = mixed[i] ^ mix[i]
		}
	}
	return ApplyTrailing(state)
}

func beRoundOutput(s [4]uint32) (o [16]byte) {
	for w := 0; w < 4; w++ {
		o[w*4+0] = byte(s[w] >> 24)
		o[w*4+1] = byte(s[w] >> 16)
		o[w*4+2] = byte(s[w] >> 8)
		o[w*4+3] = byte(s[w])
	}
	return
}

// ApplyMixColumns applies SPN#1's full GF(2)-affine MixColumns (all 4 columns).
// applyMixColumnsReference is the direct bit-by-bit reading of the matrix,
// kept as the oracle TestMixColumnsFastEquivalent checks the fast path against.
func applyMixColumnsReference(in [16]byte) [16]byte {
	var out [16]byte
	for ob := 0; ob < 128; ob++ {
		var acc byte
		row := &SPN1MixColRows[ob]
		for ib := 0; ib < 128; ib++ {
			if (row[ib>>3]>>(uint(ib)&7))&1 != 0 {
				acc ^= (in[ib>>3] >> (uint(ib) & 7)) & 1
			}
		}
		acc ^= (SPN1MixColConst[ob>>3] >> (uint(ob) & 7)) & 1
		if acc != 0 {
			out[ob>>3] |= 1 << (uint(ob) & 7)
		}
	}
	return out
}
