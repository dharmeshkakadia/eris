package monkchain

import (
	"math/big"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
)

type VMEnv struct {
	state *monkstate.State
	block *Block
	tx    *Transaction
}

func NewEnv(state *monkstate.State, tx *Transaction, block *Block) *VMEnv {
	return &VMEnv{
		state: state,
		block: block,
		tx:    tx,
	}
}

func (self *VMEnv) Origin() []byte          { return self.tx.Sender() }
func (self *VMEnv) BlockNumber() *big.Int   { return self.block.Number }
func (self *VMEnv) PrevHash() []byte        { return self.block.PrevHash }
func (self *VMEnv) Coinbase() []byte        { return self.block.Coinbase }
func (self *VMEnv) Time() int64             { return self.block.Time }
func (self *VMEnv) Difficulty() *big.Int    { return self.block.Difficulty }
func (self *VMEnv) BlockHash() []byte       { return self.block.Hash() }
func (self *VMEnv) Value() *big.Int         { return self.tx.Value }
func (self *VMEnv) State() *monkstate.State { return self.state }
func (self *VMEnv) Doug() []byte            { return genDoug.Doug() }
func (self *VMEnv) DougValidate(addr []byte, role string, state *monkstate.State) error {
	return genDoug.ValidatePerm(addr, role, state)
}
