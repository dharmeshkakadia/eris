package monkchain

import (
	"fmt"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monktrie"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
	"math/big"
	"testing"
)

type fDoug struct{}

// Populate the state
func (d *fDoug) Deploy(block *Block) ([]byte, error) {
	for _, acct := range [][]string{
		[]string{"abc123", "9876"},
		[]string{"321cba", "1234"},
	} {
		account := block.State().GetAccount(monkutil.Hex2Bytes(acct[0]))
		account.Balance = monkutil.Big(acct[1])
		block.State().UpdateStateObject(account)
	}
	block.State().Update()
	block.State().Sync()
	return nil, nil
}

func (d *fDoug) Doug() []byte { return nil }
func (d *fDoug) ValidateChainID(chainId []byte, genBlock *Block) error {
	return nil
}

func (d *fDoug) Participate(coinbase []byte, parent *Block) bool                     { return false }
func (d *fDoug) Difficulty(block, parent *Block) *big.Int                            { return nil }
func (d *fDoug) ValidatePerm(addr []byte, role string, state *monkstate.State) error { return nil }
func (d *fDoug) ValidateBlock(block *Block, bc *ChainManager) error                  { return nil }
func (d *fDoug) ValidateTx(tx *Transaction, state *monkstate.State) error            { return nil }
func (d *fDoug) CheckPoint(proposed []byte, bc *ChainManager) bool                   { return false }

// Dead simple copy (key, value) pairs of a trie
// Note this is not enough since some value's might be hashes of other tries
// For now we keep it simple
func TestSerialize(t *testing.T) {
	DB[0].Put([]byte("ChainID"), []byte{0x5, 0x4})
	cman := NewChainManager(&fDoug{})

	tr := cman.CurrentBlock().State().Trie
	trIt := tr.NewIterator()
	response := []interface{}{}
	trIt.Each(func(key string, val *monkutil.Value) {
		pair := []interface{}{[]byte(key), val.Bytes()}
		response = append(response, pair) //monkutil.NewValue(pair).Encode())
	})
	fmt.Println(response)
	fmt.Println(tr.Root)

	data := monkutil.NewValue(response)

	//data = monkutil.NewValue([]interface{}{"ethan", "jim"})
	fmt.Println("data:", data.Len())

	tr2 := monktrie.New(DB[1], "")
	for i := 0; i < data.Len(); i++ {
		l := data.Get(i)
		fmt.Println(i, l, l.Get(0).Bytes())
		tr2.Update(string(l.Get(0).Bytes()), string(l.Get(1).Bytes()))
	}
	tr2.Sync()

	trIt = tr2.NewIterator()
	response = []interface{}{}
	trIt.Each(func(key string, val *monkutil.Value) {
		pair := [2][]byte{[]byte(key), val.Bytes()}
		response = append(response, pair) //monkutil.NewValue(pair).Encode())
	})

	fmt.Println(response)
	fmt.Println(tr2.Root)

}
