package main

type match struct {
	winner string
	loser  string
}

type matchx struct {
	w player
	l player
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
			for _, duel := range matches[pi][t] {
				if duel.outcome == 1.0 {
					c <- []int{pi, duel.opponent}
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
