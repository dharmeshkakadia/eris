package monkminer

import (
	"bytes"
	"sort"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkreact"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkwire"
)

var logger = monklog.NewLogger("MINER")

type Miner struct {
	pow         monkchain.PoW
	thelonious  monkchain.NodeManager
	coinbase    []byte
	reactChan   chan monkreact.Event
	txs         monkchain.Transactions // is []*monkchain.Transaction
	uncles      []*monkchain.Block
	block       *monkchain.Block
	powChan     chan []byte
	powQuitChan chan monkreact.Event
	quitChan    chan chan error
	startChan   chan monkreact.Event

	turbo bool
}

func (self *Miner) GetPow() monkchain.PoW {
	return self.pow
}

func NewDefaultMiner(coinbase []byte, thelonious monkchain.NodeManager) *Miner {
	miner := Miner{
		pow:        &monkchain.EasyPow{},
		thelonious: thelonious,
		coinbase:   coinbase,
	}

	return &miner
}

func (self *Miner) ToggleTurbo() {
	self.turbo = !self.turbo

	self.pow.Turbo(self.turbo)
}

func (miner *Miner) Start() {
	miner.reactChan = make(chan monkreact.Event, 1)   // This is the channel that receives 'updates' when ever a new transaction or block comes in
	miner.powChan = make(chan []byte, 1)              // This is the channel that receives valid sha hashes for a given block
	miner.powQuitChan = make(chan monkreact.Event, 1) // This is the channel that can exit the miner thread
	miner.quitChan = make(chan chan error, 1)
	miner.startChan = make(chan monkreact.Event)

	// Insert initial TXs in our little miner 'pool'
	miner.txs = miner.thelonious.TxPool().Flush()
	miner.block = miner.thelonious.ChainManager().NewBlock(miner.coinbase)

	// Prepare inital block
	//miner.thelonious.BlockManager().Prepare(miner.block.State(), miner.block.State())
	go miner.listener()

	reactor := miner.thelonious.Reactor()
	reactor.Subscribe("newBlock", miner.reactChan)
	reactor.Subscribe("newTx:pre", miner.reactChan)
	reactor.Subscribe("chainReady", miner.startChan)

	// We need the quit chan to be a Reactor event.
	// The POW search method is actually blocking and if we don't
	// listen to the reactor events inside of the pow itself
	// The miner overseer will never get the reactor events themselves
	// Only after the miner will find the sha
	reactor.Subscribe("newBlock", miner.powQuitChan)
	reactor.Subscribe("newTx:pre", miner.powQuitChan)

	reactor.Post("miner:start", miner)
}

func (miner *Miner) listener() {
	// wait for the ready signal
	if miner.thelonious.ChainManager().WaitingForCheckpoint() {
		logger.Infoln("Waiting for start signal")
		<-miner.startChan
	}
	logger.Infoln("Started")
	for {
		select {
		case status := <-miner.quitChan:
			logger.Infoln("Stopped")
			status <- nil
			return
		case chanMessage := <-miner.reactChan:
			if block, ok := chanMessage.Resource.(*monkchain.Block); ok {
				miner.receiveBlock(block)
			}
			if tx, ok := chanMessage.Resource.(*monkchain.Transaction); ok {
				miner.receiveTx(tx)
			}
		default:
			miner.mineNewBlock()
		}
	}
}

func (miner *Miner) receiveTx(tx *monkchain.Transaction) {
	found := false
	for _, ctx := range miner.txs {
		if found = bytes.Compare(ctx.Hash(), tx.Hash()) == 0; found {
			break
		}
	}
	if found == false {
		// Undo all previous commits
		miner.block.Undo()
		// Apply new transactions
		miner.txs = append(miner.txs, tx)
	}
}

func (miner *Miner) receiveBlock(block *monkchain.Block) {
	//logger.Infoln("Got new block via Reactor")
	if bytes.Compare(miner.thelonious.ChainManager().CurrentBlockHash(), block.Hash()) == 0 {
		// TODO: Perhaps continue mining to get some uncle rewards
		//logger.Infoln("New top block found resetting state")

		// Filter out which Transactions we have that were not in this block
		var newtxs []*monkchain.Transaction
		for _, tx := range miner.txs {
			found := false
			for _, othertx := range block.Transactions() {
				if bytes.Compare(tx.Hash(), othertx.Hash()) == 0 {
					found = true
				}
			}
			if found == false {
				newtxs = append(newtxs, tx)
			}
		}
		miner.txs = newtxs

		// Setup a fresh state to mine on
		//miner.block = miner.thelonious.ChainManager().NewBlock(miner.coinbase, miner.txs)

	} else {
		if bytes.Compare(block.PrevHash, miner.thelonious.ChainManager().CurrentBlockPrevHash()) == 0 {
			logger.Infoln("Adding uncle block")
			miner.uncles = append(miner.uncles, block)
		}
	}
}

