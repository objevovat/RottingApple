package fpbridge

import "rottingapple/fpsap-helper/internal/fpsapcore"

// bridgeX9DataClosed is bridgeX9Data computed in closed form rather than by
// replaying the generated layers. Only x9Data[0:20] is payload-dependent; the
// rest is the constant tail the old path also appended.
//
// pkg/fpsapcore documents where the closed form came from and how it was
// checked; TestClosedFormMatchesLayers pins the two against each other.
func bridgeX9DataClosed(gp [128]byte) [64]byte {
	head := fpsapcore.BridgeX9Head(gp)
	var x9 [64]byte
	copy(x9[:], head[:])
	copy(x9[20:], bridgeX9Tail[:])
	return x9
}
