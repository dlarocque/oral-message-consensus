package db

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

type logTopic string

const (
	dClient    logTopic = "CLNT"
	dTimer     logTopic = "TIMR"
	dGossip    logTopic = "GOSP"
	dCommand   logTopic = "CMND"
	dConsensus logTopic = "CONS"
	dSet       logTopic = "SET"
	dPeer      logTopic = "PEER"
	dError     logTopic = "ERRO"
	dInfo      logTopic = "INFO"
	dTest      logTopic = "TEST"
	dWarn      logTopic = "WARN"
	dMessage   logTopic = "MESG"
)

var (
	debugStart     time.Time
	debugVerbosity int
)

func init() {
	debugVerbosity = getVerbosity()
	debugStart = time.Now()

	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
}

func Debug(topic logTopic, format string, a ...interface{}) {
	if debugVerbosity >= 1 {
		time := time.Since(debugStart).Milliseconds()
		time /= 100
		prefix := fmt.Sprintf("%06d %v ", time, string(topic))
		format = prefix + format
		log.Printf(format, a...)
	}
}

// Retrieve the verbosity level from an environment variable
func getVerbosity() int {
	v := os.Getenv("VERBOSE")
	level := 0
	if v != "" {
		var err error
		level, err = strconv.Atoi(v)
		if err != nil {
			log.Fatalf("Invalid verbosity %v", v)
		}
	}
	return level
}
