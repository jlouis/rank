package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"

	_ "github.com/bmizerany/pq"
)

type player struct {
	id string
	name string
	r	float64
	rd	float64
	sigma	float64
	active	bool
}

type duel struct {
	opponent	int
	outcome	float64
}

type tournament struct {
	id int
}

// Conf is a configuration object
type conf struct {
	r     float64
	rd    float64
	sigma float64
	tau   float64
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
	playerId  map[string]int // Mapping from the player UUID → index position in the slice
	playerName map[string]int // Mapping from the player name → index position in the slice

	players []player // Global list of players
	matches [][][]duel // [p][t] → []duel mapping, where p is a player index and t is a tournament number

	topPlayers []string = []string{"rapha", "Cypher", "DaHanG", "evil", "k1llsen", "nhd", "tox", "Av3k", "Fraze", "_ash", "Cooller"}
)

// The flags to the application have their separate variable section
var (
	dbUser = flag.String("user", "qlglicko_rank", "database user to connect as")
	dbPasswd = flag.String("passwd", "'AAAAAAAAAAAAAAAAAaaaaaaaaaaaaaaaaand-OPEN!'", "database password to use")
	dbName = flag.String("db", "qlglicko", "database to connect to")
	dbHost = flag.String("host", "192.168.1.201", "the host to connect to")

	optimize = flag.Bool("optimize", false, "run prediction code for optimization")
)

func (d duel) R() float64 {
	return players[d.opponent].r
}

func (d duel) RD() float64 {
	return players[d.opponent].rd
}

func (d duel) Sigma() float64 {
	return players[d.opponent].sigma
}

func (d duel) SJ() float64 {
	return d.outcome
}

// dbConnect connects us to the database via command line flags
func dbConnect() *sql.DB {
	connStr := fmt.Sprintf("user=%s password=%s dbname=%s host=%s sslmode=disable", *dbUser, *dbPasswd, *dbName, *dbHost)
	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	return conn
}

func addMatch(t int, winner string, loser string, m string) {
	wi := playerId[winner]
	li := playerId[loser]

	if matches[wi][t] == nil {
		matches[wi][t] = make([]duel, 0)
	}
	matches[wi][t] = append(matches[wi][t], duel{opponent: li, outcome: 1.0})

	if matches[li][t] == nil {
		matches[li][t] = make([]duel, 0)
	}
	matches[li][t] = append(matches[li][t], duel{opponent: wi, outcome: 0.0})
}

// readMatches reads matches from the database
func readMatches(db *sql.DB, t int) {
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

func readTournaments(db *sql.DB) []tournament {
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

func getPlayers(db *sql.DB) []player {
	players := make([]player, 0, initialPlayerCount)
	rows, err := db.Query("SELECT id, name FROM player")
	if err != nil {
		panic(err)
	}

	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			panic(err)
		}

		players = append(players, player{id, name, 0.0, 0.0, 0.0, false})
	}

	return players
}

// initialize will set up initial data structures needed to carry out the computations
func initialize() ([]tournament, []player, [][][]duel) {
	playerId = make(map[string]int)
	playerName = make(map[string]int)

	db := dbConnect()
	err := db.Ping()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	log.Print("Reading tournaments")
	ts := readTournaments(db)

	log.Print("Reading players")
	ps := getPlayers(db)
	log.Printf("Populating player structure with %v players", len(ps))

	for i := range ps {
		id := ps[i].id
		name := ps[i].name
		playerId[id] = i
		playerName[name] = i
	}

	matches = make([][][]duel, len(ps))
	for i := range matches {
		matches[i] = make([][]duel, len(ts))
	}

	for i := range ts {
		fmt.Printf("Getting matches for tournament: %v\n", ts[i].id)
		readMatches(db, ts[i].id)
	}

	return ts, ps, matches
}


func configPlayers(ps []player, c conf) []player {
       r := make([]player, len(ps))
       copy(r, ps)

       for i := range ps {
               r[i].r = c.r
               r[i].rd = c.rd
               r[i].sigma = c.sigma
               r[i].active = false
       }
 
       return r
}


func main() {
	flag.Parse()

	log.Print("=== INITIALIZE")
	ts, ps, ms := initialize()
	matches = ms
	log.Print("=== PREDICT")
	c := conf{1200, 264, 0.06, 0.26}
	cps := configPlayers(ps, c)
	predict(ts, cps, c)

//	for _, ply := range topPlayers {
//		fmt.Printf("%v → %v\n", ply, cps[playerName[ply]])
//	}

//	if *optimize {
//		start := [][]float64{
//			{350.0, 0.5},
//			{150.0, 0.8},
//			{400.0, 0.1}}
//		f := mkOptFun(ts, ps)
//		vals, iters, evals := nmoptim.Optimize(f, start, constrain)
//		fmt.Printf("Optimized to %v in %v iterations and %v evaluations\n", vals, iters, evals)
//	}

}
