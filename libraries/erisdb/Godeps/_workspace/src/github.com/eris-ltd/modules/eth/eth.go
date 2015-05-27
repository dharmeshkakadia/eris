package eth

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/user"
	"strconv"
	"time"

	ethutils "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/cmd/utils"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/core"
	ethtypes "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/core/types"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/crypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/eth"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/ethutil"
	ethevent "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/event"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/logger" //"github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/modules/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/chain"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/miner"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/state"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/xeth"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/modules/types"
	"github.com/eris-ltd/epm-go/utils"
)

var (
	GoPath = os.Getenv("GOPATH") // error?!
	usr, _ = user.Current()
)

var (
	GAS      = "500000"
	GASPRICE = "500"
)

//Logging
var ethlogger *logger.Logger = logger.NewLogger("EthGlue")

// implements epm Blockchain
// this will get passed to Otto (javascript vm)
// as such, it does not have "administrative" methods
type EthModule struct {
	Config      *ChainConfig
	ethereum    *eth.Ethereum
	pipe        *xeth.XEth
	keyManager  *crypto.KeyManager
	eventMux    *ethevent.TypeMux
	started     bool
	chans       map[string]chan types.Event
	subscribers map[string]ethevent.Subscription // <-chan interface{}
	newBlockSub ethevent.Subscription

	miner *miner.Miner
}

/*
   First, the functions to satisfy Module
*/

// Create a new EthModule and internal Eth, with default config.
// Accepts an ethereum instance to yield a new
// interface into the same chain.
// It will not initialize a ethereum object for you,
// giving you a chance to adjust configs before calling `Init()`
func NewEth(et *eth.Ethereum) *EthModule {
	em := new(EthModule)
	// Here we load default config and leave it to caller
	// to read a config file to overwrite
	em.Config = DefaultConfig
	if et != nil {
		em.ethereum = et
	}

	em.started = false
	return em
}

// initialize an chain
// it may or may not already have a ethereum instance
// basically gives you a pipe, local keyMang, and reactor
func (mod *EthModule) Init() error {
	// if didn't call NewEth
	if mod.Config == nil {
		mod.Config = DefaultConfig
	}

	//ethdoug.Adversary = mod.Config.Adversary

	// if no ethereum instance
	if mod.ethereum == nil {
		mod.ethConfig()
		mod.newEthereum()
	}

	// public interface
	pipe := xeth.New(mod.ethereum)
	// load keys from file. genesis block keys. convenient for testing

	mod.pipe = pipe
	mod.keyManager = mod.ethereum.KeyManager()
	mod.eventMux = mod.ethereum.EventMux()

	// subscribe to the new block
	mod.chans = make(map[string]chan types.Event)
	mod.subscribers = make(map[string]ethevent.Subscription) //<-chan interface{})
	//mod.Subscribe("newBlock", "newBlock", "")
	//mod.newBlockSub = mod.eventMux.Subscribe(core.NewBlockEvent{})

	return nil
}

// start the ethereum node
func (mod *EthModule) Start() error {
	seed := ""
	if mod.Config.UseSeed {
		seed = mod.Config.SeedAddr
	}
	mod.ethereum.Start(seed) // peer seed
	mod.started = true

	if mod.Config.Mining {
		ethutils.StartMining(mod.ethereum)
	}
	return nil
}

func (eth *EthModule) Shutdown() error {
	if !eth.started {
		return nil
	}
	eth.StopMining()
	fmt.Println("stopped mining")
	eth.ethereum.Stop()
	fmt.Println("stopped ethereum")
	eth = &EthModule{Config: eth.Config}
	logger.Reset()
	return nil
}

// ReadConfig and WriteConfig implemented in config.go

// What module is this?
func (mod *EthModule) Name() string {
	return "eth"
}

/*
   Non-interface functions that otherwise prove useful
    in standalone applications, testing, and debuging
*/

func (mod *EthModule) EthState() *state.StateDB {
	return mod.pipe.State().State()
}

/*
   Implement Blockchain
*/

func (eth *EthModule) ChainId() (string, error) {
	// TODO: implement  BlockN() !
	return "default", nil
}

func (mod *EthModule) WaitForShutdown() {
	mod.ethereum.WaitForShutdown()
}

