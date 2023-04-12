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

var (
	knownHosts = []string{"silicon.cs.umanitoba.ca", "eagle.cs.umanitoba.ca", "hawk.cs.umanitoba.ca", "osprey.cs.umanitoba.ca"}
	knownPort  = 16000
	localHosts = []string{"127.0.0.1", "127.0.0.1", "127.0.0.1", "127.0.0.1"}
	localPorts = []int{16000, 16001, 16002, 16003}
)

func Make(self *Peer, peers []*Peer, data []string) *Db {
	db := &Db{
		self:  self,
		peers: peers,
		data:  data,
	}

	rand.Seed(time.Now().UnixNano())

	Debug(dInfo, "Created DB node (%v, %v) with %v peers", self.Addr.IP, self.Name, self.Name, len(peers))
	return db
}

func (db *Db) Start() {
	// Start listening to peer messages on OS-assigned UDP port
	Debug(dInfo, "Peer listener started")
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", db.self.Addr.Port))
	if err != nil {
		Debug(dError, "UDP: Error resolving address: %v", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		Debug(dError, "UDP: Error listening: %v", err)
		return
	}

	db.conn = conn

	fmt.Printf("Peer listener started on port %v\n", db.self.Addr.Port)

	// Asychronously handle peer messages
	go db.peerListener(conn)

	db.join()

	// Start services
	go db.gossipper()
	go db.peerMaintainer()

	// Listen for incoming clients
	fmt.Println("Listening for incoming CLI connections...")

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", db.self.Addr.Port+1000))
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
			db.mu.Lock()
			res := db.peers.String()
			fmt.Fprintln(conn, res)
			db.mu.Unlock()
		case "current":
			db.mu.Lock()
			res := fmt.Sprintf("%v", db.data)
			fmt.Fprintln(conn, res)
			db.mu.Unlock()
		case "lie":
			// TODO: Implement 'lie' command
		case "truth":
			// TODO: Implement 'truth' command
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
					fmt.Fprintln(conn, "Invalid input. Please provide both index and value to set.")
				} else {
					index, err := strconv.Atoi(parts[1])
					if err != nil || index < 0 || index > 4 {
						fmt.Fprintln(conn, "Invalid input. Please provide a single digit integer index in the range 0-4.")
						continue
					}
					value := parts[2]
					fmt.Fprintln(conn, "Sending SET command to all peers. Setting index", index, "to value", value)

					go db.set(index, value)
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
		Host:      db.self.Addr.IP.String(),
		Port:      db.self.Addr.Port,
		Name:      db.self.Name,
		MessageID: uuid.New(),
	}

	jsonBytes, err := args.Json()
	Debug(dGossip, "Gossip message: %s", string(jsonBytes))
	if err != nil {
		Debug(dError, "Join: %v", err)
		panic(err)
	}

	db.mu.Lock()
	db.gossipIDs = append(db.gossipIDs, args.MessageID)

	// Send gossip to all the peers
	// TEMP
	for idx, host := range localHosts {
		wellKnownPeer, err := db.ConnectWithPeer(host, localPorts[idx], host)
		if err != nil {
			fmt.Printf("Error connecting to well-known peer: %v\n", err)
			os.Exit(1)
		}

		db.Send(wellKnownPeer, jsonBytes)
	}
	db.mu.Unlock()

	Debug(dInfo, "Successfully sent messages to all well-known peers")
}

// Send data to a peer with mutex held
func (db *Db) Send(peer *Peer, jsonData []byte) error {
	_, err := db.conn.WriteToUDP(jsonData, peer.Addr)
	if err != nil {
		Debug(dError, "Error sending data to peer %s: %v", peer.String(), err)
		return err
	}

	return nil
}

