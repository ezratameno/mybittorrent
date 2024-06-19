package main

import (
	// Uncomment this line to pass the first stage
	// "encoding/json"
	"bytes"
	"context"
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
	commandPeers  = "peers"
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

	case commandPeers:
		filePath := os.Args[2]

		return PeersCmd(filePath)

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

	// info hash in hex
	fmt.Printf("Info Hash: %x\n", file.Info.InfoHash)
	fmt.Printf("Piece Length: %+v\n", file.Info.PieceLength)
	fmt.Printf("Piece Hashes:\n")
	for _, pieceHash := range file.Info.PiecesHash {
		fmt.Println(pieceHash)
	}

	return nil
}

func PeersCmd(filePath string) error {

	file, err := NewTorrentFile(filePath)
	if err != nil {
		return err
	}

	resp, err := file.DiscoverPeers(context.Background())
	if err != nil {
		return err
	}

	for _, peer := range resp.peers {
		fmt.Printf("%s:%d\n", peer.ipAddr, peer.port)
	}
	return nil
}