func (eth *EthModule) WorldState() *types.WorldState {
	state := eth.pipe.State().State()
	stateMap := &types.WorldState{make(map[string]*types.Account), []string{}}

	it := state.Trie().Iterator()
	for it.Next() { //(func(addr string, acct *ethutil.Value) {
		addr := it.Key
		//acct := it.Value
		hexAddr := ethutil.Bytes2Hex([]byte(addr))
		stateMap.Order = append(stateMap.Order, hexAddr)
		stateMap.Accounts[hexAddr] = eth.Account(hexAddr)

	}
	return stateMap
}

func (eth *EthModule) State() *types.State {
	state := eth.pipe.State().State()
	stateMap := &types.State{make(map[string]*types.Storage), []string{}}

	it := state.Trie().Iterator()
	for it.Next() { //(func(addr string, acct *ethutil.Value) {
		addr := it.Key
		//acct := it.Value
		hexAddr := ethutil.Bytes2Hex([]byte(addr))
		stateMap.Order = append(stateMap.Order, hexAddr)
		stateMap.State[hexAddr] = eth.Storage(hexAddr)

	}
	return stateMap
}

func (eth *EthModule) Storage(addr string) *types.Storage {
	w := eth.pipe.State()
	obj := w.SafeGet(addr).StateObject
	ret := &types.Storage{make(map[string]string), []string{}}
	obj.EachStorage(func(k string, v *ethutil.Value) {
		kk := ethutil.Bytes2Hex([]byte(k))
		vv := ethutil.Bytes2Hex(v.Bytes())
		ret.Order = append(ret.Order, kk)
		ret.Storage[kk] = vv
	})
	return ret
}

func (eth *EthModule) Account(target string) *types.Account {
	w := eth.pipe.State()
	obj := w.SafeGet(target).StateObject

	bal := ethutil.NewValue(obj.Balance).String()
	nonce := obj.Nonce
	script := ethutil.Bytes2Hex(obj.Code)
	storage := eth.Storage(target)
	isscript := len(storage.Order) > 0 || len(script) > 0

	return &types.Account{
		Address:  target,
		Balance:  bal,
		Nonce:    strconv.Itoa(int(nonce)),
		Script:   script,
		Storage:  storage,
		IsScript: isscript,
	}
}

func (eth *EthModule) StorageAt(contract_addr string, storage_addr string) string {
	var saddr *big.Int
	if ethutil.IsHex(storage_addr) {
		saddr = ethutil.BigD(ethutil.Hex2Bytes(utils.StripHex(storage_addr)))
	} else {
		saddr = ethutil.Big(storage_addr)
	}

	//contract_addr = ethutil.StripHex(contract_addr)
	w := eth.pipe.State()
	ret := w.SafeGet(utils.StripHex(contract_addr)).GetStorage(saddr)
	if ret.IsNil() {
		return ""
	}
	return ethutil.Bytes2Hex(ret.Bytes())
}

func (eth *EthModule) BlockCount() int {
	return int(eth.ethereum.ChainManager().LastBlockNumber())
}

func (eth *EthModule) LatestBlock() string {
	return ethutil.Bytes2Hex(eth.ethereum.ChainManager().CurrentBlock().Hash())
}

func (eth *EthModule) Block(hash string) *types.Block {
	hashBytes := ethutil.Hex2Bytes(hash)
	block := eth.ethereum.ChainManager().GetBlock(hashBytes)
	return convertBlock(block)
}

func (eth *EthModule) IsScript(target string) bool {
	// is contract if storage is empty and no bytecode
	obj := eth.Account(target)
	storage := obj.Storage
	if len(storage.Order) == 0 && obj.Script == "" {
		return false
	}
	return true
}

// send a tx
func (eth *EthModule) Tx(addr, amt string) (string, error) {
	//keys := eth.fetchKeyPair()
	//addr = ethutil.StripHex(addr)
	if addr[:2] == "0x" {
		addr = addr[2:]
	}
	// note, NewValue will not turn a string int into a big int..
	start := time.Now()
	//tx, err := eth.pipe.Transact(keys, byte_addr, ethutil.NewValue(ethutil.Big(amt)), ethutil.NewValue(ethutil.Big("200")), ethutil.NewValue(ethutil.Big("100000")), []byte(""))
	tx, err := eth.pipe.Transact(addr, amt, GAS, GASPRICE, "")
	dif := time.Since(start)
	fmt.Println("pipe tx took ", dif)
	if err != nil {
		return "", err
	}
	return tx, nil
}