func (db *Db) peerListener(conn *net.UDPConn) {
	fmt.Println("Listening for UDP connections on", conn.LocalAddr())

	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			Debug(dError, "UDP: Error reading: %v", err)
			continue
		}

		fmt.Printf("Received message from port %d\n", addr.Port)

		// Debug(dPeer, "Received %d bytes from %s: %s\n", n, addr.String(), string(buf[:n]))

		var jsonMsg map[string]interface{}
		err = json.Unmarshal(buf[:n], &jsonMsg)
		if err != nil {
			Debug(dError, "Failed to parse peers message into JSON: %v", err)
		}

		if command, ok := jsonMsg["command"]; ok {
			db.mu.Lock()
			if command == GOSSIP {
				args, err := parseGossipArgs(buf[:n])
				if err != nil {
					Debug(dError, "Failed to parse GOSSIP command JSON: %v", err)
					db.mu.Unlock()
					continue
				}

				Debug(dPeer, "Received GOSSIP from %s", args.Name)

				// Check if we've received or sent this gossip already
				exists := false
				for _, id := range db.gossipIDs {
					if id == args.MessageID {
						exists = true
						break
					}
				}

				if exists {
					Debug(dGossip, "Duplicate gossip ID: %v, ignoring", args.MessageID)
					db.mu.Unlock()
					continue
				} else {
					// Add the new gossip to our list
					db.gossipIDs = append(db.gossipIDs, args.MessageID)
				}

				fmt.Println("GOSSIP from ", args.Name)

				// Check if first time seeing peer
				peerIdx := db.peerIdx(args.Host, args.Port)
				if peerIdx == -1 {
					Debug(dPeer, "First time seeing peer %s, adding to list of peers", args.Name)

					// Add to list of peers
					err := db.addPeer(args.Host, args.Port, args.Name)
					if err != nil {
						Debug(dError, "Failed to connect with new peer at (%s:%d)", args.Host, args.Port)
					}

					peerIdx := db.peerIdx(args.Host, args.Port)

					// Relay message to 5 other randomly selected peers
					perm := rand.Perm(len(db.peers) - 1)
					for cnt, randomIdx := range perm {
						if cnt >= 5 {
							break
						}

						Debug(dGossip, "Relayed gossip from (%s) to (%s)", args.Name, db.peers[randomIdx].Name)
						db.Send(db.peers[randomIdx], buf[:n])
					}

					// Reply to the new peer with our own information
					reply := &GossipReply{
						Command: GOSSIP_REPLY,
						Host:    db.self.Addr.IP.String(),
						Port:    db.self.Addr.Port,
						Name:    db.self.Name,
					}

					replyData, err := reply.Json()
					if err != nil {
						Debug(dError, "Failed to encode Gossip reply to JSON: %v", err)
					}

					db.Send(db.peers[peerIdx], replyData)
					Debug(dGossip, "Replied to GOSSIP from (%s) with our information", args.Name)
				} else {
					Debug(dPeer, "Peer (%s) already exists, updating expiration date", args.Name)
					// Update expiration date
					peer := db.peers[peerIdx]
					fmt.Printf("Updating expiry date for %s\n", peer.Name)
					peer.Expires = time.Now().Add(time.Second * 60 * 2)

					// Relay message to 3 other peers
					perm := rand.Perm(len(db.peers) - 1)
					for cnt, randomIdx := range perm {
						if cnt >= 3 {
							break
						}

						if len(db.peers) > randomIdx && db.peers[randomIdx] != peer {
							Debug(dGossip, "Relayed gossip from (%s) to (%s)", args.Name, db.peers[randomIdx].Name)

							// Since we may have removed a peer during the previous send
							// we have to check if the index is still in bounds.
							db.Send(db.peers[randomIdx], buf[:n])
						}
					}
				}
			} else if command == GOSSIP_REPLY {
				// Parse the reply
				reply, err := parseGossipReply(buf[:n])
				if err != nil {
					Debug(dError, "Failed to parse GOSSIP_REPLY command JSON: %v", err)
					db.mu.Unlock()
					continue
				}

				Debug(dPeer, "Received GOSSIP_REPLY command from (%s)", reply.Name)

				// Add the peer if it does not already exist
				peerIdx := db.peerIdx(reply.Host, reply.Port)
				if peerIdx == -1 {
					err := db.addPeer(reply.Host, reply.Port, reply.Name)
					if err != nil {
						Debug(dError, "Failed to connect with new peer at (%s:%d)", reply.Host, reply.Port)
					}
				} else {
					db.peers[peerIdx].Expires = time.Now().Add(time.Second * 60 * 2)
				}
			} else if command == SET {
				args, err := parseSetArgs(buf[:n])
				if err != nil {
					Debug(dError, "Failed to parse SET command JSON: %v", err)
				}

				Debug(dSet, "Received SET: (%d, %s)", args.Index, args.Value)

				// Update the value in our database if the index is in bounds
				if args.Index >= 0 && args.Index < 5 {
					// TODO: get remote address and check if any peer has it
					host, port := addr.IP.String(), addr.Port
					Debug(dPeer, "SET: host: %s, port: %d", host, port)
					peerIdx := db.peerIdx(host, port)
					if peerIdx > -1 {
						db.data[args.Index] = args.Value
						db.peers[peerIdx].RecentValue = args.Value
					} else {
						Debug(dPeer, "Received SET from unknown peer at %s:%d", host, port)
					}
				}
			} else {
				Debug(dPeer, "Received unknown command from peer")
			}

			Debug(dPeer, "Updated Peers: %v\n", db.peers.String())
			db.mu.Unlock()
		}
	}
}

