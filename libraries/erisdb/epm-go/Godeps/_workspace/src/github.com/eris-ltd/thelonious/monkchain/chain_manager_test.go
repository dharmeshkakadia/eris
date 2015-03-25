package monkchain

import (
	"container/list"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkdb"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkreact"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkwire"
)

func init() {

	initDB( //InitLogging("", "", 5, "")
	)
}

var DB = []*monkdb.MemDatabase{}

func initDB() {
	monkutil.ReadConfig(".ethtest", "/tmp/ethtest", "")
	// we need two databases, since we need two chain managers
	for i := 0; i < 2; i++ {
		db, _ := monkdb.NewMemDatabase()
		DB = append(DB, db)
	}
	monkutil.Config.Db = DB[0]
}
func setDB(i int) {
	monkutil.Config.Db = DB[i]
}

// So we can generate blocks easily
type fakePow struct{}

func (f fakePow) Search(block *Block, stop chan monkreact.Event) []byte { return nil }
func (f fakePow) Verify(hash []byte, diff *big.Int, nonce []byte) bool  { return true }
func (f fakePow) GetHashrate() int64                                    { return 0 }
func (f fakePow) Turbo(bool)                                            {}

// We need this guy because ProcessWithParent clears txs from the pool
type fakeEth struct{}

func (e *fakeEth) BlockManager() *BlockManager                            { return nil }
func (e *fakeEth) ChainManager() *ChainManager                            { return nil }
func (e *fakeEth) TxPool() *TxPool                                        { return &TxPool{} }
func (e *fakeEth) Broadcast(msgType monkwire.MsgType, data []interface{}) {}
func (e *fakeEth) Reactor() *monkreact.ReactorEngine                      { return monkreact.New() }
func (e *fakeEth) PeerCount() int                                         { return 0 }
func (e *fakeEth) IsMining() bool                                         { return false }
func (e *fakeEth) IsListening() bool                                      { return false }
func (e *fakeEth) Peers() *list.List                                      { return nil }
func (e *fakeEth) KeyManager() *monkcrypto.KeyManager                     { return nil }
func (e *fakeEth) ClientIdentity() monkwire.ClientIdentity                { return nil }
func (e *fakeEth) Db() monkutil.Database                                  { return nil }
func (e *fakeEth) Protocol() Protocol                                     { return nil }

type fakeDoug struct{}

func (d *fakeDoug) Doug() []byte { return nil }
func (d *fakeDoug) Deploy(block *Block) ([]byte, error) {
	return nil, nil
}
func (d *fakeDoug) ValidateChainID(chainId []byte, genBlock *Block) error {
	return nil
}
func (d *fakeDoug) Participate(coinbase []byte, parent *Block) bool                     { return false }
func (d *fakeDoug) Difficulty(block, parent *Block) *big.Int                            { return nil }
func (d *fakeDoug) ValidatePerm(addr []byte, role string, state *monkstate.State) error { return nil }
func (d *fakeDoug) ValidateBlock(block *Block, bc *ChainManager) error                  { return nil }
func (d *fakeDoug) ValidateTx(tx *Transaction, state *monkstate.State) error            { return nil }
func (d *fakeDoug) CheckPoint(proposed []byte, bc *ChainManager) bool                   { return false }

var (
	FakeEth  = &fakeEth{}
	FakeDoug = &fakeDoug{}
)

func newBlockFromParent(addr []byte, parent *Block) *Block {
	block := CreateBlock(
		parent.state.Trie.Root,
		parent.Hash(),
		addr,
		monkutil.BigPow(2, 32),
		nil,
		"")
	block.MinGasPrice = big.NewInt(10000000000000)
	block.Difficulty = CalcDifficulty(block, parent)
	block.Number = new(big.Int).Add(parent.Number, monkutil.Big1)
	block.GasLimit = block.CalcGasLimit(parent)
	block.Time = parent.Time + 1
	return block
}

// Actually make a block by simulating what miner would do
func makeBlock(bman *BlockManager, parent *Block, i int) *Block {
	addr := monkutil.LeftPadBytes([]byte{byte(i)}, 20)
	block := newBlockFromParent(addr, parent)
	cbase := block.State().GetOrNewStateObject(addr)
	cbase.SetGasPool(block.CalcGasLimit(parent))
	receipts, txs, _, _ := bman.ProcessTransactions(cbase, block.State(), block, block, Transactions{})
	//block.SetTransactions(txs)
	block.SetTxHash(receipts)
	block.SetReceipts(receipts, txs)
	bman.AccumelateRewards(block.State(), block, parent)
	block.State().Update()
	return block
}

