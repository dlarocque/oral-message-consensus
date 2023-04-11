package main

import (
	"consensus/internal/db"
	"fmt"
	"os"
	"strconv"
)

const (
	DB_LEN  = 5
	DB_NAME = "daniel"
)

var (
	knownHosts = []string{"silicon.cs.umanitoba.ca", "eagle.cs.umanitoba.ca", "hawk.cs.umanitoba.ca", "osprey.cs.umanitoba.ca"}
	knownPort  = 16000
)

func main() {
	localIP, err := db.GetLocalIP()
	if err != nil {
		fmt.Printf("Failed to get local IP: %v\n", err)
	}

	var self *db.Peer
	// Optionally accept port number from a command line argument
	if len(os.Args) > 1 {
		port, err := strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Println("Invalid port number:", os.Args[1])
		}

		self = &db.Peer{Host: localIP, Port: port, Name: DB_NAME}
	} else {
		self = &db.Peer{Host: localIP, Port: 0, Name: DB_NAME}
	}

	// Start up services

	peers := make([]*db.Peer, 0)
	data := make([]string, DB_LEN)
	db := db.Make(self, peers, data)
	db.Start()
}
