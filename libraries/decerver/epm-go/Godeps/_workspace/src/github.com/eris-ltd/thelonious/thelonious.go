package thelonious

import (
	"container/list"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkdoug"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkreact"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkrpc"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkwire"
)

const (
	seedTextFileUri string = "http://www.thelonious.org/servers.poc3.txt"
	//seedNodeAddress        = "162.218.65.211:30303"
	//seedNodeAddress        = "92.243.15.73:30303"
	seedNodeAddress = "localhost:30303"
)

var monklogger = monklog.NewLogger("SERV")

// not thread safe, callback should hold lock
func eachPeer(peers *list.List, callback func(*Peer, *list.Element)) {
	// Loop thru the peers and close them (if we had them)
	for e := peers.Front(); e != nil; e = e.Next() {
		//p := e.Value.(*Peer)
		callback(e.Value.(*Peer), e)
	}
}

const (
	processReapingTimeout = 60 // TODO increase
)

type Thelonious struct {
	// Channel for shutting down the thelonious
	shutdownChan chan bool
	quit         chan bool
	peerQuit     chan bool // shut down the peerHandler

	// DB interface
	db monkutil.Database
	// State manager for processing new blocks and managing the over all states
	blockManager *monkchain.BlockManager
	// The transaction pool. Transaction can be pushed on this pool
	// for later including in the blocks
	txPool *monkchain.TxPool
	// The canonical chain
	blockChain *monkchain.ChainManager
	// The block pool
	blockPool *BlockPool
	// Peers (NYI)
	peers *list.List
	// Nonce
	Nonce uint64
	// Listening addr
	Addr      net.Addr
	Port      string
	nat       NAT
	listening bool
	listener  net.Listener
	//
	peerMut sync.Mutex
	// Capabilities for outgoing peers
	serverCaps Caps
	// Specifies the desired amount of maximum peers
	MaxPeers int

	Mining bool

	reactor *monkreact.ReactorEngine

	RpcServer *monkrpc.JsonRpcServer

	keyManager *monkcrypto.KeyManager

	clientIdentity monkwire.ClientIdentity

	isUpToDate bool

	filters map[int]*monkchain.Filter

	// json based config object
	genConfig *monkdoug.GenesisConfig
	// model interface for validating actions
	protocol monkchain.Protocol
}

func New(db monkutil.Database, clientIdentity monkwire.ClientIdentity, keyManager *monkcrypto.KeyManager, caps Caps, usePnp bool, fetchPort int, checkPoint []byte, genConfig *monkdoug.GenesisConfig) (*Thelonious, error) {
	var err error
	var nat NAT

	if usePnp {
		nat, err = Discover()
		if err != nil {
			monklogger.Debugln("UPnP failed", err)
		}
	}

	bootstrapDb(db)

	monkutil.Config.Db = db

	nonce, _ := monkutil.RandomUint64()
	th := &Thelonious{
		shutdownChan:   make(chan bool),
		quit:           make(chan bool),
		peerQuit:       make(chan bool, 1),
		db:             db,
		peers:          list.New(),
		Nonce:          nonce,
		serverCaps:     caps,
		nat:            nat,
		keyManager:     keyManager,
		clientIdentity: clientIdentity,
		isUpToDate:     true,
		filters:        make(map[int]*monkchain.Filter),
	}

	protocol := th.setGenesis(genConfig)

	th.reactor = monkreact.New()

	th.blockPool = NewBlockPool(th)
	th.txPool = monkchain.NewTxPool(th)
	th.blockChain = monkchain.NewChainManager(protocol)
	th.blockManager = monkchain.NewBlockManager(th)
	th.blockChain.SetProcessor(th.blockManager)

	// Set chain's checkpoint
	if len(checkPoint) > 0 {
		th.blockChain.CheckPoint(checkPoint)
	}

	// Start the tx pool
	th.txPool.Start()

	// Start the genesis block service
	go th.ServeGenesis(fetchPort)

	return th, nil
}

