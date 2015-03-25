package monkchain

import (
	"math/big"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

/*
 * This is the special genesis block.
 */var ZeroHash256 = make([]byte, 32)
var ZeroHash160 = make([]byte, 20)
var EmptyShaList = monkcrypto.Sha3Bin(monkutil.Encode([]interface{}{}))

var GenesisHeader = []interface{}{
	// Previous hash (none)
	ZeroHash256,
	// Sha of uncles
	monkcrypto.Sha3Bin(monkutil.Encode([]interface{}{})),
	// Coinbase
	ZeroHash160,
	// Root state
	"",
	// tx sha
	"",
	// Difficulty
	//monkutil.BigPow(2, 22),
	big.NewInt(131072),
	// Number
	monkutil.Big0,
	// Block minimum gas price
	monkutil.Big0,
	// Block upper gas bound
	big.NewInt(1000000),
	// Block gas used
	monkutil.Big0,
	// Time
	monkutil.Big0,
	// Extra
	nil,
	// Nonce
	monkcrypto.Sha3Bin(big.NewInt(42).Bytes()),
}

var Genesis = []interface{}{GenesisHeader, []interface{}{}, []interface{}{}}
