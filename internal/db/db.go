package db

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

func Make(self *Peer, peers []Peer, data []int) *Db {
	db := &Db{
		self:  self,
		peers: peers,
		data:  data,
	}

	return db
}

func (db *Db) Start() {

}

// Listens, parses, and executes commands
func ClientListener() {
	fmt.Println("Listening for incoming CLI connections...")

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("An error occurred while starting the server:", err)
		os.Exit(1)
	}

	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("An error occurred while accepting a connection:", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	Debug(dClient, "New client connected: %v", conn.RemoteAddr())
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		fmt.Fprint(conn, ">> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("An error occurred while reading input. Please try again.")
			continue
		}

		input = strings.TrimSpace(input)

		switch input {
		case "peers":
			// TODO: Implement 'peers' command.
		case "current":
			// TODO: Implement 'current' command.
		case "lie":
			// TODO: Implement 'lie' command.
		case "truth":
			// TODO: Implement 'truth' command.
		case "exit":
			fmt.Fprintln(conn, "Exited.")
			return
		default:
			if strings.HasPrefix(input, "consensus ") {
				index, err := strconv.Atoi(input[10:])
				if err != nil || index < 0 || index > 4 {
					fmt.Fprintln(conn, "Invalid input. Please provide a single digit integer index in the range 0-4.")
					continue
				}
				fmt.Fprintln(conn, "Running consensus for index", index)
				// TODO: Implement 'consensus' command.
			} else if strings.HasPrefix(input, "set ") {
				parts := strings.Split(input, " ")
				if len(parts) != 3 {
					fmt.Fprintln(conn, "Invalid input. Please provide both index and word to set.")
				} else {
					index, err := strconv.Atoi(parts[1])
					if err != nil || index < 0 || index > 4 {
						fmt.Fprintln(conn, "Invalid input. Please provide a single digit integer index in the range 0-4.")
						continue
					}
					word := parts[2]
					fmt.Fprintln(conn, "Sending SET command to all peers. Setting index", index, "to word", word)
					// TODO: Implement 'set' command.
				}
			} else {
				fmt.Fprintln(conn, "Invalid input. Please try again.")
			}
		}
	}
}

func PeerListener() {
	// Resolve the UDP address to listen on.
	addr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		Debug(dError, "UDP: Error resolving address:", err)
		return
	}

	// Create a listener for UDP connections on an OS designated port.
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		Debug(dError, "UDP: Error listening:", err)
		return
	}
	defer conn.Close()

	// Get the address and port the listener is bound to.
	fmt.Println("Listening for UDP connections on", conn.LocalAddr())

	// Create a buffer to read incoming data.
	buf := make([]byte, 1024)

	// Continuously read incoming data.
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			Debug(dError, "UDP: Error reading:", err)
			continue
		}

		// Handle the incoming data.
		Debug(dPeer, "Received %d bytes from %s: %s\n", n, addr, string(buf[:n]))
	}
}

// Retrieve all values from the database
func query(args QueryArgs) {

}

// Goroutine that sets index in the database to new value, and sends
// a SET command to all active peers.
func set(index int, value string) {

}

func Gossipper() {

}

func GossipListener() {

}
