package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"time"

	_ "github.com/bmizerany/pq"
	"github.com/jlouis/glicko2"
	"github.com/jlouis/nmoptim"
)

// Representation of players in memory. We simply store the important values directly in a flat struct
type player struct {
	id     string
	name   string
	r      float64
	rd     float64
	sigma  float64
	active bool
}

// Duels represent duel matches. The player for which the duel pertains is implicit. We only track the
// index of the opponent and the outcome of the match (1.0 is a win, 0.0 is a loss).
type duel struct {
	opponent int
	outcome  float64
}

// Tournaments have a counter which represents their number in order
type tournament struct {
	id int
}

// Conf is a configuration object
// It tracks the configuration of the players.
type conf struct {
	r     float64 // starting rating for new players
	rd    float64 // starting rating deviation for new players
	sigma float64 // starting volatility (σ) for new players
	tau   float64 // The τ value used in the computations
}

// playerScratch is a temporary scratch-pad to track new rankings for a given tournament
// we move these stats into the right structure when the tournament round is over.
// We do this to avoid overwriting the current ratings while processing a new tournament.
type playerScratch struct {
	r     float64
	rd    float64
	sigma float64
}

const (
	initialPlayerCount = 150 * 1000
)

var (
	playerID   map[string]int // Mapping from the player UUID → index position in the slice
	playerName map[string]int // Mapping from the player name → index position in the slice

	players []player               // Global list of players
	matches [][][]glicko2.Opponent // [p][t] → []duel mapping, where p is a player index and t is a tournament number

	topPlayers = []string{"rapha", "Cypher", "DaHanG", "evil", "k1llsen", "nhd", "tox", "Av3k", "Fraze", "_ash", "Cooller"}
)

// The flags to the application have their separate variable section
var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile = flag.String("memprofile", "", "write memory profile to this file")

	dbUser   = flag.String("user", "qlglicko_rank", "database user to connect as")
	dbPasswd = flag.String("passwd", "'AAAAAAAAAAAAAAAAAaaaaaaaaaaaaaaaaand-OPEN!'", "database password to use")
	dbName   = flag.String("db", "qlglicko", "database to connect to")
	dbHost   = flag.String("host", "192.168.1.201", "the host to connect to")

	tournamentCount = flag.Int("tourneys", 10, "how many tournaments to process")
	optimize        = flag.Bool("optimize", false, "run prediction code for optimization")
	duelMap         = flag.String("map", "", "the map for which to rank. All maps if not set")
	csvFile         = flag.String("outfile", "", "the file to write results into")
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

func addMatch(t int, winner string, loser string) {
	wi := playerID[winner]
	li := playerID[loser]

	if matches[wi][t] == nil {
		matches[wi][t] = make([]glicko2.Opponent, 0)
	}
	matches[wi][t] = append(matches[wi][t], duel{opponent: li, outcome: 1.0})

	if matches[li][t] == nil {
		matches[li][t] = make([]glicko2.Opponent, 0)
	}
	matches[li][t] = append(matches[li][t], duel{opponent: wi, outcome: 0.0})
}

// readMatches reads matches from the database
func readMatches(db *sql.DB, t int) {
	var rows *sql.Rows
	var err error
	if *duelMap == "" {
		rows, err = db.Query("SELECT winner, loser FROM duel_match dm, tournament t WHERE t.id = $1 AND played BETWEEN t.t_from AND t.t_to", t)
	} else {
		rows, err = db.Query("SELECT winner, loser FROM duel_match dm, tournament t WHERE t.id = $1 AND played BETWEEN t.t_from AND t.t_to AND dm.map = $2", t, *duelMap)
	}

	if err != nil {
		panic(err)
	}

	for rows.Next() {
		var winner, loser string
		if err := rows.Scan(&winner, &loser); err != nil {
			panic(err)
		}

		addMatch(t-1, winner, loser) // Matches are skewed by a count of 1
	}

}

func readTournaments(db *sql.DB) []tournament {
	result := make([]tournament, 0, 250)
	rows, err := db.Query("SELECT id FROM tournament ORDER BY t_from ASC LIMIT $1", *tournamentCount)
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

	log.Printf("Read %v tournaments", len(result))
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
func initialize() ([]tournament, []player, [][][]glicko2.Opponent) {
	playerID = make(map[string]int)
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
		playerID[id] = i
		playerName[name] = i
	}

	matches = make([][][]glicko2.Opponent, len(ps))
	for i := range matches {
		matches[i] = make([][]glicko2.Opponent, len(ts))
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

// constrain will bound the optimization simplex to sensible values
func constrain(v []float64) {
	v[0] = clamp(50, v[0], 450)
	v[1] = clamp(0.1, v[1], 1.5)
}

// mkOptFun creates the optimization function which is used to optimize the results
func mkOptFun(ts []tournament, ps []player) func([]float64) float64 {
	return func(v []float64) float64 {
		c := conf{1200, v[0], 0.06, v[1]}
		cps := configPlayers(ps, c)
		return runPredict(ts, cps, c)
	}
}

func main() {
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}

		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	var doneChan chan interface{}

	if *csvFile != "" {
		if *optimize {
			panic("Can't optimize and write CSV file at the same time")
		}

		_, doneChan = initWriter()
	}

	log.Print("=== INITIALIZE")
	ts, ps, ms := initialize()
	matches = ms
	log.Print("=== PREDICT")
	c := conf{1200, 285, 0.06, 0.59}
	cps := configPlayers(ps, c)
	run(ts, cps, c)

	for _, ply := range topPlayers {
		log.Printf("%v → %v\n", ply, cps[playerName[ply]])
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}

		pprof.WriteHeapProfile(f)
		f.Close()
		return
	}

	if *optimize {
		log.Print("=== OPTIMIZE")
		start := [][]float64{
			{350.0, 0.5},
			{150.0, 0.8},
			{400.0, 0.1}}
		f := mkOptFun(ts, ps)
		startTime := time.Now()
		vals, iters, evals := nmoptim.Optimize(f, start, constrain)
		elapsed := time.Since(startTime)
		log.Printf("Optimized to %v in %v iterations and %v evaluations. Took %v\n", vals, iters, evals, elapsed)
	}

	log.Print("=== FLUSHING")
	if *csvFile != "" {
		close(writeChan)
		<-doneChan
	}
	log.Print("=== DONE")
}
