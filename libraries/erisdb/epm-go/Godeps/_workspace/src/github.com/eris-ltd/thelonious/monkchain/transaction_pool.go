package monkchain

import (
	"bytes"
	"container/list"
	"fmt"
	"math/big"
	"sync"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkwire"
)

var txplogger = monklog.NewLogger("TXP")

const (
	txPoolQueueSize = 50
)

type TxPoolHook chan *Transaction
type TxMsgTy byte

const (
	TxPre = iota
	TxPost
	minGasPrice = 0 //1000000
)

var MinGasPrice = big.NewInt(0)

type TxMsg struct {
	Tx   *Transaction
	Type TxMsgTy
}

func FindTx(pool *list.List, finder func(*Transaction, *list.Element) bool) *Transaction {
	for e := pool.Front(); e != nil; e = e.Next() {
		if tx, ok := e.Value.(*Transaction); ok {
			if finder(tx, e) {
				return tx
			}
		}
	}

	return nil
}

// The tx pool a thread safe transaction pool handler. In order to
// guarantee a non blocking pool we use a queue channel which can be
// independently read without needing access to the actual pool. If the
// pool is being drained or synced for whatever reason the transactions
// will simple queue up and handled when the mutex is freed.
type TxPool struct {
	Thelonious NodeManager
	// The mutex for accessing the Tx pool.
	mutex sync.Mutex
	// Queueing channel for reading and writing incoming
	// transactions to
	queueChan chan *Transaction
	// Quiting channel
	quit chan bool
	// The actual pool
	pool *list.List

	subscribers []chan TxMsg
}

func NewTxPool(thelonious NodeManager) *TxPool {
	return &TxPool{
		pool:       list.New(),
		queueChan:  make(chan *Transaction, txPoolQueueSize),
		quit:       make(chan bool),
		Thelonious: thelonious,
	}
}

// Blocking function. Don't use directly. Use QueueTransaction instead
// Caller should hold the lock!
func (pool *TxPool) addTransaction(tx *Transaction) {
	pool.pool.PushBack(tx)

	// Broadcast the transaction to the rest of the peers
	pool.Thelonious.Broadcast(monkwire.MsgTxTy, []interface{}{tx.RlpData()})
}

// TODO: will this panic on invalid signature? catch that
//  does not execute evm, just simple checks for adding to pool
func (pool *TxPool) ValidateTransaction(tx *Transaction) error {
	// Get the last block so we can retrieve the sender and receiver from
	// the merkle trie
	block := pool.Thelonious.ChainManager().CurrentBlock()
	// Something has gone horribly wrong if this happens
	if block == nil {
		return fmt.Errorf("[TXPL] No last block on the block chain")
	}

	if len(tx.Recipient) != 0 && len(tx.Recipient) != 20 {
		return fmt.Errorf("[TXPL] Invalid recipient. len = %d", len(tx.Recipient))
	}

	if tx.GasPrice.Cmp(MinGasPrice) < 0 {
		return fmt.Errorf("Gas price to low. Require %v > Got %v", MinGasPrice, tx.GasPrice)
	}

	// Get the sender
	//sender := pool.Thelonious.BlockManager().procState.GetAccount(tx.Sender())
	// TODO: shoudl this be TransState() ?
	sender := pool.Thelonious.BlockManager().CurrentState().GetAccount(tx.Sender())

	totAmount := new(big.Int).Set(tx.Value)
	// Make sure there's enough in the sender's account. Having insufficient
	// funds won't invalidate this transaction but simple ignores it.
	if sender.Balance.Cmp(totAmount) < 0 {
		return fmt.Errorf("[TXPL] Insufficient amount in sender's (%x) account", tx.Sender())
	}

	if tx.IsContract() {
		if tx.GasPrice.Cmp(big.NewInt(minGasPrice)) < 0 {
			return fmt.Errorf("[TXPL] Gasprice too low, %s given should be at least %d.", tx.GasPrice, minGasPrice)
		}
	}

	return nil
}

func (pool *TxPool) queueHandler() {
out:
	for {
		select {
		case tx := <-pool.queueChan:
			if pool.queueTransaction(tx) {
				continue
			}
		case <-pool.quit:
			break out
		}
	}
}

func (pool *TxPool) queueTransaction(tx *Transaction) bool {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	hash := tx.Hash()
	foundTx := FindTx(pool.pool, func(tx *Transaction, e *list.Element) bool {
		return bytes.Compare(tx.Hash(), hash) == 0
	})

	if foundTx != nil {
		return true
	}

	// Validate the transaction
	err := pool.ValidateTransaction(tx)
	if err != nil {
		txplogger.Debugln("Validating Tx failed", err)
		pool.Thelonious.Reactor().Post("newTx:pre:fail", &TxFail{tx, err})
	} else {
		// Call blocking version.
		pool.addTransaction(tx)

		tmp := make([]byte, 4)
		copy(tmp, tx.Recipient)

		txplogger.Debugf("(t) %x => %x (%v) %x\n", tx.Sender()[:4], tmp, tx.Value, tx.Hash())

		// Notify the subscribers
		pool.Thelonious.Reactor().Post("newTx:pre", tx)
	}
	return false
}

func (pool *TxPool) QueueTransaction(tx *Transaction) {
	pool.queueChan <- tx
}

func (pool *TxPool) CurrentTransactions() []*Transaction {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	txList := make([]*Transaction, pool.pool.Len())
	i := 0
	for e := pool.pool.Front(); e != nil; e = e.Next() {
		tx := e.Value.(*Transaction)

		txList[i] = tx

		i++
	}

	return txList
}

func (pool *TxPool) RemoveInvalid(state *monkstate.State) {
	for e := pool.pool.Front(); e != nil; e = e.Next() {
		tx := e.Value.(*Transaction)
		sender := state.GetAccount(tx.Sender())
		err := pool.ValidateTransaction(tx)
		if err != nil || sender.Nonce >= tx.Nonce {
			pool.pool.Remove(e)
		}
	}
}

func (self *TxPool) RemoveSet(txs Transactions) {
	self.mutex.Lock()
	defer self.mutex.Unlock()

	for _, tx := range txs {
		EachTx(self.pool, func(t *Transaction, element *list.Element) bool {
			if t == tx {
				self.pool.Remove(element)
				return true // To stop the loop
			}
			return false
		})
	}
}

func (pool *TxPool) Flush() []*Transaction {
	txList := pool.CurrentTransactions()

	// Recreate a new list all together
	// XXX Is this the fastest way?
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	pool.pool = list.New()

	return txList
}

func (pool *TxPool) Start() {
	go pool.queueHandler()
}

func (pool *TxPool) Stop() {
	close(pool.quit)

	pool.Flush()

	txplogger.Infoln("Stopped")
}

func EachTx(pool *list.List, it func(*Transaction, *list.Element) bool) {
	for e := pool.Front(); e != nil; e = e.Next() {
		if it(e.Value.(*Transaction), e) {
			break
		}
	}
}
