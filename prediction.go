package main

import "math"

const (
	predictionCount = 3
)

var (
	q = math.Log(10) / 400
)

func expectedG(rd float64) float64 {
	return 1.0 / math.Sqrt(1.0+3.0*q*q*rd*rd/(math.Pi*math.Pi))
}

func expectedScore(w, l player) float64 {
	gVal := expectedG(math.Sqrt(w.r*w.r + l.rd*l.rd))
	p := math.Pow(10.0, -gVal*(w.r-l.r)/400.0)
	return 1.0 / (1.0 + p)
}

func rateGame(y float64, expected float64) float64 {
	e := clamp(0.01, expected, 0.99)
	return -(y*math.Log10(e) + (1.0-y)*math.Log10(1.0-e))
}

type predictionData struct {
	p1, p2 int
	y      float64
}

func predictMatches(db map[string]int, ps []player, ms <-chan predictionData) float64 {
	n := 0
	s := 0.0
	for m := range ms {
		e := expectedScore(ps[m.p1], ps[m.p2])
		s += rateGame(m.y, e)
		n++
	}

	return (s / float64(n))
}

func activePlayers(lo, hi int) (players []int) {
	count := 0
	players = make([]int, 0)
	for pi := range matches {
		active := false
		for ti := lo; ti < hi; ti++ {
			if len(matches[pi][ti]) > 0 {
				active = true
				break
			}
		}

		if active {
			count++
			players = append(players, pi)
		}
	}

	return players
}

func tourneyMatches(lo, hi int, ps []int) chan predictionData {
	c := make(chan predictionData, 1000)

	go func() {
		defer close(c)

		count := 0
		for ti := lo; ti < hi; ti++ {
			for _, pi := range ps {
				for _, d := range matches[pi][ti] {
					c <- predictionData{
						p1: pi,
						p2: d.(duel).opponent,
						y:  d.(duel).outcome,
					}
					count++
				}
			}
		}
	}()

	return c
}

func runPredict(ts []tournament, ps []player, config conf) float64 {
	n := len(ts)
	activeCount := 3
	predictionCount := 3

	rank(ts[0:n-predictionCount-activeCount], ps, config.tau)
	active := activePlayers(n-predictionCount-activeCount, n-predictionCount)
	return predictMatches(playerName, ps, tourneyMatches(n-predictionCount, n, active))
}

func run(ts []tournament, ps []player, config conf) {
	rank(ts, ps, config.tau)
}
