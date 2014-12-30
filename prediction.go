package main

import (
	"math"
)

var (
	q = math.Log(10) / 400
)

func expectedG(rd float64) float64 {
	return 1 / (math.Sqrt(1.0 + 3.0*q*q*rd*rd/(math.Pi*math.Pi)))
}

func expectedScore(w, l player) float64 {
	gVal := expectedG(math.Sqrt(w.r*w.r + l.rd*l.rd))
	return 1.0 / (1.0 + math.Pow(10, -gVal*(w.r-l.r)/400.0))
}

func rateGame(y float64, expected float64) float64 {
	e := clamp(0.01, expected, 0.99)
	return -(y*math.Log10(e) + (1.0-y)*math.Log10(1.0-e))
}

func predictMatches(db map[string]int, ps []player, ms <-chan []int) float64 {
	n := 0
	s := 0.0
	for m := range ms {
		e := expectedScore(ps[m[0]], ps[m[1]])
		s += rateGame(1, e)
		n++
	}

	return (s / float64(n))
}

func tourneyMatches(t int) chan []int {
	c := make(chan []int)

	go func() {
		for pi := range matches {
			for _, d := range matches[pi][t] {
				if d.SJ() == 1.0 {
					c <- []int{pi, d.(duel).opponent}
				}
			}
		}
		close(c)
	}()
	return c
}

func predict(ts []tournament, ps []player, config conf) float64 {
	n := len(ts)

	rank(ts[0:n-1], ps, config.tau)
	return predictMatches(playerName, ps, tourneyMatches(n-1))
}
