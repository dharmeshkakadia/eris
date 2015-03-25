package monkstate

import (
	"testing"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkdb"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monktrie"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

var ZeroHash256 = make([]byte, 32)

func TestSnapshot(t *testing.T) {
	db, _ := monkdb.NewMemDatabase()
	monkutil.ReadConfig(".monktest", "/tmp/monktest", "")
	monkutil.Config.Db = db

	state := New(monktrie.New(db, ""))

	stateObject := state.GetOrNewStateObject([]byte("aa"))

	stateObject.SetStorage(monkutil.Big("0"), monkutil.NewValue(42))

	snapshot := state.Copy()

	stateObject = state.GetStateObject([]byte("aa"))
	stateObject.SetStorage(monkutil.Big("0"), monkutil.NewValue(43))

	state.Set(snapshot)

	stateObject = state.GetStateObject([]byte("aa"))
	res := stateObject.GetStorage(monkutil.Big("0"))
	if !res.Cmp(monkutil.NewValue(42)) {
		t.Error("Expected storage 0 to be 42", res)
	}
}
