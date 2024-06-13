package main

import (
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
