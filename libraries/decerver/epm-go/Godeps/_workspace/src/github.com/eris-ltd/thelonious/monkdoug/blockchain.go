package monkdoug

import (
	"bytes"
	vars "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/eris-std-lib/go-tests"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
	"math/big"
)

func (m *StdLibModel) consensus(state *monkstate.State) string {
	consensusBytes := vars.GetSingle(m.doug, "consensus", state)
	consensus := string(consensusBytes)
	return consensus
}

func (m *StdLibModel) blocktime(state *monkstate.State) int64 {
	blockTimeBytes := vars.GetSingle(m.doug, "blocktime", state)
	blockTime := monkutil.BigD(blockTimeBytes).Int64()
	return blockTime
}

// Who should the next block be mined by?
// TODO: Accomodate for a dynamic number of miners!
func (m *StdLibModel) nextCoinbase(prevblock *monkchain.Block) []byte {
	var nextcoinbase []byte
	// if its not the genesis block, get coinbase of last block
	// if it is genesis block, find first entry in linked list
	if !bytes.Equal(prevblock.PrevHash, monkchain.ZeroHash256) {
		// i = blockN % nMiners
		nblocks := prevblock.Number
		nMiners := vars.GetLinkedListLength(m.doug, "seq:name", prevblock.State())
		nB := m.base.Mod(nblocks, big.NewInt(int64(nMiners)))
		n := int(nB.Int64())
		next, _ := vars.GetLinkedListHead(m.doug, "seq:name", prevblock.State())
		for i := 0; i < n; i++ {
			next, _ = vars.GetNextLinkedListElement(m.doug, "seq:name", string(next), prevblock.State())
		}
		nextcoinbase = next
	} else {
		nextcoinbase, _ = vars.GetLinkedListHead(m.doug, "seq:name", prevblock.State())
	}
	return nextcoinbase
}

// TODO !
func (m *StdLibModel) CheckUncles(prevBlock, block *monkchain.Block) error {
	// Check each uncle's previous hash. In order for it to be valid
	// is if it has the same block hash as the current
	/*
		for _, uncle := range block.Uncles {
			if bytes.Compare(uncle.PrevHash,prevBlock.PrevHash) != 0 {
				return ValidationError("Mismatch uncle's previous hash. Expected %x, got %x",prevBlock.PrevHash, uncle.PrevHash)
			}
		}
	*/
	return nil
}

func CheckBlockTimes(prevBlock, block *monkchain.Block) error {
	diff := block.Time - prevBlock.Time
	if diff < 0 {
		return monkchain.ValidationError("Block timestamp less then prev block %v (%v - %v)", diff, block.Time, prevBlock.Time)
	}

	/* XXX
	// New blocks must be within the 15 minute range of the last block.
	if diff > int64(15*time.Minute) {
		return ValidationError("Block is too far in the future of last block (> 15 minutes)")
	}
	*/
	return nil
}
