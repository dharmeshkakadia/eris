package types

import (
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/ethdb"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/ethutil"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/trie"
)

type DerivableList interface {
	Len() int
	GetRlp(i int) []byte
}

func DeriveSha(list DerivableList) []byte {
	db, _ := ethdb.NewMemDatabase()
	trie := trie.New(nil, db)
	for i := 0; i < list.Len(); i++ {
		trie.Update(ethutil.Encode(i), list.GetRlp(i))
	}

	return trie.Root()
}
