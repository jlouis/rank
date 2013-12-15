// prediction code
package main

import (
	"math"
	"github.com/jlouis/glocko2"
)

var (
	q = math.Log(10) / 400
)

type match struct {
	winner string
	loser  string
}

type configuration struct {
	initR		float64
	initRd	float64
	initSigma	float64
	tau		float64
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

func predictMatches(db map[string]int, ps []glocko2.Player, ms []match) float64 {
	r := make([]float64, len(ms))

	for i, m := range ms {
		r[i] = expectedScore(ps[ db[m.winner] ], ps [ db[m.loser] ])
	}

	n := float64(len(r))
	s := 0.0
	for _, e := range r {
		s += rateGame(1, e)
	}
	return (s / n)
}

/*
func rank(ts []tournament, ps []player, config configuration) float64 {
	n := len(ts)
	splitPoint := n - 1
	
	// ps = Rank(ts[0:sp], ps, config)
	return predictMatches(db, ps, ts.Matches(i))
}
*/

func constrain(v []float64) []float64 {
	return []float64{ clamp(1200, v[0], 2000), clamp(50, v[1], 450), clamp(0.04, v[2], 0.09), clamp(0.1, v[3], 1.5) }
}

/*
func mkOptFun(ts []tournament, ps []glocko2.Player) {
	return func(v []float64) float64 {
		c := configuration{v[0], v[1], v[2], v[3]}
		return rank(ts, ps, c)
	}
}
*/