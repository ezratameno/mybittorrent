package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeString(t *testing.T) {

	res, err := decodeString("5:hello")
	require.NoError(t, err)
	require.Equal(t, "hello", res)

	res, err = decodeString("10:hello12345")
	require.NoError(t, err)
	require.Equal(t, "hello12345", res)

}

func TestDecodeInt(t *testing.T) {
	res, err := decodeInt("i52e")
	require.NoError(t, err)
	require.Equal(t, 52, res)

	res, err = decodeInt("i-52e")
	require.NoError(t, err)
	require.Equal(t, -52, res)
}

func TestNumDigits(t *testing.T) {

	require.Equal(t, 3, numDigits(100))
	require.Equal(t, 1, numDigits(1))

}

func TestDecodeList(t *testing.T) {
	res, err := decodeList("l5:helloi52ee")
	require.NoError(t, err)

	require.Equal(t, []any{"hello", 52}, res)

	res, err = decodeList("lli673e10:strawberryee")
	require.NoError(t, err)

	fmt.Println(res)

}
