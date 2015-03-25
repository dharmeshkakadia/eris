package monk

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"path"
	"strconv"
	"time"

	mutils "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/modules/monkutils"
	types "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/modules/types"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkdoug"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkpipe"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkreact"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
	chains "github.com/eris-ltd/epm-go/chains"
	utils "github.com/eris-ltd/epm-go/utils" //Logging
)

var logger *monklog.Logger = monklog.NewLogger("MONK")

func init() {
	utils.InitDecerverDir()
}

// implements epm.Blockchain
type MonkModule struct {
	monk          *Monk
	Config        *ChainConfig
	GenesisConfig *monkdoug.GenesisConfig
}

// implements decerver-interfaces Blockchain
// this will get passed to Otto (javascript vm)
// as such, it does not have "administrative" methods
type Monk struct {
	config     *ChainConfig
	genConfig  *monkdoug.GenesisConfig
	thelonious *thelonious.Thelonious
	pipe       *monkpipe.Pipe
	keyManager *monkcrypto.KeyManager
	reactor    *monkreact.ReactorEngine
	started    bool
	db         monkutil.Database

	chans map[string]Chan
}

type Chan struct {
	ch      chan types.Event
	reactCh chan monkreact.Event
	name    string
	event   string
	target  string
}

/*
   First, the functions to satisfy Module
*/

// Create a new MonkModule and internal Monk, with default config.
// Accepts a thelonious instance to yield a new
// interface into the same chain.
// It will not initialize the thelonious object for you though,
// so you can adjust configs before calling `Init()`
func NewMonk(th *thelonious.Thelonious) *MonkModule {
	mm := new(MonkModule)
	m := new(Monk)
	// Here we load default config and leave it to caller
	// to overwrite with config file or directly
	mm.Config = DefaultConfig
	m.config = mm.Config
	if th != nil {
		m.thelonious = th
	}
	m.started = false
	mm.monk = m
	return mm
}

// Configure the GenesisConfig struct
// If the chain already exists, use the provided genesis config
// TODO: move genconfig into db (safer than a config file)
//          but really we should reconstruct it from the genesis block
func (mod *MonkModule) ConfigureGenesis() {
	// first check if this chain already exists (and load genesis config from there)
	// (only if not working from a mem db)
	if !mod.Config.DbMem {
		if _, err := os.Stat(mod.Config.RootDir); err == nil {
			p := path.Join(mod.Config.RootDir, "genesis.json")
			if _, err = os.Stat(p); err == nil {
				mod.Config.GenesisConfig = p
			} else {
				//			exit(fmt.Errorf("Blockchain exists but missing genesis.json!"))
				utils.Copy(DefaultGenesisConfig, path.Join(mod.Config.RootDir, "genesis.json"))
			}
		}
	}

	// setup genesis config and genesis deploy handler
	if mod.GenesisConfig == nil {
		// fails if can't read json
		mod.GenesisConfig = mod.LoadGenesis(mod.Config.GenesisConfig)
	}
	if mod.GenesisConfig.Pdx != "" && !mod.GenesisConfig.NoGenDoug {
		// epm deploy through a pdx file
		mod.GenesisConfig.SetDeployer(func(block *monkchain.Block) ([]byte, error) {
			// TODO: get full path
			return epmDeploy(block, mod.GenesisConfig.Pdx)
		})
	}
	mod.monk.genConfig = mod.GenesisConfig
}

// Initialize a monkchain
// It may or may not already have a thelonious instance
// Gives you a pipe, local keyMang, and reactor
// NewMonk must have been called first
func (mod *MonkModule) Init() error {
	m := mod.monk

	if m == nil {
		return fmt.Errorf("NewMonk has not been called")
	}

	// set epm contract path
	setEpmContractPath(m.config.ContractPath)
	// set the root
	// name > chainId > rootDir > default
	mod.setRootDir()
	logger.Infoln("Root directory ", mod.Config.RootDir)
	mod.ConfigureGenesis()
	logger.Infoln("Loaded genesis configuration from: ", mod.Config.GenesisConfig)

	if !m.config.UseCheckpoint {
		m.config.LatestCheckpoint = ""
	}

	monkdoug.Adversary = mod.Config.Adversary

	// if no thelonious instance
	if m.thelonious == nil {
		mod.thConfig()
		m.newThelonious()
	}

	m.pipe = monkpipe.New(m.thelonious)
	m.keyManager = m.thelonious.KeyManager()
	m.reactor = m.thelonious.Reactor()

	// subscribe to the new block
	m.chans = make(map[string]Chan)

	return nil
}

