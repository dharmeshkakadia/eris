package genblock

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"os/user"
	"strconv"

	mutils "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/modules/monkutils"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/modules/types"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkdoug"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

var (
	GoPath         = os.Getenv("GOPATH")
	usr, _         = // error?!
	user.Current()
)

// This is a dead simple blockchain module for deploying genesis blocks from epm
// It does not have real blockchain functionality (no network, chain)
// It is simply for constructing a genesis block from txs, msgs, and contracts
// It currently expects an in-process thelonious to be running (to have setup the db)
// TODO: robustify so it can standalone

//Logging
var logger *monklog.Logger = monklog.NewLogger("GenBlock")

// Implements epm.Blockchain
// strictly for using epm to launch genesis blocks
type GenBlockModule struct {
	Config     *ChainConfig
	block      *monkchain.Block
	keyManager *monkcrypto.KeyManager
}

// Create a new genesis block module
func NewGenBlockModule(block *monkchain.Block) *GenBlockModule {
	g := new(GenBlockModule)
	g.Config = DefaultConfig
	// TODO: if block is nil, get a good one
	if block == nil {
		block = monkchain.NewBlockFromBytes(monkutil.Encode(monkchain.Genesis))
	}
	g.block = block
	return g
}

// Initialize the module by setting config and key manager
func (mod *GenBlockModule) Init() error {
	// if didn't call NewGenBlockModule
	if mod.Config == nil {
		mod.Config = DefaultConfig
	}

	mod.gConfig()

	if monkutil.Config.Db == nil {
		monkutil.Config.Db = mutils.NewDatabase(mod.Config.DbName, false)
	}

	keyManager := mutils.NewKeyManager(mod.Config.KeyStore, mod.Config.RootDir, monkutil.Config.Db)
	err := keyManager.Init(mod.Config.KeySession, mod.Config.KeyCursor, false)
	if err != nil {
		return err
	}
	mod.keyManager = keyManager

	if mod.block == nil {
		mod.block = monkchain.NewBlockFromBytes(monkutil.Encode(monkchain.Genesis))
	}
	return nil
}

// This function does nothing. There are no processes to start
func (mod *GenBlockModule) Start() error {
	return nil
}

// No processes to start, no processes to stop
func (mod *GenBlockModule) Shutdown() error {
	return nil
}

func (mod *GenBlockModule) WaitForShutdown() {

}

