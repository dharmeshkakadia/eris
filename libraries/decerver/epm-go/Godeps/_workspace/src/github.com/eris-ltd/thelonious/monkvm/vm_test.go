package monkvm

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"testing"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

type TestEnv struct {
}

func (self TestEnv) Origin() []byte                                                      { return nil }
func (self TestEnv) BlockNumber() *big.Int                                               { return nil }
func (self TestEnv) PrevHash() []byte                                                    { return nil }
func (self TestEnv) Coinbase() []byte                                                    { return nil }
func (self TestEnv) Time() int64                                                         { return 0 }
func (self TestEnv) Difficulty() *big.Int                                                { return nil }
func (self TestEnv) BlockHash() []byte                                                   { return nil }
func (self TestEnv) Value() *big.Int                                                     { return nil }
func (self TestEnv) State() *monkstate.State                                             { return nil }
func (self TestEnv) Doug() []byte                                                        { return nil }
func (self TestEnv) DougValidate(addr []byte, role string, state *monkstate.State) error { return nil }

func TestVm(t *testing.T) {
	monklog.AddLogSystem(monklog.NewStdLogSystem(os.Stdout, log.LstdFlags, monklog.LogLevel(4)))

	monkutil.ReadConfig(".monktest", "/tmp/monktest", "")

	stateObject := monkstate.NewStateObject([]byte{'j', 'e', 'f', 'f'})
	callerClosure := NewClosure(new(monkstate.Message), stateObject, stateObject, []byte{0x60, 0x01}, big.NewInt(1000000), big.NewInt(0))

	vm := New(TestEnv{})
	vm.Verbose = true

	ret, _, e := callerClosure.Call(vm, nil)
	if e != nil {
		fmt.Println("error", e)
	}
	fmt.Println(ret)
}