// Start the thelonious node
func (mod *MonkModule) Start() (err error) {
	startChan := mod.Subscribe("chainReady", "chainReady", "")

	m := mod.monk
	seed := ""
	if mod.Config.UseSeed {
		seed = m.config.RemoteHost + ":" + strconv.Itoa(m.config.RemotePort)
	}
	m.thelonious.Start(mod.Config.Listen, seed)
	RegisterInterrupt(func(sig os.Signal) {
		m.thelonious.Stop()
		monklog.Flush()
	})
	m.started = true

	if m.config.Mining {
		StartMining(m.thelonious)
	}

	if m.config.ServeRpc {
		StartRpc(m.thelonious, m.config.RpcHost, m.config.RpcPort)
	}

	m.Subscribe("newBlock", "newBlock", "")

	// wait for startup to finish
	// XXX: note for checkpoints this means waiting until
	//  the entire checkpointed state is loaded from peers...
	<-startChan
	mod.UnSubscribe("chainReady")

	return nil
}

func (mod *MonkModule) Shutdown() error {
	mod.monk.Stop()
	return nil
}

func (mod *MonkModule) ChainId() (string, error) {
	return mod.monk.ChainId()
}

func (mod *MonkModule) WaitForShutdown() {
	mod.monk.thelonious.WaitForShutdown()
}

// ReadConfig and WriteConfig implemented in config.go

// What module is this?
func (mod *MonkModule) Name() string {
	return "monk"
}

/*
   Wrapper so module satisfies Blockchain
*/

func (mod *MonkModule) WorldState() *types.WorldState {
	return mod.monk.WorldState()
}

func (mod *MonkModule) State() *types.State {
	return mod.monk.State()
}

func (mod *MonkModule) Storage(target string) *types.Storage {
	return mod.monk.Storage(target)
}

func (mod *MonkModule) Account(target string) *types.Account {
	return mod.monk.Account(target)
}

func (mod *MonkModule) StorageAt(target, storage string) string {
	return mod.monk.StorageAt(target, storage)
}

func (mod *MonkModule) BlockCount() int {
	return mod.monk.BlockCount()
}

func (mod *MonkModule) LatestBlock() string {
	return mod.monk.LatestBlock()
}

func (mod *MonkModule) Block(hash string) *types.Block {
	return mod.monk.Block(hash)
}

func (mod *MonkModule) IsScript(target string) bool {
	return mod.monk.IsScript(target)
}

func (mod *MonkModule) Tx(addr, amt string) (string, error) {
	return mod.monk.Tx(addr, amt)
}

func (mod *MonkModule) Msg(addr string, data []string) (string, error) {
	return mod.monk.Msg(addr, data)
}

func (mod *MonkModule) Script(code string) (string, error) {
	return mod.monk.Script(code)
}

func (mod *MonkModule) Transact(addr, value, gas, gasprice, data string) (string, error) {
	return mod.monk.Transact(addr, value, gas, gasprice, data)
}

func (mod *MonkModule) Subscribe(name, event, target string) chan types.Event {
	return mod.monk.Subscribe(name, event, target)
}

func (mod *MonkModule) UnSubscribe(name string) {
	mod.monk.UnSubscribe(name)
}

func (mod *MonkModule) Commit() {
	mod.monk.Commit()
}

func (mod *MonkModule) AutoCommit(toggle bool) {
	mod.monk.AutoCommit(toggle)
}

func (mod *MonkModule) IsAutocommit() bool {
	return mod.monk.IsAutocommit()
}

/*
   Module should also satisfy KeyManager
*/

func (mod *MonkModule) ActiveAddress() string {
	return mod.monk.ActiveAddress()
}

