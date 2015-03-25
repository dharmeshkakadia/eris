package monkchain

import (
	"bytes"
	"container/list"
	"fmt"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
	"log"
	"math/big"
	"sync"
	//"github.com/eris-ltd/thelonious/monkcrypto"
)

var chainlogger = monklog.NewLogger("CHAIN")

/*
   ChainManager manages the canonical blockchain and a working
       tree of forks.
   A canonical chain begins right away, with blocks saved in database.
   New blocks (from peers or miner) are placed in working tree.
   If at any time a branch in workingTree gets higher diff
       than canonical, a re-org is called, the new chain is copied
       into database, the old canonical is placed in workingTree
       and the relevant blocks are removed from workingTree if they have
       no children
       XXX: should old canonical be removed from leveldb?
           how do we even track canonical?
           can we put it in a trie?
*/

func CalcDifficulty(block, parent *Block) *big.Int {
	diff := new(big.Int)

	adjust := new(big.Int).Rsh(parent.Difficulty, 10)
	if block.Time >= parent.Time+5 {
		diff.Sub(parent.Difficulty, adjust)
	} else {
		diff.Add(parent.Difficulty, adjust)
	}

	return diff
}

type ChainManager struct {
	//Thelonious NodeManager
	processor BlockProcessor
	protocol  Protocol
	// Genesis Block and Chain ID
	genesisBlock *Block
	chainID      []byte

	// Last known total difficulty
	TD *big.Int

	// Our canonical chain
	currentBlockNumber uint64
	currentBlock       *Block
	currentBlockHash   []byte

	// Our working chains
	workingTree  map[string]*link
	workingChain *BlockChain

	// Our latest checkpoint
	latestCheckPointHash   []byte
	latestCheckPointBlock  *Block
	latestCheckPointNumber uint64
	waitingForCheckPoint   bool

	// sync access to current state (block, hash, num)
	mut sync.Mutex
	// sync access to TestChain/InsertChain
	chainMut sync.Mutex
}

func NewChainManager(protocol Protocol) *ChainManager {
	bc := &ChainManager{}
	bc.genesisBlock = NewBlockFromBytes(monkutil.Encode(Genesis))
	bc.workingTree = make(map[string]*link)
	bc.protocol = protocol

	// set last block we know of or deploy genesis
	bc.setLastBlock()
	// load the latest checkpoint
	bc.loadCheckpoint()

	return bc
}

// This is an attempt to change the chain's checkpoint
// It is valid only if the latest checkpoint is the genesis block
// or if the proposed is a known block (and satisfies some checkpoint
// satisfiability contract).
// Proposals should come after NewChainManager and come from a separate
// file or trusted process.
// TODO: allow proposals to come out of the vm
func (bc *ChainManager) CheckPoint(proposed []byte) {
	if proposed == nil {
		return
	}

	// if this is our first time booting up
	// we always accept checkpoints
	isBoot := bytes.Compare(bc.LatestCheckPointHash(), bc.Genesis().Hash()) == 0
	if isBoot {
		bc.updateCheckpoint(proposed)
		return
	}

	// check if the checkpoint satisfies consensus rules
	if bc.protocol.CheckPoint(proposed, bc) {
		bc.updateCheckpoint(proposed)
	}
}

func (bc *ChainManager) IsCheckpoint(hash []byte) bool {
	return bytes.Compare(hash, bc.LatestCheckPointHash()) == 0
}

// Receive the checkpointed block from peers
func (bc *ChainManager) ReceiveCheckPointBlock(block *Block) bool {
	if block == nil {
		return false
	}

	if bc.IsCheckpoint(block.Hash()) {
		bc.add(block)
		bc.updateCheckpoint(block.Hash())
		return true
	}
	return false
}

// This assumes this checkpoint is valid, but only writes it
// to the db if/once we have the block
func (bc *ChainManager) updateCheckpoint(checkPoint []byte) {
	bc.latestCheckPointHash = checkPoint
	b := bc.GetBlock(checkPoint)
	if b != nil {
		bc.setWaitingForCheckpoint(false)
		bc.latestCheckPointBlock = b
		bc.latestCheckPointNumber = b.Number.Uint64()
		monkutil.Config.Db.Put([]byte("LatestCheckPoint"), b.Hash())
		chainlogger.Infof("Updating checkpoint block: (#%d) %x\n", bc.latestCheckPointNumber, checkPoint)
	} else {
		// we have accepted the checkpoint but don't have the block
		// it should come in now through the block pool
		chainlogger.Infof("Updating checkpoint block %x not found. Retrieving from peers.\n", checkPoint)
		bc.setWaitingForCheckpoint(true)
	}
}

