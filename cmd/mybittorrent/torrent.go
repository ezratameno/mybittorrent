package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"

	bencode "github.com/jackpal/bencode-go"
)

type TorrentFile struct {

	// URL to a "tracker", which is a central server that keeps track of peers participating in the sharing of a torrent.
	Announce string

	Info Info
}

type Info struct {
	// size of the file in bytes, for single-file torrents
	Length int64

	// suggested name to save the file / directory as
	Name string

	// number of bytes in each piece
	PieceLength int64

	// concatenated SHA-1 hashes of each piece, 20 bytes each
	Pieces string

	// unique identifier for a torrent file. It's used when talking to trackers or peers.
	InfoHash string

	PiecesHash []string
}

// NewTorrentFile builds the torrent file from the decoded content of the torrent file
func NewTorrentFile(filePath string) (*TorrentFile, error) {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Decode the file contents
	reader := bytes.NewReader([]byte(content))
	decoded, err := bencode.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}

	decodedMap, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("wrong format, expected a map")
	}
	if _, ok := decodedMap["announce"]; !ok {
		return nil, fmt.Errorf("wrong format, announce not present")
	}

	if _, ok := decodedMap["info"]; !ok {
		return nil, fmt.Errorf("wrong format, info not present")
	}

	// TODO: validate that info is a map
	infoMap := decodedMap["info"].(map[string]any)
	length := infoMap["length"].(int64)
	name := infoMap["name"].(string)
	pieceLength := infoMap["piece length"].(int64)
	pieces := infoMap["pieces"].(string)

	// calculate the sha of the encoded info dictionary

	hash := sha1.New()
	err = bencode.Marshal(hash, infoMap)
	if err != nil {
		return nil, err
	}
	infoHash := hash.Sum(nil)

	// fmt.Println("sha: ", sha)

	// get the pieces hash

	var piecesHash []string
	var i int
	for i < len(pieces) {
		piecesHash = append(piecesHash, fmt.Sprintf("%x", pieces[i:i+20]))
		i += 20
	}

	file := &TorrentFile{

		// TODO: improve this
		Announce: decodedMap["announce"].(string),
		Info: Info{
			Length:      length,
			Name:        name,
			PieceLength: pieceLength,
			Pieces:      pieces,
			InfoHash:    fmt.Sprintf("%x", infoHash),
			PiecesHash:  piecesHash,
		},
	}

	return file, nil

}