func (s *Thelonious) Protocol() monkchain.Protocol {
	return s.protocol
}

// Loaded from genesis.json, possibly modified
// Sets the config object and the access model
func (s *Thelonious) setGenesis(genConfig *monkdoug.GenesisConfig) monkchain.Protocol {
	if s.genConfig != nil {
		fmt.Println("GenesisConfig already set")
		return nil
	}
	if genConfig.Model() == nil {
		genConfig.SetModel()
	}
	s.genConfig = genConfig
	s.protocol = genConfig.Model()
	return s.protocol
}

func (s *Thelonious) Reactor() *monkreact.ReactorEngine {
	return s.reactor
}

func (s *Thelonious) KeyManager() *monkcrypto.KeyManager {
	return s.keyManager
}

func (s *Thelonious) ClientIdentity() monkwire.ClientIdentity {
	return s.clientIdentity
}

func (s *Thelonious) ChainManager() *monkchain.ChainManager {
	return s.blockChain
}

func (s *Thelonious) BlockManager() *monkchain.BlockManager {
	return s.blockManager
}

func (s *Thelonious) TxPool() *monkchain.TxPool {
	return s.txPool
}
func (s *Thelonious) BlockPool() *BlockPool {
	return s.blockPool
}
func (self *Thelonious) Db() monkutil.Database {
	return self.db
}

func (s *Thelonious) ServerCaps() Caps {
	return s.serverCaps
}
func (s *Thelonious) IsMining() bool {
	return s.Mining
}

func (s *Thelonious) PeerCount() int {
	s.peerMut.Lock()
	defer s.peerMut.Unlock()
	return s.peers.Len()
}
func (s *Thelonious) IsUpToDate() bool {
	upToDate := true
	s.peerMut.Lock()
	defer s.peerMut.Unlock()
	eachPeer(s.peers, func(peer *Peer, e *list.Element) {
		if atomic.LoadInt32(&peer.connected) == 1 {
			if peer.catchingUp == true && peer.versionKnown {
				upToDate = false
			}
		}
	})
	return upToDate
}

func (s *Thelonious) PushPeer(peer *Peer) {
	s.peerMut.Lock()
	defer s.peerMut.Unlock()
	s.peers.PushBack(peer)
}
func (s *Thelonious) IsListening() bool {
	return s.listening
}

func (s *Thelonious) HighestTDPeer() (td *big.Int) {
	td = big.NewInt(0)

	eachPeer(s.peers, func(p *Peer, v *list.Element) {
		if p.td.Cmp(td) > 0 {
			td = p.td
		}
	})

	return
}

func (s *Thelonious) AddPeer(conn net.Conn) {
	peer := NewPeer(conn, s, true)

	if peer != nil {
		if s.peers.Len() < s.MaxPeers {
			peer.Start()
		} else {
			monklogger.Debugf("Max connected peers reached. Not adding incoming peer.")
		}
	}
}

func (s *Thelonious) ProcessPeerList(addrs []string) {
	for _, addr := range addrs {
		// TODO Probably requires some sanity checks
		s.ConnectToPeer(addr)
	}
}

func (s *Thelonious) ConnectToPeer(addr string) error {

	if s.peers.Len() < s.MaxPeers {
		var alreadyConnected bool

		ahost, aport, _ := net.SplitHostPort(addr)
		var chost string

		ips, err := net.LookupIP(ahost)

		if err != nil {
			return err
		} else {
			// If more then one ip is available try stripping away the ipv6 ones
			if len(ips) > 1 {
				var ipsv4 []net.IP
				// For now remove the ipv6 addresses
				for _, ip := range ips {
					if strings.Contains(ip.String(), "::") {
						continue
					} else {
						ipsv4 = append(ipsv4, ip)
					}
				}
				if len(ipsv4) == 0 {
					return fmt.Errorf("[SERV] No IPV4 addresses available for hostname")
				}

				// Pick a random ipv4 address, simulating round-robin DNS.
				rand.Seed(time.Now().UTC().UnixNano())
				i := rand.Intn(len(ipsv4))
				chost = ipsv4[i].String()
			} else {
				if len(ips) == 0 {
					return fmt.Errorf("[SERV] No IPs resolved for the given hostname")
					return nil
				}
				chost = ips[0].String()
			}
		}

		eachPeer(s.peers, func(p *Peer, v *list.Element) {
			if p.conn == nil {
				return
			}
			phost, _, _ := net.SplitHostPort(p.conn.RemoteAddr().String())
			pport := strconv.Itoa(int(p.port))

			if phost == chost && pport == aport {
				alreadyConnected = true
				monklogger.Debugf("Peer %s:%s already added.\n", chost, pport)
				return
			}
		})

		if alreadyConnected {
			return nil
		}
		NewOutboundPeer(addr, s, s.serverCaps)
	}

	return nil
}