func (bc *ChainManager) WaitingForCheckpoint() bool {
	bc.mut.Lock()
	defer bc.mut.Unlock()
	return bc.waitingForCheckPoint
}

func (bc *ChainManager) setWaitingForCheckpoint(s bool) {
	bc.mut.Lock()
	defer bc.mut.Unlock()
	bc.waitingForCheckPoint = s
}

// load checkpoint from db or set to genesis
func (bc *ChainManager) loadCheckpoint() {
	data, _ := monkutil.Config.Db.Get([]byte("LatestCheckPoint"))
	if len(data) != 0 {
		bc.updateCheckpoint(data)
	} else {
		bc.updateCheckpoint(bc.genesisBlock.Hash())
	}
}

func (bc *ChainManager) SetProcessor(proc BlockProcessor) {
	bc.processor = proc
}

func (bc *ChainManager) Genesis() *Block {
	return bc.genesisBlock
}

// Only called by the miner
func (bc *ChainManager) NewBlock(coinbase []byte) *Block {
	var root interface{}
	hash := ZeroHash256

	if bc.CurrentBlock() != nil {
		root = bc.CurrentBlock().state.Trie.Root
		hash = bc.CurrentBlockHash()
	}

	block := CreateBlock(
		root,
		hash,
		coinbase,
		monkutil.BigPow(2, 12),
		nil,
		"")

	// TODO: How do we feel about this
	block.MinGasPrice = big.NewInt(10000000000000)

	parent := bc.CurrentBlock()
	if parent != nil {
		block.Difficulty = genDoug.Difficulty(block, parent)
		block.Number = new(big.Int).Add(parent.Number, monkutil.Big1)
		block.GasLimit = monkutil.BigPow(10, 50) //block.CalcGasLimit(bc.CurrentBlock)

	}

	return block
}

// XXX: Only checks canonical!
func (bc *ChainManager) HasBlock(hash []byte) bool {
	data, _ := monkutil.Config.Db.Get(hash)
	return len(data) != 0
}

// TODO: At one point we might want to save a block by prevHash in the db to optimise this...
func (bc *ChainManager) HasBlockWithPrevHash(hash []byte) bool {
	block := bc.CurrentBlock()

	for ; block != nil; block = bc.GetBlock(block.PrevHash) {
		if bytes.Compare(hash, block.PrevHash) == 0 {
			return true
		}
	}
	return false
}

func (bc *ChainManager) CalculateBlockTD(block *Block) *big.Int {
	blockDiff := new(big.Int)

	for _, uncle := range block.Uncles {
		blockDiff = blockDiff.Add(blockDiff, uncle.Difficulty)
	}
	blockDiff = blockDiff.Add(blockDiff, block.Difficulty)

	return blockDiff
}

func (bc *ChainManager) GenesisBlock() *Block {
	return bc.genesisBlock
}

func (self *ChainManager) GetChainHashesFromHash(hash []byte, max uint64) (chain [][]byte) {
	block := self.GetBlock(hash)
	if block == nil {
		return
	}

	// XXX Could be optimised by using a different database which only holds hashes (i.e., linked list)
	for i := uint64(0); i < max; i++ {
		chain = append(chain, block.Hash())

		if block.Number.Cmp(monkutil.Big0) <= 0 {
			break
		}

		block = self.GetBlock(block.PrevHash)
	}

	return
}

