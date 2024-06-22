package main

import (
	// Uncomment this line to pass the first stage
	// "encoding/json"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

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
	commandDecode        = "decode"
	commandInfo          = "info"
	commandPeers         = "peers"
	commandHandshake     = "handshake"
	commandDownloadPiece = "download_piece"
	commandDownload      = "download"
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

	case commandHandshake:
		filePath := os.Args[2]
		peer := os.Args[3]
		return HandshakeCmd(filePath, peer)

	case commandDownloadPiece:
		return DownloadPieceCmd(os.Args[2:])

	case commandDownload:
		return DownloadCmd(os.Args[2:])
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

func HandshakeCmd(filePath string, peerInfo string) error {

	file, err := NewTorrentFile(filePath)
	if err != nil {
		return err
	}

	resp, err := file.DiscoverPeers(context.Background())
	if err != nil {
		return err
	}

	peerAddr := strings.Split(peerInfo, ":")[0]

	peerPort, err := strconv.Atoi(strings.Split(peerInfo, ":")[1])
	if err != nil {
		return err
	}

	var desiredPeer *Peer
	for _, peer := range resp.peers {
		if peer.ipAddr == peerAddr && peer.port == uint16(peerPort) {
			desiredPeer = peer
			break
		}
	}

	// Open a connection to the peer
	err = desiredPeer.Connect()
	if err != nil {
		return err
	}

	defer desiredPeer.Close()

	handshake, err := desiredPeer.Handshake(context.Background(), file.Info.InfoHash, []byte("00112233445566778899"))
	if err != nil {
		return err
	}

	fmt.Printf("Peer ID: %x\n", string(handshake.PeerID))
	return nil
}

func DownloadPieceCmd(args []string) error {

	fs := flag.NewFlagSet("download_piece", flag.ExitOnError)
	pieceFile := fs.String("o", "", "path to where to save the piece")
	fs.Parse(args)

	filePath := args[len(args)-2]
	pieceNum, err := strconv.Atoi(args[len(args)-1])
	if err != nil {
		return err
	}

	fmt.Println("filePath", filePath)

	file, err := NewTorrentFile(filePath)
	if err != nil {
		return err
	}

	resp, err := file.DiscoverPeers(context.Background())
	if err != nil {
		return err
	}

	// At this point all the peers contains all the pieces
	desiredPeer := resp.peers[0]

	// Open a connection to the peer
	err = desiredPeer.Connect()
	if err != nil {
		return err
	}

	defer desiredPeer.Close()

	piece, err := desiredPeer.DownloadPiece(context.Background(), file, []byte("00112233445566778899"), pieceNum)
	if err != nil {
		return err
	}

	fmt.Println("got piece !")

	return os.WriteFile(*pieceFile, piece, 0755)
}

func DownloadCmd(args []string) error {

	fs := flag.NewFlagSet("download", flag.ExitOnError)
	pathToFile := fs.String("o", "", "path to where to save the torrent file")
	fs.Parse(args)

	filePath := args[len(args)-1]

	file, err := NewTorrentFile(filePath)
	if err != nil {
		return err
	}

	resp, err := file.DiscoverPeers(context.Background())
	if err != nil {
		return err
	}

	// TODO: to imporve this find which peer has which piece
	// At this point all the peers contains all the pieces
	desiredPeer := resp.peers[0]

	// Open a connection to the peer
	err = desiredPeer.Connect()
	if err != nil {
		return err
	}

	defer desiredPeer.Close()

	var fileContent []byte
	for pieceIndex := range file.Info.PiecesHash {

		piece, err := desiredPeer.DownloadPiece(context.Background(), file, []byte("00112233445566778899"), pieceIndex)
		if err != nil {
			return err
		}

		fileContent = append(fileContent, piece...)
	}

	return os.WriteFile(*pathToFile, fileContent, 0755)
}
