package thelonious

import (
	"bytes"
	"container/list"
	//"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkreact"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkwire"
)

var poollogger = monklog.NewLogger("BPOOL")

type block struct {
	from      *Peer
	peer      *Peer
	block     *monkchain.Block
	reqAt     time.Time
	requested int
}

type BlockPool struct {
	mut sync.Mutex // TODO: should this be RW?

	eth *Thelonious

	hashPool [][]byte
	pool     map[string]*block

	td   *big.Int
	quit chan bool

	fetchingHashes    bool
	downloadStartedAt time.Time

	ChainLength, BlocksProcessed int

	peer *Peer

	start chan monkreact.Event
}

func NewBlockPool(eth *Thelonious) *BlockPool {
	return &BlockPool{
		eth:   eth,
		pool:  make(map[string]*block),
		td:    monkutil.Big0,
		quit:  make(chan bool),
		start: make(chan monkreact.Event),
	}
}

func (self *BlockPool) Len() int {
	return len(self.hashPool)
}

func (self *BlockPool) Reset() {
	self.mut.Lock()
	defer self.mut.Unlock()
	self.pool = make(map[string]*block)
	self.hashPool = nil
}

func (self *BlockPool) HasLatestHash() bool {
	self.mut.Lock()
	defer self.mut.Unlock()

	return self.pool[string(self.eth.ChainManager().CurrentBlock().Hash())] != nil
}

func (self *BlockPool) HasCommonHash(hash []byte) bool {
	cman := self.eth.ChainManager()
	if cman.WaitingForCheckpoint() && cman.IsCheckpoint(hash) {
		poollogger.Debugln("still waiting for checkpoiny...")
		return true
	}

	return cman.GetBlock(hash) != nil
}

func (self *BlockPool) Blocks() (blocks monkchain.Blocks) {
	self.mut.Lock()
	defer self.mut.Unlock()
	for _, item := range self.pool {
		if item.block != nil {
			blocks = append(blocks, item.block)
		}
	}

	return
}

// Add a hash to the hash pool
// and an empty block obj to block pool
func (self *BlockPool) AddHash(hash []byte, peer *Peer) {
	self.mut.Lock()
	defer self.mut.Unlock()

	//cman := self.eth.ChainManager()
	// if we are waiting for a checkpoint block and this
	// isn't it, do nothing
	/*if cman.WaitingForCheckpoint() && !cman.IsCheckpoint(hash){
	    poollogger.Debugln("Still waiting for checkpoint and this isn't it!")
	    return
	}*/

	if self.pool[string(hash)] == nil {
		self.pool[string(hash)] = &block{peer, nil, nil, time.Now(), 0}
		self.hashPool = append([][]byte{hash}, self.hashPool...)
	}
}

// A block has just come in from a peer
// If the block is a checkpoint block, pass it to the chain
// If we are still waiting for a checkpoint, do nothing
// If the block came before the checkpoint block, ignore it
// If we have the block already, do nothing
// If we haven't seen the hash, before, the block is unrequested
//      If we haven't seen its parents either, ignore it; TODO (better)
// If we've seen the hash, add the block to the pool
func (self *BlockPool) Add(b *monkchain.Block, peer *Peer) {
	self.mut.Lock()
	defer self.mut.Unlock()

	hash := string(b.Hash())
	cman := self.eth.ChainManager()

	// if we're still waiting for a checkpoint block,
	// check if this is it
	if cman.WaitingForCheckpoint() {
		if cman.ReceiveCheckPointBlock(b) {
			poollogger.Infof("Received checkpoint block (#%d) %x from peer", b.Number, b.Hash())
			self.eth.Broadcast(monkwire.MsgGetStateTy, []interface{}{b.Hash()})
		}
		return
	}

	// if block is before checkpoint, do nothing
	if b.Number.Uint64() < cman.LatestCheckPointNumber() {
		return
	}

	// if we have the block, do nothing
	if cman.HasBlock(b.Hash()) {
		return
	}

	// Note this doesn't check the working tree
	// Leave it to TestChain to ignore blocks already in forks
	// Also, we can one day use the information on which/howmany peers
	//  give us which blocks, in the td calculation. Hold on to your hats!
	if self.pool[hash] == nil {
		poollogger.Infof("Got unrequested block (%x...)\n", hash[0:4])

		self.hashPool = append(self.hashPool, b.Hash())
		self.pool[hash] = &block{peer, peer, b, time.Now(), 0}

		if !cman.HasBlock(b.PrevHash) && self.pool[string(b.PrevHash)] == nil && !self.fetchingHashes {
			poollogger.Infof("Unknown block, requesting parent (%x...)\n", b.PrevHash[0:4])
			//peer.QueueMessage(monkwire.NewMessage(monkwire.MsgGetBlockHashesTy, []interface{}{b.Hash(), uint32(256)}))
		}
	} else if self.pool[hash] != nil {
		self.pool[hash].block = b
	}

	self.BlocksProcessed++
}