func (bc *ChainManager) setLastBlock() {
	// check for a genesis block
	data, _ := monkutil.Config.Db.Get([]byte("GenesisBlock"))
	if len(data) != 0 {
		chainlogger.Infoln("Found genesis block")
		bc.genesisBlock = NewBlockFromBytes(data)
		data, _ = monkutil.Config.Db.Get([]byte("ChainID"))
		if len(data) == 0 {
			log.Fatal("No chainID found for genesis block.")
		}
		// chainId is leading 20 bytes of signed genblock hash
		// validated on startup
		err := bc.protocol.ValidateChainID(data, bc.genesisBlock)
		if err != nil {
			log.Fatal(err)
		}
		bc.chainID = data
	} else {
		// no genesis block found. fire up a deploy
		// save genesis and chainId to db
		chainlogger.Infoln("Genesis block not found. Deploying.")
		chainId, err := bc.protocol.Deploy(bc.genesisBlock)
		if err != nil {
			log.Fatal("Genesis deploy failed:", err)
		}
		monkutil.Config.Db.Put([]byte("GenesisBlock"), bc.genesisBlock.RlpEncode())
		monkutil.Config.Db.Put([]byte("ChainID"), chainId[:])
		bc.chainID = chainId
	}

	// check for last block.
	data, _ = monkutil.Config.Db.Get([]byte("LastBlock"))
	if len(data) != 0 {
		block := NewBlockFromBytes(data)
		bc.currentBlock = block
		bc.currentBlockHash = block.Hash()
		bc.currentBlockNumber = block.Number.Uint64()
	} else {
		bc.Reset()
	}
	// set the genDoug model (global var) for determining chain permissions
	genDoug = bc.protocol

	//bc.SetTotalDifficulty(monkutil.Big("0"))

	// Set the last know difficulty (might be 0x0 as initial value, Genesis)
	bc.TD = monkutil.BigD(monkutil.Config.Db.LastKnownTD())

	chainlogger.Infof("Last block (#%d) %x\n", bc.currentBlockNumber, bc.currentBlock.Hash())
	chainlogger.Infof("ChainID (%x) \n", bc.chainID)
	chainlogger.Infof("Genesis (%x) \n", bc.genesisBlock.Hash())
}

func (bc *ChainManager) Reset() {
	bc.add(bc.genesisBlock)
	//fk := append([]byte("bloom"), bc.genesisBlock.Hash()...)
	//bc.Ethereum.Db().Put(fk, make([]byte, 255))
	bc.currentBlock = bc.genesisBlock
	bc.currentBlockHash = bc.currentBlock.Hash()
	bc.currentBlockNumber = bc.currentBlock.Number.Uint64()
}

func (bc *ChainManager) SetTotalDifficulty(td *big.Int) {
	monkutil.Config.Db.Put([]byte("LTD"), td.Bytes())
	bc.TD = td
}

// Add a block to the canonical chain and record addition information
func (bc *ChainManager) add(block *Block) {
	bc.mut.Lock()
	defer bc.mut.Unlock()

	bc.writeBlockInfo(block)
	bc.currentBlock = block
	bc.currentBlockHash = block.Hash()

	encodedBlock := block.RlpEncode()
	monkutil.Config.Db.Put(block.Hash(), encodedBlock)
	monkutil.Config.Db.Put([]byte("LastBlock"), encodedBlock)
}

func (bc *ChainManager) ChainID() []byte {
	bc.mut.Lock()
	defer bc.mut.Unlock()
	return bc.chainID
}

func (bc *ChainManager) CurrentBlock() *Block {
	bc.mut.Lock()
	defer bc.mut.Unlock()
	return bc.currentBlock
}

func (bc *ChainManager) CurrentBlockHash() []byte {
	bc.mut.Lock()
	defer bc.mut.Unlock()
	return bc.currentBlockHash
}

func (bc *ChainManager) CurrentBlockNumber() uint64 {
	bc.mut.Lock()
	defer bc.mut.Unlock()
	return bc.currentBlockNumber
}

func (bc *ChainManager) CurrentBlockPrevHash() []byte {
	bc.mut.Lock()
	defer bc.mut.Unlock()
	return bc.currentBlock.PrevHash
}

func (bc *ChainManager) LatestCheckPointHash() []byte {
	bc.mut.Lock()
	defer bc.mut.Unlock()
	return bc.latestCheckPointHash
}

func (bc *ChainManager) LatestCheckPointBlock() *Block {
	bc.mut.Lock()
	defer bc.mut.Unlock()
	return bc.latestCheckPointBlock
}

