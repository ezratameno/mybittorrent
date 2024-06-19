package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
)

type Peer struct {
	port   uint16
	ipAddr string
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

	addr := fmt.Sprintf("%s:%d", p.ipAddr, p.port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	h := &Handshake{
		InfoHash: infoHash,
		PeerID:   peerID,
	}

	_, err = conn.Write(h.Bytes())
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 1024)

	size, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	// Parse the handshake

	handshake, err := ParseHandshake(buf[:size])
	if err != nil {
		return nil, err
	}

	// fmt.Printf("read: %+v", buf[:size])
	return handshake, err
}
