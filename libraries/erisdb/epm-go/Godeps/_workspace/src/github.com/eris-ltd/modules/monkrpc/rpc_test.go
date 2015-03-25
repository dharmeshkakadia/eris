package monkrpc

import (
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monk"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkdoug"
	"log"
	"testing"
	"time"
)

var Monk *monk.MonkModule

func setConfig() {
	Monk.Config.RpcPort = 30304
	Monk.Config.ServeRpc = true
	Monk.Config.Listen = false
	Monk.Config.Mining = false
}

func init() {
	// start a monk instance serving rpc
	Monk = monk.NewMonk(nil)
	Monk.GenesisConfig = monkdoug.DefaultGenesis
	setConfig()
	Monk.Init()
	Monk.Start()
	time.Sleep(2 * time.Second)
}

func TestRpcLocalTx(t *testing.T) {
	r := NewMonkRpcModule()
	r.Config.LogLevel = 5
	r.Config.RpcPort = 30304
	err := r.Init()
	if err != nil {
		log.Fatal(err)
	}
	r.Start()

	_, err = r.Tx("babcdef0192345678901abcdef0192345678901a", "99996665555")
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Second)
	_, err = r.Tx("aabcdef0192345678901abcdef0192345678901a", "99996665555")
	if err != nil {
		log.Fatal(err)
	}

	r.Shutdown()
}

func TestRpcRemoteTx(t *testing.T) {
	r := NewMonkRpcModule()
	r.Config.LogLevel = 5
	r.Config.RpcPort = 30304
	r.Config.Local = false
	err := r.Init()
	if err != nil {
		log.Fatal(err)
	}
	r.Start()

	r.Tx("cabcdef0192345678901abcdef0192345678901a", "99996665555")
	time.Sleep(time.Second * 2)
	r.Shutdown()
}