func (bc *ChainManager) LatestCheckPointNumber() uint64 {
	bc.mut.Lock()
	defer bc.mut.Unlock()
	return bc.latestCheckPointNumber
}

func (self *ChainManager) CalcTotalDiff(block *Block) (*big.Int, error) {
	parent := self.GetBlock(block.PrevHash)
	if parent == nil {
		return nil, fmt.Errorf("Unable to calculate total diff without known parent %x", block.PrevHash)
	}

	// this has no effect on blocks in the workingTree!
	//parentTd := parent.BlockInfo().TD
	parentTd := self.BlockInfo(parent).TD

	uncleDiff := new(big.Int)
	for _, uncle := range block.Uncles {
		uncleDiff = uncleDiff.Add(uncleDiff, uncle.Difficulty)
	}

	td := new(big.Int)
	td = td.Add(parentTd, uncleDiff)
	td = td.Add(td, block.Difficulty)

	return td, nil
}

// Tries to return the block regardless of if it's canonical
// or simply workingTree
func (self *ChainManager) GetBlock(hash []byte) *Block {
	b := self.GetBlockCanonical(hash)
	if b == nil {
		b = self.GetBlockWorking(hash)
	}
	return b
}

// Strictly returns blocks from the workingChain or workingTree
func (self *ChainManager) GetBlockWorking(hash []byte) *Block {
	if l, ok := self.workingTree[string(hash)]; ok {
		return l.block
	}

	if self.workingChain != nil {
		for e := self.workingChain.Front(); e != nil; e = e.Next() {
			if bytes.Compare(e.Value.(*link).block.Hash(), hash) == 0 {
				return e.Value.(*link).block
			}
		}
	}

	return nil
}

// Strictly returns canonical blocks
func (self *ChainManager) GetBlockCanonical(hash []byte) *Block {
	data, _ := monkutil.Config.Db.Get(hash)
	if len(data) == 0 {
		return nil
	}
	return NewBlockFromBytes(data)
}

func (self *ChainManager) GetBlockByNumber(num uint64) *Block {
	block := self.CurrentBlock()
	for ; block != nil; block = self.GetBlock(block.PrevHash) {
		if block.Number.Uint64() == num {
			break
		}
	}

	if block != nil && block.Number.Uint64() == 0 && num != 0 {
		return nil
	}

	return block
}

func (self *ChainManager) GetBlockBack(num uint64) *Block {
	block := self.CurrentBlock()

	for ; num != 0 && block != nil; num-- {
		block = self.GetBlock(block.PrevHash)
	}

	return block
}

func (bc *ChainManager) BlockInfoByHash(hash []byte) BlockInfo {
	bi := BlockInfo{}
	data, _ := monkutil.Config.Db.Get(append(hash, []byte("Info")...))
	bi.RlpDecode(data)

	return bi
}

func (bc *ChainManager) BlockInfo(block *Block) BlockInfo {
	bi := BlockInfo{}
	data, _ := monkutil.Config.Db.Get(append(block.Hash(), []byte("Info")...))
	if len(data) == 0 {
		if l, ok := bc.workingTree[string(block.Hash())]; ok {
			b := l.block
			return BlockInfo{Number: b.Number.Uint64(), Hash: b.Hash(), Parent: b.PrevHash, TD: l.td}
		}

		if bc.workingChain != nil {
			// Check the temp chain
			for e := bc.workingChain.Front(); e != nil; e = e.Next() {
				l := e.Value.(*link)
				b := l.block
				if bytes.Compare(b.Hash(), block.Hash()) == 0 {
					return BlockInfo{Number: b.Number.Uint64(), Hash: b.Hash(), Parent: b.PrevHash, TD: l.td}
				}
			}
		}
	}
	bi.RlpDecode(data)

	return bi
}

// Unexported method for writing extra non-essential block info to the db
// not thread safe (caller should lock)
func (bc *ChainManager) writeBlockInfo(block *Block) {
	if block.Number.Cmp(big.NewInt(0)) != 0 {
		bc.currentBlockNumber++
	}
	bi := BlockInfo{Number: bc.currentBlockNumber, Hash: block.Hash(), Parent: block.PrevHash, TD: bc.TD}

	// For now we use the block hash with the words "info" appended as key
	monkutil.Config.Db.Put(append(block.Hash(), []byte("Info")...), bi.RlpEncode())
}

