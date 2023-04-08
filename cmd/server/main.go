package main

import (
	"consensus/internal/db"
)

const (
	NUM_PEERS = 5
	NUM_DATA  = 5
	DB_HOST   = "localhost"
	DB_PORT   = 0 // OS assigned
	DB_NAME   = "daniel"
)

func main() {
	// Start up services
	self := &db.Peer{Host: DB_HOST, Port: DB_PORT, Name: DB_NAME}
	peers := make([]db.Peer, NUM_PEERS)
	data := make([]int, NUM_DATA)

	db := db.Make(self, peers, data)
	db.Start()
}
