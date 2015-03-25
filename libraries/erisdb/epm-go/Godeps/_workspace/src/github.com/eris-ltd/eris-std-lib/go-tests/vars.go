package vars

import (
	"bytes"
	"container/ring"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto" // each var comes with 3 permissions: add, rm, mod
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
	"math/big"
)

var StdVarSize = 4

// location of a var is 1 followed by the first 8 bytes of its sha3
func VariName(name string) []byte {
	h := monkcrypto.Sha3Bin(monkutil.PackTxDataArgs2(name))
	base := append([]byte{1}, h[:8]...)
	base = append(base, bytes.Repeat([]byte{0}, 32-len(base))...)
	return base
}

/*
   All interfaces are in strings, but array indices are ints.
*/

// Single Type Variable.
// (+ @@variname 1)
func GetSingle(addr []byte, name string, state *monkstate.State) []byte {
	obj := state.GetStateObject(addr)
	base := VariName(name)
	base[31] = byte(StdVarSize + 1)
	return (obj.GetStorage(monkutil.BigD(base))).Bytes()
}

// Return an element from an array type
func GetArrayElement(addr []byte, name string, index int, state *monkstate.State) []byte {
	return GetKeyedArrayElement(addr, name, "0x0", index, state)
}

// Return the entire array
func GetArray(addr []byte, name string, state *monkstate.State) [][]byte {
	return GetKeyedArray(addr, name, "0x0", state)
}

// key must come as hex!!!!!!
// Return an element from keyed array type
func GetKeyedArrayElement(addr []byte, name, key string, index int, state *monkstate.State) []byte {
	bigBase := big.NewInt(0)
	bigBase2 := big.NewInt(0)

	obj := state.GetStateObject(addr)
	base := VariName(name)

	// how big are the elements stored in this array:
	sizeLocator := make([]byte, len(base))
	copy(sizeLocator, base)
	sizeLocator = append(sizeLocator[:31], byte(StdVarSize+1))
	elementSizeBytes := (obj.GetStorage(monkutil.BigD(sizeLocator))).Bytes()
	elementSize := monkutil.BigD(elementSizeBytes).Uint64()

	// key should be trailing 20 bytes
	if len(key) >= 2 && key[:2] == "0x" {
		key = key[2:]
	}
	if l := len(key); l > 40 {
		key = key[l-40:]
	}

	// what slot does the array start at:
	keyBytes := monkutil.PackTxDataArgs2("0x" + key)
	keyBytesShift := append(keyBytes[3:], []byte{1, 0, 0}...)
	slotBig := bigBase.Add(monkutil.BigD(base), monkutil.BigD(keyBytesShift))

	//numElements := obj.GetStorage(slotBig)

	// which slot (row), and where in that slot (col) is the element we want:
	entriesPerRow := int64(256 / elementSize)
	rowN := int64(index) / entriesPerRow
	colN := int64(index) % entriesPerRow

	row := bigBase.Add(big.NewInt(1), bigBase.Add(slotBig, big.NewInt(rowN))).Bytes()
	rowStorage := (obj.GetStorage(monkutil.BigD(row))).Bytes()
	rowStorageBig := monkutil.BigD(rowStorage)

	elSizeBig := monkutil.BigD(elementSizeBytes)
	// row storage gives us a big number, from which we need to pull
	// an element of size elementsize.
	// so divide it by 2^(colN*elSize) and take modulo 2^elsize
	// divide row storage by 2^(colN*elSize)
	colBig := bigBase.Exp(big.NewInt(2), bigBase.Mul(elSizeBig, big.NewInt(colN)), nil)
	r := bigBase.Div(rowStorageBig, colBig)
	w := bigBase2.Exp(big.NewInt(2), elSizeBig, nil)
	v := bigBase.Mod(r, w)
	return v.Bytes()
}

// Return entire keyed array
func GetKeyedArray(addr []byte, name, key string, state *monkstate.State) [][]byte {
	return nil
}

