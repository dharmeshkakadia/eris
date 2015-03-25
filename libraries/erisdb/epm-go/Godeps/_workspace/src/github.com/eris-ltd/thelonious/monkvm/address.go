package monkvm

import (
	"math/big"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

type Address interface {
	Call(in []byte) []byte
}

type PrecompiledAddress struct {
	Gas *big.Int
	fn  func(in []byte) []byte
}

func (self PrecompiledAddress) Call(in []byte) []byte {
	return self.fn(in)
}

var Precompiled = map[string]*PrecompiledAddress{
	"ecrecover": &PrecompiledAddress{big.NewInt(500), ecrecoverFunc},
	"sha256":    &PrecompiledAddress{big.NewInt(100), sha256Func},
	"ripemd160": &PrecompiledAddress{big.NewInt(100), ripemd160Func},
}

func sha256Func(in []byte) []byte {
	return monkcrypto.Sha256(in)
}

func ripemd160Func(in []byte) []byte {
	return monkutil.RightPadBytes(monkcrypto.Ripemd160(in), 32)
}

func ecrecoverFunc(in []byte) []byte {
	// In case of an invalid sig. Defaults to return nil
	defer func() { recover() }()

	addr := monkcrypto.Ecrecover(in)
	// we want to pad the return (its only 20 bytes)
	return monkutil.LeftPadBytes(addr, 32)
}
