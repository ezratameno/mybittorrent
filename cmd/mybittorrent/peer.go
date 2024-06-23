package main

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

type Peer struct {
	// make sure we don't write and read at the same time

	sync.Once

	port   uint16
	ipAddr string
	conn   net.Conn

	handshake *Handshake

	availablePiecesIndexes []int

	// If the peer is choked then we can't request any pieces from him
	chocked bool

	unChokedChan chan struct{}

	// TODO: add chan that we pass the messages through him
	msgChan      chan []byte
	pieceMsgChan chan []byte

	// lock sync.Mutex
}

func NewPeer(port uint16, ipAddr string) *Peer {
	return &Peer{
		port:         port,
		ipAddr:       ipAddr,
		msgChan:      make(chan []byte),
		unChokedChan: make(chan struct{}),
		pieceMsgChan: make(chan []byte),
		// lock:         sync.Mutex{},
	}
}

const (
	blockSize = 16 * 1024
)

func (p *Peer) Connect(infoHash []byte) error {
	addr := fmt.Sprintf("%s:%d", p.ipAddr, p.port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	p.conn = conn

	// unique identifier of our peer
	peerID := []byte("00112233445566778899")
	p.handshake, err = p.Handshake(context.Background(), infoHash, peerID)
	if err != nil {
		return err
	}

	go p.handleConnection()

	go p.handleMessage()

	return nil
}

func (p *Peer) Close() error {
	return p.conn.Close()
}

const (
	handshakeSize = 68
)

type Handshake struct {

	// length of the protocol string (BitTorrent protocol) which is 19 (1 byte)
	ProtocolLength byte

	// BitTorrent protocol - 19 bytes
	ProtocolName string

	// sha1 info hash - 20 bytes
	InfoHash []byte

	// 20 bytes
	PeerID []byte
}

func (h *Handshake) Bytes() []byte {
	var buf bytes.Buffer

	// length of the protocol
	buf.WriteByte(19)

	// name of the protocol
	buf.WriteString("BitTorrent protocol")

	// eight reserved bytes, which are all set to zero (8 bytes)
	for i := 0; i < 8; i++ {
		buf.WriteByte(0)
	}

	buf.WriteString(string(h.InfoHash))
	buf.WriteString(string(h.PeerID))

	return buf.Bytes()
}

func ParseHandshake(buf []byte) (*Handshake, error) {
	if len(buf) != handshakeSize {
		return nil, fmt.Errorf("wrong size, expected %d, got %d", handshakeSize, len(buf))
	}

	peerID := buf[handshakeSize-20 : handshakeSize]
	peerHashInfo := buf[handshakeSize-40 : handshakeSize-20]

	return &Handshake{
		PeerID:   peerID,
		InfoHash: peerHashInfo,
	}, nil
}

func (p *Peer) Handshake(ctx context.Context, infoHash []byte, peerID []byte) (*Handshake, error) {

	h := &Handshake{
		InfoHash: infoHash,
		PeerID:   peerID,
	}

	_, err := p.conn.Write(h.Bytes())
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 1024)

	size, err := p.conn.Read(buf)
	if err != nil {
		return nil, err
	}

	// Parse the handshake

	handshake, err := ParseHandshake(buf[:size])
	if err != nil {
		return nil, err
	}

	return handshake, err
}

func (p *Peer) DownloadPiece(ctx context.Context, file *TorrentFile, pieceIndex int) ([]byte, error) {

	// Send interested message to start
	_, err := p.Write([]byte{0, 0, 0, 1, messageIDInterested})
	if err != nil {
		return nil, err
	}

	fmt.Println("sent interested message")

	piece, err := p.downloadPiece(file, pieceIndex)
	if err != nil {
		return nil, err
	}

	// validate the hash of the piece

	expectedPieceHash := file.Info.PiecesHash[pieceIndex]

	fmt.Printf("expectedPieceHash: %s, piece index: %d\n", expectedPieceHash, pieceIndex)

	hash := sha1.New()

	_, err = hash.Write(piece)
	if err != nil {
		return nil, err
	}

	pieceHash := fmt.Sprintf("%x", hash.Sum(nil))

	// fmt.Println("pieceHash", pieceHash)

	// fmt.Println("piece len", len(piece))

	if pieceHash != expectedPieceHash {
		return nil, errors.New("piece hash doesn't match expected hash")
	}

	return piece, nil

}

func (p *Peer) handleConnection() error {
	buf := make([]byte, blockSize+13)

	for {

		size, err := p.Read(buf)
		if err != nil {
			// Connection was closed
			if errors.Is(err, io.EOF) {

				fmt.Println("eof")
				return nil
			}

			return err
		}

		fmt.Println("message size", size)
		msg := buf[:size]

		p.msgChan <- msg
	}
}

