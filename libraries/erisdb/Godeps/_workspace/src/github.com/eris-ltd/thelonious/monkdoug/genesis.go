package monkdoug

import (
	"encoding/json"
	"fmt"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
	"github.com/eris-ltd/epm-go/utils"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"reflect"
	"strconv"
)

var douglogger = monklog.NewLogger("DOUG")

//   Configure a new genesis block from genesis.json
//   Deploy the genesis block

type Account struct {
	Address     string         `json:"address"`
	byteAddr    []byte         //convenience, but not from json
	Name        string         `json:"name"`
	Balance     string         `json:"balance"`
	Permissions map[string]int `json:"permissions"`
	Stake       int            `json:"stake"`
}

type GenesisConfig struct {
	/*
	   MetaGenDoug
	*/
	// 20 ASCI bytes of gendoug addr
	Address string `json:"address"`
	// Should we use epm deploy?
	Pdx string `json:"pdx"`
	// Path to lll doug contract
	DougPath string `json:"doug"`
	// Should gendoug be unique (set true in production)
	Unique bool `json:"unique"`
	// A private key to seed uniqueness (otherwise is random)
	PrivateKey string `json:"private-key"`
	// Name of the gendoug access model (yes, no, std, vm, eth)
	ModelName string `json:"model"`
	// Turn off gendoug
	NoGenDoug bool `json:"no-gendoug"`

	/*
	   Global GenDoug Singles
	*/
	// Consensus/difficulty mechanism (stake, robin, constant, eth)
	Consensus string `json:"consensus"`
	// Starting difficulty level
	Difficulty int `json:"difficulty"`
	// Allow anyone to mine
	PublicMine int `json:"public:mine"`
	// Allow anyone to create contracts
	PublicCreate int `json:"public:create"`
	// Allow anyone to transact
	PublicTx int `json:"public:tx"`
	// Max gas per tx
	MaxGasTx string `json:"maxgastx"`
	// Proof of work difficulty for transactions/
	TaPoW int `json:"tapow"`
	// Target block time (shaky...)
	BlockTime int `json:"blocktime"`

	// Paths to lll consensus contracts (if ModelName = vm)
	Vm *VmConsensus `json:"vm"`

	// Accounts (permissions and stake)
	Accounts []*Account `json:"accounts"`

	// for convenience, not filled in by json
	hexAddr      string
	byteAddr     []byte
	contractPath string

	// Gendoug based protocol interface
	// for verifying blocks/txs
	protocol  monkchain.Protocol
	consensus monkchain.Consensus

	// Signed genesis block (hex)
	chainId string

	// so we can register a deployer function (which might import monkdoug)
	deployer func(block *monkchain.Block) ([]byte, error)
}

// A protocol level call executed through the vm
type SysCall struct {
	// Path to lll code for this function
	CodePath string `json:"code-path"`
	// Addr of this contract (left padded ascii of its VmConsensys name)
	byteAddr []byte
}

func NewSysCall(codePath string, byteAddr []byte) SysCall {
	return SysCall{codePath, byteAddr}
}

type VmConsensus struct {
	// Name of a suite of contracts
	SuiteName string `json:"suite-name"`
	// Path to lll permission verify contract
	PermissionVerify SysCall `json:"permission-verify"`
	// Path to lll block verify contract
	BlockVerify SysCall `json:"block-verify"`
	// Path to lll tx verify contract
	TxVerify SysCall `json:"tx-verify"`
	// Path to lll compute difficulty contract
	// Calculate difficulty for block from parent (and storage)
	ComputeDifficulty SysCall `json:"compute-difficulty"`
	// Path to lll participate contract
	// Determine if a coinbase should participate in consensus
	ComputeParticipate SysCall `json:"compute-participate"`
	// Participate/Pledge contract
	Participate SysCall `json:"participate"`
	// Contract to run at the beginning of a block
	PreCall SysCall `json:"precall"`
	// Contract to run at the end of a block
	PostCall SysCall `json:"postcall"`
	// Other contracts for arbitrary functionality
	Other []SysCall `json:"other"`
}

func (g *GenesisConfig) Model() monkchain.Protocol {
	return g.protocol
}

func (g *GenesisConfig) SetModel() {
	g.protocol = NewProtocol(g)
}