func (bc *ChainManager) Stop() {
	if bc.CurrentBlock() != nil {
		chainlogger.Infoln("Stopped")
	}
}

// a link in the working tree
type link struct {
	block *Block
	//messages state.Messages
	td *big.Int

	// everyone has a parent in the working tree
	//  except the first link off canonical.
	parent *link
	// links are removed if they have no children and part of
	//  a chain that fails
	//  on re-orgs, new canonical blocks with more than one child
	//      point the non-canonical child's parent at nil
	//      (ie, its a branch off canonical now)
	children []*link
}

// Blockchain coming in from the block pool or from miners
type BlockChain struct {
	*list.List
}

// Create a linked-list of links
func NewChain(blocks Blocks) *BlockChain {
	chain := &BlockChain{list.New()}
	for _, block := range blocks {
		chain.PushBack(&link{block, nil, nil, nil})
	}
	return chain
}

// Validate the new chain with respect to its parent
// First set workingChain. If passes and is a fork, add to workingTree
// TODO: Note this will sync new states (we may not want that, but it shouldn't
//  get in the way, it's just storage we dont need to keep around. Also, if there's a fork attack, we can study it later :) )
func (self *ChainManager) TestChain(chain *BlockChain) (td *big.Int, err error) {
	self.chainMut.Lock()
	defer self.chainMut.Unlock()

	self.workingChain = chain
	defer func(cm *ChainManager) { cm.workingChain = nil }(self)

	var parent *Block
	var fork bool

	parent, fork = self.detectFork(chain)
	if fork {
		fmt.Println("Fork!")
		if _, ok := self.workingTree[string(parent.Hash())]; !ok {
			chainlogger.Infof("New fork detected off parent %x at height %d. Head %x at %d", parent.Hash(), parent.Number, self.CurrentBlockHash(), self.CurrentBlockNumber())
		} else {
			chainlogger.Infoln("Extending a fork...")
		}
	}

	// Process the chain starting from its parent to ensure its valid
	for e := chain.Front(); e != nil; e = e.Next() {
		l := e.Value.(*link)
		block := l.block

		// parent may be on canonical or a fork on workingTree
		parent := self.GetBlock(block.PrevHash)

		if parent == nil {
			err = fmt.Errorf("incoming chain broken on hash %x\n", block.PrevHash[0:4])
			return
		}

		//var messages state.Messages
		td, err = self.processor.ProcessWithParent(block, parent)
		if err != nil {
			chainlogger.Infoln(err)
			chainlogger.Debugf("Block #%v failed (%x...)\n", block.Number, block.Hash()[0:4])
			chainlogger.Debugln(block)
			err = fmt.Errorf("incoming chain failed %v\n", err)
			return
		} else {
			chainlogger.Debugf("Block #%v passed (%x...)\n", block.Number, block.Hash()[0:4])
		}

		l.td = td
		//l.messages = messages
	}

	// If this is a fork, we add to the persistent tree
	if fork {
		self.addChainToWorkingTree(chain)
	}

	if td.Cmp(self.TD) <= 0 {
		err = &TDError{td, self.TD}
		return
	}

	return
}

func (self *ChainManager) insertChain(chain *BlockChain) {
	self.chainMut.Lock()
	defer self.chainMut.Unlock()

	// We are lengthening canonical!
	// for each block, set the new difficulty, add to chain
	for e := chain.Front(); e != nil; e = e.Next() {
		link := e.Value.(*link)

		self.SetTotalDifficulty(link.td)
		self.add(link.block)

		// XXX: Post. Do we do this here? Prob better for caller ...
		//self.Thelonious.Reactor().Post(NewBlockEvent{link.block})
		//self.Thelonious.Reactor().Post(link.messages)
	}

	// summarize
	b, e := chain.Front(), chain.Back()
	if b != nil && e != nil {
		front, back := b.Value.(*link).block, e.Value.(*link).block
		chainlogger.Infof("Imported %d blocks. #%v (%x) / %#v (%x)", chain.Len(), front.Number, front.Hash()[0:4], back.Number, back.Hash()[0:4])
	}
	return

}

