package fairplayhash

// SPN#1 stage 2 — the register-only "MixColumns" combine — fully unwound.
//
// It is a single, round-independent GF(2)-affine map on the 128-bit state:
//
//	stage2 = Stage2Mix . stage1  XOR  Stage2Const
//
// where Stage2Mix is a 128x128 GF(2) matrix (each output bit is an XOR of a
// subset of input bits) and Stage2Const is a 128-bit constant. This is the
// white-box "nibble-XOR / TypeIV" form: linear over GF(2), NOT over GF(2^8)
// (standard AES MixColumns fails 0/16; a GF(2^8) matrix fails ~66% held-out;
// this GF(2) map verifies 0-mismatch on held-out samples for all 9 rounds and
// is bit-identical across rounds).
//
// The 128 rows are stored as [16]byte masks: bit b of Stage2MixRows[out][b>>3]
// selects input bit b for output bit `out`. Values are generated from the raw
// interpreter by TestExtractStage2 and checked in by TestStage2MatrixMatchesConst.

// stage2Bit returns bit b (0..127) of a 16-byte vector.
func stage2Bit(v *[16]byte, b int) byte { return (v[b>>3] >> (uint(b) & 7)) & 1 }

// ApplyStage2 applies the solved GF(2)-affine mixing map to a post-substitution
// 16-byte state, returning the post-mixing state (before AddRoundKey).
func ApplyStage2(stage1 [16]byte) [16]byte {
	var out [16]byte
	for ob := 0; ob < 128; ob++ {
		var acc byte
		row := &Stage2MixRows[ob]
		for ib := 0; ib < 128; ib++ {
			if (row[ib>>3]>>(uint(ib)&7))&1 != 0 {
				acc ^= stage2Bit(&stage1, ib)
			}
		}
		acc ^= (Stage2Const[ob>>3] >> (uint(ob) & 7)) & 1
		if acc != 0 {
			out[ob>>3] |= 1 << (uint(ob) & 7)
		}
	}
	return out
}
