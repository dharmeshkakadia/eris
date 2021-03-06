package xeth

import "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/state"

type State struct {
	xeth *XEth
}

func NewState(xeth *XEth) *State {
	return &State{xeth}
}

func (self *State) State() *state.StateDB {
	return self.xeth.chainManager.TransState()
}

func (self *State) Get(addr string) *Object {
	return &Object{self.State().GetStateObject(fromHex(addr))}
}

func (self *State) SafeGet(addr string) *Object {
	return &Object{self.safeGet(addr)}
}

func (self *State) safeGet(addr string) *state.StateObject {
	object := self.State().GetStateObject(fromHex(addr))
	if object == nil {
		object = state.NewStateObject(fromHex(addr), self.xeth.eth.Db())
	}

	return object
}