func (mod *MonkModule) Address(n int) (string, error) {
	return mod.monk.Address(n)
}

func (mod *MonkModule) SetAddress(addr string) error {
	return mod.monk.SetAddress(addr)
}

func (mod *MonkModule) SetAddressN(n int) error {
	return mod.monk.SetAddressN(n)
}

func (mod *MonkModule) NewAddress(set bool) string {
	return mod.monk.NewAddress(set)
}

func (mod *MonkModule) AddressCount() int {
	return mod.monk.AddressCount()
}

/*
   Module should satisfy a P2P interface
   Not in decerver-interfaces yet but prototyping here
*/

func (mod *MonkModule) Listen(should bool) {
	mod.monk.Listen(should)
}

/*
   Non-interface functions that otherwise prove useful
    in standalone applications, testing, and debuging
*/

// Load genesis json file (so calling pkg need not import monkdoug)
func (mod *MonkModule) LoadGenesis(file string) *monkdoug.GenesisConfig {
	g := monkdoug.LoadGenesis(file)
	return g
}

// Set the genesis json object. This can only be done once
func (mod *MonkModule) SetGenesis(genJson *monkdoug.GenesisConfig) {
	// reset the permission model struct (since config may have changed)
	//genJson.SetModel(monkdoug.NewPermModel(genJson))
	mod.GenesisConfig = genJson
}

func (mod *MonkModule) MonkState() *monkstate.State {
	return mod.monk.pipe.World().State()
}

/*
   Implement Blockchain
*/

func (monk *Monk) ChainId() (string, error) {
	// get the chain id
	data, err := monkutil.Config.Db.Get([]byte("ChainID"))
	if err != nil {
		return "", err
	} else if len(data) == 0 {
		return "", fmt.Errorf("ChainID is empty!")
	}
	chainId := monkutil.Bytes2Hex(data)
	return chainId, nil
}

func (monk *Monk) WorldState() *types.WorldState {
	state := monk.pipe.World().State()
	stateMap := &types.WorldState{make(map[string]*types.Account), []string{}}

	trieIterator := state.Trie.NewIterator()
	trieIterator.Each(func(addr string, acct *monkutil.Value) {
		hexAddr := monkutil.Bytes2Hex([]byte(addr))
		stateMap.Order = append(stateMap.Order, hexAddr)
		stateMap.Accounts[hexAddr] = monk.Account(hexAddr)

	})
	return stateMap
}

func (monk *Monk) State() *types.State {
	state := monk.pipe.World().State()
	stateMap := &types.State{make(map[string]*types.Storage), []string{}}

	trieIterator := state.Trie.NewIterator()
	trieIterator.Each(func(addr string, acct *monkutil.Value) {
		hexAddr := monkutil.Bytes2Hex([]byte(addr))
		stateMap.Order = append(stateMap.Order, hexAddr)
		stateMap.State[hexAddr] = monk.Storage(hexAddr)

	})
	return stateMap
}

func (monk *Monk) Storage(addr string) *types.Storage {
	w := monk.pipe.World()
	obj := w.SafeGet(monkutil.UserHex2Bytes(addr)).StateObject
	ret := &types.Storage{make(map[string]string), []string{}}
	obj.EachStorage(func(k string, v *monkutil.Value) {
		kk := monkutil.Bytes2Hex([]byte(k))
		v.Decode()
		vv := monkutil.Bytes2Hex(v.Bytes())
		ret.Order = append(ret.Order, kk)
		ret.Storage[kk] = vv
	})
	return ret
}