func (miner *Miner) Stop() {
	logger.Infoln("Stopping...")

	miner.powQuitChan <- monkreact.Event{}

	status := make(chan error)
	miner.quitChan <- status
	<-status

	reactor := miner.thelonious.Reactor()
	reactor.Unsubscribe("newBlock", miner.powQuitChan)
	reactor.Unsubscribe("newTx:pre", miner.powQuitChan)
	reactor.Unsubscribe("newBlock", miner.reactChan)
	reactor.Unsubscribe("newTx:pre", miner.reactChan)

	reactor.Post("miner:stop", miner)
}

func (self *Miner) mineNewBlock() {
	stateManager := self.thelonious.BlockManager()
	chainMan := self.thelonious.ChainManager()
	self.block = chainMan.NewBlock(self.coinbase)

	parent := self.thelonious.ChainManager().GetBlock(self.block.PrevHash)

	// if parent is not built yet, return
	if parent == nil {
		return
	}

	// check if we should even bother mining (potential energy savings)
	if !self.thelonious.Protocol().Participate(self.coinbase, parent) {
		return
	}

	// Apply uncles
	if len(self.uncles) > 0 {
		self.block.SetUncles(self.uncles)
	}

	// Sort the transactions by nonce in case of odd network propagation
	sort.Sort(monkchain.TxByNonce{self.txs})

	// Accumulate all valid transactions and apply them to the new state
	// Error may be ignored. It's not important during mining
	coinbase := self.block.State().GetOrNewStateObject(self.block.Coinbase)
	coinbase.SetGasPool(self.block.CalcGasLimit(parent))
	receipts, txs, unhandledTxs, err := stateManager.ProcessTransactions(coinbase, self.block.State(), self.block, self.block, self.txs)
	if err != nil {
		logger.Debugln(err)
	}
	self.txs = append(txs, unhandledTxs...)
	self.block.SetTxHash(receipts)

	// Set the transactions to the block so the new SHA3 can be calculated
	self.block.SetReceipts(receipts, txs)

	// Accumulate the rewards included for this block
	stateManager.AccumelateRewards(self.block.State(), self.block, parent)

	self.block.State().Update()

	logger.Infof("Mining on block %d. Includes %v transactions", self.block.Number, len(self.txs))

	// Find a valid nonce
	self.block.Nonce = self.pow.Search(self.block, self.powQuitChan)
	if self.block.Nonce != nil {
		// sign the block
		keypair := self.thelonious.KeyManager().KeyPair()
		self.block.Sign(keypair.PrivateKey)
		// process the completed block
		lchain := monkchain.NewChain(monkchain.Blocks{self.block})
		_, err := chainMan.TestChain(lchain)
		if err != nil {
			logger.Infoln(err)
		} else {
			chainMan.InsertChain(lchain)
			//	self.thelonious.EventMux().Post(chain.NewBlockEvent{block})
			logger.Infoln("posting new block!")
			self.thelonious.Reactor().Post("newBlock", self.block)
			self.thelonious.Broadcast(monkwire.MsgBlockTy, []interface{}{self.block.Value().Val})

			logger.Infof("🔨  Mined block %x\n", self.block.Hash())
			logger.Infoln(self.block)
			self.txs = self.thelonious.TxPool().CurrentTransactions()
		}

		// go self.mineNewBlock()
		/*
			err := self.thelonious.BlockManager().Process(self.block, false)
			if err != nil {
				logger.Infoln(err)
			} else {
				self.thelonious.Broadcast(monkwire.MsgBlockTy, []interface{}{self.block.Value().Val})
				logger.Infof("🔨  Mined block %x\n", self.block.Hash())
				logger.Infoln(self.block)
				// Gather the new batch of transactions currently in the tx pool
				self.txs = self.thelonious.TxPool().CurrentTransactions()
			}*/
	}
}
