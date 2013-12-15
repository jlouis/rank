package main

import (
	"database/sql"
	"fmt"
	_ "github.com/bmizerany/pq"
	"github.com/jlouis/glocko2"
)

// Conf is a configuration object
type Conf struct {
	initR	float64
	initRd float64
	initSigma float64
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

// rank() ranks players for all matches
func rank(ts []tournament, ps []glocko2.Player) {

	ranked := 0
	activated := 0

	scratch := make([]playerScratch, len(ps))

	for ti := range ts {
		fmt.Printf("Ranking: %v", ti)
		for pi, player := range ps {
			if pi%1000 == 0 {
				fmt.Printf(".")
			}
			opponents := matches[pi][ti]
			/*
			if player.Name == "rapha" {
				fmt.Printf("Tournament %v\n", ti)
				fmt.Printf("  Opponents:\n")
				for _, o := range(opponents) {
					fmt.Printf("    %v Score: %v\n", ps[o.Idx].Name, o.Sj)
				}
			}
			*/
			if opponents == nil {
				if player.Active {
					mu, phi := glocko2.Scale(player.R, player.Rd)
					phi = glocko2.PhiStar(player.Sigma, phi)
					_, phi = glocko2.Unscale(mu, phi)
					player.Rd = phi
				}
				scratch[pi].r, scratch[pi].rd, scratch[pi].sigma = player.R, player.Rd, player.Sigma
			} else {
				r, rd, sigma := player.Rank(opponents, ps)
				if player.Active == false {
					ps[pi].Active = true
					activated++
				}
				ranked++
				scratch[pi].r = clamp(0, r, 3000)
				scratch[pi].rd = clamp(0, rd, 400)
				scratch[pi].sigma = clamp(0, sigma, 0.1)
			}
		}
		fmt.Printf("x\n")
		closeRound(scratch, ps)
		fmt.Printf("Ranked %v playes and activated %v new players\n", ranked, activated)
		activated = 0
		ranked = 0
		fmt.Printf("Rapha's strength: %v\n", ps[playerName["rapha"]])
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

func getPlayers(db *sql.DB, c *Conf) []glocko2.Player {
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

		players = append(players, glocko2.Player{id, name, c.initR, c.initRd, c.initSigma, false})
	}

	return players
}

func runRank(c Conf) {
	playerMap = make(map[string]int)
	playerName = make(map[string]int)

	db := dbConnect()
	err := db.Ping()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ts := getTournaments(db)

	ps := getPlayers(db, &c)
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

	rank(ts, ps)
}

func main() {
	runRank(Conf{1500.0, 350.0, 0.06})
}
