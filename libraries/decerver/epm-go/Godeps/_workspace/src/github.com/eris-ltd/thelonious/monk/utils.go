package monk

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/decerver/interfaces/dapps"
	mutils "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/modules/monkutils"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/modules/types"
	eth "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkdoug"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkminer"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkpipe"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkrpc"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/kardianos/osext"
	"github.com/eris-ltd/epm-go/chains" // this is basically go-etheruem/utils
	// TODO: use the interupts...
	"github.com/eris-ltd/epm-go/epm" //"github.com/eris-ltd/modules/genblock"
	"github.com/eris-ltd/epm-go/utils"
)

//var logger = monklog.NewLogger("CLI")
var interruptCallbacks = []func(os.Signal){}

// Register interrupt handlers callbacks
func RegisterInterrupt(cb func(os.Signal)) {
	interruptCallbacks = append(interruptCallbacks, cb)
}

// go routine that call interrupt handlers in order of registering
func HandleInterrupt() {
	c := make(chan os.Signal, 1)
	go func() {
		signal.Notify(c, os.Interrupt)
		for sig := range c {
			logger.Errorf("Shutting down (%v) ... \n", sig)
			RunInterruptCallbacks(sig)
		}
	}()
}

func RunInterruptCallbacks(sig os.Signal) {
	for _, cb := range interruptCallbacks {
		cb(sig)
	}
}

func confirm(message string) bool {
	fmt.Println(message, "Are you sure? (y/n)")
	var r string
	fmt.Scanln(&r)
	for ; ; fmt.Scanln(&r) {
		if r == "n" || r == "y" {
			break
		} else {
			fmt.Printf("Yes or no?", r)
		}
	}
	return r == "y"
}

// TODO: dwell on this more too
func InitConfig(ConfigFile string, Datadir string, EnvPrefix string) *monkutil.ConfigManager {
	utils.InitDataDir(Datadir)
	return monkutil.ReadConfig(ConfigFile, Datadir, EnvPrefix)
}

func exit(err error) {
	status := 0
	if err != nil {
		fmt.Println(err)
		logger.Errorln("Fatal: ", err)
		status = 1
	}
	monklog.Flush()
	os.Exit(status)
}

func ShowGenesis(ethereum *eth.Thelonious) {
	logger.Infoln(ethereum.ChainManager().Genesis())
	exit(nil)
}

// TODO: work this baby
func DefaultAssetPath() string {
	var assetPath string
	// If the current working directory is the go-ethereum dir
	// assume a debug build and use the source directory as
	// asset directory.
	pwd, _ := os.Getwd()
	if pwd == path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "ethereal") {
		assetPath = path.Join(pwd, "assets")
	} else {
		switch runtime.GOOS {
		case "darwin":
			// Get Binary Directory
			exedir, _ := osext.ExecutableFolder()
			assetPath = filepath.Join(exedir, "../Resources")
		case "linux":
			assetPath = "/usr/share/ethereal"
		case "windows":
			assetPath = "./assets"
		default:
			assetPath = "."
		}
	}
	return assetPath
}

// TODO: use this...
func KeyTasks(keyManager *monkcrypto.KeyManager, KeyRing string, GenAddr bool, SecretFile string, ExportDir string, NonInteractive bool) {

	var err error
	switch {
	case GenAddr:
		if NonInteractive || confirm("This action overwrites your old private key.") {
			err = keyManager.Init(KeyRing, 0, true)
		}
		exit(err)
	case len(SecretFile) > 0:
		SecretFile = monkutil.ExpandHomePath(SecretFile)

		if NonInteractive || confirm("This action overwrites your old private key.") {
			err = keyManager.InitFromSecretsFile(KeyRing, 0, SecretFile)
		}
		exit(err)
	case len(ExportDir) > 0:
		err = keyManager.Init(KeyRing, 0, false)
		if err == nil {
			err = keyManager.Export(ExportDir)
		}
		exit(err)
	default:
		// Creates a keypair if none exists
		err = keyManager.Init(KeyRing, 0, false)
		if err != nil {
			exit(err)
		}
	}
}

