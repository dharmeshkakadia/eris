package monkchain

import (
	"bytes"
	"container/list"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkreact"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkwire"
)

var statelogger = monklog.NewLogger("STATE")

type BlockProcessor interface {
	ProcessWithParent(block, parent *Block) (*big.Int, error)
}

type Peer interface {
	Inbound() bool
	LastSend() time.Time
	LastPong() int64
	Host() []byte
	Port() uint16
	Version() string
	PingTime() string
	Connected() *int32
	Caps() *monkutil.Value
}

type NodeManager interface {
	BlockManager() *BlockManager
	ChainManager() *ChainManager
	TxPool() *TxPool
	Broadcast(msgType monkwire.MsgType, data []interface{})
	Reactor() *monkreact.ReactorEngine
	PeerCount() int
	IsMining() bool
	IsListening() bool
	Peers() *list.List
	KeyManager() *monkcrypto.KeyManager
	ClientIdentity() monkwire.ClientIdentity
	Db() monkutil.Database
	Protocol() Protocol
}

type Protocol interface {
	// Permissions based consensus
	Consensus
	// GenDoug address
	Doug() []byte
	// deploy genesis block containing protocol rules
	// returns 20-byte chainId
	Deploy(block *Block) ([]byte, error)
	// validate the chain's Id
	// (typically requires other info, like signatures)
	ValidateChainID(chainId []byte, genesisBlock *Block) error
}

// Model defining the consensus
type Consensus interface {
	// determine whether to attempt participating in consensus
	Participate(coinbase []byte, parent *Block) bool
	// required difficulty of a block
	Difficulty(block, parent *Block) *big.Int
	// determine if an address has a permission
	ValidatePerm(addr []byte, role string, state *monkstate.State) error
	// validate a block
	ValidateBlock(block *Block, bc *ChainManager) error
	// validate a tx
	ValidateTx(tx *Transaction, state *monkstate.State) error
	// determine whether or not this checkpoint should be accepted
	CheckPoint(proposed []byte, bc *ChainManager) bool
}

// Private global genDoug variable for checking permissions on arbitrary
// chain related actions. Set by setLastBlock when we boot up the blockchain
var genDoug Protocol

// Public function so we can validate permissions using the genDoug from outside this package
func DougValidatePerm(addr []byte, role string, state *monkstate.State) error {
	return genDoug.ValidatePerm(addr, role, state)
}

type BlockManager struct {
	// Mutex for state not kept by chain manager
	mutex sync.Mutex
	// Canonical block chain
	bc *ChainManager
	// non-persistent key/value memory storage
	mem map[string]*big.Int
	// Proof of work used for validating
	Pow PoW
	// The thelonious manager interface
	th NodeManager
	// The managed states
	// Official state. Tracks the highest TD block
	// after the block is synced
	state *monkstate.State
	// Transiently state. The trans state isn't ever saved, validated and
	// is used to keep track between block processing
	// It should only be non-nil in a loop over
	transState *monkstate.State
	// Mining state. The mining state is used purely and solely by the mining
	// operation.
	miningState *monkstate.State

	// The last attempted block is mainly used for debugging purposes
	// This does not have to be a valid block and will be set during
	// 'Process' & canonical validation.
	lastAttemptedBlock *Block
}

func NewBlockManager(thelonious NodeManager) *BlockManager {
	sm := &BlockManager{
		mem: make(map[string]*big.Int),
		Pow: &EasyPow{},
		th:  thelonious,
		bc:  thelonious.ChainManager(),
	}
	sm.state = thelonious.ChainManager().CurrentBlock().State().Copy()
	sm.miningState = thelonious.ChainManager().CurrentBlock().State().Copy()
	sm.transState = nil

	return sm
}

func (sm *BlockManager) CurrentState() *monkstate.State {
	return sm.th.ChainManager().CurrentBlock().State()
}

func (sm *BlockManager) TransState() *monkstate.State {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	return sm.state
}

func (sm *BlockManager) MiningState() *monkstate.State {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	return sm.miningState
}

func (sm *BlockManager) NewMiningState() *monkstate.State {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.miningState = sm.th.ChainManager().CurrentBlock().State().Copy()

	return sm.miningState
}

func (sm *BlockManager) ChainManager() *ChainManager {
	return sm.bc
}

func (self *BlockManager) ProcessTransactions(coinbase *monkstate.StateObject, state *monkstate.State, block, parent *Block, txs Transactions) (Receipts, Transactions, Transactions, error) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	var (
		receipts           Receipts
		handled, unhandled Transactions
		totalUsedGas       = big.NewInt(0)
		err                error
	)

