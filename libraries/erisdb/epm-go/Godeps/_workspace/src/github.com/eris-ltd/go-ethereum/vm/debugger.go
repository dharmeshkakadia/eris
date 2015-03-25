package vm

import "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/state"

type Debugger interface {
	BreakHook(step int, op OpCode, mem *Memory, stack *Stack, object *state.StateObject) bool
	StepHook(step int, op OpCode, mem *Memory, stack *Stack, object *state.StateObject) bool
	BreakPoints() []int64
	SetCode(byteCode []byte)
}
