package mint

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/logger"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/modules/types"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/confer"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/account"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/common"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/config"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/consensus"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/events"
	tmlog "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/logger"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/mempool"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/node"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/p2p"
	rpccore "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/rpc/core"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/state"
	blk "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/types"
	"github.com/eris-ltd/epm-go/utils"
)

var (
	GASLIMIT = uint64(1000000)
	FEE      = uint64(5000)
)

//Logging
var mintlogger *logger.Logger = logger.NewLogger("MintLogger")

// implements decerver-interfaces Blockchain
// this will get passed to Otto (javascript vm)
// as such, it does not have "administrative" methods
type MintModule struct {
	Config         *ChainConfig
	ConsensusState *consensus.ConsensusState
	MempoolReactor *mempool.MempoolReactor
	//	State      *state.State
	App     *confer.Config
	started bool

	node     *node.Node
	listener p2p.Listener
	evsw     *events.EventSwitch

	priv *account.PrivAccount
}

/*
   First, the functions to satisfy Module
*/

func NewMint() *MintModule {
	m := new(MintModule)
	m.Config = DefaultConfig
	m.started = false
	return m
}

// initialize an chain
func (mod *MintModule) Init() error {
	// config should be loaded by epm

	// transform epm json based config to tendermint config
	Config2Config(mod.Config)

	tmlog.Reset()

	keySession := mod.Property("KeySession").(string)
	keyFile := mod.Property("KeyFile").(string)
	rootDir := mod.Property("RootDir").(string)
	prv, err := loadOrCreateKey(keySession, keyFile, rootDir)
	if err != nil {
		return err
	}
	mod.priv = prv
	if err := writeValidatorFile(rootDir, mod.priv); err != nil {
		return err
	}

	// Create & start node
	n := node.NewNode()
	l := p2p.NewDefaultListener("tcp", config.App().GetString("ListenAddr"), false)
	n.AddListener(l)

	mod.listener = l
	mod.node = n
	mod.App = config.App()
	mod.ConsensusState = n.ConsensusState()
	mod.MempoolReactor = n.MempoolReactor()
	mod.evsw = n.EventSwitch()

	return nil
}

// start the tendermint node
func (mod *MintModule) Start() error {
	mod.node.Start()

	// If seedNode is provided by config, dial out.
	if config.App().GetString("SeedNode") != "" {
		mod.node.DialSeed()
	}

	// Run the RPC server.
	if config.App().GetString("RPC.HTTP.ListenAddr") != "" {
		mod.node.StartRPC()
	}

	mod.started = true

	return nil
}

func (mod *MintModule) Shutdown() error {
	mod.node.Stop()
	return nil
}

func (mod *MintModule) WaitForShutdown() {
	// Sleep forever and then...
	trapSignal(func() {
		mod.node.Stop()
	})
}

// ReadConfig and WriteConfig implemented in config.go

// What module is this?
func (mod *MintModule) Name() string {
	return "tendermint"
}

/*
 *  Implement Blockchain
 */

func (mint *MintModule) ChainId() (string, error) {
	// TODO: genhash + network
	return "TODO" + mint.App.GetString("Network"), nil
}

func (mint *MintModule) WorldState() *types.WorldState {
	stateMap := &types.WorldState{make(map[string]*types.Account), []string{}}
	state := mint.ConsensusState.GetState()
	//blockHeight = state.LastBlockHeight
	state.GetAccounts().Iterate(func(key interface{}, value interface{}) bool {
		acc := value.(*account.Account)
		hexAddr := hex.EncodeToString(acc.Address)
		stateMap.Order = append(stateMap.Order, hexAddr)
		accTy := &types.Account{
			Address: hexAddr,
			Balance: strconv.Itoa(int(acc.Balance)),
			Nonce:   strconv.Itoa(int(acc.Sequence)),
			// TODO:
			//Script:   script,
			//Storage:  storage,
			//IsScript: isscript,
		}
		stateMap.Accounts[hexAddr] = accTy
		return false
	})
	return stateMap
}

// tendermint/tendermint/merkel/iavl_node.go
// traverse()
func (mint *MintModule) State() *types.State {
	return nil
}

