package core

import (
	bc "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/blockchain"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/consensus"
	mempl "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/mempool"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/p2p"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/state"
)

var blockStore *bc.BlockStore
var consensusState *consensus.ConsensusState
var consensusReactor *consensus.ConsensusReactor
var mempoolReactor *mempl.MempoolReactor
var p2pSwitch *p2p.Switch

func SetBlockStore(bs *bc.BlockStore) {
	blockStore = bs
}

func SetConsensusState(cs *consensus.ConsensusState) {
	consensusState = cs
}

func SetConsensusReactor(cr *consensus.ConsensusReactor) {
	consensusReactor = cr
}

func SetMempoolReactor(mr *mempl.MempoolReactor) {
	mempoolReactor = mr
}

func SetSwitch(sw *p2p.Switch) {
	p2pSwitch = sw
}

// JAE Why is this here?
func SetPrivValidator(priv *state.PrivValidator) {
	consensusReactor.SetPrivValidator(priv)
}