func (g *GenesisConfig) Deployer(block *monkchain.Block) ([]byte, error) {
	return g.deployer(block)
}

func (g *GenesisConfig) SetDeployer(f func(block *monkchain.Block) ([]byte, error)) {
	g.deployer = f
}

// Load the genesis block info from genesis.json
func LoadGenesis(file string) *GenesisConfig {
	douglogger.Infoln("Loading genesis config:", file)
	b, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println("err reading genesis.json", err)
		os.Exit(0)
	}

	g := new(GenesisConfig)
	err = json.Unmarshal(b, g)
	if err != nil {
		fmt.Println("error unmarshalling genesis.json", err)
		os.Exit(0)
	}

	// move address into accounts, in bytes
	for _, acc := range g.Accounts {
		acc.byteAddr = monkutil.UserHex2Bytes(acc.Address)
	}

	g.byteAddr = []byte(g.Address)
	g.hexAddr = monkutil.Bytes2Hex(g.byteAddr)
	g.contractPath = path.Join(utils.ErisLtd, "eris-std-lib")

	g.Init()

	return g
}

// Initialize the Protocol and Deployer for a populated GenesisConfig
func (g *GenesisConfig) Init() {
	// set doug model
	g.protocol = NewProtocol(g)

	// set default deploy function
	douglogger.Debugf("Setting Doug Deployer: NoGenDoug=%v, ModelName=%s, Path=%s", g.NoGenDoug, g.ModelName, g.DougPath)
	g.SetDeployer(g.Deploy)
}

// Deploy the genesis block
// Converts the GenesisConfig into a populated and functional doug contract in the genesis block
func (g *GenesisConfig) Deploy(block *monkchain.Block) ([]byte, error) {
	block.Difficulty = monkutil.BigPow(2, g.Difficulty)

	// Keys for creating valid txs and for signing
	// the final genblock. Will also give us uniqueness
	keys, err := g.selectKeyPair()
	if err != nil {
		return nil, err
	}

	if g.NoGenDoug {
		// simple bankroll accounts
		g.bankRoll(block)
		chainId := g.chainIdFromBlock(block, keys)
		block.State().Update()
		block.State().Sync()
		return chainId, nil
	}

	douglogger.Infoln("Deploying GenDoug:", g.Address, " ", g.DougPath)

	// create the genesis doug
	codePath := path.Join(g.contractPath, g.DougPath)
	_, _, err = MakeApplyTx(codePath, g.byteAddr, nil, keys, block)
	if err != nil {
		return nil, err
	}

	// set the global vars
	g.setValues(keys, block)

	// set balances and permissions
	g.bankRollAndPerms(keys, block)

	// set verification contracts for "vm" consensus
	if g.ModelName == "vm" {
		if g.Vm == nil {
			return nil, fmt.Errorf("Model=vm requires non-nil VmConsensus obj")
		}
		g.hookVmDeploy(keys, block)
	}

	block.State().Update()
	block.State().Sync()
	chainId := g.chainIdFromBlock(block, keys)

	return chainId, nil
}

func (g *GenesisConfig) chainIdFromBlock(block *monkchain.Block, keys *monkcrypto.KeyPair) []byte {
	// ChainId is leading 20 bytes of SHA3 of 65-byte ecdsa sig
	// Note this means verification requires provision of pubkey
	// We choose 20 bytes so the hashes can be keys on mainline DHT
	douglogger.Debugf("Using signing address %x for deploy\n", keys.Address())
	sig := block.Sign(keys.PrivateKey)
	chainId := monkcrypto.Sha3Bin(sig)[:20]
	g.chainId = monkutil.Bytes2Hex(chainId)
	return chainId
}

/*
   Deploy utilities
*/

// bankroll the accounts
func (g *GenesisConfig) bankRoll(block *monkchain.Block) {
	// no genesis doug, deploy simple
	for _, account := range g.Accounts {
		// direct state modification to create accounts and balances
		AddAccount(account.byteAddr, account.Balance, block)
	}
}