func (mint *MintModule) Storage(addr string) *types.Storage {
	addr = utils.StripHex(addr)
	addrBytes, err := hex.DecodeString(addr)
	if err != nil {
		return nil
	}
	acc := mint.ConsensusState.GetState().GetAccount(addrBytes)
	_ = acc
	// TODO: iterate and grab storage
	return nil
}

func (mint *MintModule) Account(target string) *types.Account {
	target = utils.StripHex(target)
	targetBytes, err := hex.DecodeString(target)
	if err != nil {
		return nil
	}
	fmt.Println("TARGET:", target, targetBytes)
	acc := mint.ConsensusState.GetState().GetAccount(targetBytes)
	return &types.Account{
		Address: target,
		Balance: strconv.Itoa(int(acc.Balance)),
		Nonce:   strconv.Itoa(int(acc.Sequence)),
		// TODO:
		//Script:   script,
		//Storage:  storage,
		//IsScript: isscript,
	}
}

func (mint *MintModule) StorageAt(contract_addr string, storage_addr string) string {
	contract_addr = utils.StripHex(contract_addr)
	storage_addr = utils.StripHex(storage_addr)
	fmt.Println("STORAGE AT:", contract_addr, storage_addr)
	caddr, err := hex.DecodeString(contract_addr)
	if err != nil {
		return ""
	}
	saddr, err := hex.DecodeString(storage_addr)
	if err != nil {
		return ""
	}
	cache := mint.MempoolReactor.Mempool.GetCache()
	// block cache will not have account available to return
	// something meaningful from GetStorage unless we
	// call GetAccount first
	acc := cache.GetAccount(caddr)
	fmt.Println("ACC:", acc)
	b := cache.GetStorage(common.LeftPadWord256(caddr), common.RightPadWord256(flip(saddr)))
	fmt.Printf("CADDR, SADDR, b: %x, %x, %x", caddr, common.RightPadWord256(flip(saddr)), b)
	return hex.EncodeToString(flip(b.Bytes()))
}

func (mint *MintModule) BlockCount() int {
	return int(mint.ConsensusState.GetState().LastBlockHeight)
}

func (mint *MintModule) LatestBlock() string {
	return hex.EncodeToString(mint.ConsensusState.GetState().LastBlockHash)
}

func (mint *MintModule) Block(hash string) *types.Block {
	// TODO
	return nil
}

func (mint *MintModule) IsScript(target string) bool {
	// TODO
	return false
}

// TODO: move these to use rpccore!

// send a tx
func (mint *MintModule) Tx(addr, amt string) (string, error) {
	addr = utils.StripHex(addr)
	addrB, err := hex.DecodeString(addr)
	if err != nil {
		return "", err
	}
	acc := mint.MempoolReactor.Mempool.GetCache().GetAccount(mint.priv.Address)
	nonce := 0
	if acc != nil {
		nonce = int(acc.Sequence) + 1
	}

	amtInt, err := strconv.Atoi(amt)
	if err != nil {
		return "", err
	}
	amtUint64 := uint64(amtInt)

	tx := &blk.SendTx{
		Inputs: []*blk.TxInput{
			&blk.TxInput{
				Address:   mint.priv.Address,
				Amount:    amtUint64,
				Sequence:  uint(nonce),
				Signature: account.SignatureEd25519{},
				PubKey:    mint.priv.PubKey,
			},
		},
		Outputs: []*blk.TxOutput{
			&blk.TxOutput{
				Address: addrB,
				Amount:  amtUint64,
			},
		},
	}
	tx.Inputs[0].Signature = mint.priv.PrivKey.Sign(account.SignBytes(tx))
	err = mint.MempoolReactor.BroadcastTx(tx)
	return hex.EncodeToString(account.HashSignBytes(tx)), err
	return "", err
}