func (monk *Monk) Account(target string) *types.Account {
	w := monk.pipe.World()
	obj := w.SafeGet(monkutil.UserHex2Bytes(target)).StateObject

	bal := obj.Balance.String()
	nonce := obj.Nonce
	script := monkutil.Bytes2Hex(obj.Code)
	storage := monk.Storage(target)
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

func (monk *Monk) StorageAt(contract_addr string, storage_addr string) string {
	var saddr *big.Int
	if monkutil.IsHex(storage_addr) {
		saddr = monkutil.BigD(monkutil.Hex2Bytes(monkutil.StripHex(storage_addr)))
	} else {
		saddr = monkutil.Big(storage_addr)
	}

	contract_addr = monkutil.StripHex(contract_addr)
	caddr := monkutil.Hex2Bytes(contract_addr)
	w := monk.pipe.World()
	ret := w.SafeGet(caddr).GetStorage(saddr)
	if ret.IsNil() {
		return ""
	}
	return monkutil.Bytes2Hex(ret.Bytes())
}

func (monk *Monk) BlockCount() int {
	return int(monk.thelonious.ChainManager().CurrentBlockNumber())
}

func (monk *Monk) LatestBlock() string {
	return monkutil.Bytes2Hex(monk.thelonious.ChainManager().CurrentBlockHash())
}

func (monk *Monk) Block(hash string) *types.Block {
	hashBytes := monkutil.Hex2Bytes(hash)
	block := monk.thelonious.ChainManager().GetBlock(hashBytes)
	return convertBlock(block)
}

func (monk *Monk) IsScript(target string) bool {
	// is contract if storage is empty and no bytecode
	obj := monk.Account(target)
	storage := obj.Storage
	if len(storage.Order) == 0 && obj.Script == "" {
		return false
	}
	return true
}

// send a tx
func (monk *Monk) Tx(addr, amt string) (string, error) {
	keys := monk.fetchKeyPair()
	addr = monkutil.StripHex(addr)
	if addr[:2] == "0x" {
		addr = addr[2:]
	}
	byte_addr := monkutil.Hex2Bytes(addr)
	// note, NewValue will not turn a string int into a big int..
	//start := time.Now()
	//hash, err := monk.pipe.Transact(keys, byte_addr, monkutil.NewValue(monkutil.Big(amt)), monkutil.NewValue(monkutil.Big("20000000000")), monkutil.NewValue(monkutil.Big("100000")), "")
	hash, err := monk.pipe.Transact(keys, byte_addr, monkutil.NewValue(monkutil.Big(amt)), monkutil.NewValue(monkutil.Big("200000000000000")), monkutil.NewValue(monkutil.Big("0")), "")
	//dif := time.Since(start)
	//fmt.Println("pipe tx took ", dif)
	if err != nil {
		return "", err
	}
	return monkutil.Bytes2Hex(hash), nil
}

func (monk *MonkModule) Reactor() bool {
	return monk.monk.reactor.Running()
}

// send a message to a contract
func (monk *Monk) Msg(addr string, data []string) (string, error) {
	packed := PackTxDataArgs(data...)
	keys := monk.fetchKeyPair()
	addr = monkutil.StripHex(addr)
	byte_addr := monkutil.Hex2Bytes(addr)
	hash, err := monk.pipe.Transact(keys, byte_addr, monkutil.NewValue(monkutil.Big("0")), monkutil.NewValue(monkutil.Big("200000000000000")), monkutil.NewValue(monkutil.Big("0")), packed)
	if err != nil {
		return "", err
	}
	return monkutil.Bytes2Hex(hash), nil
}

func (monk *Monk) Script(code string) (string, error) {
	code = monkutil.StripHex(code)

	keys := monk.fetchKeyPair()

	// well isn't this pretty! barf
	contract_addr, err := monk.pipe.Transact(keys, nil, monkutil.NewValue(monkutil.Big("0")), monkutil.NewValue(monkutil.Big("200000000000000")), monkutil.NewValue(monkutil.Big("0")), code)
	if err != nil {
		return "", err
	}
	return monkutil.Bytes2Hex(contract_addr), nil
}

func (monk *Monk) Transact(addr, amt, gas, gasprice, data string) (string, error) {
	keys := monk.fetchKeyPair()
	addr = monkutil.StripHex(addr)
	byte_addr := monkutil.Hex2Bytes(addr)
	hash, err := monk.pipe.Transact(keys, byte_addr, monkutil.NewValue(monkutil.Big(amt)), monkutil.NewValue(monkutil.Big(gas)), monkutil.NewValue(monkutil.Big(gasprice)), data)
	if err != nil {
		return "", err
	}
	return monkutil.Bytes2Hex(hash), nil
}

// returns a chanel that will fire when address is updated
func (monk *Monk) Subscribe(name, event, target string) chan types.Event {
	th_ch := make(chan monkreact.Event, 1)
	if target != "" {
		addr := string(monkutil.Hex2Bytes(target))
		monk.reactor.Subscribe("object:"+addr, th_ch)
	} else {
		monk.reactor.Subscribe(event, th_ch)
	}

	ch := make(chan types.Event)
	c := Chan{
		ch:      ch,
		reactCh: th_ch,
		name:    name,
		event:   event,
		target:  target,
	}
	monk.chans[name] = c
	//monk.chans[name] = ch
	//monk.reactchans[name] = th_ch

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
				Source:    "monk",
				TimeStamp: time.Now(),
			}
			// cast resource to appropriate type
			resource := eve.Resource
			if block, ok := resource.(*monkchain.Block); ok {
				returnEvent.Resource = convertBlock(block)
			} else if tx, ok := resource.(*monkchain.Transaction); ok {
				returnEvent.Resource = convertTx(tx)
			} else if txFail, ok := resource.(*monkchain.TxFail); ok {
				tx := convertTx(txFail.Tx)
				tx.Error = txFail.Err.Error()
				returnEvent.Resource = tx
			} else if s, ok := resource.(string); ok {
				returnEvent.Resource = s
			} else {
				logger.Errorln("Invalid event resource type", resource)
			}
			ch <- returnEvent
		}
	}()
	return ch
}

