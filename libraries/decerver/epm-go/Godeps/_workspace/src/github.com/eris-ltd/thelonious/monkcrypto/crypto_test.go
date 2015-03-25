package monkcrypto

import (
	"bytes"
	"testing"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

// FIPS 202 test (reverted back to FIPS 180)
func TestSha3(t *testing.T) {
	const exp = "4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45"
	sha3_256 := Sha3Bin([]byte("abc"))
	if bytes.Compare(sha3_256, monkutil.Hex2Bytes(exp)) != 0 {
		t.Errorf("Sha3_256 failed. Incorrect result %x", sha3_256)
	}
}
