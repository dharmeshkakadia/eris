package mint

import (
	"encoding/hex"
	"path"
	"strconv"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/account"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/state"
	blk "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/types"
	"github.com/eris-ltd/epm-go/utils"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/modules/types" // implements epm Blockchain
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/rpc/core_client"
)

type MintRPCModule struct {
	Config   *ChainConfig
	priv     *account.PrivAccount
	client   core_client.Client
	wsClient *core_client.WSClient
}

/*
   First, the functions to satisfy Module
*/

func NewMintRPC() *MintRPCModule {
	m := new(MintRPCModule)
	m.Config = DefaultConfig
	return m
}

// initialize an chain
func (mod *MintRPCModule) Init() error {
	p := path.Dir(mod.Property("RootDir").(string))
	p = path.Join(p, "config.json")

	Config2Config(mod.Config)

	mod.rConfig()

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

	rpcAddr := mod.Config.RpcHost + ":" + strconv.Itoa(mod.Config.RpcPort)
	mintlogger.Infoln(rpcAddr)
	mod.client = core_client.NewClient("http://"+rpcAddr, "JSONRPC")
	mod.wsClient = core_client.NewWSClient("ws://" + rpcAddr + "/events")
	_, err = mod.wsClient.Dial()
	if err != nil {
		return err
	}

	return nil
}

// start the tendermint node
func (mod *MintRPCModule) Start() error {
	mintlogger.Infoln("Started")
	return nil
}

func (mod *MintRPCModule) Shutdown() error {
	//mod.wsClient.Close()
	return nil
}

func (mod *MintRPCModule) WaitForShutdown() {
}

// ReadConfig and WriteConfig implemented in config.go

// What module is this?
func (mod *MintRPCModule) Name() string {
	return "tendermint-rpc"
}

/*
 *  Implement Blockchain
 */

func (mint *MintRPCModule) ChainId() (string, error) {
	r, err := mint.client.NetInfo()
	if err != nil {
		return "", err
	}
	return r.Network, nil
}

func (mint *MintRPCModule) WorldState() *types.WorldState {
	return nil
}

func (mint *MintRPCModule) State() *types.State {
	return nil
}

func (mint *MintRPCModule) Storage(addr string) *types.Storage {
	return nil
}

func (mint *MintRPCModule) Account(target string) *types.Account {
	return nil
}

func (mint *MintRPCModule) StorageAt(contract_addr string, storage_addr string) string {
	return ""
}

func (mint *MintRPCModule) BlockCount() int {
	r, err := mint.client.Status()
	if err != nil {
		mintlogger.Errorln(err)
		return -1
	}
	return int(r.LatestBlockHeight)
}

func (mint *MintRPCModule) LatestBlock() string {
	r, err := mint.client.Status()
	if err != nil {
		mintlogger.Errorln(err)
		return ""
	}
	return hex.EncodeToString(r.LatestBlockHash)
}

func (mint *MintRPCModule) Block(hash string) *types.Block {
	// TODO
	return nil
}

func (mint *MintRPCModule) IsScript(target string) bool {
	// TODO
	return false
}

// send a tx
func (mint *MintRPCModule) Tx(addr, amt string) (string, error) {
	addr = utils.StripHex(addr)
	addrB, err := hex.DecodeString(addr)
	if err != nil {
		return "", err
	}
	r, err := mint.client.GetAccount(mint.priv.Address)
	if err != nil {
		return "", err
	}
	acc := r.Account
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
	_, err = mint.client.BroadcastTx(tx)
	return hex.EncodeToString(account.HashSignBytes(tx)), err
}

// send a message to a contract
// data is prepacked by epm
func (mint *MintRPCModule) Msg(addr string, data []string) (string, error) {
	packed := data[0]
	packedBytes, _ := hex.DecodeString(packed)
	addr = utils.StripHex(addr)
	addrB, err := hex.DecodeString(addr)
	if err != nil {
		return "", err
	}
	r, err := mint.client.GetAccount(mint.priv.Address)
	if err != nil {
		return "", err
	}
	acc := r.Account
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
	_, err = mint.client.BroadcastTx(tx)
	return hex.EncodeToString(account.HashSignBytes(tx)), err
}

// send a simulated message to a contract
// data is prepacked by epm
func (mint *MintRPCModule) Call(addr string, data []string) (string, error) {
	packed := data[0]
	packedBytes, _ := hex.DecodeString(packed)
	addr = utils.StripHex(addr)
	addrB, err := hex.DecodeString(addr)
	if err != nil {
		return "", err
	}

	resp, err := mint.client.Call(addrB, packedBytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(resp.Return), err
}

func (mint *MintRPCModule) Script(script string) (string, string, error) {
	script = utils.StripHex(script)
	code, err := hex.DecodeString(script)
	if err != nil {
		return "", "", err
	}
	r, err := mint.client.GetAccount(mint.priv.Address)
	if err != nil {
		return "", "", err
	}
	acc := r.Account
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
	_, err = mint.client.BroadcastTx(tx)
	return hex.EncodeToString(account.HashSignBytes(tx)), hex.EncodeToString(state.NewContractAddress(mint.priv.Address, uint64(nonce))), err
}

// returns a chanel that will fire when address is updated
func (mint *MintRPCModule) Subscribe(name, event, target string) chan types.Event {
	return nil
}

func (mint *MintRPCModule) UnSubscribe(name string) {
}

// Mine a block
func (m *MintRPCModule) Commit() {
	err := m.wsClient.Subscribe("NewBlock")
	if err != nil {
		mintlogger.Errorln(err)
		return
	}
	ch := m.wsClient.Read()
	<-ch
	<-ch
}

// start and stop continuous mining
func (m *MintRPCModule) AutoCommit(toggle bool) {
}

func (m *MintRPCModule) IsAutocommit() bool {
	return false
}

/*
   Blockchain interface should also satisfy KeyManager
   All values are hex encoded
*/

// Return the active address
func (mint *MintRPCModule) ActiveAddress() string {
	return ""
}

// Return the nth address in the ring
func (mint *MintRPCModule) Address(n int) (string, error) {
	return "", nil
}

// Set the address
func (mint *MintRPCModule) SetAddress(addr string) error {
	return nil
}

// Set the address to be the nth in the ring
func (mint *MintRPCModule) SetAddressN(n int) error {
	return nil
}

// Generate a new address
func (mint *MintRPCModule) NewAddress(set bool) string {
	return ""
}

// Return the number of available addresses
func (mint *MintRPCModule) AddressCount() int {
	return 0
}

/*
   Helper functions
*/

func (mint *MintRPCModule) StartMining() bool {
	return false
}

func (mint *MintRPCModule) StopMining() bool {
	return false
}

func (mint *MintRPCModule) StartListening() {
	//eth.ethereum.StartListening()
}

func (mint *MintRPCModule) StopListening() {
	//eth.ethereum.StopListening()
}