func (self *BlockPool) Remove(hash []byte) {
	self.mut.Lock()
	defer self.mut.Unlock()

	self.hashPool = monkutil.DeleteFromByteSlice(self.hashPool, hash)
	delete(self.pool, string(hash))
}

// Distribute pool hashes over peers and fetch them
func (self *BlockPool) DistributeHashes() {
	// TODO: can we do better than locking up everything to run this?
	self.mut.Lock()
	defer self.mut.Unlock()

	var (
		peerLen = self.eth.PeerCount()
		amount  = 256 * peerLen
		dist    = make(map[*Peer][][]byte)
	)

	// min (amount, len(pool))
	num := int(math.Min(float64(amount), float64(len(self.pool))))
	for i, j := 0, 0; i < self.Len() && j < num; i++ {
		hash := self.hashPool[i]
		item := self.pool[string(hash)]

		// if we have the item but not the block
		//  - if we have a peer, try to get block
		//  - if not, find a peer
		//  - append hash to dist[peer]
		if item != nil && item.block == nil {
			var peer *Peer
			lastFetchFailed := time.Since(item.reqAt) > 5*time.Second

			// Handle failed requests
			if lastFetchFailed && item.requested > 5 && item.peer != nil {
				if item.requested < 100 {
					// Select peer the hash was retrieved off
					peer = item.from
				} else {
					// Remove it
					self.hashPool = monkutil.DeleteFromByteSlice(self.hashPool, hash)
					delete(self.pool, string(hash))
				}
			} else if lastFetchFailed || item.peer == nil {
				// Find a suitable, available peer
				eachPeer(self.eth.peers, func(p *Peer, v *list.Element) {
					if peer == nil && len(dist[p]) < amount/peerLen {
						peer = p
					}
				})
			}

			if peer != nil {
				item.reqAt = time.Now()
				item.peer = peer
				item.requested++

				dist[peer] = append(dist[peer], hash)
			}
		}
	}

	for peer, hashes := range dist {
		peer.FetchBlocks(hashes)
	}

	if len(dist) > 0 {
		self.downloadStartedAt = time.Now()
	}
}

func (self *BlockPool) Start() {
	self.eth.Reactor().Subscribe("chainReady", self.start)
	go self.downloadThread()
	go self.chainThread()
}

func (self *BlockPool) Stop() {
	close(self.quit)
}

func (self *BlockPool) downloadThread() {
	serviceTimer := time.NewTicker(100 * time.Millisecond)
	flushedTimer := time.NewTicker(5 * time.Second)
out:
	for {
		select {
		case <-self.quit:
			break out
		case <-serviceTimer.C:
			// If we're not catching up.
			if !self.areWeFetchingHashes() {
				// If we're waiting for a checkpoint, request it from
				// all peers
				cman := self.eth.ChainManager()
				if cman.WaitingForCheckpoint() {
					eachPeer(self.eth.peers, func(p *Peer, v *list.Element) {
						p.FetchBlocks([][]byte{cman.LatestCheckPointHash()})
					})
				} else {
					// distribute the hashes to peers
					// and download the blockchain
					self.DistributeHashes()
				}
			}

			self.setChainLength()
		case <-flushedTimer.C:
			// if pool is empty, get hashes
			if self.Len() == 0 {
				eachPeer(self.eth.peers, func(p *Peer, v *list.Element) {
					//p.FetchHashes()
				})
			}
		}
	}
}

