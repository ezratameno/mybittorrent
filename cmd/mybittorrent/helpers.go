package main

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

const (
	typeUnknown = iota
	typeString
	typeInt
	typeList
)

// bencodeType returns the type of value
func bencodeType(bencodedString string) int {
	switch {
	case unicode.IsDigit(rune(bencodedString[0])):
		return typeString

	case strings.HasPrefix(bencodedString, "i"):
		return typeInt
	case strings.HasPrefix(bencodedString, "l"):
		return typeList
	default:
		fmt.Println("bencodedString", bencodedString)
		return typeUnknown

	}
}

// Bencode (pronounced Bee-encode) is a serialization format used in the BitTorrent protocol.
//
//	It is used in torrent files and in communication between trackers and peers.
func decodeBencode(bencodedString string) (interface{}, error) {

	switch bencodeType(bencodedString) {
	case typeString:
		return decodeString(bencodedString)

	case typeInt:
		return decodeInt(bencodedString)

	case typeList:
		return decodeList(bencodedString)
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

func decodeList(bencodedString string) ([]any, error) {

	res := make([]any, 0)
	bencodedString = strings.TrimPrefix(bencodedString, "l")
	bencodedString = strings.TrimSuffix(bencodedString, "e")

	for len(bencodedString) > 0 {
		switch bencodeType(bencodedString) {
		case typeString:

			str, err := decodeString(bencodedString)
			if err != nil {
				return nil, err
			}

			res = append(res, str)

			// the len of the portion that we read to get the str is the len of the string +1 (':') +
			// the len of the number which is the length of the string
			lenStr := len(str) + 1 + numDigits(len(str))

			// advance the string
			bencodedString = bencodedString[lenStr:]

		case typeInt:
			val, err := decodeInt(bencodedString)
			if err != nil {
				return nil, err
			}

			// len of the number is the number of digits + the 2 (i+e)
			valLen := numDigits(val) + 2
			// for the -
			if val < 0 {
				valLen += 1
			}

			res = append(res, val)

			bencodedString = bencodedString[valLen:]

		case typeList:

			list, err := decodeList(bencodedString)
			if err != nil {
				return nil, err
			}

			fmt.Println("list", list)
		default:
			return nil, fmt.Errorf("unknown type")

		}
	}

	return res, nil
}

func numDigits(i int) int {
	var count int

	for i > 0 {
		i = i / 10
		count++
	}

	return count
}
