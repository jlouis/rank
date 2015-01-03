package main

import (
	"strconv"
	"sync"

	"github.com/jlouis/glicko2"
)

// closeRound() ends a tournament round by moving rankings to the player structure
func closeRound(scratch []playerScratch, ps []player) {
	for i := range ps {
		ps[i].r = scratch[i].r
		ps[i].rd = scratch[i].rd
		ps[i].sigma = scratch[i].sigma
	}
}

func rankChunk(ti int, lo int, hi int, ps []player, scratch []playerScratch, tau float64) {
	for i := lo; i < hi; i++ {
		player := ps[i]

		opponents := matches[i][ti]
		if opponents == nil {
			if player.active {
				player.rd = glicko2.Skip(player.r, player.rd, player.sigma)
			}

			scratch[i].r, scratch[i].rd, scratch[i].sigma = player.r, player.rd, player.sigma
		} else {
			r, rd, sigma := glicko2.Rank(player.r, player.rd, player.sigma, []glicko2.Opponent(opponents), tau)
			if player.active == false {
				ps[i].active = true
			}
			scratch[i].r = clamp(0, r, 3000)
			scratch[i].rd = clamp(0, rd, 400)
			scratch[i].sigma = clamp(0, sigma, 0.1)
		}
	}
}

// rank() ranks players for all matches
func rank(ts []tournament, ps []player, tau float64) {
	var wg sync.WaitGroup

	scratch := make([]playerScratch, len(ps))

	for ti := range ts {
		players = ps

		for i := 0; i < len(ps); i += 5000 {
			lo := i
			var hi int
			if i+5000 < len(ps) {
				hi = i + 5000
			} else {
				hi = len(ps)
			}

			wg.Add(1)
			go func() {
				defer wg.Done()

				rankChunk(ti, lo, hi, ps, scratch, tau)
			}()
		}

		wg.Wait()
		closeRound(scratch, ps)
		if *csvFile != "" {
			writeTournament(ti, ps)
		}
	}
}

func writeTournament(i int, players []player) {
	var m string
	if *duelMap == "" {
		m = "all"
	} else {
		m = *duelMap
	}

	for pi := range players {
		p := players[pi]
		if p.active {
			fields := []string{
				strconv.Itoa(i),
				p.name,
				m,
				strconv.FormatFloat(p.r, 'e', 6, 64),
				strconv.FormatFloat(p.rd, 'e', 6, 64),
				strconv.FormatFloat(p.sigma, 'e', 6, 64),
			}

			writeChan <- fields
		}
	}
}
