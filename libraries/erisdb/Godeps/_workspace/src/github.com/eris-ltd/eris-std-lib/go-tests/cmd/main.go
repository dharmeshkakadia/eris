package main

import (
	"fmt"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/eris-std-lib/go-tests"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monk"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

func main() {
	e := monk.NewMonk(nil)
	e.ReadConfig("eth-config.json")
	g := e.LoadGenesis(e.Config.GenesisConfig)
	g.NoGenDoug = true
	g.ModelName = "yes"
	e.SetGenesis(g)
	e.Init()

	e.Start()

	addrHex, _ := e.Script("var-tests.lll", "lll")
	addr := monkutil.Hex2Bytes(addrHex)

	fmt.Println(addr)

	e.Commit()

	state := e.MonkState()

	// test single
	s := vars.GetSingle(addr, "mySingle", state)
	fmt.Println(monkutil.Bytes2Hex(s))

	// test array
	t := vars.GetArrayElement(addr, "myArray", 2, state)
	fmt.Println(monkutil.Bytes2Hex(t))
	t = vars.GetArrayElement(addr, "myArray", 5, state)
	fmt.Println(monkutil.Bytes2Hex(t))
	t = vars.GetArrayElement(addr, "myArray", 6, state)
	fmt.Println(monkutil.Bytes2Hex(t))

	// test linked list
	k, l := vars.GetLinkedListHead(addr, "myLL", state)
	fmt.Println(monkutil.Bytes2Hex(l))
	l = vars.GetLinkedListElement(addr, "myLL", "balls", state)
	fmt.Println(monkutil.Bytes2Hex(l))
	l = vars.GetLinkedListElement(addr, "myLL", "paws", state)
	fmt.Println(monkutil.Bytes2Hex(l))
	n := vars.GetLinkedListLength(addr, "myLL", state)
	fmt.Println(n)
	r := vars.GetLinkedList(addr, "myLL", state)
	for i := 0; i < r.Len(); i++ {
		fmt.Println(r.Value)
		r = r.Next()
	}
	k, l = vars.GetNextLinkedListElement(addr, "myLL", "balls", state)
	fmt.Println(monkutil.Bytes2Hex(k), l)
	k, l = vars.GetNextLinkedListElement(addr, "myLL", "paws", state)
	fmt.Println(monkutil.Bytes2Hex(k), l)
	k, l = vars.GetNextLinkedListElement(addr, "myLL", "monkeyturd", state)
	fmt.Println(monkutil.Bytes2Hex(k), l)

	for i := 0; i < 10; i++ {
		k, l = vars.GetNextLinkedListElement(addr, "myLL", string(k), state)
		fmt.Println(string(k), l)

	}
}