// send a message to a contract
// data is prepacked by epm
func (eth *EthModule) Msg(addr string, data []string) (string, error) {
	packed := data[0]
	//packed := PackTxDataArgs(data...)
	//packed = abi + packed[2:]
	//fmt.Println("PACKED:", packed)
	//keys := eth.fetchKeyPair()
	//addr = ethutil.StripHex(addr)
	tx, err := eth.pipe.Transact(addr, "0", GAS, GASPRICE, packed)
	if err != nil {
		return "", err
	}
	return tx, nil
}

// simulate sending msg to contract
func (eth *EthModule) Call(addr string, data []string) (string, error) {
	packed := data[0]
	tx, err := eth.pipe.Call(addr, "0", GAS, GASPRICE, packed)
	if err != nil {
		return "", err
	}
	return tx, nil
}

func (eth *EthModule) Script(script string) (string, string, error) {
	txid, addr, err := eth.pipe.Create("0", GAS, GASPRICE, script)
	if err != nil {
		return "", "", err
	}
	return txid, addr, nil
}

// returns a chanel that will fire when address is updated
func (eth *EthModule) Subscribe(name, event, target string) chan types.Event {
	var eventObj interface{}
	var subscriber ethevent.Subscription
	switch event {
	case "newBlock":
		eventObj = core.NewBlockEvent{}
		subscriber = eth.eventMux.Subscribe(eventObj)
	}

	th_ch := subscriber.Chan()

	ch := make(chan types.Event)
	eth.chans[name] = ch
	eth.subscribers[name] = subscriber

	// fire up a goroutine and broadcast module specific chan on our main chan
	go func() {
		for {
			eve, more := <-th_ch
			if !more {
				break
			}
			returnEvent := types.Event{
				Event:     event,
				Target:    target,
				Source:    "eth",
				TimeStamp: time.Now(),
			}
			switch eve := eve.(type) {
			case core.NewBlockEvent:
				block := eve.Block
				returnEvent.Resource = convertBlock(block)
			case core.TxPreEvent:
			}
			// cast resource to appropriate type
			/*
				resource := eve.Resource
				} else if tx, ok := resource.(chain.Transaction); ok {
					returnEvent.Resource = convertTx(&tx)
				} else if txFail, ok := resource.(chain.TxFail); ok {
					tx := convertTx(txFail.Tx)
					tx.Error = txFail.Err.Error()
					returnEvent.Resource = tx
				} else {
					ethlogger.Errorln("Invalid event resource type", resource)
				}*/
			ch <- returnEvent
		}
	}()
	return ch
	return nil
}

func (eth *EthModule) UnSubscribe(name string) {
	if c, ok := eth.subscribers[name]; ok {
		c.Unsubscribe()
		delete(eth.subscribers, name)
	}
	if c, ok := eth.chans[name]; ok {
		close(c)
		delete(eth.chans, name)
	}
}

// Mine a block
func (m *EthModule) Commit() {
	c := m.eventMux.Subscribe(core.NewBlockEvent{})
	m.StartMining()
	_ = <-c.Chan()
	m.StopMining()
	c.Unsubscribe()
}

// start and stop continuous mining
func (m *EthModule) AutoCommit(toggle bool) {
	if toggle {
		m.StartMining()
	} else {
		m.StopMining()
	}
}

func (m *EthModule) IsAutocommit() bool {
	return m.ethereum.IsMining()
}

/*
   Blockchain interface should also satisfy KeyManager
   All values are hex encoded
*/

// Return the active address
func (eth *EthModule) ActiveAddress() string {
	keypair := eth.keyManager.KeyPair()
	addr := ethutil.Bytes2Hex(keypair.Address())
	return addr
}