func StartRpc(ethereum *eth.Thelonious, RpcHost string, RpcPort int) {
	var err error
	rpcAddr := RpcHost + ":" + strconv.Itoa(RpcPort)
	ethereum.RpcServer, err = monkrpc.NewJsonRpcServer(monkpipe.NewJSPipe(ethereum), rpcAddr)
	if err != nil {
		logger.Errorf("Could not start RPC interface (port %v): %v", RpcPort, err)
	} else {
		go ethereum.RpcServer.Start()
	}
}

var minerLock = new(sync.Mutex)
var miner *monkminer.Miner

func GetMiner() *monkminer.Miner {
	return miner
}

func StartMining(ethereum *eth.Thelonious) bool {

	if !ethereum.Mining {
		ethereum.Mining = true
		addr := ethereum.KeyManager().Address()

		go func() {
			logger.Infoln("Start mining")
			minerLock.Lock()
			if miner == nil {
				miner = monkminer.NewDefaultMiner(addr, ethereum)
			}
			// Give it some time to connect with peers
			time.Sleep(3 * time.Second)
			for !ethereum.IsUpToDate() {
				time.Sleep(5 * time.Second)
			}
			miner.Start()
			minerLock.Unlock()
		}()
		RegisterInterrupt(func(os.Signal) {
			StopMining(ethereum)
		})
		return true
	}
	return false
}

func FormatTransactionData(data string) []byte {
	d := monkutil.StringToByteFunc(data, func(s string) (ret []byte) {
		slice := regexp.MustCompile("\\n|\\s").Split(s, 1000000000)
		for _, dataItem := range slice {
			d := monkutil.FormatData(dataItem)
			ret = append(ret, d...)
		}
		return
	})

	return d
}

func StopMining(ethereum *eth.Thelonious) bool {
	minerLock.Lock()
	defer minerLock.Unlock()
	if ethereum.Mining && miner != nil {
		miner.Stop()
		logger.Infoln("Stopped mining")
		ethereum.Mining = false
		miner = nil
		return true
	}

	return false
}

// Replay block
func BlockDo(ethereum *eth.Thelonious, hash []byte) error {
	block := ethereum.ChainManager().GetBlock(hash)
	if block == nil {
		return fmt.Errorf("unknown block %x", hash)
	}

	parent := ethereum.ChainManager().GetBlock(block.PrevHash)

	_, err := ethereum.BlockManager().ApplyDiff(parent.State(), parent, block)
	if err != nil {
		return err
	}

	return nil

}

// If an address is empty, load er up
// vestige of ye old key days
func CheckZeroBalance(pipe *monkpipe.Pipe, keyMang *monkcrypto.KeyManager) {
	keys := keyMang.KeyRing()
	masterPair := keys.GetKeyPair(0)
	logger.Infoln("master has ", pipe.Balance(keys.GetKeyPair(keys.Len()-1).Address()))
	for i := 0; i < keys.Len(); i++ {
		k := keys.GetKeyPair(i).Address()
		val := pipe.Balance(k)
		logger.Infoln("key ", i, " ", monkutil.Bytes2Hex(k), " ", val)
		v := val.Int()
		if v < 100 {
			_, err := pipe.Transact(masterPair, k, monkutil.NewValue(monkutil.Big("10000000000000000000")), monkutil.NewValue(monkutil.Big("1000")), monkutil.NewValue(monkutil.Big("1000")), "")
			if err != nil {
				logger.Infoln("Error transfering funds to ", monkutil.Bytes2Hex(k))
			}
		}
	}
}

// Set the EPM contract root
func setEpmContractPath(p string) {
	epm.ContractPath = p
}

// Deploy a pdx onto a block
// This is used as a monkdoug deploy function
func epmDeploy(block *monkchain.Block, pkgDef string) ([]byte, error) {
	// TODO: use epm here
	/*
		m := genblock.NewGenBlockModule(block)
		m.Config.LogLevel = 5
		err := m.Init()
		if err != nil {
			return nil, err
		}
		m.Start()
		epm.ErrMode = epm.ReturnOnErr
		e, err := epm.NewEPM(m, ".epm-log")
		if err != nil {
			return nil, err
		}
		err = e.Parse(pkgDef)
		if err != nil {
			return nil, err
		}
		err = e.ExecuteJobs()
		if err != nil {
			return nil, err
		}
		e.Commit()
		chainId, err := m.ChainId()
		if err != nil {
			return nil, err
		}
		return chainId, nil
	*/
	return nil, nil
}

