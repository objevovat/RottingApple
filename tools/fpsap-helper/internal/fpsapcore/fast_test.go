package fpsapcore

import (
	"math/rand"
	"testing"
)

func TestDescriptorFastMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	for i := 0; i < 3000; i++ {
		var body [128]byte
		rng.Read(body[:])
		if got, want := descriptorFast(&body), fpsapDescriptorForSAP(localSAP, body); got != want {
			t.Fatalf("body %d: fast %x reference %x", i, got, want)
		}
	}
}
