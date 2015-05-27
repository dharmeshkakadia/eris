package monkpipe

import (
	"testing"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

func Val(v interface{}) *monkutil.Value {
	return monkutil.NewValue(v)
}

func _TestNew(t *testing.T) {
	pipe := New(nil)

	var addr, privy, recp, data []byte
	var object *Object
	var key *monkcrypto.KeyPair

	world := pipe.World()
	world.Get(addr)
	world.Coinbase()
	world.IsMining()
	world.IsListening()
	world.State()
	peers := world.Peers()
	peers.Len()

	// Shortcut functions
	pipe.Balance(addr)
	pipe.Nonce(addr)
	pipe.Block(addr)
	pipe.Storage(addr, addr)
	pipe.ToAddress(privy)
	pipe.Exists(addr)
	// Doesn't change state
	pipe.Execute(addr, nil, Val(0), Val(1000000), Val(10))
	// Doesn't change state
	pipe.ExecuteObject(object, nil, Val(0), Val(1000000), Val(10))

	conf := world.Config()
	namereg := conf.Get("NameReg")
	namereg.Storage(addr)

	var err error
	// Transact
	_, err = pipe.Transact(key, recp, monkutil.NewValue(0), monkutil.NewValue(0), monkutil.NewValue(0), "")
	if err != nil {
		t.Error(err)
	}
	// Create
	_, err = pipe.Transact(key, nil, monkutil.NewValue(0), monkutil.NewValue(0), monkutil.NewValue(0), string(data))
	if err != nil {
		t.Error(err)
	}
}
