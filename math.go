package main

// clamp bounds a floating64 point value from above and below
// If the value is less than low, low is returned. If the value is higher than high, then high is returned.
// Otherwise the clamp returns the value.
// We use this to bound certain movements in order to avoid moving to silly values.
func clamp(low float64, v float64, high float64) float64 {
	if v < low {
		return low
	} else if v > high {
		return high
	} else {
		return v
	}
}
