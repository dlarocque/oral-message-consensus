package main

import (
	"consensus/internal/db"
	"fmt"
	"os"
)

const (
	DB_LEN  = 5
	DB_HOST = "localhost"
	DB_PORT = 0 // OS assigned
	DB_NAME = "daniel"
)

var (
	knownHosts = []string{"silicon.cs.umanitoba.ca", "eagle.cs.umanitoba.ca", "hawk.cs.umanitoba.ca", "osprey.cs.umanitoba.ca"}
	knownPort  = 16000
)

func main() {
	// Start up services
	self := &db.Peer{Host: DB_HOST, Port: DB_PORT, Name: DB_NAME}
	peers := make([]*db.Peer, 0)

	for _, host := range knownHosts {
		wellKnownPeer, err := db.ConnectWithPeer(host, knownPort, host)
		if err != nil {
			fmt.Printf("Error connecting to well-known peer: %v", err)
			os.Exit(1)
		}
		peers = append(peers, wellKnownPeer)
	}

	data := make([]string, DB_LEN)
	db := db.Make(self, peers, data)
	db.Start()
}
