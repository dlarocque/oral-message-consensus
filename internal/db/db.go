package db

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func Make(self *Peer, peers []*Peer, data []string) *Db {
	db := &Db{
		self:  self,
		peers: peers,
		data:  data,
	}

	rand.Seed(time.Now().UnixNano())

	Debug(dInfo, "Created DB node (%v, %v, %v) with %v peers", self.Host, self.Port, self.Name, len(peers))
	return db
}

func (db *Db) Start() {
	db.join()

	// Start services
	go db.peerListener()
	go db.gossipper()

	// Listen for incoming clients
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

		go db.handleConnection(conn)
	}
}

func (db *Db) handleConnection(conn net.Conn) {
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

// Should not require mutex since should be only thread running
func (db *Db) join() {
	Debug(dInfo, "Attempting to join network")
	args := &GossipArgs{
		Command:   GOSSIP,
		Host:      db.self.Host,
		Port:      db.self.Port,
		Name:      db.self.Name,
		MessageID: uuid.New(),
	}

	jsonBytes, err := args.Json()
	if err != nil {
		Debug(dError, "Join: %v", err)
		panic(err)
	}

	// Send gossip to all the peers
	for _, peer := range db.peers {
		db.Send(peer, jsonBytes)
	}
}

// Send data to a peer with mutex held
func (db *Db) Send(peer *Peer, jsonData []byte) error {
	// Check if the connection has expired
	if peer.Expires.After(time.Now()) {
		Debug(dPeer, "Connection has expired with peer %s, not sending", peer.ToString())

		// Remove the expired peer from the list of peers
		var updatedPeers []*Peer
		for _, oldPeer := range db.peers {
			if oldPeer != peer {
				updatedPeers = append(updatedPeers, oldPeer)
			}
		}

		db.peers = updatedPeers
		Debug(dPeer, "Removed expired peer from list of peers")

		return nil
	}

	_, err := peer.Conn.Write(jsonData)
	if err != nil {
		Debug(dError, "Error sending data to peer %s: %v", peer.ToString(), err)
		return err
	}

	Debug(dPeer, "Sent message to peer %s", peer.ToString())
	return nil
}

func (db *Db) peerListener() {
	Debug(dInfo, "Peer listener started")
	addr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		Debug(dError, "UDP: Error resolving address: %v", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		Debug(dError, "UDP: Error listening: %v", err)
		return
	}
	defer conn.Close()

	fmt.Println("Listening for UDP connections on", conn.LocalAddr())

	buf := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			Debug(dError, "UDP: Error reading: %v", err)
			continue
		}

		Debug(dPeer, "Received %d bytes from %s: %s\n", n, addr.String(), string(buf[:n]))

		var jsonMsg map[string]interface{}
		err = json.Unmarshal(buf[:n], &jsonMsg)
		if err != nil {
			Debug(dError, "Failed to parse peers message into JSON: %v", err)
		}

		if command, ok := jsonMsg["command"]; ok {
			if command == GOSSIP {
				Debug(dPeer, "Received GOSSIP")

				args, err := parseGossipReply(buf[:n])
				if err != nil {
					Debug(dError, "Failed to parse GOSSIP command JSON: %v", err)
					return
				}

				db.mu.Lock()

				// Check if first time seeing peer
				isNew := true
				peerIdx := 0
				for idx, peer := range db.peers {
					if peer.Host == args.Host && peer.Port == args.Port {
						isNew = false
						peerIdx = idx
					}
				}

				if isNew {
					Debug(dPeer, "First time seeing peer")

					// Add to list of peers
					newPeer, err := ConnectWithPeer(args.Host, args.Port, args.Name)
					if err != nil {
						Debug(dError, "Failed to connect with new peer at (%s:%d)", args.Host, args.Port)
					}

					db.peers = append(db.peers, newPeer)

					// Relay message to 5 other randomly selected peers
					Debug(dGossip, "Relaying gossip to 5 random peers")
					perm := rand.Perm(len(db.peers) - 1)
					for cnt, randomIdx := range perm {
						if cnt >= 5 {
							break
						}

						if db.peers[randomIdx] != newPeer {
							db.Send(db.peers[randomIdx], buf[:n])
						}
					}

					// Reply to the new peer with our own information
					reply := &GossipReply{
						Command: GOSSIP,
						Host:    db.self.Host,
						Port:    db.self.Port,
						Name:    db.self.Name,
					}

					replyData, err := reply.Json()
					if err != nil {
						Debug(dError, "Failed to encode Gossip reply to JSON: %v", err)
					}

					go db.Send(newPeer, replyData)
				} else {
					Debug(dPeer, "Peer already exists")
					// Update expiration date
					db.peers[peerIdx].Expires = time.Now().Add(time.Second * 2)

					// Relay message to 3 other peers
					Debug(dGossip, "Relaying gossip to 3 random peers")
					perm := rand.Perm(len(db.peers) - 1)
					for cnt, randomIdx := range perm {
						if cnt >= 3 {
							break
						}

						if db.peers[randomIdx] != db.peers[peerIdx] {
							db.Send(db.peers[randomIdx], buf[:n])
						}
					}
				}

				db.mu.Unlock()
			} else {
				Debug(dPeer, "Received unknown command from peer")
			}
		}
	}
}

// Goroutine that periodically sends gossip messages to all active peers
func (db *Db) gossipper() {
	Debug(dInfo, "Gossipper started")
	for {
		Debug(dGossip, "Sending Gossip messages to all peers")
		args := &GossipArgs{
			Command:   GOSSIP,
			Host:      db.self.Host,
			Port:      db.self.Port,
			Name:      db.self.Name,
			MessageID: uuid.New(),
		}

		jsonBytes, err := args.Json()
		if err != nil {
			Debug(dError, "Join: %v", err)
			panic(err)
		}

		db.mu.Lock()
		// Send gossip to all the peers
		for _, peer := range db.peers {
			db.Send(peer, jsonBytes)
		}

		db.mu.Unlock()

		Debug(dInfo, "Gossipper sleeping for 1s")
		time.Sleep(time.Second * 1)
	}
}

// Retrieve all values from the database
func (db *Db) query(args QueryArgs) []string {
	db.mu.Lock()
	defer db.mu.Unlock()

	return db.data
}

// Goroutine that sets index in the database to new value, and sends
// a SET command to all active peers.
func (db *Db) set(index int, value string) {
	/*
		db.mu.Lock()
		db.data[index] = value
		db.mu.Unlock()

		args := &SetArgs{
			Command: SET,
			Index:   index,
			Value:   value,
		}

		for _, peer := range db.peers {
			go db.sendSet(peer, args)
		}
	*/

}
