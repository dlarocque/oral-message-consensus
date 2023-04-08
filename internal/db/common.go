package db

import "github.com/google/uuid"

// State of the peer to peer distributed database
type Db struct {
	self  *Peer  // The information of this peer
	peers []Peer // Known active peers
	data  []int  // Database of five values
}

// State of a peer in the network
type Peer struct {
	Host string
	Port int
	Name string
}

type QueryArgs struct {
	command string // Command name
}

type QueryReply struct {
	command string // Command name
	data    []int  // All five values from the database of the replying peer
}

type SetArgs struct {
	command string // Command name
	index   int    // Database index
	value   string // Value to replaced what is at the index
}

type GossipArgs struct {
	command   string    // Command name
	host      string    // Host of the sending peer
	port      int       // Port of the sending peer
	name      string    // Name of the sending peer
	messageID uuid.UUID // Unique message ID
}

type GossipReply struct {
	command string // Command name
	host    string // Host of the replying peer
	port    int    // Port of the replying peer
	name    string // Name of the replying peer
}

type ConsensusArgs struct {
	command   string    // Command name
	OM        int       // OM level, should always be the highest possible
	index     int       // Index that consensus is being performed on
	value     string    // Value contained at index
	peers     []string  // Peers in string format host:port
	messageID uuid.UUID // Unique message ID
	due       int       // Unix time to reply by
}

type ConsensusReply struct {
	command string
	value   string
	replyTo uuid.UUID `json:"reply-to"`
}