func (s *Thelonious) OutboundPeers() []*Peer {
	// Create a new peer slice with at least the length of the total peers
	outboundPeers := make([]*Peer, s.peers.Len())
	length := 0
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		if !p.inbound && p.conn != nil {
			outboundPeers[length] = p
			length++
		}
	})

	return outboundPeers[:length]
}

func (s *Thelonious) InboundPeers() []*Peer {
	// Create a new peer slice with at least the length of the total peers
	inboundPeers := make([]*Peer, s.peers.Len())
	length := 0
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		if p.inbound {
			inboundPeers[length] = p
			length++
		}
	})

	return inboundPeers[:length]
}

func (s *Thelonious) InOutPeers() []*Peer {
	// Reap the dead peers first
	s.reapPeers()

	// Create a new peer slice with at least the length of the total peers
	inboundPeers := make([]*Peer, s.peers.Len())
	length := 0
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		// Only return peers with an actual ip
		if len(p.host) > 0 {
			inboundPeers[length] = p
			length++
		}
	})

	return inboundPeers[:length]
}

func (s *Thelonious) Broadcast(msgType monkwire.MsgType, data []interface{}) {
	msg := monkwire.NewMessage(msgType, data)
	s.BroadcastMsg(msg)
}

func (s *Thelonious) BroadcastMsg(msg *monkwire.Msg) {
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		p.QueueMessage(msg)
	})
}

func (s *Thelonious) Peers() *list.List {
	return s.peers
}

func (s *Thelonious) reapPeers() {
	s.peerMut.Lock()
	defer s.peerMut.Unlock()
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		if atomic.LoadInt32(&p.disconnect) == 1 || (p.inbound && (time.Now().Unix()-p.lastPong) > int64(5*time.Minute)) {
			s.removePeerElement(e)
		}
	})
}

func (s *Thelonious) removePeerElement(e *list.Element) {
	s.peerMut.Lock()
	defer s.peerMut.Unlock()

	s.peers.Remove(e)

	s.reactor.Post("peerList", s.peers)
}

func (s *Thelonious) RemovePeer(p *Peer) {
	eachPeer(s.peers, func(peer *Peer, e *list.Element) {
		if peer == p {
			s.removePeerElement(e)
		}
	})
}

func (s *Thelonious) ReapDeadPeerHandler() {
	reapTimer := time.NewTicker(processReapingTimeout * time.Second)

	for {
		select {
		case <-reapTimer.C:
			s.reapPeers()
		}
	}
}

// Start thelonious
func (s *Thelonious) Start(listen bool, seed string) {
	s.reactor.Start()
	s.blockPool.Start()
	if listen {
		s.StartListening()
		monklogger.Infoln("Server started")
	}

	if s.nat != nil {
		go s.upnpUpdateThread()
	}

	// Start the reaping processes
	go s.ReapDeadPeerHandler()
	go s.update()
	go s.filterLoop()

	if seed != "" {
		s.Seed(seed)
	}
	monklogger.Infoln("Peer handling started")

	if !s.ChainManager().WaitingForCheckpoint() {
		s.Reactor().Post("chainReady", "Chain is ready!")
	}
}