done:
	for i, tx := range txs {
		txGas := new(big.Int).Set(tx.Gas)

		cb := state.GetStateObject(coinbase.Address())
		// TODO: deal with this
		st := NewStateTransitionEris(cb, tx, state, block, self.bc.Genesis()) // ERIS
		err = st.TransitionState()
		if err != nil {
			statelogger.Infoln(err)
			switch {
			case IsNonceErr(err):
				self.th.Reactor().Post("newTx:post:fail", &TxFail{tx, err})
				err = nil // ignore error
				continue
			case IsGasLimitTxErr(err):
				self.th.Reactor().Post("newTx:post:fail", &TxFail{tx, err})
				err = nil // ignore error
				continue
			case IsGasLimitErr(err):
				unhandled = txs[i:]
				for _, t := range unhandled {
					self.th.Reactor().Post("newTx:post:fail", &TxFail{t, err})
				}
				break done
			default:
				statelogger.Infoln("this tx registered an error and may have failed:", err)
				err = nil
				// TODO: should this have a tx:fail ?
				//return nil, nil, nil, err
			}
		}

		if st.msg != nil {
			// if msg is nil, an error should have triggered above
			// publish return value
			self.th.Reactor().Post("tx:"+string(tx.Hash())+":return", st.msg.Output)
		}

		// Notify all subscribers
		self.th.Reactor().Post("newTx:post", tx)

		// Update the state with pending changes
		state.Update()

		txGas.Sub(txGas, st.gas)
		accumelative := new(big.Int).Set(totalUsedGas.Add(totalUsedGas, txGas))
		receipt := &Receipt{tx, monkutil.CopyBytes(state.Root().([]byte)), accumelative}

		if i < len(block.Receipts()) {
			original := block.Receipts()[i]
			if !original.Cmp(receipt) {
				if monkutil.Config.Diff {
					os.Exit(1)
				}

				err := fmt.Errorf("#%d receipt failed (r) %v ~ %x  <=>  (c) %v ~ %x (%x...)", i+1, original.CumulativeGasUsed, original.PostState[0:4], receipt.CumulativeGasUsed, receipt.PostState[0:4], receipt.Tx.Hash()[0:4])

				return nil, nil, nil, err
			}
		}

		receipts = append(receipts, receipt)
		handled = append(handled, tx)

		if monkutil.Config.Diff && monkutil.Config.DiffType == "all" {
			state.CreateOutputForDiff()
		}
	}

	//parent.GasUsed = totalUsedGas

	return receipts, handled, unhandled, err
}

// Not thread safe
// Should only be called from TestChain
//  which holds the ChainManager's lock
func (sm *BlockManager) ProcessWithParent(block, parent *Block) (td *big.Int, err error) {

	// is this a new run or are we already working on a chain?
	var state *monkstate.State
	if sm.transState != nil && bytes.Compare(sm.lastAttemptedBlock.Hash(), parent.Hash()) == 0 {
		state = sm.transState
	} else {
		state = parent.State().Copy()
	}

	sm.lastAttemptedBlock = block

	// Defer the Undo on the Trie. If the block processing happened
	// we don't want to undo but since undo only happens on dirty
	// nodes this won't happen because Commit would have been called
	// before that.
	defer func() {
		if err != nil {
			sm.transState = nil
		}
	}()

	if monkutil.Config.Diff && monkutil.Config.DiffType == "all" {
		fmt.Printf("## %x %x ##\n", block.Hash(), block.Number)
	}

	var receipts Receipts
	receipts, err = sm.ApplyDiff(state, parent, block)
	if err != nil {
		return
	}

	txSha := CreateTxSha(receipts)
	if bytes.Compare(txSha, block.TxSha) != 0 {
		err = fmt.Errorf("Error validating tx sha. Received %x, got %x", block.TxSha, txSha)
		return
	}

	// Block validation
	if err = sm.ValidateBlock(block); err != nil {
		statelogger.Errorln("Error validating block:", err)
		return
	}

	if err = sm.AccumelateRewards(state, block, parent); err != nil {
		statelogger.Errorln("Error accumulating reward", err)
		return
	}

	state.Update()

	if !block.State().Cmp(state) {
		err = fmt.Errorf("Invalid merkle root.\nrec: %x\nis:  %x", block.State().Trie.Root, state.Trie.Root)
		return
	}

	// Calculate the new total difficulty and sync back to the db
	var ok bool
	if td, ok = sm.CalculateTD(block); ok {
		// Sync the current block's state to the database and cancelling out the deferred Undo
		state.Sync()

		//if dontReact == false {
		sm.th.Reactor().Post("newBlock", block)
		state.Manifest().Reset()
		//}

		statelogger.Infof("Processed block #%d (%x...)\n", block.Number, block.Hash()[0:4])
		sm.transState = nil
		sm.state = state.Copy()
		/*
			// Create a bloom bin for this block
			filter := sm.createBloomFilter(state)
			// Persist the data
			fk := append([]byte("bloom"), block.Hash()...)
			sm.Thelonious.Db().Put(fk, filter.Bin())
		*/
		//sm.Thelonious.TxPool().RemoveInvalid(state)
		sm.th.TxPool().RemoveSet(block.Transactions())
		return
	} else {
		sm.transState = state
		return
	}

	return nil, nil
}

