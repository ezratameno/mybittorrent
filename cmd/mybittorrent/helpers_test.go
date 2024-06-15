package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeString(t *testing.T) {

	res, lenStr, err := decodeString("5:hello")
	require.NoError(t, err)
	require.Equal(t, "hello", res)
	require.Equal(t, 7, lenStr)

	res, lenStr, err = decodeString("10:hello12345")
	require.NoError(t, err)
	require.Equal(t, "hello12345", res)
	require.Equal(t, 13, lenStr)

}

func TestDecodeInt(t *testing.T) {
	res, resLen, err := decodeInt("i52e")
	require.NoError(t, err)
	require.Equal(t, 52, res)
	require.Equal(t, 4, resLen)

	res, resLen, err = decodeInt("i-52e")
	require.NoError(t, err)
	require.Equal(t, 5, resLen)
}

func TestNumDigits(t *testing.T) {

	require.Equal(t, 3, numDigits(100))
	require.Equal(t, 1, numDigits(1))
	require.Equal(t, 3, numDigits(-100))

}

func TestDecodeList(t *testing.T) {
	res, resLen, err := decodeList("l5:helloi52ee")
	require.NoError(t, err)

	fmt.Println(resLen)

	require.Equal(t, []any{"hello", 52}, res)

	res, resLen, err = decodeList("lli673e10:strawberryee")
	require.NoError(t, err)
	fmt.Println(resLen)

	fmt.Printf("res :%+v\n", res)

}
