package main

// clampPct keeps a computed percentage within the [0, 100] bound the
// check-in payload schema enforces server-side, guarding against rounding
// or edge-case arithmetic pushing it slightly out of range.
func clampPct(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