func (s *Thelonious) Seed(seed string) {
	ips := PastPeers()
	if len(ips) > 0 {
		for _, ip := range ips {
			monklogger.Infoln("Connecting to previous peer ", ip)
			s.ConnectToPeer(ip)
		}
	}
	monklogger.Infoln("Connecting to peer: ", seed)
	s.ConnectToPeer(seed)
	/* else {
		monklogger.Debugln("Retrieving seed nodes")

		// Eth-Go Bootstrapping
		ips, er := net.LookupIP("seed.bysh.me")
		if er == nil {
			peers := []string{}
			for _, ip := range ips {
				node := fmt.Sprintf("%s:%d", ip.String(), 30303)
				monklogger.Debugln("Found DNS Go Peer:", node)
				peers = append(peers, node)
			}
			s.ProcessPeerList(peers)
		}

		// Official DNS Bootstrapping
		_, nodes, err := net.LookupSRV("eth", "tcp", "ethereum.org")
		if err == nil {
			peers := []string{}
			// Iterate SRV nodes
			for _, n := range nodes {
				target := n.Target
				port := strconv.Itoa(int(n.Port))
				// Resolve target to ip (Go returns list, so may resolve to multiple ips?)
				addr, err := net.LookupHost(target)
				if err == nil {
					for _, a := range addr {
						// Build string out of SRV port and Resolved IP
						peer := net.JoinHostPort(a, port)
						monklogger.Debugln("Found DNS Bootstrap Peer:", peer)
						peers = append(peers, peer)
					}
				} else {
					monklogger.Debugln("Couldn't resolve :", target)
				}
			}
			// Connect to Peer list
			s.ProcessPeerList(peers)
		}

		// XXX tmp
		s.ConnectToPeer(seedNodeAddress)
	}*/
}

func (s *Thelonious) StartListening() {
	ln, err := net.Listen("tcp", ":"+s.Port)
	if err != nil {
		monklogger.Warnf("Port %s in use. Connection listening disabled. Acting as client", s.Port)
		s.listening = false
	} else {
		s.listening = true
		// add listener to thelonious so we can close it later
		s.listener = ln
		// Starting accepting connections
		monklogger.Infoln("Ready and accepting connections")
		// Start the peer handler

		// if we're in a container, we want to broadcast our public port
		// (which should be mapped to the port we listen on )
		if p := os.Getenv("PUBLIC_PORT"); p != "" {
			s.Port = p
		}

		go s.peerHandler(ln)
	}
}

// use to toggle listening
func (s *Thelonious) StopListening() {
	if s.listening {
		s.peerQuit <- true
		// does not kill already established peer go routines (just stops listening)
		s.listener.Close()
		s.listening = false
	}
}

func (s *Thelonious) peerHandler(listener net.Listener) {
out:
	for {
		select {
		case <-s.quit:
			break out
		case <-s.peerQuit: // so we can quit the listener without quiting the whole node
			break out
		default:
			// to stop, call s.listener.Close(). if a quit/peerQuit has been fired, itll catch and exit the loop
			conn, err := listener.Accept()
			if err != nil {
				monklogger.Debugln(err)
				continue
			}
			go s.AddPeer(conn)
		}
	}
	//listener.Close()
}

func (s *Thelonious) Stop() {
	// Close the database
	defer s.db.Close()

	var ips []string
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		ips = append(ips, p.conn.RemoteAddr().String())
	})

	if len(ips) > 0 {
		d, _ := json.MarshalIndent(ips, "", "    ")
		monkutil.WriteFile(path.Join(monkutil.Config.ExecPath, "known_peers.json"), d)
	}

	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		p.Stop()
	})

	close(s.quit)

	if s.listening {
		s.listener.Close() // release the listening port
		s.listening = false
	}

	if s.RpcServer != nil {
		s.RpcServer.Stop()
	}
	s.txPool.Stop()
	s.blockManager.Stop()
	s.reactor.Flush()
	s.reactor.Stop()
	s.blockPool.Stop()

	monklogger.Infoln("Server stopped")
	close(s.shutdownChan)
}

