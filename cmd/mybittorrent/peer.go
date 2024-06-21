package main

import (
	"bytes"
	"context"
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

	// TODO: add chan that we pass the messages through him
}

const (
	blockSize = 16 * 1024
)

func (p *Peer) Connect() error {
	addr := fmt.Sprintf("%s:%d", p.ipAddr, p.port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	p.conn = conn
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

func (p *Peer) DownloadPiece(ctx context.Context, file *TorrentFile, peerID []byte, pieceIndex int) (*any, error) {

	_, err := p.Handshake(context.Background(), file.Info.InfoHash, peerID)
	if err != nil {
		return nil, err
	}

	go p.handleDownloadPiece(file, pieceIndex)

	time.Sleep(100 * time.Second)

	// wait for bitfield message

	// send interested message

	// wait for unchoke message

	// Break the piece into blocks of 16 kiB (16 * 1024 bytes) and send a request message for each block

	return nil, nil
}

func (p *Peer) handleDownloadPiece(file *TorrentFile, pieceIndex int) error {
	buf := make([]byte, blockSize*5)

	for {

		size, err := p.conn.Read(buf)
		if err != nil {
			// Connection was closed
			if errors.Is(err, io.EOF) {

				fmt.Println("eof")
				return nil

			}

			fmt.Println("err", err.Error())
			return err
		}

		content := buf[:size]

		// fmt.Println("content: ", content)

		switch messageID := content[4]; messageID {

		// send interested message
		case messageIDBitfield:

			// TODO: improve this
			p.conn.Write([]byte{0, 0, 0, 1, messageIDInterested})
			fmt.Println("sent msg")

		case messageIDUnchoke:

			// Break the piece into blocks of 16 kiB (16 * 1024 bytes) and send a request message for each block

			fmt.Println("unchoke")

			numBlocks := file.Info.PieceLength / blockSize

			// If the file has some part that is less then the standard block size
			if file.Info.PieceLength%blockSize != 0 {
				numBlocks++
			}
			fmt.Printf("num of blocks in a piece: %d\n", numBlocks)

			fmt.Println("piece length", file.Info.PieceLength)

			for i := 0; i < int(numBlocks); i++ {

				index := uint32(pieceIndex)
				begin := uint32(i * blockSize)
				length := uint32(blockSize)

				// The length of the last piece can be less then the others
				if i == int(numBlocks)-1 && file.Info.PieceLength%blockSize != 0 {
					length = uint32(file.Info.PieceLength % blockSize)
				}

				fmt.Printf("begin: %d, block num: %d\n", begin, i)
				fmt.Printf("length: %d, block num: %d\n", length, i)

				var b []byte

				b = binary.BigEndian.AppendUint32(b, 13)
				b = append(b, uint8(messageIDRequest))
				b = binary.BigEndian.AppendUint32(b, index)
				b = binary.BigEndian.AppendUint32(b, begin)
				b = binary.BigEndian.AppendUint32(b, length)

				fmt.Println(b)

				// Send the request
				_, err = p.conn.Write(b)
				if err != nil {
					fmt.Println("error:", err.Error())
				}
				fmt.Println("wrote data", i)

				// Read the response

				d := make([]byte, length+13)
				s, err := p.conn.Read(d)
				if err != nil {
					fmt.Println("err", err.Error())

					return err
				}

				resp := d[:s]
				fmt.Println(resp)

				respIndex := binary.BigEndian.Uint32(resp[5:9])
				respBegin := binary.BigEndian.Uint32(resp[9:13])
				respBlock := resp[14:]
				_ = respBlock
				fmt.Println("respIndex", respIndex)
				fmt.Println("respBegin", respBegin)
				// fmt.Println("respBlock", respBlock)
			}

		default:
			fmt.Println("size", size)
			// fmt.Println("content", content)
			fmt.Println("message id", content[4])
		}

	}
}
