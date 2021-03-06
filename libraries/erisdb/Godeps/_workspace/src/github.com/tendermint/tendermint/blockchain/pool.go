package blockchain

import (
	"sync"
	"sync/atomic"
	"time"

	. "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/common"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/types"
)

const (
	maxTries              = 3
	inputsChannelCapacity = 200
	requestIntervalMS     = 500
	maxPendingRequests    = 200
	maxTotalRequests      = 300
	maxRequestsPerPeer    = 300
)

// numTotal = numPending + blocks in the pool we havnt synced yet

var (
	requestTimeoutSeconds = time.Duration(3)
)

/*
	Peers self report their heights when a new peer joins the block pool.
	Starting from whatever we've got (pool.height), we request blocks
	in sequence from peers that reported higher heights than ours.
	Every so often we ask peers what height they're on so we can keep going.

	Requests are continuously made for blocks of heigher heights until
	the limits. If most of the requests have no available peers, and we
	are not at peer limits, we can probably switch to consensus reactor
*/

type BlockPool struct {
	// block requests
	requestsMtx sync.Mutex
	requests    map[uint]*bpRequest
	peerless    int32 // number of requests without peers
	height      uint  // the lowest key in requests.
	numPending  int32
	numTotal    int32

	// peers
	peersMtx sync.Mutex
	peers    map[string]*bpPeer

	requestsCh chan<- BlockRequest
	timeoutsCh chan<- string
	repeater   *RepeatTimer

	running int32 // atomic
}

func NewBlockPool(start uint, requestsCh chan<- BlockRequest, timeoutsCh chan<- string) *BlockPool {
	return &BlockPool{
		peers: make(map[string]*bpPeer),

		requests:   make(map[uint]*bpRequest),
		height:     start,
		numPending: 0,
		numTotal:   0,

		requestsCh: requestsCh,
		timeoutsCh: timeoutsCh,
		repeater:   NewRepeatTimer("", requestIntervalMS*time.Millisecond),

		running: 0,
	}
}

func (pool *BlockPool) Start() {
	if atomic.CompareAndSwapInt32(&pool.running, 0, 1) {
		log.Info("Starting BlockPool")
		go pool.run()
	}
}

func (pool *BlockPool) Stop() {
	if atomic.CompareAndSwapInt32(&pool.running, 1, 0) {
		log.Info("Stopping BlockPool")
		pool.repeater.Stop()
	}
}

func (pool *BlockPool) IsRunning() bool {
	return atomic.LoadInt32(&pool.running) == 1
}

// Run spawns requests as needed.
func (pool *BlockPool) run() {
RUN_LOOP:
	for {
		if atomic.LoadInt32(&pool.running) == 0 {
			break RUN_LOOP
		}
		_, numPending, numTotal := pool.GetStatus()
		if numPending >= maxPendingRequests {
			// sleep for a bit.
			time.Sleep(requestIntervalMS * time.Millisecond)
		} else if numTotal >= maxTotalRequests {
			// sleep for a bit.
			time.Sleep(requestIntervalMS * time.Millisecond)
		} else {
			// request for more blocks.
			height := pool.nextHeight()
			pool.makeRequest(height)
		}
	}
}

func (pool *BlockPool) GetStatus() (uint, int32, int32) {
	pool.requestsMtx.Lock() // Lock
	defer pool.requestsMtx.Unlock()

	return pool.height, pool.numPending, pool.numTotal
}

// We need to see the second block's Validation to validate the first block.
// So we peek two blocks at a time.
func (pool *BlockPool) PeekTwoBlocks() (first *types.Block, second *types.Block) {
	pool.requestsMtx.Lock() // Lock
	defer pool.requestsMtx.Unlock()

	if r := pool.requests[pool.height]; r != nil {
		first = r.block
	}
	if r := pool.requests[pool.height+1]; r != nil {
		second = r.block
	}
	return
}

// Pop the first block at pool.height
// It must have been validated by 'second'.Validation from PeekTwoBlocks().
func (pool *BlockPool) PopRequest() {
	pool.requestsMtx.Lock() // Lock
	defer pool.requestsMtx.Unlock()

	if r := pool.requests[pool.height]; r == nil || r.block == nil {
		panic("PopRequest() requires a valid block")
	}

	delete(pool.requests, pool.height)
	pool.height++
	pool.numTotal--
}

// Invalidates the block at pool.height.
// Remove the peer and request from others.
func (pool *BlockPool) RedoRequest(height uint) {
	pool.requestsMtx.Lock() // Lock
	defer pool.requestsMtx.Unlock()

	request := pool.requests[height]
	if request.block == nil {
		panic("Expected block to be non-nil")
	}
	// TODO: record this malfeasance
	// maybe punish peer on switch (an invalid block!)
	pool.RemovePeer(request.peerId) // Lock on peersMtx.
	request.block = nil
	request.peerId = ""
	pool.numPending++
	pool.peerless++

	go requestRoutine(pool, height)
}

func (pool *BlockPool) hasBlock(height uint) bool {
	pool.requestsMtx.Lock() // Lock
	defer pool.requestsMtx.Unlock()

	request := pool.requests[height]
	return request != nil && request.block != nil
}

func (pool *BlockPool) setPeerForRequest(height uint, peerId string) {
	pool.requestsMtx.Lock() // Lock
	defer pool.requestsMtx.Unlock()

	request := pool.requests[height]
	if request == nil {
		return
	}
	pool.peerless--
	request.peerId = peerId
}

