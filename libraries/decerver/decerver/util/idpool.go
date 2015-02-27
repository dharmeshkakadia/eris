package util

// Simple id pool. Lets you get and release ids.
import (
	"container/list"
)

type IdPool struct {
	ids *list.List
}

// Keeps a pool of integers.
func NewIdPool(totNum uint32) *IdPool {
	idPool := &IdPool{}
	idPool.init(totNum)
	return idPool
}

// We start from 1, so that 0 is not used as an id.
func (idp *IdPool) init(totNum uint32) {
	idp.ids = list.New()
	for i := uint32(1); i <= totNum; i++ {
		idp.ids.PushBack(i)
	}
}

func (idp *IdPool) GetId() uint32 {
	val := idp.ids.Front()
	idp.ids.Remove(val)
	num, _ := val.Value.(uint32)
	return num
}

func (idp *IdPool) ReleaseId(id uint32) {
	idp.ids.PushBack(id)
}
