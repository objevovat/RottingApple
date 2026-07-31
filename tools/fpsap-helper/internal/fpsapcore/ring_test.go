package fpsapcore

import "testing"

// The ring loop is optimised in two ways, and neither is covered by the
// end-to-end vectors in a way that would localise a fault: the first 155 steps
// read tabulated indices, and the remaining 685 abandon the tables for four
// counters walking contiguous spans. Both are checked here against the naive
// derivation they replace.
//
// The derivation is only interesting because i is uint32: for i < 155 the
// subtraction wraps through 2^32 before the modulo, so (i-155)%210 is (i-109)
// mod 210 rather than (i+55) mod 210, because 2^32 mod 210 == 46. A port to any
// language with different integer promotion gets this wrong silently.
func naiveRingIndices(i uint32) (x, y, z, w uint8) {
	return uint8((i - 155) % 210), uint8((i - 57) % 210), uint8((i - 13) % 210), uint8(i % 210)
}

func TestRingTablesMatchNaiveDerivation(t *testing.T) {
	for i := uint32(0); i < 840; i++ {
		x, y, z, w := naiveRingIndices(i)
		if ringX[i] != x || ringY[i] != y || ringZ[i] != z || ringW[i] != w {
			t.Fatalf("i=%d: tables (%d,%d,%d,%d), naive (%d,%d,%d,%d)",
				i, ringX[i], ringY[i], ringZ[i], ringW[i], x, y, z, w)
		}
	}
}

// TestRingWrapIsIrregular is the control for the test above. If the wrap were
// ever "simplified" to (i+55)%210 the tables would still be self-consistent, so
// assert the two genuinely disagree over the first 155 steps.
func TestRingWrapIsIrregular(t *testing.T) {
	differences := 0
	for i := uint32(0); i < 155; i++ {
		if x, _, _, _ := naiveRingIndices(i); x != uint8((i+55)%210) {
			differences++
		}
	}
	if differences == 0 {
		t.Fatal("the uint32 wrap is not being exercised; the tables prove nothing")
	}
	t.Logf("wrap is irregular on %d of the first 155 steps", differences)
}

// TestRingSegmentedCountersMatchTables covers the second optimisation: from
// i=155 the loop drops the tables and advances four counters in contiguous
// segments, cut so no counter crosses 210 mid-segment.
func TestRingSegmentedCountersMatchTables(t *testing.T) {
	xi, yi, zi, wi := 0, 98, 142, 155
	for i := 155; i < 840; {
		n := 840 - i
		for _, idx := range [4]int{xi, yi, zi, wi} {
			if r := 210 - idx; r < n {
				n = r
			}
		}
		for k := 0; k < n; k++ {
			step := uint32(i + k)
			if uint8(xi+k) != ringX[step] || uint8(yi+k) != ringY[step] ||
				uint8(zi+k) != ringZ[step] || uint8(wi+k) != ringW[step] {
				t.Fatalf("step %d: counters (%d,%d,%d,%d), tables (%d,%d,%d,%d)",
					step, xi+k, yi+k, zi+k, wi+k,
					ringX[step], ringY[step], ringZ[step], ringW[step])
			}
		}
		i += n
		xi = (xi + n) % 210
		yi = (yi + n) % 210
		zi = (zi + n) % 210
		wi = (wi + n) % 210
	}
}