func (sm *BlockManager) ApplyDiff(state *monkstate.State, parent, block *Block) (receipts Receipts, err error) {
	coinbase := state.GetOrNewStateObject(block.Coinbase)
	coinbase.SetGasPool(block.CalcGasLimit(parent))

	// Process the transactions on to current block
	receipts, _, _, err = sm.ProcessTransactions(coinbase, state, block, parent, block.Transactions())
	if err != nil {
		return nil, err
	}

	return receipts, nil
}

// TODO: this is a sham...
func (sm *BlockManager) CalculateTD(block *Block) (*big.Int, bool) {
	td, err := sm.bc.CalcTotalDiff(block)
	if err != nil {
		fmt.Println(err)
		return nil, false
	}
	// The new TD will only be accepted if the new difficulty is
	// is greater than the previous.
	if td.Cmp(sm.bc.TD) > 0 {
		// Set the new total difficulty back to the block chain
		//sm.bc.SetTotalDifficulty(td)

		return td, true
	}

	return td, false
}

// Validates the current block. Returns an error if the block was invalid,
// an uncle or anything that isn't on the current block chain.
// Validation validates easy over difficult (dagger takes longer time = difficult)
func (sm *BlockManager) ValidateBlock(block *Block) error {
	// all validation is done through the genDoug
	return sm.bc.protocol.ValidateBlock(block, sm.bc)
}

func (sm *BlockManager) AccumelateRewards(state *monkstate.State, block, parent *Block) error {
	reward := new(big.Int).Set(BlockReward)

	knownUncles := monkutil.Set(parent.Uncles)
	nonces := monkutil.NewSet(block.Nonce)
	for _, uncle := range block.Uncles {
		if nonces.Include(uncle.Nonce) {
			// Error not unique
			return UncleError("Uncle not unique")
		}

		uncleParent := sm.bc.GetBlock(uncle.PrevHash)
		if uncleParent == nil {
			return UncleError("Uncle's parent unknown")
		}

		if uncleParent.Number.Cmp(new(big.Int).Sub(parent.Number, big.NewInt(6))) < 0 {
			return UncleError("Uncle too old")
		}

		if knownUncles.Include(uncle.Hash()) {
			return UncleError("Uncle in chain")
		}

		nonces.Insert(uncle.Nonce)

		r := new(big.Int)
		r.Mul(BlockReward, big.NewInt(15)).Div(r, big.NewInt(16))

		uncleAccount := state.GetAccount(uncle.Coinbase)
		uncleAccount.AddAmount(r)

		reward.Add(reward, new(big.Int).Div(BlockReward, big.NewInt(32)))
	}
	// Get the account associated with the coinbase
	account := state.GetAccount(block.Coinbase)
	// Reward amount of junk to the coinbase address
	account.AddAmount(reward)

	return nil
}

func (sm *BlockManager) Stop() {
	sm.bc.Stop()
}

// Manifest will handle both creating notifications and generating bloom bin data
func (sm *BlockManager) createBloomFilter(state *monkstate.State) *BloomFilter {
	bloomf := NewBloomFilter(nil)

	for _, msg := range state.Manifest().Messages {
		bloomf.Set(msg.To)
		bloomf.Set(msg.From)
	}

	sm.th.Reactor().Post("messages", state.Manifest().Messages)

	return bloomf
}

func (sm *BlockManager) GetMessages(block *Block) (messages []*monkstate.Message, err error) {
	if !sm.bc.HasBlock(block.PrevHash) {
		return nil, ParentError(block.PrevHash)
	}

	sm.lastAttemptedBlock = block

	var (
		parent = sm.bc.GetBlock(block.PrevHash)
		state  = parent.State().Copy()
	)

	defer state.Reset()

	sm.ApplyDiff(state, parent, block)

	sm.AccumelateRewards(state, block, parent)

	return state.Manifest().Messages, nil
}

func printTrie(state *monkstate.State) {
	//	s := monkutil.NewValue(st).Bytes()
	//	sta, _ := monkutil.Config.Db.Get(s)
	it := state.Trie.NewIterator()
	it.Each(func(k string, n *monkutil.Value) {
		addr := monkutil.Address([]byte(k))
		obj := state.GetAccount(addr)
		PrettyPrintAccount(obj)
	})
}

func PrettyPrintAccount(obj *monkstate.StateObject) {
	fmt.Println("Address", monkutil.Bytes2Hex(obj.Address())) //monkutil.Bytes2Hex([]byte(addr)))
	fmt.Println("\tBalance", obj.Balance)
}