// Return an element from a linked list
func GetLinkedListElement(addr []byte, name, key string, state *monkstate.State) []byte {

	bigBase := big.NewInt(0)

	obj := state.GetStateObject(addr)
	base := VariName(name)

	// key should be trailing 20 bytes
	if l := len(key); l > 20 {
		key = key[l-20:]
	}

	// get slot for this keyed element of linked list
	keyBytes := monkutil.PackTxDataArgs2(key)
	keyBytesShift := append(keyBytes, []byte{1, 0, 0}...)[3:]
	slotBig := bigBase.Add(monkutil.BigD(base), monkutil.BigD(keyBytesShift))

	// value is right at slot
	v := obj.GetStorage(slotBig)
	return v.Bytes()
}

func traverseLink(addr []byte, name, key string, state *monkstate.State, dir int) ([]byte, []byte) {
	bigBase := big.NewInt(0)

	obj := state.GetStateObject(addr)
	base := VariName(name)

	// key should be trailing 20 bytes
	if l := len(key); l > 20 {
		key = key[l-20:]
	}

	// get slot for this keyed element of linked list
	keyBytes := monkutil.PackTxDataArgs2(key)
	keyBytesShift := append(keyBytes, []byte{1, 0, 0}...)[3:]
	slotBig := bigBase.Add(monkutil.BigD(base), monkutil.BigD(keyBytesShift))

	// next locator is this slot plus 2
	// prev locator is this slot plus 1
	slotNextLoc := bigBase.Add(slotBig, big.NewInt(int64(dir)))
	slotNext := obj.GetStorage(slotNextLoc)
	if slotNext.IsNil() {
		slotNextLoc = monkutil.BigD(append(base[:len(base)-1], byte(StdVarSize+1)))
		slotNext = obj.GetStorage(slotNextLoc)
	}

	keyB := slotNext.Bytes()
	keyB = keyB[9:29]
	// value is right at slot
	v := obj.GetStorage(slotNext.BigInt())
	return keyB, v.Bytes()
}

// return the key and value for the next element in the linked list
// NOTE: key here is ascii bytes!!!
func GetNextLinkedListElement(addr []byte, name, key string, state *monkstate.State) ([]byte, []byte) {
	return traverseLink(addr, name, key, state, 2)
}

func GetPrevLinkedListElement(addr []byte, name, key string, state *monkstate.State) ([]byte, []byte) {
	return traverseLink(addr, name, key, state, 1)
}

func GetLinkedListLength(addr []byte, name string, state *monkstate.State) int {
	obj := state.GetStateObject(addr)
	base := VariName(name)
	return getLinkedListLength(obj, base)
}

func getLinkedListLength(obj *monkstate.StateObject, base []byte) int {
	nEleLoc := append(base[:len(base)-1], byte(StdVarSize+3))
	nEl := obj.GetStorage(monkutil.BigD(nEleLoc))
	return int(nEl.Uint())
}

func GetLinkedListHead(addr []byte, name string, state *monkstate.State) ([]byte, []byte) {
	obj := state.GetStateObject(addr)
	base := VariName(name)
	return getLinkedListHead(obj, base)
}

func getLinkedListHead(obj *monkstate.StateObject, base []byte) ([]byte, []byte) {
	headLocLoc := append(base[:len(base)-1], byte(StdVarSize+1))
	headLoc := obj.GetStorage(monkutil.BigD(headLocLoc))
	headKey := headLoc.Bytes()[9:29]
	head := obj.GetStorage(headLoc.BigInt())
	return headKey, head.Bytes()
}

func GetLinkedList(addr []byte, name string, state *monkstate.State) *ring.Ring {
	bigBase := big.NewInt(0)

	obj := state.GetStateObject(addr)
	base := VariName(name)

	n := getLinkedListLength(obj, base)

	nextLocLoc := monkutil.BigD(append(base[:len(base)-1], byte(StdVarSize+1)))
	nextLoc := obj.GetStorage(nextLocLoc).BigInt()

	r := ring.New(n)
	for i := 0; i < n; i++ {
		r.Value = obj.GetStorage(nextLoc)
		r = r.Next()

		nextLocLoc = bigBase.Add(nextLoc, big.NewInt(2))
		nextLoc = obj.GetStorage(nextLocLoc).BigInt()
	}

	return r
}
