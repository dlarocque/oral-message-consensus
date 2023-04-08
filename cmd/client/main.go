package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

var (
	peersFlag     bool
	currentFlag   bool
	consensusFlag int
	lieFlag       bool
	truthFlag     bool
	setIndex      int
	setWord       string
)

func init() {
	// Command line arguments
	flag.BoolVar(&peersFlag, "peers", false, "list all known peers, with last words received from each")
	flag.BoolVar(&currentFlag, "current", false, "the current word list")
	flag.IntVar(&consensusFlag, "consensus", -1, "run a consensus for this word index")
	flag.BoolVar(&lieFlag, "lie", false, "begin lying")
	flag.BoolVar(&truthFlag, "truth", false, "stop lying")
	flag.IntVar(&setIndex, "set", -1, "send a SET command to all known peers, setting the value x to word y")
	flag.StringVar(&setWord, "word", "", "the word to set for the SET command")
	flag.Parse()
}

func main() {
	fmt.Println("TCP Client started:")

	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("An error occurred while connecting to the server:", err)
		os.Exit(1)
	}

	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(">> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("An error occurred while reading input. Please try again.")
			continue
		}

		input = strings.TrimSpace(input)

		fmt.Fprintln(conn, input)

		response, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			fmt.Println("An error occurred while reading response from the server:", err)
			if err.Error() == "EOF" {
				fmt.Println("Connection to the server has been closed. Exiting...")
				os.Exit(0)
			}
			continue
		}

		fmt.Print(response)
	}
}