func (monk *Monk) UnSubscribe(name string) {
	if c, ok := monk.chans[name]; ok {
		monk.reactor.Unsubscribe(c.event, c.reactCh)
		// drain channels
		select {
		case <-c.reactCh:
		default:
		}
		close(c.reactCh)

		select {
		case <-c.ch:
		default:
		}
		close(c.ch)
		delete(monk.chans, name)
	}
}

// Mine a block
func (m *Monk) Commit() {
	m.StartMining()
	_ = <-m.chans["newBlock"].ch
	v := false
	for !v {
		v = m.StopMining()
	}
	select {
	case _ = <-m.chans["newBlock"].ch:
	default:
	}
}

// start and stop continuous mining
func (m *Monk) AutoCommit(toggle bool) {
	if toggle {
		m.StartMining()
	} else {
		m.StopMining()
	}
}

func (m *Monk) IsAutocommit() bool {
	return m.thelonious.IsMining()
}

/*
   Blockchain interface should also satisfy KeyManager
   All values are hex encoded
*/

// Return the active address
func (monk *Monk) ActiveAddress() string {
	keypair := monk.keyManager.KeyPair()
	addr := monkutil.Bytes2Hex(keypair.Address())
	return addr
}

// Return the nth address in the ring
func (monk *Monk) Address(n int) (string, error) {
	ring := monk.keyManager.KeyRing()
	if n >= ring.Len() {
		return "", fmt.Errorf("cursor %d out of range (0..%d)", n, ring.Len())
	}
	pair := ring.GetKeyPair(n)
	addr := monkutil.Bytes2Hex(pair.Address())
	return addr, nil
}

// Set the address
func (monk *Monk) SetAddress(addr string) error {
	n := -1
	i := 0
	ring := monk.keyManager.KeyRing()
	ring.Each(func(kp *monkcrypto.KeyPair) {
		a := monkutil.Bytes2Hex(kp.Address())
		if a == addr {
			n = i
		}
		i += 1
	})
	if n == -1 {
		return fmt.Errorf("Address %s not found in keyring", addr)
	}
	return monk.SetAddressN(n)
}

// Set the address to be the nth in the ring
func (monk *Monk) SetAddressN(n int) error {
	return monk.keyManager.SetCursor(n)
}

// Generate a new address
func (monk *Monk) NewAddress(set bool) string {
	newpair := monkcrypto.GenerateNewKeyPair()
	addr := monkutil.Bytes2Hex(newpair.Address())
	ring := monk.keyManager.KeyRing()
	ring.AddKeyPair(newpair)
	if set {
		monk.SetAddressN(ring.Len() - 1)
	}
	return addr
}

