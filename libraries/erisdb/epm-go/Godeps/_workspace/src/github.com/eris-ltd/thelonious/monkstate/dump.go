package monkstate

import (
	"encoding/json"
	"fmt"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

type Account struct {
	Balance  string            `json:"balance"`
	Nonce    uint64            `json:"nonce"`
	CodeHash string            `json:"codeHash"`
	Storage  map[string]string `json:"storage"`
}

type World struct {
	Root     string             `json:"root"`
	Accounts map[string]Account `json:"accounts"`
}

func (self *State) Dump() []byte {
	world := World{
		Root:     monkutil.Bytes2Hex(self.Trie.Root.([]byte)),
		Accounts: make(map[string]Account),
	}

	self.Trie.NewIterator().Each(func(key string, value *monkutil.Value) {
		stateObject := NewStateObjectFromBytes([]byte(key), value.Bytes())

		account := Account{Balance: stateObject.Balance.String(), Nonce: stateObject.Nonce, CodeHash: monkutil.Bytes2Hex(stateObject.codeHash)}
		account.Storage = make(map[string]string)

		stateObject.EachStorage(func(key string, value *monkutil.Value) {
			value.Decode()
			account.Storage[monkutil.Bytes2Hex([]byte(key))] = monkutil.Bytes2Hex(value.Bytes())
		})
		world.Accounts[monkutil.Bytes2Hex([]byte(key))] = account
	})

	json, err := json.MarshalIndent(world, "", "    ")
	if err != nil {
		fmt.Println("dump err", err)
	}

	return json
}
