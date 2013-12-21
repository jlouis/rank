// prediction code
package main

import (
	"github.com/jlouis/glocko2"
	"math"
)

var (
	q = math.Log(10) / 400
)

type match struct {
	winner string
	loser  string
}

func rateGame(y float64, e float64) float64 {
	return -(y*math.Log10(e) + (1.0-y)*math.Log10(1.0-e))
}

func expectedG(rd float64) float64 {
	return 1 / (math.Sqrt(1.0 + 3.0*q*q*rd*rd/(math.Pi*math.Pi)))
}

func expectedScore(w glocko2.Player, l glocko2.Player) float64 {
	gVal := expectedG(math.Sqrt(w.Rd*w.Rd + l.Rd*l.Rd))
	return 1.0 / (1.0 + math.Pow(10, -gVal*(w.R-l.R)/400.0))
}

func predictMatches(db map[string]int, ps []glocko2.Player, ms <-chan []int) float64 {
	n := 0
	s := 0.0
	for m := range ms {
		e := expectedScore(ps[m[0]], ps[m[1]])
		s += rateGame(1, e)
		n++
	}

	return (s / float64(n))
}

func predict(ts []tournament, ps []glocko2.Player, config Conf) float64 {
	n := len(ts)

	rank(ts[0:n-1], ps, config.Tau)
	return predictMatches(playerName, ps, tourneyMatches(n-1))
}

func constrain(v []float64) {
	v[0] = clamp(1200, v[0], 2500)
	v[1] = clamp(50, v[1], 450)
	v[2] = clamp(0.04, v[2], 0.1)
	v[3] = clamp(0.1, v[3], 1.5)
}

func mkOptFun(ts []tournament, ps []glocko2.Player) func([]float64) float64 {
	return func(v []float64) float64 {
		c := Conf{v[0], v[1], v[2], v[3]}
		cps := configPlayers(ps, c)
		return predict(ts, cps, c)
	}
}
