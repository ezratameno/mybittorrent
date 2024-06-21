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
