package main

import (
	"math"
)

var (
	q = math.Log(10) / 400
)

func clamp(low float64, v float64, high float64) float64 {
	if v < low {
		return low
	} else if v > high {
		return high
	} else {
		return v
	}
}

func rateGame(y float64, expected float64) float64 {
	e := clamp(0.01, expected, 0.99)
	return -(y*math.Log10(e) + (1.0-y)*math.Log10(1.0-e))
}

func expectedG(rd float64) float64 {
	return 1 / (math.Sqrt(1.0 + 3.0*q*q*rd*rd/(math.Pi*math.Pi)))
}

func expectedScore(w, l player) float64 {
	gVal := expectedG(math.Sqrt(w.r*w.r + l.rd*l.rd))
	return 1.0 / (1.0 + math.Pow(10, -gVal*(w.r-l.r)/400.0))
}
