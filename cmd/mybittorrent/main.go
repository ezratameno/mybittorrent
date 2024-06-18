package main

import (
	// Uncomment this line to pass the first stage
	// "encoding/json"
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

func main() {
	err := run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

const (
	commandDecode = "decode"
	commandInfo   = "info"
)

func run() error {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	// fmt.Println("Logs from your program will appear here!")

	command := os.Args[1]

	switch command {
	case commandDecode:

		bencodedValue := os.Args[2]

		reader := bytes.NewReader([]byte(bencodedValue))

		decoded, err := bencode.Decode(reader)
		if err != nil {
			return fmt.Errorf("decode error: %w", err)
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))

	case commandInfo:
		filePath := os.Args[2]

		err := InfoCmd(filePath)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown command %s", command)
	}

	return nil
}

func InfoCmd(filePath string) error {

	file, err := NewTorrentFile(filePath)
	if err != nil {
		return err
	}

	fmt.Printf("Tracker URL: %+v\n", file.Announce)
	fmt.Printf("Length: %+v\n", file.Info.Length)
	fmt.Printf("Info Hash: %+v\n", file.Hash)

	return nil
}