func (p *Peer) handleMessage() error {
	for {

		msg := <-p.msgChan

		if len(msg) < 5 {

			// If len is 4, then it's a keep alive message
			// It serves to maintain active connections by signaling that the connection is still alive and should remain open.
			fmt.Println("msg:", msg)
			return fmt.Errorf("message in wrong format, expected len at least 5, %s", msg)
		}

		msgID := msg[4]

		switch msgID {

		case messageIDChoke:
			fmt.Println("msg choke")
			p.chocked = true

		case messageIDUnchoke:
			fmt.Println("msg unchoke")
			p.chocked = false
			p.unChokedChan <- struct{}{}

		case messageIDInterested:
			fmt.Println("msg Interested")

		case messageIDNotInterested:
			fmt.Println("msg NotInterested")

		case messageIDHave:
			fmt.Println("msg Have")

			// get all the pieces that the peer has
		case messageIDBitfield:
			fmt.Println("msg Bitfield")

			err := p.handleBitfieldMessage(msg)
			if err != nil {
				return fmt.Errorf("failed to handle bitfield message: %w", err)
			}
		case messageIDRequest:
			fmt.Println("msg Request")

		case messageIDPiece:
			fmt.Println("msg Piece")
			p.pieceMsgChan <- msg

		case messageIDCancel:
			fmt.Println("msg Cancel")

		default:
			fmt.Println("unknown message")
		}

	}
}

// make sure we don't write and read at the same time
func (p *Peer) Write(b []byte) (int, error) {
	// fmt.Println("try lock")

	// p.lock.Lock()
	// fmt.Println("locked")
	// defer p.lock.Unlock()

	return p.conn.Write(b)
}

// make sure we don't write and read at the same time
func (p *Peer) Read(b []byte) (int, error) {

	// p.lock.Lock()
	// defer p.lock.Unlock()

	return p.conn.Read(b)
}
func (p *Peer) downloadPiece(file *TorrentFile, pieceIndex int) ([]byte, error) {

	// TODO: block until not choked

	<-p.unChokedChan

	fmt.Println("unchoke starting to download")

	var completedPiece []byte
	pieceLen := file.Info.PieceLength

	if len(file.Info.PiecesHash)-1 == pieceIndex {
		pieceLen = file.Info.Length % file.Info.PieceLength
	}
	numBlocks := pieceLen / blockSize

	// If the file has some part that is less then the standard block size
	if pieceLen%blockSize != 0 {
		numBlocks++
	}
	// fmt.Printf("num of blocks in a piece: %d\n", numBlocks)
	// fmt.Println("piece length", pieceLen)

	for i := 0; i < int(numBlocks); i++ {

		index := uint32(pieceIndex)
		begin := uint32(i * blockSize)
		length := uint32(blockSize)

		// The length of the last piece can be less then the others
		if i == int(numBlocks)-1 && pieceLen%blockSize != 0 {

			// fmt.Println(`file.Info.PieceLength % blockSize:`, file.Info.PieceLength%blockSize)
			length = uint32(pieceLen % blockSize)
		}

		// fmt.Printf("begin: %d, block num: %d\n", begin, i)
		fmt.Printf("length: %d, begin: %d, block num: %d\n", length, begin, i)

		var request []byte

		request = binary.BigEndian.AppendUint32(request, 13)
		request = append(request, uint8(messageIDRequest))
		request = binary.BigEndian.AppendUint32(request, index)
		request = binary.BigEndian.AppendUint32(request, begin)
		request = binary.BigEndian.AppendUint32(request, length)

		// Send the request

		fmt.Println("trying to send request")
		_, err := p.Write(request)
		if err != nil {
			return nil, fmt.Errorf("failed to write: %w", err)
		}

		fmt.Println("sent request message")

		// fmt.Printf("wrote: %d bytes\n", n)
		// fmt.Println("wrote data", i)

		// Read the response
		// resp := make([]byte, length+13)
		// var respSize int

		// err = withRetry(3, 1300*time.Millisecond, func() error {
		// 	respSize, err = io.ReadFull(p.conn, resp)
		// 	return err
		// })
		// if err != nil {
		// 	return nil, err
		// }

		resp := <-p.pieceMsgChan

		// resp = resp[:length+13]

		fmt.Println("got piece message")

		// fmt.Println("respSize", respSize)

		// resp = resp[:respSize]
		// fmt.Println("resp:", resp)

		// respIndex := binary.BigEndian.Uint32(resp[5:9])
		// respBegin := binary.BigEndian.Uint32(resp[9:13])
		respBlock := resp[13:]

		// fmt.Println("resp Length", binary.BigEndian.Uint32(resp[:5]))
		// fmt.Println("respIndex", respIndex)
		// fmt.Println("respBegin", respBegin)

		completedPiece = append(completedPiece, respBlock...)
		// fmt.Println("respBlock", respBlock)
	}

	return completedPiece, nil
}

func (p *Peer) handleBitfieldMessage(msg []byte) error {

	piecesBinRep := fmt.Sprintf("%b", msg[5:])
	for i, pieceIndex := range piecesBinRep {
		if pieceIndex == '1' {
			p.availablePiecesIndexes = append(p.availablePiecesIndexes, i)
		}
	}
	return nil
}