// This function assumes you've done your checking. No validity checking is done at this stage anymore
// This will either extend canonical or cause a reorg.
func (self *ChainManager) InsertChain(chain *BlockChain) {

	var (
		oldest       = chain.Front().Value.(*link).block
		branchParent = self.GetBlock(oldest.PrevHash)
		head         = self.CurrentBlock()
	)

	// Check if parent is top block on canonical
	// if so, extend canonical
	if bytes.Compare(head.Hash(), branchParent.Hash()) == 0 {
		self.insertChain(chain)
		return
	}

	// Looks like it's time for a re-org!
	var td *big.Int

	// get td from end of list
	td = chain.Back().Value.(*link).td

	// if the new chain is crowned most gangsta (sanity check)

	if td.Cmp(self.TD) > 0 {
		chainlogger.Infoln("A fork has overtaken canonical. Time for a reorg!")
		self.reOrg(chain)
	}
}

func (self *ChainManager) reOrg(chain *BlockChain) {
	// Find branch point
	// Pop them off the top of canonical into a chain
	//  add the chain to working tree
	// Pop the new canonical chain out of workingTree and into databas
	// Create array of blocks from new head back to branch point
	// Deletes them from workingTree
	// Uses memory links. Maybe we should use prev hashes?
	chainlogger.Debugln("Popping blocks off working tree")
	bchain := &BlockChain{list.New()}
	for l := chain.Back().Value.(*link); l != nil; l = l.parent {
		bchain.PushFront(&link{l.block, nil, nil, nil})
		// TODO: only delete if no children left
		//      remove children as they are killed
		//      children should maybe be a map..
		delete(self.workingTree, string(l.block.Hash()))
	}

	ancestorHash := bchain.Front().Value.(*link).block.PrevHash
	ancestor := self.GetBlockCanonical(ancestorHash)

	oldHeadHash := self.CurrentBlockHash()
	oldHead := self.GetBlockCanonical(oldHeadHash)

	// revert the blockchain
	chainlogger.Infof("Reverting blockchain to block %x at height %d, a reversion of %d blocks", ancestorHash, ancestor.Number, new(big.Int).Sub(oldHead.Number, ancestor.Number))

	self.mut.Lock()
	self.currentBlock = ancestor
	self.currentBlockHash = ancestorHash
	self.mut.Unlock()

	// process the new chain on top
	// we've already done this
	// but we're also paranoid
	chainlogger.Infof("Testing new chain (redundant, we know...)")
	_, err := self.TestChain(bchain)
	if err != nil {
		chainlogger.Infoln("Reorg failed as new chain failed processing. This shouldn't have happened and may mean trouble")
		self.mut.Lock()
		self.currentBlock = oldHead
		self.currentBlockHash = oldHeadHash
		self.mut.Unlock()
		return
	}
	chainlogger.Infof("Inserting chain")
	self.InsertChain(bchain)

	// move old canonical into workingTree chain
	bchain = &BlockChain{list.New()}
	for b := oldHead; bytes.Compare(b.Hash(), ancestorHash) != 0; b = self.GetBlock(b.PrevHash) {
		bchain.PushFront(&link{b, nil, nil, nil})
		// TODO: remove from database
	}

	// again, we have already processed, since its fucking canonical
	// but this is easy for now, gives an extra check
	_, err = self.TestChain(bchain)
	if err != nil && !IsTDError(err) {
		chainlogger.Infoln("Adding the old canonical chain to the workingTree failed. This shouldn't happen, and may imply that Jesus has returned")
		return
	}
}