// Make a chain with real blocks
// Runs ProcessWithParent to get proper state roots
func makeChain(bman *BlockManager, parent *Block, max int) *BlockChain {
	bman.bc.currentBlock = parent
	bman.bc.currentBlockHash = parent.Hash()
	blocks := make(Blocks, max)
	td := bman.bc.BlockInfo(parent).TD
	var err error
	for i := 0; i < max; i++ {
		block := makeBlock(bman, parent, i)
		// add the parent and its difficulty to the working chain
		// so ProcessWithParent can access it
		bman.bc.workingChain = NewChain(Blocks{parent})
		bman.bc.workingChain.Back().Value.(*link).td = td
		td, err = bman.ProcessWithParent(block, parent)
		if err != nil {
			fmt.Println("process with parent failed", err)
			panic(err)
		}
		blocks[i] = block
		parent = block
	}
	lchain := NewChain(blocks)
	return lchain
}

// Make a new canonical chain by running TestChain and InsertChain
// on result of makeChain
func newCanonical(n int) (*BlockManager, error) {
	bman := &BlockManager{bc: newChainManager(nil, FakeDoug), Pow: fakePow{}, th: FakeEth}
	bman.bc.SetProcessor(bman)
	parent := bman.bc.CurrentBlock()
	if n == 0 {
		return bman, nil
	}
	lchain := makeChain(bman, parent, n)

	_, err := bman.bc.TestChain(lchain)
	if err != nil {
		return nil, err
	}
	bman.bc.InsertChain(lchain)
	return bman, nil
}

// Create a new chain manager starting from given block
// Effectively a fork factory
func newChainManager(block *Block, protocol Protocol) *ChainManager {
	bc := &ChainManager{}
	bc.protocol = protocol
	bc.genesisBlock = NewBlockFromBytes(monkutil.Encode(Genesis))
	bc.workingTree = make(map[string]*link)
	genDoug = bc.protocol
	if block == nil {
		bc.protocol.Deploy(bc.genesisBlock)
		bc.Reset()
		bc.TD = monkutil.Big("0")
	} else {
		bc.currentBlock = block
		bc.SetTotalDifficulty(monkutil.Big("0"))
		bc.TD = block.BlockInfo().TD
	}
	return bc
}

// flush the blocks so they point to the current DB
func flushChain(chain *BlockChain) *BlockChain {
	for e := chain.Front(); e != nil; e = e.Next() {
		l := e.Value.(*link)
		b := l.block
		encode := b.RlpEncode()
		b.RlpDecode(encode)
	}
	return chain
}

// Test fork of length N starting from block i
func testFork(t *testing.T, bman *BlockManager, i, N int, f func(td1, td2 *big.Int)) {
	var b *Block = nil
	if i > 0 {
		b = bman.bc.GetBlockByNumber(uint64(i))
	}
	_ = b
	// switch databases to process the new chain
	setDB(1)
	// copy old chain up to i into new db with deterministic canonical
	bman2, err := newCanonical(i) //&BlockManager{bc: newChainManager(b, FakeDoug), Pow: fakePow{}, th: &fakeEth{}}
	if err != nil {
		t.Fatal("could not make new canonical in testFork", err)
	}
	bman2.bc.SetProcessor(bman2)
	parent := bman2.bc.CurrentBlock()

	chainB := makeChain(bman2, parent, N)
	bman2.bc.TestChain(chainB)
	bman2.bc.InsertChain(chainB)

	// test second chain against first
	// switch back to first chain's db
	setDB(0)
	// now, chainB's blocks have states that point to DB 1
	// we need to remake the chain with some fresh rlp decode/encode
	chainB = flushChain(chainB)

	td2, err := bman.bc.TestChain(chainB)
	if err != nil && !IsTDError(err) {
		t.Fatal("expected chainB not to give errors:", err)
	}
	// Compare difficulties
	f(bman.bc.TD, td2)
}

