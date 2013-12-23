package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/bmizerany/pq"
	"github.com/jlouis/glocko2"
	"github.com/jlouis/nmoptim"
	"sync"
)

// Conf is a configuration object
type Conf struct {
	R     float64
	Rd    float64
	Sigma float64
	Tau   float64
}

// playerScratch is a temporary scratch-pad to track new rankings for a given tournament
// we move these stats into the right structure when the tournament round is over
type playerScratch struct {
	r     float64
	rd    float64
	sigma float64
}

const (
	initialPlayerCount = 150 * 1000
)

var (
	playerMap  map[string]int // Mapping from the player name → index position in the slice
	playerName map[string]int // Mapping from the player name → index position in the slice

	matches [][][]glocko2.Opponent

	topPlayers []string = []string{"rapha", "Cypher", "DaHanG", "evil", "k1llsen", "nhd", "tox", "Av3k", "Fraze", "_ash", "Cooller"}

	optimize = flag.Bool("optimize", false, "run prediction code for optimization")
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

func rankChunk(ti int, lo int, hi int, ps []glocko2.Player, scratch []playerScratch, tau float64) {
	for i := lo; i < hi; i++ {
		player := ps[i]

		opponents := matches[i][ti]
		if opponents == nil {
			if player.Active {
				mu, phi := glocko2.Scale(player.R, player.Rd)
				phi = glocko2.PhiStar(player.Sigma, phi)
				_, phi = glocko2.Unscale(mu, phi)
				player.Rd = phi
			}

			scratch[i].r, scratch[i].rd, scratch[i].sigma = player.R, player.Rd, player.Sigma
		} else {
			r, rd, sigma := player.Rank(opponents, ps, tau)
			if player.Active == false {
				ps[i].Active = true
			}
			scratch[i].r = clamp(0, r, 3000)
			scratch[i].rd = clamp(0, rd, 400)
			scratch[i].sigma = clamp(0, sigma, 0.1)
		}
	}
}

// rank() ranks players for all matches
func rank(ts []tournament, ps []glocko2.Player, tau float64) {
	var wg sync.WaitGroup

	scratch := make([]playerScratch, len(ps))

	for ti := range ts {
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
	}
}

// closeRound() ends a tournament round by moving rankings to the player structure
func closeRound(scratch []playerScratch, ps []glocko2.Player) {
	for i := range ps {
		ps[i].R = scratch[i].r
		ps[i].Rd = scratch[i].rd
		ps[i].Sigma = scratch[i].sigma
	}
}

func getOpponents(t int, p int) []glocko2.Opponent {
	os := matches[p][t]

	return os
}

type tournament struct {
	id int
}

type player struct {
	id   string
	name string
}

func dbConnect() *sql.DB {
	conn, err := sql.Open("postgres", "user=qlglicko_rank password=shijMebs dbname=qlglicko host=dragon.lan sslmode=disable")
	if err != nil {
		panic(err)
	}

	return conn
}

func getTournaments(db *sql.DB) []tournament {
	result := make([]tournament, 0, 250)
	rows, err := db.Query("SELECT id FROM tournament ORDER BY t_from ASC")
	if err != nil {
		panic(err)
	}

	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			panic(err)
		}

		result = append(result, tournament{id})
	}

	return result
}

func getMatches(db *sql.DB, t int) {
	rows, err := db.Query("SELECT winner, loser, map FROM duel_match dm, tournament t WHERE t.id = $1 AND played BETWEEN t.t_from AND t.t_to", t)
	if err != nil {
		panic(err)
	}

	for rows.Next() {
		var winner, loser, m string
		if err := rows.Scan(&winner, &loser, &m); err != nil {
			panic(err)
		}

		addMatch(t-1, winner, loser, m) // Matches are skewed by a count of 1
	}

}

func addMatch(t int, winner string, loser string, m string) {
	wi := playerMap[winner]
	li := playerMap[loser]

	if matches[wi][t] == nil {
		matches[wi][t] = make([]glocko2.Opponent, 0)
	}
	matches[wi][t] = append(matches[wi][t], glocko2.Opponent{li, 1.0})

	if matches[li][t] == nil {
		matches[li][t] = make([]glocko2.Opponent, 0)
	}
	matches[li][t] = append(matches[li][t], glocko2.Opponent{wi, 0.0})
}

func getPlayers(db *sql.DB) []glocko2.Player {
	players := make([]glocko2.Player, 0, initialPlayerCount)
	rows, err := db.Query("SELECT id, name FROM player")
	if err != nil {
		panic(err)
	}

	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			panic(err)
		}

		players = append(players, glocko2.Player{id, name, 0.0, 0.0, 0.0, false})
	}

	return players
}

func setup() ([]tournament, []glocko2.Player, [][][]glocko2.Opponent) {
	playerMap = make(map[string]int)
	playerName = make(map[string]int)

	db := dbConnect()
	err := db.Ping()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ts := getTournaments(db)

	ps := getPlayers(db)
	for i := range ps {
		id := ps[i].Id
		name := ps[i].Name
		playerMap[id] = i
		playerName[name] = i
	}

	matches = make([][][]glocko2.Opponent, len(ps))
	for i := range matches {
		matches[i] = make([][]glocko2.Opponent, len(ts))
	}

	for i := range ts {
		fmt.Printf("Getting matches for tournament: %v\n", ts[i].id)
		getMatches(db, ts[i].id)
	}

	return ts, ps, matches
}

func configPlayers(ps []glocko2.Player, c Conf) []glocko2.Player {
	r := make([]glocko2.Player, len(ps))
	copy(r, ps)

	for i := range ps {
		r[i].R = c.R
		r[i].Rd = c.Rd
		r[i].Sigma = c.Sigma
		r[i].Active = false
	}

	return r
}

func tourneyMatches(t int) chan []int {
	c := make(chan []int)

	go func() {
		for pi := range matches {
			for _, opp := range matches[pi][t] {
				if opp.Sj == 1.0 {
					c <- []int{pi, opp.Idx}
				}
			}
		}
		close(c)
	}()
	return c
}

func main() {
	flag.Parse()

	ts, ps, _ := setup()
	c := Conf{1200, 325, 0.06, 0.1}
	cps := configPlayers(ps, c)
	predict(ts, cps, c)

	for _, ply := range topPlayers {
		fmt.Printf("%v → %v\n", ply, cps[playerName[ply]])
	}

	if *optimize {
		start := [][]float64{
			{350.0, 0.5},
			{150.0, 0.8},
			{400.0, 0.1}}
		f := mkOptFun(ts, ps)
		vals, iters, evals := nmoptim.Optimize(f, start, constrain)
		fmt.Printf("Optimized to %v in %v iterations and %v evaluations\n", vals, iters, evals)
	}

}
