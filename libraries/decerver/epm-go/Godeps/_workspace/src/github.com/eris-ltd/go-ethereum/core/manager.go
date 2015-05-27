package core

import (
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/crypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/ethutil"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/event"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/p2p"
)

type EthManager interface {
	BlockProcessor() *BlockProcessor
	ChainManager() *ChainManager
	TxPool() *TxPool
	PeerCount() int
	IsMining() bool
	IsListening() bool
	Peers() []*p2p.Peer
	KeyManager() *crypto.KeyManager
	ClientIdentity() p2p.ClientIdentity
	Db() ethutil.Database
	EventMux() *event.TypeMux
}
