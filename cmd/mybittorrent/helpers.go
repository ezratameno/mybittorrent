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
		str, _, err := decodeString(bencodedString)
		return str, err

	case typeInt:
		intVal, _, err := decodeInt(bencodedString)
		return intVal, err

	case typeList:
		list, _, err := decodeList(bencodedString)
		return list, err

	default:
		return "", fmt.Errorf("only strings are supported at the moment")
	}

}

// Example:
// - 5:hello -> hello
// - 10:hello12345 -> hello12345
// decodeString decode the bencoded string and returns the end index of the string
func decodeString(bencodedString string) (string, int, error) {

	lenEndIndex := strings.Index(bencodedString, ":")
	if lenEndIndex == -1 {
		return "", 0, fmt.Errorf("string in the wrong format, expected at least one ':' ")
	}

	stringLen, err := strconv.Atoi(bencodedString[:lenEndIndex])
	if err != nil {
		return "", 0, err
	}

	startOfContentIndex := lenEndIndex + 1

	stringContent := bencodedString[startOfContentIndex : stringLen+startOfContentIndex]

	// the len of the portion that we read to get the str is the len of the string +1 (':') +
	// the len of the number which is the length of the string
	lenStr := len(stringContent) + 1 + numDigits(len(stringContent))
	return stringContent, lenStr, nil

}

// format - i<number>e
// decodeInt decode the bencoded int and return the end index of the int.
func decodeInt(bencodedString string) (int, int, error) {
	bencodedString = strings.TrimPrefix(bencodedString, "i")

	endOfIntIndex := strings.Index(bencodedString, "e")
	bencodedString = bencodedString[:endOfIntIndex]

	intVal, err := strconv.Atoi(bencodedString)
	if err != nil {
		return 0, 0, err
	}

	valLen := numDigits(intVal) + 2
	// for the -
	if intVal < 0 {
		valLen += 1
	}
	return intVal, valLen, nil
}

func decodeList(bencodedString string) ([]any, int, error) {

	res := make([]any, 0)
	listLen := 2
	bencodedString = strings.TrimPrefix(bencodedString, "l")
	bencodedString = strings.TrimSuffix(bencodedString, "e")

	for len(bencodedString) > 0 {
		switch bencodeType(bencodedString) {
		case typeString:
			fmt.Println("string: ", bencodedString)

			str, lenStr, err := decodeString(bencodedString)
			if err != nil {
				return nil, 0, err
			}

			res = append(res, str)
			// advance the string
			bencodedString = bencodedString[lenStr:]

			listLen += lenStr

		case typeInt:
			fmt.Println("int: ", bencodedString)

			intVal, intLen, err := decodeInt(bencodedString)
			if err != nil {
				return nil, 0, err
			}

			res = append(res, intVal)

			bencodedString = bencodedString[intLen:]
			listLen += intLen

		case typeList:

			fmt.Println("list: ", bencodedString)

			list, len, err := decodeList(bencodedString)
			if err != nil {
				return nil, 0, err
			}

			listLen += len

			res = append(res, list)

			bencodedString = bencodedString[len:]
			fmt.Println("list", list)
			fmt.Println("list len", len)

		default:
			return nil, 0, fmt.Errorf("unknown type")

		}
	}

	return res, listLen, nil
}

func numDigits(i int) int {
	var count int

	if i < 0 {
		i = -1 * i
	}
	for i > 0 {
		i = i / 10
		count++
	}

	return count
}
