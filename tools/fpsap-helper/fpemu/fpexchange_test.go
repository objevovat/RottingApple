package fpemu

import (
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"os"
	"testing"

	"rottingapple/fpsap-helper/internal/fpbridge"
)

// These exercise the exported seam — the two functions main.go and
// crates/rotten-crypto depend on — rather than the internals. The helper had no
// tests before, so a regression in the exchange would previously have surfaced
// only as an Apple TV refusing to pair.

const goldenCSV = "../testdata/golden_vectors.csv"

// TestFPSAPExchangeStandaloneGoldenVectors runs the archived corpus of 142
// challenge/response pairs. A missing corpus fails the test rather than
// skipping it: a vector file that silently disappears turns this into a test
// that passes by doing nothing.
func TestFPSAPExchangeStandaloneGoldenVectors(t *testing.T) {
	f, err := os.Open(goldenCSV)
	if err != nil {
		t.Fatalf("golden vectors are required, not optional: %v", err)
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("read golden vectors: %v", err)
	}
	if len(rows) < 2 {
		t.Fatal("golden vector file has no data rows")
	}

	checked := 0
	for i, row := range rows[1:] { // skip header
		if len(row) < 4 {
			t.Fatalf("row %d: expected 4 columns, got %d", i+2, len(row))
		}
		payloadHex, wantHex := row[2], row[3]

		raw, err := hex.DecodeString(payloadHex)
		if err != nil {
			t.Fatalf("row %d: bad payload hex: %v", i+2, err)
		}
		if len(raw) != 128 {
			t.Fatalf("row %d: payload is %d bytes, want 128", i+2, len(raw))
		}

		var payload [128]byte
		copy(payload[:], raw)

		got := FPSAPExchangeStandalone(payload)
		if gotHex := hex.EncodeToString(got[:]); gotHex != wantHex {
			t.Errorf("row %d (%s): got %s, want %s", i+2, row[0], gotHex, wantHex)
		}
		checked++
	}

	if checked != 142 {
		t.Errorf("checked %d vectors, expected 142", checked)
	}
	t.Logf("%d/%d golden vectors pass", checked, checked)
}

// TestFPSAPExchangeStandaloneIndependentVectors cross-checks against vectors
// published by two unrelated AirPlay senders that compute this exchange by
// emulating Apple's binary. Neither derives from the reverse engineering behind
// this implementation, so agreement is independent evidence rather than a
// restatement of our own assumptions.
//
//	nored/airfry          rust/fpemu/tests/vectors.txt  (CORE_IN/CORE_OUT)
//	omarroth/doubletake   internal/airplay/fpsap_test.go
func TestFPSAPExchangeStandaloneIndependentVectors(t *testing.T) {
	capturedM2 := mustHex(t, "46504c59030102000000008202034a114c26b77d4e2eec2c8f89fdb653b5b3"+
		"2d3576bc176816d110a14c3f53c08dbb936183bfdfe0a4f3c12e85216003b46f738c40c54da6c436d29d1"+
		"b342d63c7b314309ae79a33bb1787709ef077cbfe4190117a3423e270fd1a2eac44da1a7934f59dc681d1"+
		"b70783f228c4d077c2d495f5285c3bf8df586fc2ebfe17fb5b65")

	fill := func(v byte) (p [128]byte) {
		for i := range p {
			p[i] = v
		}
		return
	}
	sparse := func(i int) (p [128]byte) {
		p[i] = 0x42
		return
	}
	ramp := func() (p [128]byte) {
		for i := range p {
			p[i] = byte(i)
		}
		return
	}
	fromM2 := func() (p [128]byte) {
		copy(p[:], capturedM2[14:142])
		return
	}

	for _, tc := range []struct {
		name    string
		source  string
		payload [128]byte
		want    string
	}{
		{"ramp", "airfry", ramp(), "84449e19d306930b66942aacfb71395a903878ef"},
		{"all-zeros", "doubletake", [128]byte{}, "6f627565f3e77f5b5ede91beee7baf92e4241e0b"},
		{"all-ff", "doubletake", fill(0xff), "dc2cc74f2ed55484f59f95b96082f0f5c017dd17"},
		{"captured-m2", "doubletake", fromM2(), "4b911e48af23d8406368aeafbb61bfcd569e3e55"},
		{"0x42-at-0", "doubletake", sparse(0), "9bfb9556b8659c2ac94b7ef9e587d71e159ea624"},
		{"0x42-at-63", "doubletake", sparse(63), "150d9fa4eb456e73ba48de5779c5c996b16b3b23"},
		{"0x42-at-64", "doubletake", sparse(64), "a167db30424ff8890d085c0f1c92b2c5cc06fc45"},
		{"0x42-at-127", "doubletake", sparse(127), "d246ec5e7adc8118994b8df77146529486ac7caf"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := FPSAPExchangeStandalone(tc.payload)
			if gotHex := hex.EncodeToString(got[:]); gotHex != tc.want {
				t.Fatalf("disagrees with %s: got %s, want %s", tc.source, gotHex, tc.want)
			}
		})
	}
}