// Deploy sequence (done through monk interface for simplicity):
//  - Create .temp/ for database in current dir
//  - Read genesis.json and populate struct
//  - Deploy genesis block and return chainId
//  - Move .temp/ into ~/.decerver/blockchain/thelonious/chainID
//  - write name to index file if provided and no conflict
func DeploySequence(name, genesis, config string) (string, error) {
	root := ".temp"
	chainId, err := DeployChain(root, genesis, config)
	if err != nil {
		return "", err
	}
	if err := InstallChain(root, genesis, config, chainId); err != nil {
		return "", err
	}

	return chainId, nil
}

func splitHostPort(peerServer string) (string, int, error) {
	spl := strings.Split(peerServer, ":")
	if len(spl) < 2 {
		return "", 0, fmt.Errorf("Impromerly formatted peer server. Should be <host>:<port>")
	}
	host := spl[0]
	port, err := strconv.Atoi(spl[1])
	if err != nil {
		return "", 0, fmt.Errorf("Bad port number: ", spl[1])
	}
	return host, port, nil
}

// Deploy a chain from a genesis block but overwrite the chain Id
// This is someone else's chain we are fetching to catch up with
func FetchInstallChain(dappName string) error { //chainId, peerServer, genesisJson string) error{
	dappDir := path.Join(utils.Dapps, dappName)
	var err error

	p, err := chains.CheckGetPackageFile(dappDir)
	if err != nil {
		return err
	}
	logger.Warnf("SHUTING DOWN")

	var (
		host    string
		port    int
		chainId string
	)
	for _, dep := range p.ModuleDependencies {
		if dep.Name == "monk" {
			d := &dapps.MonkData{}
			if err = json.Unmarshal(*(dep.Data), d); err != nil {
				return err
			}
			host, port, err = splitHostPort(d.PeerServerAddress)
			if err != nil {
				return err
			}
			chainId = d.ChainId
		}
	}

	m := NewMonk(nil)

	m.Config.RootDir = path.Join(Thelonious, chainId)
	if err := utils.InitDataDir(m.Config.RootDir); err != nil {
		return err
	}

	genesisJson := path.Join(dappDir, "genesis.json")
	if err := utils.Copy(genesisJson, path.Join(m.Config.RootDir, "genesis.json")); err != nil {
		return err
	}

	m.Config.UseSeed = true
	m.Config.GenesisConfig = path.Join(m.Config.RootDir, "genesis.json")
	m.Config.RemoteHost = host
	m.Config.RemotePort = port

	if err := utils.WriteJson(m.Config, path.Join(m.Config.RootDir, "config.json")); err != nil {
		return err
	}

	err = m.Init()
	if err != nil {
		return err
	}

	monkutil.Config.Db.Put([]byte("ChainID"), monkutil.Hex2Bytes(chainId))

	return nil
}

// Read config, set deployment root, config genesis block,
// init, return chainId
func DeployChain(root, genesis, config string) (string, error) {
	// startup and deploy
	m := NewMonk(nil)
	m.ReadConfig(config)
	m.Config.RootDir = root

	if strings.HasSuffix(genesis, ".pdx") || strings.HasSuffix(genesis, ".gdx") {
		m.GenesisConfig = &monkdoug.GenesisConfig{Address: "0000000000THISISDOUG", NoGenDoug: false, Pdx: genesis}
		m.GenesisConfig.Init()
	} else {
		m.Config.GenesisConfig = genesis
	}

	if err := m.Init(); err != nil {
		return "", err
	}

	// get the chain id
	return m.ChainId()
}

