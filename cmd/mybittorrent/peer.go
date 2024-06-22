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
	"time"
)

type Peer struct {
	port   uint16
	ipAddr string
	conn   net.Conn

	handshake *Handshake

	availablePiecesIndexes []int

	// TODO: add chan that we pass the messages through him
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

	// after handshake the peer will send a bitfield message telling us which pieces he has
	buf := make([]byte, 1024)
	size, err := p.conn.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read bitfield message: %w", err)
	}

	buf = buf[:size]

	piecesBinRep := fmt.Sprintf("%b", buf[5:])
	for i, pieceIndex := range piecesBinRep {
		if pieceIndex == '1' {
			p.availablePiecesIndexes = append(p.availablePiecesIndexes, i)
		}
	}
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

	piece, err := p.handleDownloadPiece(file, pieceIndex)
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

func (p *Peer) handleDownloadPiece(file *TorrentFile, pieceIndex int) ([]byte, error) {

	// Send interested message to start
	_, err := p.conn.Write([]byte{0, 0, 0, 1, messageIDInterested})
	if err != nil {
		return nil, err
	}

	fmt.Println("sent interested message")

	buf := make([]byte, blockSize*5)

	var piece []byte
	for {

		size, err := p.conn.Read(buf)
		if err != nil {
			// Connection was closed
			if errors.Is(err, io.EOF) {

				fmt.Println("eof")
				return nil, nil
			}

			return nil, err
		}

		content := buf[:size]

		fmt.Println("content:", content)

		// http://www.kristenwidman.com/blog/71/how-to-write-a-bittorrent-client-part-2/#:~:text=Bitfield%20messages%20are%20optional%20and,pieces%20that%20a%20peer%20has.
		// Can be because of the choke message
		if len(content) < 5 {
			continue
		}

		switch messageID := content[4]; messageID {

		case messageIDUnchoke:

			// Break the piece into blocks of 16 kiB (16 * 1024 bytes) and send a request message for each block

			fmt.Println("unchoke")

			// This piece can be the last piece in the file
			// so the piece length can be less the the standard piece length

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
				// fmt.Printf("length: %d, begin: %d, block num: %d\n", length, begin, i)

				var request []byte

				request = binary.BigEndian.AppendUint32(request, 13)
				request = append(request, uint8(messageIDRequest))
				request = binary.BigEndian.AppendUint32(request, index)
				request = binary.BigEndian.AppendUint32(request, begin)
				request = binary.BigEndian.AppendUint32(request, length)

				// Send the request
				_, err := p.conn.Write(request)
				if err != nil {
					return nil, fmt.Errorf("failed to write: %w", err)
				}

				// fmt.Printf("wrote: %d bytes\n", n)
				// fmt.Println("wrote data", i)

				// Read the response
				resp := make([]byte, length+13)
				var respSize int

				err = withRetry(3, 1300*time.Millisecond, func() error {
					respSize, err = io.ReadFull(p.conn, resp)
					return err
				})
				if err != nil {
					return nil, err
				}

				// fmt.Println("respSize", respSize)

				resp = resp[:respSize]
				// fmt.Println("resp:", resp)

				// respIndex := binary.BigEndian.Uint32(resp[5:9])
				// respBegin := binary.BigEndian.Uint32(resp[9:13])
				respBlock := resp[13:]

				// fmt.Println("resp Length", binary.BigEndian.Uint32(resp[:5]))
				// fmt.Println("respIndex", respIndex)
				// fmt.Println("respBegin", respBegin)

				piece = append(piece, respBlock...)
				// fmt.Println("respBlock", respBlock)
			}

			// Finish getting all the blocks of the pieces
			return piece, nil

		default:
			fmt.Println("size", size)
			// fmt.Println("content", content)
			fmt.Println("message id", content[4])
		}

	}
}
