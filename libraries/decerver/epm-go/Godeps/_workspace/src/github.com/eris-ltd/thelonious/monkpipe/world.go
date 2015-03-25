package monkpipe

import (
	"container/list"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
)

type World struct {
	pipe *Pipe
	cfg  *Config
}

func NewWorld(pipe *Pipe) *World {
	world := &World{pipe, nil}
	world.cfg = &Config{pipe}

	return world
}

func (self *Pipe) World() *World {
	return self.world
}

func (self *World) State() *monkstate.State {
	return self.pipe.stateManager.TransState() //CurrentState()
}

func (self *World) Get(addr []byte) *Object {
	return &Object{self.State().GetStateObject(addr)}
}

func (self *World) SafeGet(addr []byte) *Object {
	return &Object{self.safeGet(addr)}
}

func (self *World) safeGet(addr []byte) *monkstate.StateObject {
	object := self.State().GetStateObject(addr)
	if object == nil {
		object = monkstate.NewStateObject(addr)
	}

	return object
}

func (self *World) Coinbase() *monkstate.StateObject {
	return nil
}

func (self *World) IsMining() bool {
	return self.pipe.obj.IsMining()
}

func (self *World) IsListening() bool {
	return self.pipe.obj.IsListening()
}

func (self *World) Peers() *list.List {
	return self.pipe.obj.Peers()
}

func (self *World) Config() *Config {
	return self.cfg
}
