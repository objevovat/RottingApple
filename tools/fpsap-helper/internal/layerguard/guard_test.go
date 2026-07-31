// Package layerguard holds the two invariants of the generated Phase-1 layers
// that no single layer's own tests can express: that they still compute what
// they computed before, and that they carry none of Apple's address space.
//
// The second is what makes the first necessary. The emitted code is a
// transliteration of a traced window, so it was born full of dyld-shared-cache
// addresses -- as the bases its access arithmetic was written against, and as
// data it spills into scratch memory. scratch/layera_sanitize.py takes them
// out. None of them can matter: the shared cache is slid on every boot, so a
// computation whose result depended on an address value would return a
// different SAP response after a reboot, and Apple's does not. TestLayerDigest
// is the check on that reasoning rather than the reasoning itself.
package layerguard

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"rottingapple/fpsap-helper/internal/layera"
	"rottingapple/fpsap-helper/internal/layerb"
	"rottingapple/fpsap-helper/internal/layerc"
)

// layerDigest folds all five message derivations, both deltas and the output
// encoding over one pseudorandom payload stream into a single value, so the
// whole native Phase-1 surface is pinned without carrying vectors for each.
func layerDigest(n int) string {
	h := fnv.New128a()
	rng := rand.New(rand.NewSource(0xA11CE))
	buf := make([]byte, 64)
	for i := 0; i < n; i++ {
		rng.Read(buf)
		var idx [4]byte
		binary.LittleEndian.PutUint32(idx[:], uint32(i))
		h.Write(idx[:])

		d4, d5 := layera.DeltaB4(buf[:47]), layera.DeltaB5(buf[:64])
		h.Write(d4[:])
		h.Write(d5[:])

		b3, b4, b5 := layerb.MsgB3(buf[:47]), layerb.MsgB4(buf[:64]), layerb.MsgB5(buf[:17])
		c3, c4 := layerb.MsgC3(buf[:47]), layerb.MsgC4(buf[:64])
		for _, m := range [][64]byte{b3, b4, b5, c3, c4} {
			h.Write(m[:])
		}

		x9 := layerc.EncodeX9(buf[:64])
		h.Write(x9[:])
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Recorded from the generated files as they stood before sanitising, at commit
// faf72f0. Any regeneration or rewrite that changes this has changed behaviour.
const layerDigest512 = "5c618f3e1c9d8e307661000262c9518c"

func TestLayerDigest(t *testing.T) {
	if got := layerDigest(512); got != layerDigest512 {
		t.Fatalf("layer digest changed\n want %s\n got  %s", layerDigest512, got)
	}
}

var (
	genFiles = []string{
		"layera/deltab4_gen.go", "layera/deltab5_gen.go",
		"layerb/msgB3_gen.go", "layerb/msgB4_gen.go", "layerb/msgB5_gen.go",
		"layerb/msgC3_gen.go", "layerb/msgC4_gen.go",
		"layerc/encodex9_gen.go",
	}
	hexLit    = regexp.MustCompile(`0x[0-9a-f]+`)
	bakedPage = regexp.MustCompile(`^\t\{(\d+), "([0-9a-f]*)"\},$`)
)

// inCache reports whether v lands in the dyld shared cache window the trace ran
// under. Nothing generated should hold such a value, as a literal or as data.
func inCache(v uint64) bool { return v >= 0x180000000 && v < 0x1c0000000 }

func TestNoAppleAddresses(t *testing.T) {
	for _, rel := range genFiles {
		src, err := os.ReadFile(filepath.Join("..", rel))
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		lits, baked := 0, 0
		for n, line := range strings.Split(string(src), "\n") {
			if m := bakedPage.FindStringSubmatch(line); m != nil {
				for off := 0; off+16 <= len(m[2]); off += 16 {
					if inCache(leU64(m[2][off : off+16])) {
						baked++
					}
				}
				continue
			}
			for _, lit := range hexLit.FindAllString(line, -1) {
				if v, err := strconv.ParseUint(lit[2:], 16, 64); err == nil && inCache(v) {
					lits++
					if lits <= 3 {
						t.Errorf("%s:%d: Apple address %s", rel, n+1, lit)
					}
				}
			}
		}
		if lits > 3 {
			t.Errorf("%s: %d Apple address literals in all", rel, lits)
		}
		if baked > 0 {
			t.Errorf("%s: %d pointer-shaped values in the baked memory image", rel, baked)
		}
	}
}

// leU64 decodes 16 hex characters as a little-endian 64-bit value.
func leU64(h string) uint64 {
	var v uint64
	for i := 7; i >= 0; i-- {
		b, err := strconv.ParseUint(h[i*2:i*2+2], 16, 8)
		if err != nil {
			return 0
		}
		v = v<<8 | b
	}
	return v
}