// Copy files and deploy directory into global tree. Set configuration values for root dir and chain id.
func InstallChain(root, genesis, config, chainId string) error {
	monkutil.Config.Db.Close()
	home := path.Join(Thelonious, chainId)
	logger.Infoln("Install directory ", home)
	// move datastore and configs
	// be sure to copy paths into config
	if err := utils.InitDataDir(home); err != nil {
		return err
	}
	if err := rename(root, home); err != nil {
		return err
	}

	logger.Infoln("Loading and setting chainId ", config)
	// set chainId and rootdir values in config file
	b, err := ioutil.ReadFile(config)
	if err != nil {
		return err
	}
	var configT ChainConfig
	if err = json.Unmarshal(b, &configT); err != nil {
		return err
	}

	configT.ChainId = chainId
	//configT.ChainName = name
	configT.RootDir = home
	configT.GenesisConfig = path.Join(home, "genesis.json")

	if err := utils.WriteJson(configT, config); err != nil {
		return err
	}

	if err := rename(config, path.Join(home, "config.json")); err != nil {
		return err
	}

	/*if err := rename(genesis, path.Join(home, "genesis.json")); err != nil {
		return err
	}*/

	// update refs
	/*if name != "" {
		err := chains.AddRef("thelonious", chainId, name)
		if err != nil {
			return err
		}
		logger.Infof("Created ref %s to point to chain %s\n", name, chainId)
	}*/

	return nil
}

func ChainIdFromDb(root string) (string, error) {
	monkutil.Config = &monkutil.ConfigManager{ExecPath: root, Debug: true, Paranoia: true}
	db := mutils.NewDatabase("database", false)
	monkutil.Config.Db = db
	data, err := monkutil.Config.Db.Get([]byte("ChainID"))
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("Empty ChainID!")
	}
	return monkutil.Bytes2Hex(data), nil
}

func rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func copy(oldpath, newpath string) error {
	return utils.Copy(oldpath, newpath)
}

// some convenience functions

// get users home directory
func homeDir() string {
	usr, _ := user.Current()
	return usr.HomeDir
}

// convert a big int from string to hex
func BigNumStrToHex(s string) string {
	bignum := monkutil.Big(s)
	bignum_bytes := monkutil.BigToBytes(bignum, 16)
	return monkutil.Bytes2Hex(bignum_bytes)
}

// takes a string, converts to bytes, returns hex
func SHA3(tohash string) string {
	h := monkcrypto.Sha3Bin([]byte(tohash))
	return monkutil.Bytes2Hex(h)
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
			x := monkutil.Hex2Bytes(t)
			//fmt.Println(x)
			l := len(x)
			ret = append(ret, monkutil.LeftPadBytes(x, 32*((l+31)/32))...)
		} else {
			x := []byte(s)
			l := len(x)
			// TODO: just changed from right to left. yabadabadoooooo take care!
			ret = append(ret, monkutil.LeftPadBytes(x, 32*((l+31)/32))...)
		}
	}
	return "0x" + monkutil.Bytes2Hex(ret)
	// return ret
}

// convert thelonious block to modules block
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

// convert thelonious tx to modules tx
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

func PrettyPrintAccount(obj *monkstate.StateObject) {
	fmt.Println("Address", monkutil.Bytes2Hex(obj.Address())) //monkutil.Bytes2Hex([]byte(addr)))
	fmt.Println("\tNonce", obj.Nonce)
	fmt.Println("\tBalance", obj.Balance)
	if true { // only if contract, but how?!
		fmt.Println("\tInit", monkutil.Bytes2Hex(obj.InitCode))
		fmt.Println("\tCode", monkutil.Bytes2Hex(obj.Code))
		fmt.Println("\tStorage:")
		obj.EachStorage(func(key string, val *monkutil.Value) {
			val.Decode()
			fmt.Println("\t\t", monkutil.Bytes2Hex([]byte(key)), "\t:\t", monkutil.Bytes2Hex([]byte(val.Str())))
		})
	}
}

// print all accounts and storage in a block
func PrettyPrintBlockAccounts(block *monkchain.Block) {
	state := block.State()
	it := state.Trie.NewIterator()
	it.Each(func(key string, value *monkutil.Value) {
		addr := monkutil.Address([]byte(key))
		//        obj := monkstate.NewStateObjectFromBytes(addr, value.Bytes())
		obj := block.State().GetAccount(addr)
		PrettyPrintAccount(obj)
	})
}

// print all accounts and storage in the latest block
func PrettyPrintChainAccounts(mod *MonkModule) {
	curchain := mod.monk.thelonious.ChainManager()
	block := curchain.CurrentBlock()
	PrettyPrintBlockAccounts(block)
}
