package main

import (
	"consensus/internal/db"
	"fmt"
	"net"
	"os"
	"strconv"
)

const (
	DB_LEN  = 5
	DB_NAME = "daniel"
)

var (
	knownHosts = []string{"silicon.cs.umanitoba.ca", "eagle.cs.umanitoba.ca", "hawk.cs.umanitoba.ca", "osprey.cs.umanitoba.ca"}
	localHosts = []string{"127.0.0.1", "127.0.0.1", "127.0.0.1", "127.0.0.1"}
	knownPort  = 16000
)

func main() {
	knownHosts = localHosts // TEMP

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
		rAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", localIP, port))
		self = &db.Peer{Addr: rAddr, Name: DB_NAME}
	} else {
		rAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", localIP, 0))
		self = &db.Peer{Addr: rAddr, Name: DB_NAME}
	}

	// Start up services

	peers := make([]*db.Peer, 0)
	data := make([]string, DB_LEN)
	db := db.Make(self, peers, data)
	db.Start()
}
