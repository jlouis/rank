package main

import (
	"database/sql"
	_ "github.com/bmizerany/pq"
)

type tournament struct {
	id string
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
		var id string
		if err := rows.Scan(&id); err != nil {
			panic(err)
		}

		result = append(result, tournament{id})
	}

	return result
}

func getPlayers(db *sql.DB) []player {
	result := make([]player, 0, 100*1000)
	rows, err := db.Query("SELECT id, name FROM player")
	if err != nil {
		panic(err)
	}

	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id); err != nil {
			panic(err)
		}

		if err := rows.Scan(&name); err != nil {
			panic(err)
		}

		result = append(result, player{id, name})
	}

	return result
}

func main() {
	db := dbConnect()
	err := db.Ping()
	if err != nil {
		panic(err)
	}
	defer db.Close()
}