// Add chain to working tree by connecting link blocks
// Add parent, child, and TD for each link
func (self *ChainManager) addChainToWorkingTree(chain *BlockChain) {
	base := new(big.Int)
	for e := chain.Front(); e != nil; e = e.Next() {
		l := e.Value.(*link)
		block := l.block
		s := string(block.Hash())
		// check if we've seen this block
		// XXX: if we never get passed something we've seen,
		//  we can remove
		if _, ok := self.workingTree[s]; ok {
			// remove from chain so it won't be processed or removed
			chain.Remove(e)
		} else {
			// Add parent and sum TD
			// parent is in the new chain or
			//  for first element is in workingTree or canonical
			if f := e.Prev(); f != nil {
				l.parent = f.Value.(*link)
				l.td = base.Add(f.Value.(*link).td, block.Difficulty)
			} else {
				// extending a fork
				if p := self.workingTree[string(block.PrevHash)]; p != nil {
					l.parent = p
					l.td = base.Add(p.td, block.Difficulty)
				} else {
					// new fork
					// sanity check that parent is on canonical
					b := self.GetBlockCanonical(block.PrevHash)
					if b == nil {
						fmt.Println("Chain does not have known parent")
						return
					}
					// use nil as marker for branch off canonical
					l.parent = nil
					parentDiff := b.BlockInfo().TD
					l.td = base.Add(parentDiff, b.Difficulty)
				}
			}
			// add child
			l.children = []*link{}
			if f := e.Next(); f != nil {
				l.children = append(l.children, f.Value.(*link))
			}
			// add to tree
			self.workingTree[s] = l
		}
	}
}

/*
   // Sum difficulties along this chain in the workingTree
   // TODO: Can we do this in TestChain?
   base := new(big.Int)
   for e := chain.Front(); e != nil; e = e.Next(){
       l := e.Value.(*link)
       block := l.block
       // add up difficulty
       td = base.Add(td, block.Difficulty)
       // this block should already be known
       b := self.workingTree[string(block.Hash())]
       b.td = td
   }

*/

/*
func (self *ChainManager) addChainToWorkingTree(chain *BlockChain){
    for e := chain.Front(); e != nil; e = e.Next(){
        l := e.Value.(*link)
        block := l.block
        s := string(block.Hash())
        // check if we've seen this block
        if _, ok := self.workingTree[s]; ok{
            // remove from chain so it won't be processed or removed
            chain.Remove(e)
        } else{
            // add parent
            if f := e.Prev(); f != nil{
                l.parent = f.Value.(*link)
            } else {
                // the parent is either in workingTree or on canonical
                if p := self.workingTree[string(block.PrevHash)]; p!=nil{
                    l.parent = p
                } else{
                    // sanity check that parent is on canonical
                    if b := self.GetBlockCanonical(block.PrevHash); b==nil{
                        return nil, fmt.Errorf("Chain does not have known parent")
                    }
                    // use nil as marker for branch off canonical
                    l.parent = nil
                }
            }
            // add child
            l.children = []*link{}
            if f := e.Next(); f != nil{
                l.children = append(l.children, f.Value.(*link))
            }
            // add to tree
            self.workingTree[s] = l
        }
    }
}

func (self *ChainManager) finalizeChainOnWorkingTree(chain *BlockChain, err error){
    if err != nil{
        for e := chain.Front(); e != nil; e = e.Next(){
            l := e.Value.(*link)
            block := l.block
            delete(self.workingTree, string(block.Hash()))
        }
    } else{
        e := chain.Front()
        if parent := e.Value.(*link).parent; parent != nil{
            // TODO: make sure this works (pointers ...)
            parent.children = append(parent.children, e.Value.(*link))
            // TODO: sum difficulties here?
        }
}*/
/*
func (self *ChainManager) sumDifficulties(chain *BlockChain){
    // Sum difficulties along this chain in the workingTree
    base := new(big.Int)
    for e := chain.Front(); e != nil; e = e.Next(){
        l := e.Value.(*link)
        block := l.block
        b := self.workingTree[string(block.Hash())]
        // sanity
        if b == nil{
            fmt.Println("An impossible tradgedy! tried to finalize an unknown block")
        } else {
            // add up difficulty
            b.td = base.Add(td, block.Difficulty)
        }
    }
}*/

// Detect if this chain extends or creates a fork
// ie. does the first block in the chain point to the head of
// canonical or not?
func (self *ChainManager) detectFork(chain *BlockChain) (*Block, bool) {
	var (
		oldest       = chain.Front().Value.(*link).block
		branchParent = self.GetBlock(oldest.PrevHash)
		head         = self.CurrentBlock()
	)

	if branchParent == nil {
		return nil, false
	}

	if bytes.Compare(head.Hash(), branchParent.Hash()) == 0 {
		return branchParent, false
	}

	return branchParent, true
}
