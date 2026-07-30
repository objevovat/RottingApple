package fpbridge

import (
	"encoding/csv"
	"encoding/hex"
	"os"
	"testing"
)

// goldenCSV holds the archived ground-truth FairPlay SAP vectors
// (128-byte challenge payload -> 20-byte m3 response hash), bundled with this
// module so validation is fully self-contained.
const goldenCSV = "../../testdata/golden_vectors.csv"

func loadGolden(t *testing.T) [][]string {
	t.Helper()
	f, err := os.Open(goldenCSV)
	if err != nil {
		// Fail rather than skip: a missing vector file must not look like a pass.
		t.Fatalf("golden vectors missing (%v) — this module is not validated without them", err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("read golden vectors: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("golden vectors empty")
	}
	return rows[1:] // skip header
}

// TestGoldenPayloadToHash validates FPExchangeBlobless: 128-byte payload ->
// 20-byte response, across every archived vector.
func TestGoldenPayloadToHash(t *testing.T) {
	rows := loadGolden(t)
	pass, fail := 0, 0
	for _, rec := range rows {
		if len(rec) < 4 {
			continue
		}
		category, payloadHex, wantHex := rec[0], rec[2], rec[3]
		pb, err := hex.DecodeString(payloadHex)
		if err != nil || len(pb) != 128 {
			t.Fatalf("%s: bad payload hex", category)
		}
		want, err := hex.DecodeString(wantHex)
		if err != nil || len(want) != 20 {
			t.Fatalf("%s: bad hash hex", category)
		}
		var payload [128]byte
		copy(payload[:], pb)

		got := FPExchangeBlobless(payload)
		if string(got[:]) != string(want) {
			fail++
			if fail <= 3 {
				t.Errorf("%s: got %x, want %x", category, got, want)
			}
			continue
		}
		pass++
	}
	if fail != 0 {
		t.Fatalf("payload->hash: %d passed, %d FAILED", pass, fail)
	}
	if pass == 0 {
		t.Fatal("no vectors executed")
	}
	t.Logf("✅ %d/%d golden vectors pass (payload -> 20-byte hash)", pass, pass)
}

// TestGoldenM2ToM3 validates the wire-level entry point a sender actually calls:
// a full m2 message -> the 164-byte m3 frame (144-byte constant prefix + hash).
func TestGoldenM2ToM3(t *testing.T) {
	rows := loadGolden(t)
	pass, fail := 0, 0
	for _, rec := range rows {
		if len(rec) < 4 {
			continue
		}
		category, payloadHex, wantHex := rec[0], rec[2], rec[3]
		pb, _ := hex.DecodeString(payloadHex)
		want, _ := hex.DecodeString(wantHex)
		if len(pb) != 128 || len(want) != 20 {
			t.Fatalf("%s: malformed vector", category)
		}

		// A receiver places the challenge at bytes 14..142 of m2.
		m2 := make([]byte, 142)
		copy(m2[14:142], pb)

		m3, err := FPSAPExchangeM3(m2)
		if err != nil {
			t.Fatalf("%s: FPSAPExchangeM3: %v", category, err)
		}
		if len(m3) != 164 {
			t.Fatalf("%s: m3 is %d bytes, want 164", category, len(m3))
		}
		if string(m3[:4]) != "FPLY" {
			t.Errorf("%s: missing FPLY framing: % x", category, m3[:4])
		}
		if string(m3[144:]) != string(want) {
			fail++
			if fail <= 3 {
				t.Errorf("%s: response %x, want %x", category, m3[144:], want)
			}
			continue
		}
		pass++
	}
	if fail != 0 {
		t.Fatalf("m2->m3: %d passed, %d FAILED", pass, fail)
	}
	t.Logf("✅ %d/%d golden vectors pass (m2 -> 164-byte m3 frame)", pass, pass)
}

// TestShortM2Rejected checks malformed input is rejected rather than panicking;
// a receiver can send anything.
func TestShortM2Rejected(t *testing.T) {
	for _, n := range []int{0, 13, 141} {
		if _, err := FPSAPExchangeM3(make([]byte, n)); err == nil {
			t.Errorf("m2 of %d bytes: expected an error, got nil", n)
		}
	}
}

// TestGPBufferSane checks Phase 1 is deterministic and payload-dependent.
func TestGPBufferSane(t *testing.T) {
	var a, b [128]byte
	for i := range a {
		a[i] = byte(i*7 + 3)
		b[i] = byte(i*3 + 1)
	}
	if GPBuffer(a) == GPBuffer(b) {
		t.Error("GP buffer identical for different payloads")
	}
	if GPBuffer(a) != GPBuffer(a) {
		t.Error("GP buffer not deterministic")
	}
}