// This function will wait for a shutdown and resumes main thread execution
func (s *Thelonious) WaitForShutdown() {
	<-s.shutdownChan
}

func (s *Thelonious) upnpUpdateThread() {
	// Go off immediately to prevent code duplication, thereafter we renew
	// lease every 15 minutes.
	timer := time.NewTimer(5 * time.Minute)
	lport, _ := strconv.ParseInt(s.Port, 10, 16)
	first := true
out:
	for {
		select {
		case <-timer.C:
			var err error
			_, err = s.nat.AddPortMapping("TCP", int(lport), int(lport), "eth listen port", 20*60)
			if err != nil {
				monklogger.Debugln("can't add UPnP port mapping:", err)
				break out
			}
			if first && err == nil {
				_, err = s.nat.GetExternalAddress()
				if err != nil {
					monklogger.Debugln("UPnP can't get external address:", err)
					continue out
				}
				first = false
			}
			timer.Reset(time.Minute * 15)
		case <-s.quit:
			break out
		}
	}

	timer.Stop()

	if err := s.nat.DeletePortMapping("TCP", int(lport), int(lport)); err != nil {
		monklogger.Debugln("unable to remove UPnP port mapping:", err)
	} else {
		monklogger.Debugln("succesfully disestablished UPnP port mapping")
	}
}

func (self *Thelonious) update() {
	upToDateTimer := time.NewTicker(1 * time.Second)

out:
	for {
		select {
		case <-upToDateTimer.C:
			if self.IsUpToDate() && !self.isUpToDate {
				self.reactor.Post("chainSync", false)
				self.isUpToDate = true
			} else if !self.IsUpToDate() && self.isUpToDate {
				self.reactor.Post("chainSync", true)
				self.isUpToDate = false
			}
		case <-self.quit:
			break out
		}
	}
}

var filterId = 0

func (self *Thelonious) InstallFilter(object map[string]interface{}) (*monkchain.Filter, int) {
	defer func() { filterId++ }()

	filter := monkchain.NewFilterFromMap(object, self)
	self.filters[filterId] = filter

	return filter, filterId
}

func (self *Thelonious) UninstallFilter(id int) {
	delete(self.filters, id)
}

func (self *Thelonious) GetFilter(id int) *monkchain.Filter {
	return self.filters[id]
}

func (self *Thelonious) filterLoop() {
	blockChan := make(chan monkreact.Event, 5)
	messageChan := make(chan monkreact.Event, 5)
	// Subscribe to events
	reactor := self.Reactor()
	reactor.Subscribe("newBlock", blockChan)
	reactor.Subscribe("messages", messageChan)
out:
	for {
		select {
		case <-self.quit:
			break out
		case block := <-blockChan:
			if block, ok := block.Resource.(*monkchain.Block); ok {
				for _, filter := range self.filters {
					if filter.BlockCallback != nil {
						filter.BlockCallback(block)
					}
				}
			}
		case msg := <-messageChan:
			if messages, ok := msg.Resource.(monkstate.Messages); ok {
				for _, filter := range self.filters {
					if filter.MessageCallback != nil {
						msgs := filter.FilterMessages(messages)
						if len(msgs) > 0 {
							filter.MessageCallback(msgs)
						}
					}
				}
			}
		}
	}
}

func bootstrapDb(db monkutil.Database) {
	d, _ := db.Get([]byte("ProtocolVersion"))
	protov := monkutil.NewValue(d).Uint()

	if protov == 0 {
		db.Put([]byte("ProtocolVersion"), monkutil.NewValue(ProtocolVersion).Bytes())
	}
}

func PastPeers() []string {
	var ips []string
	data, _ := monkutil.ReadAllFile(path.Join(monkutil.Config.ExecPath, "known_peers.json"))
	json.Unmarshal([]byte(data), &ips)

	return ips
}
