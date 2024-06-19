package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

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
	InfoHash []byte

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
			InfoHash:    infoHash,
			PiecesHash:  piecesHash,
		},
	}

	return file, nil

}

type DiscoverPeersRequest struct {
}

type DiscoverPeersResponse struct {
	// indicating how often your client should make a request to the tracker.
	interval int64

	// A string, which contains list of peers that your client can connect to.
	// Each peer is represented using 6 bytes.
	//  The first 4 bytes are the peer's IP address and the last 2 bytes are the peer's port number.
	peers []Peer
}

type Peer struct {
	port   uint16
	ipAddr string
}

func (tf *TorrentFile) DiscoverPeers(ctx context.Context) (*DiscoverPeersResponse, error) {

	u, err := url.Parse(tf.Announce)
	if err != nil {
		return nil, err
	}

	q := u.Query()

	q.Set("info_hash", string(tf.Info.InfoHash))

	// unique identifier for your client
	// A string of length 20 that you get to pick.
	q.Set("peer_id", "00112233445566778899")

	// the port your client is listening on
	// you will not have to support this functionality during this challenge.
	q.Set("port", "6881")

	// the total amount uploaded so far
	// Since your client hasn't uploaded anything yet, you can set this to 0.
	q.Set("uploaded", "0")

	// the total amount downloaded so far
	// Since your client hasn't downloaded anything yet, you can set this to 0
	q.Set("downloaded", "0")

	// the number of bytes left to download
	// Since you client hasn't downloaded anything yet, this'll be the total length of the file
	q.Set("left", fmt.Sprintf("%d", tf.Info.Length))

	// whether the peer list should use the compact representation
	q.Set("compact", "1")

	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code %d", resp.StatusCode)
	}

	decodedResp, err := bencode.Decode(resp.Body)
	if err != nil {
		return nil, err
	}

	if _, ok := decodedResp.(map[string]any); !ok {
		return nil, fmt.Errorf("response in the wrong format")
	}

	decodedRespMap := decodedResp.(map[string]any)

	if _, ok := decodedRespMap["interval"].(int64); !ok {
		return nil, fmt.Errorf("expected interval to be a int64")
	}

	discoverResp := &DiscoverPeersResponse{
		interval: decodedRespMap["interval"].(int64),
	}

	if _, ok := decodedRespMap["peers"].(string); !ok {
		return nil, fmt.Errorf("expected peers to be a string")
	}

	peers := decodedRespMap["peers"].(string)

	var i int

	// every 6 bytes is a new peer
	for i < len(peers) {

		var buf bytes.Buffer
		for j := 0; j < 4; j++ {
			buf.WriteString(fmt.Sprintf("%+v.", peers[j+i]))
		}
		port := binary.BigEndian.Uint16([]byte(peers[i+4 : i+6]))

		discoverResp.peers = append(discoverResp.peers, Peer{
			port:   port,
			ipAddr: strings.TrimSuffix(buf.String(), "."),
		})
		i += 6
	}

	return discoverResp, nil

}