func TestExtendCanonical(t *testing.T) {
	initDB()
	// make first chain starting from genesis
	bman, err := newCanonical(5)
	if err != nil {
		t.Fatal("Could not make new canonical chain:", err)
	}

	f := func(td1, td2 *big.Int) {
		if td2.Cmp(td1) <= 0 {
			t.Error("expected chainB to have higher difficulty. Got", td2, "expected more than", td1)
		}
	}

	// Start fork from current height (5)
	testFork(t, bman, 5, 1, f)
	testFork(t, bman, 5, 2, f)
	testFork(t, bman, 5, 5, f)
	testFork(t, bman, 5, 10, f)

}

func TestShorterFork(t *testing.T) {
	initDB()
	// make first chain starting from genesis
	bman, err := newCanonical(10)
	if err != nil {
		t.Fatal("Could not make new canonical chain:", err)
	}

	f := func(td1, td2 *big.Int) {
		if td2.Cmp(td1) >= 0 {
			t.Error("expected chainB to have lower difficulty. Got", td2, "expected less than", td1)
		}
	}

	// Sum of numbers must be less than 10
	// for this to be a shorter fork
	//testFork(t, bman, 0, 3, f)
	//testFork(t, bman, 0, 7, f)
	//testFork(t, bman, 1, 1, f)
	//testFork(t, bman, 1, 7, f)
	testFork(t, bman, 5, 3, f)
	testFork(t, bman, 5, 4, f)
}

func TestLongerFork(t *testing.T) {
	initDB()
	// make first chain starting from genesis
	bman, err := newCanonical(10)
	if err != nil {
		t.Fatal("Could not make new canonical chain:", err)
	}

	f := func(td1, td2 *big.Int) {
		if td2.Cmp(td1) <= 0 {
			t.Error("expected chainB to have higher difficulty. Got", td2, "expected more than", td1)
		}
	}

	// Sum of numbers must be greater than 10
	// for this to be a longer fork
	testFork(t, bman, 0, 11, f)
	testFork(t, bman, 0, 15, f)
	testFork(t, bman, 1, 10, f)
	testFork(t, bman, 1, 12, f)
	testFork(t, bman, 5, 6, f)
	testFork(t, bman, 5, 8, f)
}

func TestEqualFork(t *testing.T) {
	initDB()
	bman, err := newCanonical(10)
	if err != nil {
		t.Fatal("Could not make new canonical chain:", err)
	}

	f := func(td1, td2 *big.Int) {
		if td2.Cmp(td1) != 0 {
			t.Error("expected chainB to have equal difficulty. Got", td2, "expected less than", td1)
		}
	}

	// Sum of numbers must be equal to 10
	// for this to be an equal fork
	testFork(t, bman, 0, 10, f)
	testFork(t, bman, 1, 9, f)
	testFork(t, bman, 2, 8, f)
	testFork(t, bman, 5, 5, f)
	testFork(t, bman, 6, 4, f)
	testFork(t, bman, 9, 1, f)
}

func TestBrokenChain(t *testing.T) {
	initDB()
	bman, err := newCanonical(10)
	if err != nil {
		t.Fatal("Could not make new canonical chain:", err)
	}

	bman2 := &BlockManager{bc: NewChainManager(FakeDoug), Pow: fakePow{}, th: FakeEth}
	bman2.bc.SetProcessor(bman2)
	parent := bman2.bc.CurrentBlock()

	chainB := makeChain(bman2, parent, 5)
	chainB.Remove(chainB.Front())

	_, err = bman.bc.TestChain(chainB)
	if err == nil {
		t.Error("expected broken chain to return error")
	}
}

func BenchmarkChainTesting(b *testing.B) {
	initDB()
	const chainlen = 1000

	bman, err := newCanonical(5)
	if err != nil {
		b.Fatal("Could not make new canonical chain:", err)
	}

	bman2 := &BlockManager{bc: NewChainManager(FakeDoug), Pow: fakePow{}, th: FakeEth}
	bman2.bc.SetProcessor(bman2)
	parent := bman2.bc.CurrentBlock()

	chain := makeChain(bman2, parent, chainlen)

	stime := time.Now()
	bman.bc.TestChain(chain)
	fmt.Println(chainlen, "took", time.Since(stime))
}
func InitLogging(Datadir string, LogFile string, LogLevel int, DebugFile string) {
	var writer io.Writer
	writer = os.Stdout
	monklog.AddLogSystem(monklog.NewStdLogSystem(writer, log.LstdFlags, monklog.LogLevel(LogLevel)))
}
