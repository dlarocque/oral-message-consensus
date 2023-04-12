package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	GOSSIP       = "GOSSIP"
	GOSSIP_REPLY = "GOSSIP_REPLY"
	SET          = "SET"
)

// State of the peer to peer distributed database
type Db struct {
	mu        sync.Mutex
	conn      *net.UDPConn
	self      *Peer       // The information of this peer
	peers     PeerList    // Known active peers
	data      []string    // Database of five values
	gossipIDs []uuid.UUID // UUIDs of all gossips we have seen
	// consensusStates map[uuid.UUID]ConsensusArgs
}

// State of a peer in the network
type Peer struct {
	Addr        *net.UDPAddr
	Name        string
	Expires     time.Time
	RecentValue string // Most recent value received from peer
}

type PeerList []*Peer

type QueryArgs struct {
	Command string `json:"command"` // Command name
}

type QueryReply struct {
	Command string `json:"command"` // Command name
	Data    []int  `json:"data"`    // All five values from the database of the replying peer
}

type SetArgs struct {
	Command string `json:"command"` // Command name
	Index   int    `json:"index"`   // Database index
	Value   string `json:"value"`   // Value to replaced what is at the index
}

type GossipArgs struct {
	Command   string    `json:"command"`   // Command name
	Host      string    `json:"host"`      // Host of the sending peer
	Port      int       `json:"port"`      // Port of the sending peer
	Name      string    `json:"name"`      // Name of the sending peer
	MessageID uuid.UUID `json:"messageID"` // Unique message ID
}

type GossipReply struct {
	Command string `json:"command"` // Command name
	Host    string `json:"host"`    // Host of the replying peer
	Port    int    `json:"port"`    // Port of the replying peer
	Name    string `json:"name"`    // Name of the replying peer
}

type ConsensusArgs struct {
	Command   string    `json:"command"`   // Command name
	OM        int       `json:"OM"`        // OM level, should always be the highest possible
	Index     int       `json:"index"`     // Index that consensus is being performed on
	Value     string    `json:"value"`     // Value contained at index
	Peers     []string  `json:"peers"`     // Peers in string format host:port
	MessageID uuid.UUID `json:"messageID"` // Unique message ID
	Due       int       `json:"due"`       // Unix time to reply by
}

type ConsensusReply struct {
	Command string    `json:"command"`
	Value   string    `json:"value"`
	PeplyTo uuid.UUID `json:"reply-to"`
}

func (db *Db) ConnectWithPeer(host string, port int, name string) (*Peer, error) {
	// Attempt to establish connection with the peer
	rAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		Debug(dError, "Error resolving address:", err)
		return nil, err
	}

	peer := &Peer{
		Addr:        rAddr,
		Name:        name,
		Expires:     time.Now().Add(time.Minute * 2),
		RecentValue: "",
	}

	return peer, nil
}

func GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String(), nil
		}
	}
	return "", errors.New("No local IP address")
}

func (args *GossipArgs) Json() ([]byte, error) {
	jsonBytes, err := json.Marshal(args)
	if err != nil {
		err = errors.New(fmt.Sprintf("failed to marshal GossipArgs to JSON: %s", err.Error()))
		return nil, err
	}

	return jsonBytes, nil
}

func (args *GossipReply) Json() ([]byte, error) {
	jsonBytes, err := json.Marshal(args)
	if err != nil {
		err = errors.New(fmt.Sprintf("failed to marshal GossipReply to JSON: %s", err.Error()))
		return nil, err
	}

	return jsonBytes, nil
}

func parseGossipArgs(data []byte) (*GossipArgs, error) {
	reply := &GossipArgs{}
	err := json.Unmarshal(data, reply)
	if err != nil {
		return nil, fmt.Errorf("error decoding JSON: %v", err)
	}
	return reply, nil
}

func parseGossipReply(data []byte) (*GossipReply, error) {
	reply := &GossipReply{}
	err := json.Unmarshal(data, reply)
	if err != nil {
		return nil, fmt.Errorf("error decoding JSON: %v", err)
	}
	return reply, nil
}

func (args *SetArgs) Json() ([]byte, error) {
	jsonBytes, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SetArgs to JSON: %s", err.Error())
	}

	return jsonBytes, nil
}

func parseSetArgs(data []byte) (*SetArgs, error) {
	args := &SetArgs{}
	err := json.Unmarshal(data, args)
	if err != nil {
		return nil, fmt.Errorf("error decoding JSON: %v", err)
	}
	return args, nil
}

func (p *Peer) String() string {
	str := fmt.Sprintf("[%s - Recent Value: %s]", p.Name, p.RecentValue)
	return str
}

func (pl *PeerList) String() string {
	peerStrs := make([]string, len(*pl))
	for i, peer := range *pl {
		peerStrs[i] = peer.String()
	}
	return fmt.Sprintf("(%v)", strings.Join(peerStrs, ","))
}