func (mod *GenBlockModule) ChainId() (string, error) {
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

// What module is this?
func (mod *GenBlockModule) Name() string {
	return "genblock"
}

/*
   Implement Blockchain
*/

// Return the world state
func (mod *GenBlockModule) WorldState() *types.WorldState {
	state := mod.block.State()
	stateMap := &types.WorldState{make(map[string]*types.Account), []string{}}

	trieIterator := state.Trie.NewIterator()
	trieIterator.Each(func(addr string, acct *monkutil.Value) {
		hexAddr := monkutil.Bytes2Hex([]byte(addr))
		stateMap.Order = append(stateMap.Order, hexAddr)
		stateMap.Accounts[hexAddr] = mod.Account(hexAddr)

	})
	return stateMap
}

func (mod *GenBlockModule) State() *types.State {
	state := mod.block.State()
	stateMap := &types.State{make(map[string]*types.Storage), []string{}}

	trieIterator := state.Trie.NewIterator()
	trieIterator.Each(func(addr string, acct *monkutil.Value) {
		hexAddr := monkutil.Bytes2Hex([]byte(addr))
		stateMap.Order = append(stateMap.Order, hexAddr)
		stateMap.State[hexAddr] = mod.Storage(hexAddr)

	})
	return stateMap
}

// Return the entire storage of an address
func (mod *GenBlockModule) Storage(addr string) *types.Storage {
	state := mod.block.State()
	obj := state.GetOrNewStateObject(monkutil.UserHex2Bytes(addr))
	ret := &types.Storage{make(map[string]string), []string{}}
	obj.EachStorage(func(k string, v *monkutil.Value) {
		kk := monkutil.Bytes2Hex([]byte(k))
		vv := monkutil.Bytes2Hex(v.Bytes())
		ret.Order = append(ret.Order, kk)
		ret.Storage[kk] = vv
	})
	return ret
}

// Return the account associated with an address
func (mod *GenBlockModule) Account(target string) *types.Account {
	state := mod.block.State()
	obj := state.GetOrNewStateObject(monkutil.UserHex2Bytes(target))

	bal := obj.Balance.String()
	nonce := obj.Nonce
	script := monkutil.Bytes2Hex(obj.Code)
	storage := mod.Storage(target)
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

// Return a specific storage slot at a contract address
func (mod *GenBlockModule) StorageAt(contract_addr string, storage_addr string) string {
	var saddr *big.Int
	if monkutil.IsHex(storage_addr) {
		saddr = monkutil.BigD(monkutil.Hex2Bytes(monkutil.StripHex(storage_addr)))
	} else {
		saddr = monkutil.Big(storage_addr)
	}

	contract_addr = monkutil.StripHex(contract_addr)
	caddr := monkutil.Hex2Bytes(contract_addr)
	state := mod.block.State()
	ret := state.GetOrNewStateObject(caddr).GetStorage(saddr)
	if ret.IsNil() {
		return ""
	}
	return monkutil.Bytes2Hex(ret.Bytes())
}

// This is always 0
func (mod *GenBlockModule) BlockCount() int {
	return 0
}

// Hash of the latest state of the genesis block
func (mod *GenBlockModule) LatestBlock() string {
	return monkutil.Bytes2Hex(mod.block.Hash())
}

// Return the genesis block
func (mod *GenBlockModule) Block(hash string) *types.Block {
	return convertBlock(mod.block)
}

// Is this account a contract?
func (mod *GenBlockModule) IsScript(target string) bool {
	// is contract if storage is empty and no bytecode
	obj := mod.Account(target)
	storage := obj.Storage
	if len(storage.Order) == 0 && obj.Script == "" {
		return false
	}
	return true
}

// Send a transaction to increase an accounts balance.
func (mod *GenBlockModule) Tx(addr, amt string) (string, error) {
	account := mod.block.State().GetAccount(monkutil.UserHex2Bytes(addr))
	account.Balance = monkutil.Big(amt)
	mod.block.State().UpdateStateObject(account)

	return addr, nil
}

// Send a message to a contract.
func (mod *GenBlockModule) Msg(addr string, data []string) (string, error) {
	monkdoug.SetValue(monkutil.UserHex2Bytes(addr), data, mod.fetchKeyPair(), mod.block)
	return addr, nil
}

// Deploy a new contract. Note the addresses of core contracts must be stored in gendoug if
// thelonious is expected to find them. Also note the gendoug contract must have `gendoug` in the name!
func (mod *GenBlockModule) Script(code string) (string, error) {
	/*if strings.Contains(file, "gendoug") {
		addr := []byte("0000000000THISISDOUG")
	}*/

	// TODO!!!!
	// need away to find out if this is the genduog contract so we can set the address!!!

	tx, _, err := monkdoug.MakeApplyTx(code, nil, nil, mod.fetchKeyPair(), mod.block)
	if err != nil {
		fmt.Println("script deploy err:", err)
		return "", err
	}
	return monkutil.Bytes2Hex(tx.CreationAddress()), nil
}

// There is nothing to subscribe to
func (mod *GenBlockModule) Subscribe(name, event, target string) chan types.Event {
	return nil
}

// There is nothing to unsubscribe from
func (mod *GenBlockModule) UnSubscribe(name string) {
}

// Commit the current state to the database by syncing the genesis block's trie
func (m *GenBlockModule) Commit() {
	m.block.State().Trie.Sync()
}

// There is nothing to autocommit over
func (m *GenBlockModule) AutoCommit(toggle bool) {
	// TODO: sync after every change?
}

// There is nothing to autocommit over
func (m *GenBlockModule) IsAutocommit() bool {
	return false
}

/*
TODO: should we use this instead?
func (mod *GenBlockModule) ChainId() ([]byte, error) {
	keys, err := mod.selectKeyPair()
	if err != nil {
		return nil, err
	}
	sig := mod.block.Sign(keys.PrivateKey)
	return monkcrypto.Sha3Bin(sig)[:20], nil
}*/

func (mod *GenBlockModule) selectKeyPair() (*monkcrypto.KeyPair, error) {
	var keys *monkcrypto.KeyPair
	var err error
	if mod.Config.Unique {
		if mod.Config.PrivateKey != "" {
			// TODO: some kind of encryption here ...
			decoded := monkutil.Hex2Bytes(mod.Config.PrivateKey)
			keys, err = monkcrypto.NewKeyPairFromSec(decoded)
			if err != nil {
				return nil, fmt.Errorf("Invalid private key", err)
			}
		} else {
			keys = monkcrypto.GenerateNewKeyPair()
		}
	} else {
		static := []byte("11111111112222222222333333333322")
		keys, err = monkcrypto.NewKeyPairFromSec(static)
		if err != nil {
			return nil, fmt.Errorf("Invalid static private", err)
		}
	}
	return keys, nil
}

/*
   Blockchain interface should also satisfy KeyManager
   All values are hex encoded
*/

// Return the active address
func (mod *GenBlockModule) ActiveAddress() string {
	keypair := mod.keyManager.KeyPair()
	addr := monkutil.Bytes2Hex(keypair.Address())
	return addr
}

// Return the nth address in the ring
func (mod *GenBlockModule) Address(n int) (string, error) {
	ring := mod.keyManager.KeyRing()
	if n >= ring.Len() {
		return "", fmt.Errorf("cursor %d out of range (0..%d)", n, ring.Len())
	}
	pair := ring.GetKeyPair(n)
	addr := monkutil.Bytes2Hex(pair.Address())
	return addr, nil
}

// Set the address
func (mod *GenBlockModule) SetAddress(addr string) error {
	n := -1
	i := 0
	ring := mod.keyManager.KeyRing()
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
	return mod.SetAddressN(n)
}

// Set the address to be the nth in the ring
func (mod *GenBlockModule) SetAddressN(n int) error {
	return mod.keyManager.SetCursor(n)
}

// Generate a new address
func (mod *GenBlockModule) NewAddress(set bool) string {
	newpair := monkcrypto.GenerateNewKeyPair()
	addr := monkutil.Bytes2Hex(newpair.Address())
	ring := mod.keyManager.KeyRing()
	ring.AddKeyPair(newpair)
	if set {
		mod.SetAddressN(ring.Len() - 1)
	}
	return addr
}

// Return the number of available addresses
func (mod *GenBlockModule) AddressCount() int {
	return mod.keyManager.KeyRing().Len()
}

/*
   some key management stuff
*/

func (mod *GenBlockModule) fetchPriv() string {
	keypair := mod.keyManager.KeyPair()
	priv := monkutil.Bytes2Hex(keypair.PrivateKey)
	return priv
}

func (mod *GenBlockModule) fetchKeyPair() *monkcrypto.KeyPair {
	return mod.keyManager.KeyPair()
}

// some convenience functions

// get users home directory
func homeDir() string {
	usr, _ := user.Current()
	return usr.HomeDir
}

// convert thelonious block to types block
func convertBlock(block *monkchain.Block) *types.Block {
	if block == nil {
		return nil
	}
	b := &types.Block{}
	b.Coinbase = hex.EncodeToString(block.Coinbase)
	b.Difficulty = block.Difficulty.String()
	b.GasLimit = block.GasLimit.String()
	b.GasUsed = block.GasUsed.String()
	b.Hash = hex.EncodeToString(block.Hash())
	b.MinGasPrice = block.MinGasPrice.String()
	b.Nonce = hex.EncodeToString(block.Nonce)
	b.Number = block.Number.String()
	b.PrevHash = hex.EncodeToString(block.PrevHash)
	b.Time = int(block.Time)
	txs := make([]*types.Transaction, len(block.Transactions()))
	for idx, tx := range block.Transactions() {
		txs[idx] = convertTx(tx)
	}
	b.Transactions = txs
	b.TxRoot = hex.EncodeToString(block.TxSha)
	b.UncleRoot = hex.EncodeToString(block.UncleSha)
	b.Uncles = make([]string, len(block.Uncles))
	for idx, u := range block.Uncles {
		b.Uncles[idx] = hex.EncodeToString(u.Hash())
	}
	return b
}

// convert thelonious tx to types tx
func convertTx(monkTx *monkchain.Transaction) *types.Transaction {
	tx := &types.Transaction{}
	tx.ContractCreation = monkTx.CreatesContract()
	tx.Gas = monkTx.Gas.String()
	tx.GasCost = monkTx.GasPrice.String()
	tx.Hash = hex.EncodeToString(monkTx.Hash())
	tx.Nonce = fmt.Sprintf("%d", monkTx.Nonce)
	tx.Recipient = hex.EncodeToString(monkTx.Recipient)
	tx.Sender = hex.EncodeToString(monkTx.Sender())
	tx.Value = monkTx.Value.String()
	return tx
}
