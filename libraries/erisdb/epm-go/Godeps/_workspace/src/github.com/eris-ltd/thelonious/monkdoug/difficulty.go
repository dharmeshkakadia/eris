package monkdoug

import (
	"bytes"
	"math/big"

	vars "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/eris-std-lib/go-tests"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

// Base difficulty of the chain is 2^($difficulty), with $difficulty
// stored in GenDoug
func (m *StdLibModel) baseDifficulty(state *monkstate.State) *big.Int {
	difv := vars.GetSingle(m.doug, "difficulty", state)
	return monkutil.BigPow(2, int(monkutil.ReadVarInt(difv)))
}

// Adjust difficulty to meet block time
// TODO: testing and robustify. this is leaky
func adjustDifficulty(oldDiff *big.Int, oldTime, newTime, target int64) *big.Int {
	diff := new(big.Int)
	adjust := new(big.Int).Rsh(oldDiff, 8)
	if newTime >= oldTime+target {
		diff.Sub(oldDiff, adjust)
	} else {
		diff.Add(oldDiff, adjust)
	}
	return diff
}

// Difficulty for miners in a round robin
func (m *StdLibModel) RoundRobinDifficulty(block, parent *monkchain.Block) *big.Int {
	state := parent.State()
	// get base difficulty
	newdiff := m.baseDifficulty(state)
	// get target block time
	blockTimeBytes := vars.GetSingle(m.doug, "blocktime", parent.State())
	blockTime := monkutil.BigD(blockTimeBytes).Int64()
	// adjust difficulty in pursuit of holy target block time
	newdiff = adjustDifficulty(newdiff, parent.Time, block.Time, blockTime)
	// find relative position of coinbase in the linked list (i)
	// difficulty should be (base difficulty)*2^i
	var i int
	nMiners := vars.GetLinkedListLength(m.doug, "seq:name", state)
	// this is the proper next coinbase
	next := m.nextCoinbase(parent)
	for i = 0; i < nMiners; i++ {
		if bytes.Equal(next, block.Coinbase) {
			break
		}
		next, _ = vars.GetNextLinkedListElement(m.doug, "seq:name", string(next), state)
	}
	newdiff = big.NewInt(0).Mul(monkutil.BigPow(2, i), newdiff)
	return newdiff
}

func (m *StdLibModel) StakeDifficulty(block, parent *monkchain.Block) *big.Int {
	//TODO
	return nil
}

// difficulty targets a specific block time
func EthDifficulty(timeTarget int64, block, parent *monkchain.Block) *big.Int {
	return adjustDifficulty(parent.Difficulty, parent.Time, block.Time, timeTarget)
}