// TestFPSAPExchangeM3Framing checks the wrapper's length handling and that the
// response tail tracks the payload. It deliberately does not assert that the
// 144-byte prefix is correct for a live session: it is a captured constant, and
// the doc comment on FPSAPExchangeM3 explains why that matters.
func TestFPSAPExchangeM3Framing(t *testing.T) {
	if _, err := FPSAPExchangeM3(make([]byte, 141)); err == nil {
		t.Error("expected an error for a short m2, got nil")
	}

	var challenge [128]byte
	m2 := fpbridge.NewFPSAPM2(fpbridge.SupportedFPSAPMode, challenge)
	a, err := FPSAPExchangeM3(m2)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 164 {
		t.Fatalf("m3 is %d bytes, want 164", len(a))
	}

	challenge[6] ^= 0xff
	m2 = fpbridge.NewFPSAPM2(fpbridge.SupportedFPSAPMode, challenge)
	b, err := FPSAPExchangeM3(m2)
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(a[144:]) == hex.EncodeToString(b[144:]) {
		t.Error("m3 tail did not change with the payload; the exchange is ignoring its input")
	}
}

// TestRejectsUnsupportedModes is the point of parsing rather than slicing. This
// helper implements mode 3 only, and the earlier code answered every m2 as
// though it had asked for mode 3 -- so a mode-0 challenge got confident wrong
// bytes back instead of an error.
func TestRejectsUnsupportedModes(t *testing.T) {
	var challenge [128]byte
	for _, mode := range []byte{0, 1, 2} {
		if _, err := ParseFPSAPM2(fpbridge.NewFPSAPM2(mode, challenge)); err == nil {
			t.Errorf("mode %d was accepted; only mode %d is implemented", mode, fpbridge.SupportedFPSAPMode)
		}
	}
	if _, err := ParseFPSAPM2(fpbridge.NewFPSAPM2(fpbridge.SupportedFPSAPMode, challenge)); err != nil {
		t.Errorf("mode %d rejected: %v", fpbridge.SupportedFPSAPMode, err)
	}
}

// TestParseRejectsMalformedFraming covers what hand-slicing bytes 14:142 could
// not: a buffer of the right length that is not an m2 at all.
func TestParseRejectsMalformedFraming(t *testing.T) {
	var challenge [128]byte
	good := fpbridge.NewFPSAPM2(fpbridge.SupportedFPSAPMode, challenge)

	for _, tc := range []struct {
		name string
		mut  func([]byte)
	}{
		{"bad magic", func(b []byte) { b[0] = 'X' }},
		{"bad version", func(b []byte) { b[4] = 9 }},
		{"bad length field", func(b []byte) { b[11] = 0 }},
		{"bad payload marker", func(b []byte) { b[12] = 7 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := append([]byte(nil), good...)
			tc.mut(bad)
			if _, err := ParseFPSAPM2(bad); err == nil {
				t.Errorf("%s was accepted", tc.name)
			}
		})
	}
}

// TestSessionsDoNotRepeat is the reason NewFPSAPSession exists: a receiver that
// checks the m3 body rejects a replayed local SAP.
func TestSessionsDoNotRepeat(t *testing.T) {
	var challenge [128]byte
	m2 := fpbridge.NewFPSAPM2(fpbridge.SupportedFPSAPMode, challenge)

	first, err := NewFPSAPSession(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewFPSAPSession(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	a, err := first.ExchangeM3(m2)
	if err != nil {
		t.Fatal(err)
	}
	b, err := second.ExchangeM3(m2)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 164 || len(b) != 164 {
		t.Fatalf("m3 lengths %d and %d, want 164", len(a), len(b))
	}
	if hex.EncodeToString(a) == hex.EncodeToString(b) {
		t.Error("two sessions produced the same m3; the local SAP is not per-session")
	}
	if hex.EncodeToString(a[:144]) == hex.EncodeToString(frozen(t, m2)[:144]) {
		t.Error("session m3 reused the frozen prefix")
	}
}

func frozen(t *testing.T, m2 []byte) []byte {
	t.Helper()
	m3, err := FPSAPExchangeM3(m2)
	if err != nil {
		t.Fatal(err)
	}
	return m3
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad test vector hex: %v", err)
	}
	return b
}
