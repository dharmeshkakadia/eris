package epm

import (
	"fmt"
	"github.com/eris-ltd/modules/types"
)

func (e *EPM) CurrentState() types.State { //map[string]string{
	if e.chain == nil {
		return types.State{}
	}
	return *(e.chain.State())
}

func (e *EPM) checkTakeStateDiff(i int) {
	if _, ok := e.diffSched[i]; !ok {
		return
	}
	e.chain.Commit()
	scheds := e.diffSched[i]
	names := e.diffName[i]
	for j, sched := range scheds {
		name := names[j]
		if sched == 0 {
			// store state
			e.states[name] = e.CurrentState()
		} else {
			// take diff
			e.chain.Commit()
			PrintDiff(name, e.states[name], e.CurrentState())
		}
	}
}

func StorageDiff(pre, post types.State) types.State { //map[string]string) map[string]map[string]string{
	diff := types.State{make(map[string]*types.Storage), []string{}}
	// for each account in post, compare all elements.
	for _, addr := range post.Order {
		acct := post.State[addr]
		diff.State[addr] = &types.Storage{make(map[string]string), []string{}}
		diff.Order = append(diff.Order, addr)
		acct2, ok := pre.State[addr]
		if !ok {
			// if this account didnt exist in pre
			diff.State[addr] = acct
			continue
		}
		// for each storage in the post acct, check for diff in 2.
		for _, k := range acct.Order {
			v := acct.Storage[k]
			v2, ok := acct2.Storage[k]
			// if its not in the pre-state or its different, add to diff
			if !ok || v2 != v {
				diff.State[addr].Storage[k] = v
				st := diff.State[addr]
				st.Order = append(diff.State[addr].Order, k)
				diff.State[addr] = st
			}
		}
	}
	return diff
}

func PrettyPrintAcctDiff(dif types.State) string { //map[string]string) string{
	result := ""
	for _, addr := range dif.Order {
		acct := dif.State[addr]
		if len(acct.Order) == 0 {
			continue
		}
		result += addr + ":\n"
		for _, store := range acct.Order {
			v := acct.Storage[store]
			val := v
			result += "\t" + store + ": " + val + "\n"
		}
	}
	return result
}

func PrintDiff(name string, pre, post types.State) { //map[string]string) {
	/*
	   fmt.Println("pre")
	   fmt.Println(PrettyPrintAcctDiff(pre))
	   fmt.Println("\n\n")
	   fmt.Println("post")
	   fmt.Println(PrettyPrintAcctDiff(post))
	   fmt.Println("\n\n")
	*/
	fmt.Println("diff:", name)
	diff := StorageDiff(pre, post)
	fmt.Println(PrettyPrintAcctDiff(diff))
}
