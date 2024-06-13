package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Bencode (pronounced Bee-encode) is a serialization format used in the BitTorrent protocol.
//
//	It is used in torrent files and in communication between trackers and peers.
func decodeBencode(bencodedString string) (interface{}, error) {

	switch {
	case unicode.IsDigit(rune(bencodedString[0])):
		return decodeString(bencodedString)

	case strings.HasPrefix(bencodedString, "i"):
		return decodeInt(bencodedString)

	default:
		return "", fmt.Errorf("only strings are supported at the moment")

	}

}

// Example:
// - 5:hello -> hello
// - 10:hello12345 -> hello12345
func decodeString(bencodedString string) (string, error) {

	lenEndIndex := strings.Index(bencodedString, ":")
	if lenEndIndex == -1 {
		return "", fmt.Errorf("string in the wrong format, expected at least one ':' ")
	}

	stringLen, err := strconv.Atoi(bencodedString[:lenEndIndex])
	if err != nil {
		return "", err
	}

	startOfContentIndex := lenEndIndex + 1

	stringContent := bencodedString[startOfContentIndex : stringLen+startOfContentIndex]
	return stringContent, nil

}

// format - i<number>e
func decodeInt(bencodedString string) (int, error) {

	bencodedString = strings.TrimPrefix(bencodedString, "i")
	bencodedString = strings.TrimSuffix(bencodedString, "e")

	return strconv.Atoi(bencodedString)
}
