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
	"time"
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
	chockedCh chan struct{}

	unChokedCh chan struct{}

	// If we are unchoked then we can download from the peer
	choked bool

	// TODO: add chan that we pass the messages through him
	msgChan chan []byte

	// Pass piece messages from the peer
	pieceMsgChan chan []byte

	downloadedPieceChan chan downloadPieceChan
}

type downloadPieceChan struct {
	content []byte
	err     error
}

func NewPeer(port uint16, ipAddr string) *Peer {
	return &Peer{
		port:                port,
		ipAddr:              ipAddr,
		msgChan:             make(chan []byte),
		chockedCh:           make(chan struct{}),
		unChokedCh:          make(chan struct{}),
		pieceMsgChan:        make(chan []byte),
		downloadedPieceChan: make(chan downloadPieceChan),
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

	select {
	case <-p.unChokedCh:
		return nil
	case <-time.After(3 * time.Second):
		return fmt.Errorf("timed out to receive unchoke message")
	}

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

	// // Then it's not the first time, we need to send an interested message
	// if len(p.availablePiecesIndexes) > 0 {
	// 	// Send interested message to start
	// 	_, err := p.conn.Write([]byte{0, 0, 0, 1, messageIDInterested})
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	fmt.Println("sent interested message")
	// }
	// send in the background
	go p.downloadPiece(file, pieceIndex)

	// wait for the choke message or the piece to be downloaded
	select {
	case <-p.chockedCh:
		return nil, fmt.Errorf("peer %s:%d is choked", p.ipAddr, p.port)

	case pieceRes := <-p.downloadedPieceChan:
		if pieceRes.err != nil {
			return nil, pieceRes.err
		}

		// validate the hash of the piece

		expectedPieceHash := file.Info.PiecesHash[pieceIndex]

		hash := sha1.New()

		_, err := hash.Write(pieceRes.content)
		if err != nil {
			return nil, err
		}

		pieceHash := fmt.Sprintf("%x", hash.Sum(nil))

		if pieceHash != expectedPieceHash {
			return nil, errors.New("piece hash doesn't match expected hash")
		}

		return pieceRes.content, nil

	}

}

func (p *Peer) handleConnection() error {
	buf := make([]byte, 5)

	for {

		// Read the first 5 bytes to get the size and message id

		size, err := p.conn.Read(buf)
		if err != nil {
			// Connection was closed
			if errors.Is(err, io.EOF) {
				fmt.Println("eof")
				return nil
			}

			return err
		}
		buf = buf[:size]

		// Keep alive
		if len(buf) < 5 {
			continue
		}

		messageSize := binary.BigEndian.Uint32(buf[:4])

		// Because we read the message ID already
		messageSize = messageSize - 1

		msg := buf

		// Read the payload if exist
		if messageSize > 0 {
			payloadBuf := make([]byte, messageSize)

			_, err = io.ReadFull(p.conn, payloadBuf)
			if err != nil {
				return fmt.Errorf("failed to read payload")
			}
			msg = append(msg, payloadBuf...)

		}

		p.msgChan <- msg
	}
}

func (p *Peer) handleMessage() error {
	for {

		msg := <-p.msgChan

		msgID := msg[4]

		switch msgID {

		case messageIDChoke:
			fmt.Println("msg choke")
			p.chockedCh <- struct{}{}
			p.choked = true

		case messageIDUnchoke:

			fmt.Println("msg unchoke")
			p.choked = false
			p.unChokedCh <- struct{}{}

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

func (p *Peer) downloadPiece(file *TorrentFile, pieceIndex int) {

	// TODO: block until not choked
	// Wait X time
	// if p.choked {
	// 	p.downloadedPieceChan <- downloadPieceChan{
	// 		err: fmt.Errorf("trying to download while choked"),
	// 	}
	// }

	fmt.Println("unchoke starting to download")

	pieceLen := file.Info.PieceLength

	if len(file.Info.PiecesHash)-1 == pieceIndex {
		pieceLen = file.Info.Length % file.Info.PieceLength
	}
	var completedPiece []byte

	numBlocks := pieceLen / blockSize

	// If the file has some part that is less then the standard block size
	if pieceLen%blockSize != 0 {
		numBlocks++
	}
	fmt.Printf("num of blocks in a piece: %d\n", numBlocks)
	fmt.Println("piece length", pieceLen)

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
		_, err := p.conn.Write(request)
		if err != nil {
			p.downloadedPieceChan <- downloadPieceChan{
				err: fmt.Errorf("failed to write: %w", err),
			}
			return
		}

		fmt.Println("sent request message")

		// Read the response

		resp := <-p.pieceMsgChan

		resp = resp[:length+13]

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

	p.downloadedPieceChan <- downloadPieceChan{
		content: completedPiece,
	}

}

func (p *Peer) handleBitfieldMessage(msg []byte) error {

	piecesBinRep := fmt.Sprintf("%b", msg[5:])
	for i, pieceIndex := range piecesBinRep {
		if pieceIndex == '1' {
			p.availablePiecesIndexes = append(p.availablePiecesIndexes, i)
		}
	}

	fmt.Printf("availablePiecesIndexes: %+v\n", p.availablePiecesIndexes)
	// Send interested message to start
	_, err := p.conn.Write([]byte{0, 0, 0, 1, messageIDInterested})
	if err != nil {
		return err
	}

	fmt.Println("sent interested message")
	return nil
}
