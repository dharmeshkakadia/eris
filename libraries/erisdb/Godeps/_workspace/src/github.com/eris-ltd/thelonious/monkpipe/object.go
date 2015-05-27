package monkpipe

import (
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

type Object struct {
	*monkstate.StateObject
}

func (self *Object) StorageString(str string) *monkutil.Value {
	if monkutil.IsHex(str) {
		return self.Storage(monkutil.Hex2Bytes(str[2:]))
	} else {
		return self.Storage(monkutil.RightPadBytes([]byte(str), 32))
	}
}

func (self *Object) StorageValue(addr *monkutil.Value) *monkutil.Value {
	return self.Storage(addr.Bytes())
}

func (self *Object) Storage(addr []byte) *monkutil.Value {
	return self.StateObject.GetStorage(monkutil.BigD(addr))
}