// send a message to a contract
// data is prepacked by epm
func (mint *MintModule) Msg(addr string, data []string) (string, error) {
	packed := data[0]
	packedBytes, _ := hex.DecodeString(packed)
	addr = utils.StripHex(addr)
	addrB, err := hex.DecodeString(addr)
	if err != nil {
		return "", err
	}
	acc := mint.MempoolReactor.Mempool.GetCache().GetAccount(mint.priv.Address)
	nonce := 0
	if acc != nil {
		nonce = int(acc.Sequence) + 1
	}

	tx := &blk.CallTx{
		Input: &blk.TxInput{
			Address:   mint.priv.Address,
			Amount:    FEE,
			Sequence:  uint(nonce),
			Signature: account.SignatureEd25519{},
			PubKey:    mint.priv.PubKey,
		},
		Address:  addrB,
		GasLimit: GASLIMIT,
		Fee:      FEE,
		Data:     packedBytes,
	}
	tx.Input.Signature = mint.priv.PrivKey.Sign(account.SignBytes(tx))
	err = mint.MempoolReactor.BroadcastTx(tx)
	return hex.EncodeToString(account.HashSignBytes(tx)), err
}

// send a simulated message to a contract
// data is prepacked by epm
func (mint *MintModule) Call(addr string, data []string) (string, error) {
	packed := data[0]
	packedBytes, _ := hex.DecodeString(packed)
	addr = utils.StripHex(addr)
	addrB, err := hex.DecodeString(addr)
	if err != nil {
		return "", err
	}

	resp, err := rpccore.Call(addrB, packedBytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(resp.Return), err
}

func (mint *MintModule) Script(script string) (string, string, error) {
	script = utils.StripHex(script)
	code, err := hex.DecodeString(script)
	if err != nil {
		return "", "", err
	}
	acc := mint.MempoolReactor.Mempool.GetCache().GetAccount(mint.priv.Address)
	nonce := 0
	if acc != nil {
		nonce = int(acc.Sequence) + 1
	}

	tx := &blk.CallTx{
		Input: &blk.TxInput{
			Address:   mint.priv.Address,
			Amount:    FEE,
			Sequence:  uint(nonce),
			Signature: account.SignatureEd25519{},
			PubKey:    mint.priv.PubKey,
		},
		Address:  nil,
		GasLimit: GASLIMIT,
		Fee:      FEE,
		Data:     code,
	}
	tx.Input.Signature = mint.priv.PrivKey.Sign(account.SignBytes(tx))
	err = mint.MempoolReactor.BroadcastTx(tx)
	return hex.EncodeToString(account.HashSignBytes(tx)), hex.EncodeToString(state.NewContractAddress(mint.priv.Address, uint64(nonce))), err
}

// returns a chanel that will fire when address is updated
func (mint *MintModule) Subscribe(name, event, target string) chan types.Event {
	return nil
}

func (mint *MintModule) UnSubscribe(name string) {
}

// Mine a block
func (m *MintModule) Commit() {
	ch := make(chan struct{})
	m.evsw.AddListenerForEvent("mint-module", "NewBlock", func(msg interface{}) {
		ch <- struct{}{}
	})
	<-ch
	<-ch
	m.evsw.RemoveListener("mint-module")

}

// start and stop continuous mining
func (m *MintModule) AutoCommit(toggle bool) {
	if toggle {
		// m.StartMining()
	} else {
		// m.StopMining()
	}
}

func (m *MintModule) IsAutocommit() bool {
	return false
}

/*
   Blockchain interface should also satisfy KeyManager
   All values are hex encoded
*/

// Return the active address
func (mint *MintModule) ActiveAddress() string {
	return ""
}

// Return the nth address in the ring
func (mint *MintModule) Address(n int) (string, error) {
	return "", nil
}

// Set the address
func (mint *MintModule) SetAddress(addr string) error {
	return nil
}

// Set the address to be the nth in the ring
func (mint *MintModule) SetAddressN(n int) error {
	return nil
}

// Generate a new address
func (mint *MintModule) NewAddress(set bool) string {
	return ""
}

// Return the number of available addresses
func (mint *MintModule) AddressCount() int {
	return 0
}

/*
   Helper functions
*/

func (mint *MintModule) StartMining() bool {
	return false
}

func (mint *MintModule) StopMining() bool {
	return false
}

func (mint *MintModule) StartListening() {
	//eth.ethereum.StartListening()
}

func (mint *MintModule) StopListening() {
	//eth.ethereum.StopListening()
}

func (mint *MintModule) fetchPriv() string {
	return ""
}

// convert ethereum block to types block
/*
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
}*/

// convert ethereum tx to types tx
/*
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
}
*/
