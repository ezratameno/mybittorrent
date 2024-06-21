package main

// https://www.bittorrent.org/beps/bep_0003.html#peer-messages
const (
	messageIDChoke = iota
	messageIDUnchoke
	messageIDInterested
	messageIDNotInterested
	messageIDHave
	messageIDBitfield
	messageIDRequest
	messageIDPiece
	messageIDCancel
)

type PeerMessage struct {
	// 4 bytes
	Length []byte
	ID     byte
}

type BitfieldMessage struct {
	PeerMessage

	// Payload of some kind
}

type InterestedMessage struct {
	PeerMessage
}

type RequestMessage struct {
	PeerMessage
	Payload RequestPayload
}

type RequestPayload struct {
	//  the zero-based piece index
	Index int

	// the zero-based byte offset within the piece
	// This'll be 0 for the first block, 2^14 for the second block, 2*2^14 for the third block etc.
	// Break the piece into blocks of 16 kiB (16 * 1024 bytes)
	Begin int

	// the length of the block in bytes
	// This'll be 2^14 (16 * 1024) for all blocks except the last one.
	// The last block will contain 2^14 bytes or less, you'll need calculate this value using the piece length.
	Length int
}

type PieceMessage struct {
	PeerMessage
}

type PiecePayload struct {
	//  the zero-based piece index
	Index int

	//  the zero-based byte offset within the piece
	Begin int

	// the data for the piece, usually 2^14 bytes long
	Block []byte
}