func (self *BlockPool) setChainLength() {
	self.mut.Lock()
	defer self.mut.Unlock()
	if self.ChainLength < len(self.hashPool) {
		self.ChainLength = len(self.hashPool)
	}
}

func (self *BlockPool) areWeFetchingHashes() bool {
	self.eth.peerMut.Lock()
	defer self.eth.peerMut.Unlock()
	fetchingHashes := false
	eachPeer(self.eth.peers, func(p *Peer, v *list.Element) {
		if p.statusKnown && p.FetchingHashes() {
			self.fetchingHashes = true
		}
	})
	self.fetchingHashes = fetchingHashes

	return fetchingHashes
}

// Sort blocks in pool by number
// Find first with prevhash in canonical
// Find first consecutive chain
// TestChain (add blocks to workingTree, remove if any fail)
// InsertChain (add to canonical
//      or      sum difficulties of fork
//      and     possibly cause re-org
func (self *BlockPool) chainThread() {
	// wait for the start signal from the state
	if self.eth.ChainManager().WaitingForCheckpoint() {
		<-self.start
	}
	procTimer := time.NewTicker(500 * time.Millisecond)
out:
	for {
		select {
		case <-self.quit:
			break out
		case <-procTimer.C:
			// We'd need to make sure that the pools are properly protected by a mutex
			blocks := self.Blocks()
			monkchain.BlockBy(monkchain.Number).Sort(blocks)

			// Find first block with prevhash in canonical
			for i, block := range blocks {
				if self.eth.ChainManager().HasBlock(block.PrevHash) {
					blocks = blocks[i:]
					break
				}
			}

			// Find first conescutive chain
			if len(blocks) > 0 {
				// Find chain of blocks
				if self.eth.ChainManager().HasBlock(blocks[0].PrevHash) {
					for i, block := range blocks[1:] {
						// NOTE: The Ith element in this loop refers to the previous block in
						// outer "blocks"
						if bytes.Compare(block.PrevHash, blocks[i].Hash()) != 0 {
							blocks = blocks[:i]
							break
						}
					}
				} else {
					blocks = nil
				}
			}

			// TODO figure out whether we were catching up
			// If caught up and just a new block has been propagated:
			// sm.eth.EventMux().Post(NewBlockEvent{block})
			// otherwise process and don't emit anything
			if len(blocks) > 0 {
				chainManager := self.eth.ChainManager()

				// sling blocks into a list
				bchain := monkchain.NewChain(blocks)
				// validate the chain
				_, err := chainManager.TestChain(bchain)

				// If validation failed, we flush the pool
				// and punish the peer
				if err != nil && !monkchain.IsTDError(err) {
					poollogger.Debugln(err)

					self.Reset()
					//self.punishPeer()
				} else {
					// Validation was successful
					// Sum-difficulties, insert chain
					// Possibly re-org
					chainManager.InsertChain(bchain)
					// Remove all blocks from pool
					for _, block := range blocks {
						self.Remove(block.Hash())
					}
				}
			}

			/* Do not propagate to the network on catchups
			if amount == 1 {
				block := self.eth.ChainManager().CurrentBlock
				self.eth.Broadcast(monkwire.MsgBlockTy, []interface{}{block.Value().Val})
			}*/
		}
	}
}

func (self *BlockPool) punishPeer() {
	/*
		                        TODO: fix this peer handling!
							if self.peer != nil && self.peer.conn != nil {
								poollogger.Debugf("Punishing peer for supplying bad chain (%v)\n", self.peer.conn.RemoteAddr())

							// This peer gave us bad hashes and made us fetch a bad chain, therefor he shall be punished.
							//self.eth.BlacklistPeer(self.peer)
							//self.peer.StopWithReason(DiscBadPeer)
		                        self.peer.Stop()
		                        self.td = monkutil.Big0
		                        self.peer = nil
							}*/

}