func (pool *BlockPool) removePeerForRequest(height uint, peerId string) {
	pool.requestsMtx.Lock() // Lock
	defer pool.requestsMtx.Unlock()

	request := pool.requests[height]
	if request == nil {
		return
	}
	pool.peerless++
	request.peerId = ""
}

func (pool *BlockPool) AddBlock(block *types.Block, peerId string) {
	pool.requestsMtx.Lock() // Lock
	defer pool.requestsMtx.Unlock()

	request := pool.requests[block.Height]
	if request == nil {
		return
	}
	if request.peerId != peerId {
		return
	}
	if request.block != nil {
		return
	}
	request.block = block
	pool.numPending--
}

func (pool *BlockPool) getPeer(peerId string) *bpPeer {
	pool.peersMtx.Lock() // Lock
	defer pool.peersMtx.Unlock()

	peer := pool.peers[peerId]
	return peer
}

// Sets the peer's alleged blockchain height.
func (pool *BlockPool) SetPeerHeight(peerId string, height uint) {
	pool.peersMtx.Lock() // Lock
	defer pool.peersMtx.Unlock()

	peer := pool.peers[peerId]
	if peer != nil {
		peer.height = height
	} else {
		peer = &bpPeer{
			height:      height,
			id:          peerId,
			numRequests: 0,
		}
		pool.peers[peerId] = peer
	}
}

func (pool *BlockPool) RemovePeer(peerId string) {
	pool.peersMtx.Lock() // Lock
	defer pool.peersMtx.Unlock()

	delete(pool.peers, peerId)
}

// Pick an available peer with at least the given minHeight.
// If no peers are available, returns nil.
func (pool *BlockPool) pickIncrAvailablePeer(minHeight uint) *bpPeer {
	pool.peersMtx.Lock()
	defer pool.peersMtx.Unlock()

	for _, peer := range pool.peers {
		if peer.numRequests >= maxRequestsPerPeer {
			continue
		}
		if peer.height < minHeight {
			continue
		}
		peer.numRequests++
		return peer
	}
	return nil
}

func (pool *BlockPool) decrPeer(peerId string) {
	pool.peersMtx.Lock()
	defer pool.peersMtx.Unlock()

	peer := pool.peers[peerId]
	if peer == nil {
		return
	}
	peer.numRequests--
}

func (pool *BlockPool) nextHeight() uint {
	pool.requestsMtx.Lock() // Lock
	defer pool.requestsMtx.Unlock()

	// we make one request per height.
	return pool.height + uint(pool.numTotal)
}

func (pool *BlockPool) makeRequest(height uint) {
	pool.requestsMtx.Lock() // Lock
	defer pool.requestsMtx.Unlock()

	request := &bpRequest{
		height: height,
		peerId: "",
		block:  nil,
	}
	pool.requests[height] = request

	pool.peerless++

	nextHeight := pool.height + uint(pool.numTotal)
	if nextHeight == height {
		pool.numTotal++
		pool.numPending++
	}

	go requestRoutine(pool, height)
}

func (pool *BlockPool) sendRequest(height uint, peerId string) {
	if atomic.LoadInt32(&pool.running) == 0 {
		return
	}
	pool.requestsCh <- BlockRequest{height, peerId}
}

func (pool *BlockPool) sendTimeout(peerId string) {
	if atomic.LoadInt32(&pool.running) == 0 {
		return
	}
	pool.timeoutsCh <- peerId
}

func (pool *BlockPool) debug() string {
	pool.requestsMtx.Lock() // Lock
	defer pool.requestsMtx.Unlock()

	str := ""
	for h := pool.height; h < pool.height+uint(pool.numTotal); h++ {
		if pool.requests[h] == nil {
			str += Fmt("H(%v):X ", h)
		} else {
			str += Fmt("H(%v):", h)
			str += Fmt("B?(%v) ", pool.requests[h].block != nil)
		}
	}
	return str
}

//-------------------------------------

type bpPeer struct {
	id          string
	height      uint
	numRequests int32
}

type bpRequest struct {
	height uint
	peerId string
	block  *types.Block
}

//-------------------------------------

// Responsible for making more requests as necessary
// Returns only when a block is found (e.g. AddBlock() is called)
func requestRoutine(pool *BlockPool, height uint) {
	for {
		var peer *bpPeer = nil
	PICK_LOOP:
		for {
			if !pool.IsRunning() {
				log.Debug("BlockPool not running. Stopping requestRoutine", "height", height)
				return
			}
			peer = pool.pickIncrAvailablePeer(height)
			if peer == nil {
				//log.Debug("No peers available", "height", height)
				time.Sleep(requestIntervalMS * time.Millisecond)
				continue PICK_LOOP
			}
			break PICK_LOOP
		}

		// set the peer, decrement peerless
		pool.setPeerForRequest(height, peer.id)

		for try := 0; try < maxTries; try++ {
			pool.sendRequest(height, peer.id)
			time.Sleep(requestTimeoutSeconds * time.Second)
			// if successful the block is either in the pool,
			if pool.hasBlock(height) {
				pool.decrPeer(peer.id)
				return
			}
			// or already processed and we've moved past it
			bpHeight, _, _ := pool.GetStatus()
			if height < bpHeight {
				pool.decrPeer(peer.id)
				return
			}
		}

		// unset the peer, increment peerless
		pool.removePeerForRequest(height, peer.id)

		// this peer failed us, try again
		pool.RemovePeer(peer.id)
		pool.sendTimeout(peer.id)
	}
}

//-------------------------------------

type BlockRequest struct {
	Height uint
	PeerId string
}