// Return the nth address in the ring
func (eth *EthModule) Address(n int) (string, error) {
	ring := eth.keyManager.KeyRing()
	if n >= ring.Len() {
		return "", fmt.Errorf("cursor %d out of range (0..%d)", n, ring.Len())
	}
	pair := ring.GetKeyPair(n)
	addr := ethutil.Bytes2Hex(pair.Address())
	return addr, nil
}

// Set the address
func (eth *EthModule) SetAddress(addr string) error {
	n := -1
	i := 0
	ring := eth.keyManager.KeyRing()
	ring.Each(func(kp *crypto.KeyPair) {
		a := ethutil.Bytes2Hex(kp.Address())
		if a == addr {
			n = i
		}
		i += 1
	})
	if n == -1 {
		return fmt.Errorf("Address %s not found in keyring", addr)
	}
	return eth.SetAddressN(n)
}

// Set the address to be the nth in the ring
func (eth *EthModule) SetAddressN(n int) error {
	return eth.keyManager.SetCursor(n)
}

// Generate a new address
func (eth *EthModule) NewAddress(set bool) string {
	newpair := crypto.GenerateNewKeyPair()
	addr := ethutil.Bytes2Hex(newpair.Address())
	ring := eth.keyManager.KeyRing()
	ring.AddKeyPair(newpair)
	if set {
		eth.SetAddressN(ring.Len() - 1)
	}
	return addr
}

// Return the number of available addresses
func (eth *EthModule) AddressCount() int {
	return eth.keyManager.KeyRing().Len()
}

/*
   Helper functions
*/

// create a new ethereum instance
// expects ethConfig to already have been called!
// init db, nat/upnp, ethereum struct, reactorEngine, txPool, blockChain, stateManager
func (m *EthModule) newEthereum() {
	db := NewDatabase(m.Config.DbName)

	keyManager := NewKeyManager(m.Config.KeyStore, m.Config.RootDir, db)
	err := keyManager.Init(m.Config.KeySession, m.Config.KeyCursor, false)
	if err != nil {
		log.Fatal(err)
	}
	m.keyManager = keyManager

	c := new(eth.Config)
	m.fillConfig(c)

	// create the ethereum obj
	//th, err := eth.New(db, clientIdentity, m.keyManager, eth.CapDefault, false)
	th, err := eth.New(c)

	if err != nil {
		log.Fatal("Could not start node: %s\n", err)
	}

	m.ethereum = th
}

func (m *EthModule) fillConfig(c *eth.Config) {
	c.Port = strconv.Itoa(m.Config.Port)
	//c.Name = m.Config.
	c.Version = m.Config.Version
	c.Identifier = m.Config.ClientIdentifier
	c.KeyStore = m.Config.KeyStore
	c.DataDir = m.Config.RootDir
	c.LogFile = m.Config.LogFile
	c.LogLevel = m.Config.LogLevel
	c.KeyRing = m.Config.KeySession
	c.MaxPeers = m.Config.MaxPeers
	//c.NATType =
	//c.PMPGateway
	c.Shh = false
	c.Dial = false
}

// returns hex addr of gendoug
/*
func (eth *EthModule) GenDoug() string {
	return ethutil.Bytes2Hex(ethdoug.GenDougByteAddr)
}*/

func (eth *EthModule) StartMining() bool {
	if !eth.ethereum.Mining {
		eth.ethereum.Mining = true
		addr := eth.ethereum.KeyManager().Address()

		go func() {
			ethlogger.Infoln("Start mining")
			// Give it some time to connect with peers
			time.Sleep(3 * time.Second)
			if eth.miner == nil {
				eth.miner = miner.New(addr, eth.ethereum)
			}
			eth.miner.Start()
		}()
		RegisterInterrupt(func(os.Signal) {
			eth.StopMining()
		})
		return true

	}
	return false
}

func (eth *EthModule) StopMining() bool {
	if eth.ethereum.Mining && eth.miner != nil {
		eth.miner.Stop()
		ethlogger.Infoln("Stopped mining")
		eth.ethereum.Mining = false
		eth.miner = nil
		return true
	}
	return false
}

func (eth *EthModule) StartListening() {
	//eth.ethereum.StartListening()
}

func (eth *EthModule) StopListening() {
	//eth.ethereum.StopListening()
}

/*
   some key management stuff
*/

