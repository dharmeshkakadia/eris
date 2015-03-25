package helper

import "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/ethutil"

func FromHex(h string) []byte {
	if ethutil.IsHex(h) {
		h = h[2:]
	}

	return ethutil.Hex2Bytes(h)
}