// Bank roll accounts and add permissions and stake
func (g *GenesisConfig) bankRollAndPerms(keys *monkcrypto.KeyPair, block *monkchain.Block) {
	permsApplied := make(map[string]bool)
	for _, account := range g.Accounts {
		// direct state modification to create accounts and balances
		AddAccount(account.byteAddr, account.Balance, block)
		if g.protocol != nil {
			// make sure permission variable exists first!
			for p, _ := range account.Permissions {
				if ok := permsApplied[p]; !ok {
					data := monkutil.PackTxDataArgs2("addperm", p)
					MakeApplyTx("", g.byteAddr, data, keys, block)
					permsApplied[p] = true
				}
			}

			// issue txs to set perms according to the model
			SetPermissions(g.byteAddr, account.byteAddr, account.Permissions, block, keys)
			if account.Permissions["mine"] != 0 {
				SetValue(g.byteAddr, []string{"addminer", account.Name, "0x" + account.Address, "0x" + strconv.Itoa(account.Stake)}, keys, block)
			}
			douglogger.Debugln("Setting permissions for ", account.Address)
		}
	}
}

// XXX: Must be unique for production use!
func (g *GenesisConfig) selectKeyPair() (*monkcrypto.KeyPair, error) {
	var keys *monkcrypto.KeyPair
	var err error
	if g.Unique {
		if g.PrivateKey != "" {
			// TODO: some kind of encryption here ...
			decoded := monkutil.Hex2Bytes(g.PrivateKey)
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

// Set some global values in gendoug
func (g *GenesisConfig) setValues(keys *monkcrypto.KeyPair, block *monkchain.Block) {
	SetValue(g.byteAddr, []string{"initvar", "consensus", "single", g.Consensus}, keys, block)
	SetValue(g.byteAddr, []string{"initvar", "difficulty", "single", "0x" + monkutil.Bytes2Hex(big.NewInt(int64(g.Difficulty)).Bytes())}, keys, block)
	//SetValue(g.byteAddr, []string{"initvar", "public:mine", "single", "0x" + strconv.Itoa(g.PublicMine)}, keys, block)
	//SetValue(g.byteAddr, []string{"initvar", "public:create", "single", "0x" + strconv.Itoa(g.PublicCreate)}, keys, block)
	//SetValue(g.byteAddr, []string{"initvar", "public:tx", "single", "0x" + strconv.Itoa(g.PublicTx)}, keys, block)
	SetValue(g.byteAddr, []string{"setglobal", "mine", "0x" + strconv.Itoa(g.PublicMine)}, keys, block)
	SetValue(g.byteAddr, []string{"setglobal", "create", "0x" + strconv.Itoa(g.PublicCreate)}, keys, block)
	SetValue(g.byteAddr, []string{"setglobal", "transact", "0x" + strconv.Itoa(g.PublicTx)}, keys, block)
	SetValue(g.byteAddr, []string{"initvar", "maxgastx", "single", g.MaxGasTx}, keys, block)
	SetValue(g.byteAddr, []string{"initvar", "blocktime", "single", "0x" + strconv.Itoa(g.BlockTime)}, keys, block)
}

// Options for hooking consensus to the vm
const (
	VmDefTy = iota
	VmScriptTy
)

// Hook a set of contracts into the protocol for deployment
// by filling in the VmModel.contracts map.
// Contracts can be specified explictly or by suite name,
// or else the (TODO: non-secure) builtin defaults
// Addresses are stored at standard locations in gendoug
func (g *GenesisConfig) hookVmDeploy(keys *monkcrypto.KeyPair, block *monkchain.Block) {
	// TODO: add some logs!

	// grab the suite, if any
	suite := suites["default"]
	if s, ok := suites[g.Vm.SuiteName]; ok {
		suite = s
	}

	// loop through g.Vm fields
	// deploy the non-nil ones
	// fall back order: g.Vm > suite > defaults
	m := g.protocol.(*Protocol).consensus.(*VmModel)
	gvm := reflect.ValueOf(g.Vm).Elem()
	svm := reflect.ValueOf(suite).Elem()
	// Skip first and last (suite name, others)
	for i := 1; i < gvm.NumField()-1; i++ {
		// default mode (if a contract is provided)
		mode := VmDefTy
		// grab fields from struct
		_, tag, codePath := nameTagPath(gvm, i)
		if codePath != "" {
			mode = VmScriptTy
		} else if suite != nil {
			_, _, codePath = nameTagPath(svm, i)
			if codePath != "" {
				mode = VmScriptTy
			}
		}

		if mode > 0 {
			absCodePath := path.Join(g.contractPath, codePath)
			tx, _, err := MakeApplyTx(absCodePath, nil, nil, keys, block)
			if err == nil {
				s := SysCall{
					byteAddr: tx.CreationAddress(),
					CodePath: absCodePath,
				}
				m.contract[tag] = s
				douglogger.Infof("Setting contract address in GENDOUG for %s (%s) : %x\n", tag, codePath, s.byteAddr)
				SetValue(g.byteAddr, []string{"initvar", tag, "single", "0x" + monkutil.Bytes2Hex(s.byteAddr)}, keys, block)
			}
		}
	}
	//TODO handle final element in Vm struct (list of SysCalls)
}

// return field name, tag, and codepath
// value of f is a SysCall struct
func nameTagPath(gvm reflect.Value, i int) (string, string, string) {
	// value of f is a SysCall struct
	f := gvm.Field(i)
	typeOf := gvm.Type()
	name := typeOf.Field(i).Name
	tag := typeOf.Field(i).Tag.Get("json")
	v := f.FieldByName("CodePath")
	val := v.String()
	return name, tag, val
}

// hook the contracts into consensus for a running node
// by looking up their addresses in gendoug
func (g *GenesisConfig) hookVm(block *monkchain.Block) {
	m := g.protocol.(*Protocol).consensus.(*VmModel)
	gvm := reflect.ValueOf(g.Vm).Elem()
	for i := 1; i < gvm.NumField()-1; i++ {
		_, tag, _ := nameTagPath(gvm, i)
		// address of contract from gendoug
		addr := GetValue(g.byteAddr, tag, block.State())
		s := SysCall{
			byteAddr: addr,
		}
		m.contract[tag] = s
	}
}

// set balance of an account (does not commit)
func AddAccount(addr []byte, balance string, block *monkchain.Block) {
	account := block.State().GetAccount(addr)
	account.Balance = monkutil.Big(balance) //monkutil.BigPow(2, 200)
	block.State().UpdateStateObject(account)
}

//
func NewProtocol(g *GenesisConfig) monkchain.Protocol {
	consensus := NewPermModel(g)
	p := &Protocol{g: g, consensus: consensus}
	return p
}

// Return a new permissions model
// Only "std" and "vm" care about gendoug
// NoGendoug defaults to the "yes" model
func NewPermModel(g *GenesisConfig) (model monkchain.Consensus) {
	modelName := g.ModelName
	if g.NoGenDoug {
		modelName = "yes"
	}
	switch modelName {
	case "std":
		// gendoug-v2
		// uses eris-std-lib/gotests/vars for reading
		// from gendoug
		model = NewStdLibModel(g)
	case "vm":
		// run processing through the vm
		model = NewVmModel(g)
	case "yes":
		// everyone allowed everything
		model = NewYesModel(g)
	case "no":
		// noone allowed anything
		model = NewNoModel(g)
	case "eth":
		// ethereum
		g.NoGenDoug = true
		model = NewEthModel(g)
	default:
		// default to yes
		model = NewYesModel(g)
	}
	return
}

// A default genesis.json
// TODO: make a lookup-able suite of these
var DefaultGenesis = defaultGenesis()

func defaultGenesis() *GenesisConfig {
	g := &GenesisConfig{
		Address:    "0000000000THISISDOUG",
		NoGenDoug:  true,
		Difficulty: 15,
		Accounts: []*Account{
			&Account{
				Address:  "0xbbbd0256041f7aed3ce278c56ee61492de96d001",
				byteAddr: monkutil.Hex2Bytes("bbbd0256041f7aed3ce278c56ee61492de96d001"),
				Balance:  "1000000000000000000000000000000000000",
			},
		},
	}
	g.Init()
	return g
}

// Contract suites for vm based protocol
var suites = map[string]*VmConsensus{
	"std": &VmConsensus{
		SuiteName:          "std",
		PermissionVerify:   NewSysCall("", nil),
		BlockVerify:        NewSysCall("Protocol/block-verify.lll", nil),
		TxVerify:           NewSysCall("Protocol/tx-verify.lll", nil),
		ComputeDifficulty:  NewSysCall("Protocol/compute-difficulty.lll", nil),
		ComputeParticipate: NewSysCall("", nil),
		Participate:        NewSysCall("", nil),
		PreCall:            NewSysCall("", nil),
		PostCall:           NewSysCall("", nil),
	},
}