// With mutex, retrieve the index of peer with parameters, or -1 if it does not exist
func (db *Db) peerIdx(host string, port int) int {
	peerIdx := -1
	for idx, peer := range db.peers {
		if peer.Addr.IP.String() == host && peer.Addr.Port == port {
			peerIdx = idx
		}
	}

	return peerIdx
}

// With mutex, connect to a new peer and add it to the list of peers
func (db *Db) addPeer(host string, port int, name string) error {
	// Add to list of peers
	newPeer, err := db.ConnectWithPeer(host, port, name)
	if err != nil {
		return err
	}

	fmt.Printf("New peer: %s\n", newPeer.Name)
	db.peers = append(db.peers, newPeer)
	return nil
}

// Goroutine that periodically sends gossip messages to all active peers
func (db *Db) gossipper() {
	Debug(dInfo, "Gossipper started")
	for {
		Debug(dGossip, "Sending Gossip messages to all peers")
		args := &GossipArgs{
			Command:   GOSSIP,
			Host:      db.self.Addr.IP.String(),
			Port:      db.self.Addr.Port,
			Name:      db.self.Name,
			MessageID: uuid.New(),
		}

		jsonBytes, err := args.Json()
		Debug(dGossip, "Gossip message: %s", string(jsonBytes))
		if err != nil {
			Debug(dError, "Join: %v", err)
			panic(err)
		}

		db.mu.Lock()
		db.gossipIDs = append(db.gossipIDs, args.MessageID)
		// Send gossip to all the peers
		for _, peer := range db.peers {
			Debug(dGossip, "Sending gossip to %s", peer.Name)
			db.Send(peer, jsonBytes)
		}

		db.mu.Unlock()

		Debug(dInfo, "Gossipper sleeping for 1s")
		time.Sleep(time.Second * 60)
	}
}

func (db *Db) peerMaintainer() {
	for {
		time.Sleep(time.Millisecond * 50)

		db.mu.Lock()
		var updatedPeers []*Peer
		for _, peer := range db.peers {
			// if peer.Expires.After(time.Now()) {
			if time.Since(peer.Expires) < 0 {
				updatedPeers = append(updatedPeers, peer)
			}
		}

		db.peers = updatedPeers
		// fmt.Printf("Peers: %d\n", len(db.peers))

		db.mu.Unlock()
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
	db.mu.Lock()
	db.data[index] = value
	Debug(dSet, "Updated database: %v", db.data)

	args := &SetArgs{
		Command: SET,
		Index:   index,
		Value:   value,
	}

	jsonBytes, err := args.Json()
	if err != nil {
		Debug(dError, "Failed to parse set command: %v", err)
	}

	// Send the set command to all peers
	for _, peer := range db.peers {
		Debug(dSet, "Sent SET to %v", peer)
		db.Send(peer, jsonBytes)
	}

	db.mu.Unlock()

}
