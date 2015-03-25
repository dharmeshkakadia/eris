package monkrpc

import (
	"encoding/hex"
	"fmt"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"os/user"
	"strconv"

	mutils "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/modules/monkutils"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/modules/types"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkrpc"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

const ( // Some defaults because we are bad ;)
	VALUE    = "10"
	GAS      = "100000"
	GASPRICE = "100000"
)

// This is a dead simple blockchain module for making rpc calls to an ethereum or thelonious client
// It is mostly designed for use by EPM, and hence will otherwise not be particularly
// functional, as we don't care to well implement everything

var logger *monklog.Logger = monklog.NewLogger("MonkRpc")

// Implements epm.Blockchain
type MonkRpcModule struct {
	Config     *RpcConfig
	client     *rpc.Client
	keyManager *monkcrypto.KeyManager
}

// Create a new rpc module
func NewMonkRpcModule() *MonkRpcModule {
	g := new(MonkRpcModule)
	g.Config = DefaultConfig
	return g
}

// Initialize the module by setting config and key manager
func (mod *MonkRpcModule) Init() error {
	// if didn't call NewMonkRpcModule
	if mod.Config == nil {
		mod.Config = DefaultConfig
	}

	mod.rConfig()

	keyManager := mutils.NewKeyManager(mod.Config.KeyStore, mod.Config.RootDir, monkutil.Config.Db)
	err := keyManager.Init(mod.Config.KeySession, mod.Config.KeyCursor, false)
	if err != nil {
		return err
	}
	mod.keyManager = keyManager

	// need this here becaus we want to be able to get ChainId after Init() without calling Start()
	// TODO: deal with this better somehow?
	rpcAddr := mod.Config.RpcHost + ":" + strconv.Itoa(mod.Config.RpcPort)
	logger.Infoln(rpcAddr)
	client, err := jsonrpc.Dial("tcp", rpcAddr)
	if err != nil {
		fmt.Println("dial http failed: ", err)
		os.Exit(0)
	}
	mod.client = client

	return nil
}

// Connect to rpc server
func (mod *MonkRpcModule) Start() error {
	logger.Infoln("Started")

	return nil
}

func (mod *MonkRpcModule) Shutdown() error {
	return mod.client.Close()
}

func (mod *MonkRpcModule) WaitForShutdown() {

}

func (mod *MonkRpcModule) ChainId() (string, error) {
	args := monkrpc.ChainIdArgs{}
	var res string //monkrpc.ChainIdRes
	err := mod.client.Call("TheloniousApi.ChainId", args, &res)
	if err != nil {
		return "", err
	}
	return res, nil
}

// What module is this?
func (mod *MonkRpcModule) Name() string {
	return "genblock"
}

/*
   Implement Blockchain
*/

// Return the world state
func (mod *MonkRpcModule) WorldState() *types.WorldState {
	return nil
}

func (mod *MonkRpcModule) State() *types.State {
	return nil
}

// Return the entire storage of an address
func (mod *MonkRpcModule) Storage(addr string) *types.Storage {
	return nil
}

// Return the account associated with an address
func (mod *MonkRpcModule) Account(target string) *types.Account {
	return nil
}

// Return a specific storage slot at a contract address
func (mod *MonkRpcModule) StorageAt(contract_addr string, storage_addr string) string {
	args := monkrpc.GetStorageArgs{contract_addr, storage_addr}
	var res monkrpc.GetStorageAtRes
	err := mod.client.Call("TheloniousApi.GetStorageAt", args, &res)
	if err != nil {
		return ""
	}
	return res.Value
}

func (mod *MonkRpcModule) BlockCount() int {
	return -1
}

// Hash of the latest state of the genesis block
func (mod *MonkRpcModule) LatestBlock() string {
	return ""
}

func (mod *MonkRpcModule) Block(hash string) *types.Block {
	args := monkrpc.GetBlockArgs{Hash: hash}
	var res *string
	err := mod.client.Call("TheloniousApi.GetBlock", args, res)
	if err != nil {
		fmt.Println("Err on getblock:", err)
		return nil
	}

	// get block from res (a string?!)

	return nil
}

// Is this account a contract?
func (mod *MonkRpcModule) IsScript(target string) bool {
	// TODO
	return false
}

// Send a transaction to increase an accounts balance.
func (mod *MonkRpcModule) Tx(addr, amt string) (string, error) {
	if mod.Config.Local {
		args := mod.newLocalTx(addr, amt, GAS, GASPRICE, "")
		return mod.rpcLocalTxCall(args)
	}
	// send a signed and serialized tx to a remote server
	keys := mod.keyManager.KeyPair()
	args := mod.newRemoteTx(keys, addr, amt, GAS, GASPRICE, "")
	return mod.rpcRemoteTxCall(args)
}

// Send a message to a contract.
func (mod *MonkRpcModule) Msg(addr string, data []string) (string, error) {
	dataArgs := monkutil.Bytes2Hex(monkutil.PackTxDataArgs(data...))
	if mod.Config.Local {
		args := mod.newLocalTx(addr, VALUE, GAS, GASPRICE, dataArgs)
		return mod.rpcLocalTxCall(args)
	}
	keys := mod.keyManager.KeyPair()
	args := mod.newRemoteTx(keys, addr, VALUE, GAS, GASPRICE, dataArgs)
	return mod.rpcRemoteTxCall(args)
}

// Deploy a new contract.
func (mod *MonkRpcModule) Script(scriptHex string) (string, error) {
	//logger.Debugln("Deploying script: ", file)

	/*var scriptHex string
	if lang == "lll-literal" {
		scriptHex = mutils.Compile(file)
	}
	if lang == "lll" {
		scriptHex = mutils.Compile(file, false) // if lll, compile and pass along
	} else if lang == "mutan" {
		s, _ := ioutil.ReadFile(file) // if mutan, pass along and pipe will compile
		scriptHex = string(s)
	} else if lang == "serpent" {

	} else {
		scriptHex = file
	}*/

	if mod.Config.Local {
		args := mod.newLocalTx("", VALUE, GAS, GASPRICE, scriptHex)
		return mod.rpcLocalCreateCall(args)
	}
	keys := mod.keyManager.KeyPair()
	args := mod.newRemoteTx(keys, "", VALUE, GAS, GASPRICE, scriptHex)
	return mod.rpcRemoteTxCall(args)
}

// There is nothing to subscribe to
func (mod *MonkRpcModule) Subscribe(name, event, target string) chan types.Event {
	return nil
}

// There is nothing to unsubscribe from
func (mod *MonkRpcModule) UnSubscribe(name string) {
}

// Rpc doesn't give us this kind of control
func (m *MonkRpcModule) Commit() {
}

// There is nothing to autocommit over
func (m *MonkRpcModule) AutoCommit(toggle bool) {
}

// There is nothing to autocommit over
func (m *MonkRpcModule) IsAutocommit() bool {
	return false
}

/*
   Blockchain interface should also satisfy KeyManager
   All values are hex encoded
*/

// Return the active address
func (mod *MonkRpcModule) ActiveAddress() string {
	keypair := mod.keyManager.KeyPair()
	addr := monkutil.Bytes2Hex(keypair.Address())
	return addr
}

// Return the nth address in the ring
func (mod *MonkRpcModule) Address(n int) (string, error) {
	ring := mod.keyManager.KeyRing()
	if n >= ring.Len() {
		return "", fmt.Errorf("cursor %d out of range (0..%d)", n, ring.Len())
	}
	pair := ring.GetKeyPair(n)
	addr := monkutil.Bytes2Hex(pair.Address())
	return addr, nil
}

// Set the address
func (mod *MonkRpcModule) SetAddress(addr string) error {
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
func (mod *MonkRpcModule) SetAddressN(n int) error {
	return mod.keyManager.SetCursor(n)
}

// Generate a new address
func (mod *MonkRpcModule) NewAddress(set bool) string {
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
func (mod *MonkRpcModule) AddressCount() int {
	return mod.keyManager.KeyRing().Len()
}

/*
   some key management stuff
*/

func (mod *MonkRpcModule) fetchPriv() string {
	keypair := mod.keyManager.KeyPair()
	priv := monkutil.Bytes2Hex(keypair.PrivateKey)
	return priv
}

func (mod *MonkRpcModule) fetchKeyPair() *monkcrypto.KeyPair {
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