// Return the number of available addresses
func (monk *Monk) AddressCount() int {
	return monk.keyManager.KeyRing().Len()
}

/*
   P2P interface
*/

// Start and stop listening on the port
func (monk *Monk) Listen(should bool) {
	if should {
		monk.StartListening()
	} else {
		monk.StopListening()
	}
}

/*
   Helper functions
*/

// create a new thelonious instance
// expects thConfig to already have been called!
// init db, nat/upnp, thelonious struct, reactorEngine, txPool, blockChain, stateManager
func (m *Monk) newThelonious() {
	db := mutils.NewDatabase(m.config.DbName, m.config.DbMem)
	m.db = db

	keyManager := mutils.NewKeyManager(m.config.KeyStore, m.config.RootDir, db)
	err := keyManager.Init(m.config.KeySession, m.config.KeyCursor, false)
	if err != nil {
		log.Fatal(err)
	}
	m.keyManager = keyManager

	clientIdentity := mutils.NewClientIdentity(m.config.ClientIdentifier, m.config.Version, m.config.Identifier)
	logger.Infoln("Identity created")

	checkpoint := monkutil.UserHex2Bytes(m.config.LatestCheckpoint)

	// create the thelonious obj
	th, err := thelonious.New(db, clientIdentity, m.keyManager, thelonious.CapDefault, false, m.config.FetchPort, checkpoint, m.genConfig)

	if err != nil {
		log.Fatal("Could not start node: %s\n", err)
	}

	logger.Infoln("Created thelonious node")

	th.Port = strconv.Itoa(m.config.ListenPort)
	th.MaxPeers = m.config.MaxPeers

	m.thelonious = th
}

// returns hex addr of gendoug
/*
func (monk *Monk) GenDoug() string {
	return monkutil.Bytes2Hex(monkdoug.GenDougByteAddr)
}*/

func (monk *Monk) StartMining() bool {
	return StartMining(monk.thelonious)
}

func (monk *Monk) StopMining() bool {
	return StopMining(monk.thelonious)
}

func (monk *Monk) StartListening() {
	monk.thelonious.StartListening()
}

func (monk *Monk) StopListening() {
	monk.thelonious.StopListening()
}

/*
   some key management stuff
*/

func (monk *Monk) fetchPriv() string {
	keypair := monk.keyManager.KeyPair()
	priv := monkutil.Bytes2Hex(keypair.PrivateKey)
	return priv
}

func (monk *Monk) fetchKeyPair() *monkcrypto.KeyPair {
	return monk.keyManager.KeyPair()
}

// this is bad but I need it for testing
// TODO: deprecate!
func (monk *Monk) FetchPriv() string {
	return monk.fetchPriv()
}

func (mod *MonkModule) Restart() error {
	if err := mod.Shutdown(); err != nil {
		return err
	}

	mk := mod.monk
	mod = NewMonk(nil)
	mod.monk = mk
	mod.Config = mk.config

	if err := mod.Init(); err != nil {
		return err
	}
	if err := mod.Start(); err != nil {
		return err
	}

	return nil

}

func (monk *Monk) Stop() {
	if !monk.started {
		logger.Infoln("can't stop: haven't even started...")
		if monk.db != nil {
			monk.db.Close()
		}
		return
	}
	monk.StopMining()
	monk.thelonious.Stop()
	time.Sleep(time.Second)
	for n, _ := range monk.chans {
		monk.UnSubscribe(n)
	}
	monk = &Monk{config: monk.config}
	monklog.Reset()
	monk.started = false
	logger.Warnln("Shutdown monk")
}

// Set the root. If it's already set, check if the
func (mod *MonkModule) setRootDir() {
	c := mod.Config
	// if RootDir is set, we're done
	if c.RootDir != "" {
		/*
			if _, err := os.Stat(path.Join(c.RootDir, "config.json")); err == nil {
				mod.ReadConfig(path.Join(c.RootDir, "config.json"))
			}*/
		return
	}

	root, _ := chains.ResolveChainDir("thelonious", c.ChainName, c.ChainId)
	if root == "" {
		c.RootDir = DefaultRoot
	} else {
		c.RootDir = root
	}
}