func (eth *EthModule) fetchPriv() string {
	keypair := eth.keyManager.KeyPair()
	priv := ethutil.Bytes2Hex(keypair.PrivateKey)
	return priv
}

func (eth *EthModule) fetchKeyPair() *crypto.KeyPair {
	return eth.keyManager.KeyPair()
}

// this is bad but I need it for testing
// TODO: deprecate!
func (eth *EthModule) FetchPriv() string {
	return eth.fetchPriv()
}

// compile LLL file into evm bytecode
// returns hex
func CompileLLL(filename string, literal bool) string {
	/*
		code, err := ethutil.CompileLLL(filename, literal)
		if err != nil {
			fmt.Println("error compiling lll!", err)
			return ""
		}*/
	return "0x" //+ ethutil.Bytes2Hex(code)
}

// some convenience functions

// get users home directory
func homeDir() string {
	usr, _ := user.Current()
	return usr.HomeDir
}

// convert a big int from string to hex
func BigNumStrToHex(s string) string {
	bignum := ethutil.Big(s)
	bignum_bytes := ethutil.BigToBytes(bignum, 16)
	return ethutil.Bytes2Hex(bignum_bytes)
}

// takes a string, converts to bytes, returns hex
func SHA3(tohash string) string {
	h := crypto.Sha3([]byte(tohash))
	return ethutil.Bytes2Hex(h)
}

// pack data into acceptable format for transaction
// TODO: make sure this is ok ...
// TODO: this is in two places, clean it up you putz
func PackTxDataArgs(args ...string) string {
	//fmt.Println("pack data:", args)
	ret := *new([]byte)
	for _, s := range args {
		if s[:2] == "0x" {
			t := s[2:]
			if len(t)%2 == 1 {
				t = "0" + t
			}
			x := ethutil.Hex2Bytes(t)
			//fmt.Println(x)
			l := len(x)
			ret = append(ret, ethutil.LeftPadBytes(x, 32*((l+31)/32))...)
		} else {
			x := []byte(s)
			l := len(x)
			// TODO: just changed from right to left. yabadabadoooooo take care!
			ret = append(ret, ethutil.LeftPadBytes(x, 32*((l+31)/32))...)
		}
	}
	return "0x" + ethutil.Bytes2Hex(ret)
	// return ret
}

// convert ethereum block to types block
func convertBlock(block *ethtypes.Block) *types.Block {
	if block == nil {
		return nil
	}
	b := &types.Block{}
	b.Coinbase = hex.EncodeToString(block.Coinbase())
	b.Difficulty = block.Difficulty().String()
	b.GasLimit = block.GasLimit().String()
	b.GasUsed = block.GasUsed().String()
	b.Hash = hex.EncodeToString(block.Hash())
	//b.MinGasPrice = block.MinGasPrice.String()
	b.Nonce = hex.EncodeToString(block.Nonce())
	b.Number = block.Number().String()
	b.PrevHash = hex.EncodeToString(block.ParentHash())
	b.Time = int(block.Time())
	txs := make([]*types.Transaction, len(block.Transactions()))
	for idx, tx := range block.Transactions() {
		txs[idx] = convertTx(tx)
	}
	b.Transactions = txs
	b.TxRoot = hex.EncodeToString(block.TxHash())
	b.UncleRoot = hex.EncodeToString(block.UncleHash())
	b.Uncles = make([]string, len(block.Uncles()))
	for idx, u := range block.Uncles() {
		b.Uncles[idx] = hex.EncodeToString(u.Hash())
	}
	return b
}

// convert ethereum tx to types tx
func convertTx(ethTx *ethtypes.Transaction) *types.Transaction {
	tx := &types.Transaction{}
	tx.ContractCreation = ethtypes.IsContractAddr(ethTx.To())
	tx.Gas = ethTx.Gas().String()
	tx.GasCost = ethTx.GasPrice().String()
	tx.Hash = hex.EncodeToString(ethTx.Hash())
	tx.Nonce = fmt.Sprintf("%d", ethTx.Nonce)
	tx.Recipient = hex.EncodeToString(ethTx.To())
	tx.Sender = hex.EncodeToString(ethTx.From())
	tx.Value = ethTx.Value().String()
	return tx
}
