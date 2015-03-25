package eth

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	eth "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/crypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/ethdb"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/ethutil"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/logger"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/miner"

	//"github.com/eris-ltd/go-ethereum/xeth"
	//"github.com/eris-ltd/go-ethereum/monkrpc"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/wire"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/kardianos/osext"
)

// this is basically go-etheruem/utils
// i think for now we only use StartMining, but there's porbably other goodies...

//var logger = logger.NewLogger("CLI")
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
			ethlogger.Errorf("Shutting down (%v) ... \n", sig)
			RunInterruptCallbacks(sig)
		}
	}()
}

func RunInterruptCallbacks(sig os.Signal) {
	for _, cb := range interruptCallbacks {
		cb(sig)
	}
}

func AbsolutePath(Datadir string, filename string) string {
	if path.IsAbs(filename) {
		return filename
	}
	return path.Join(Datadir, filename)
}

func openLogFile(Datadir string, filename string) *os.File {
	path := AbsolutePath(Datadir, filename)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("error opening log file '%s': %v", filename, err))
	}
	return file
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

func InitDataDir(Datadir string) {
	_, err := os.Stat(Datadir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Data directory '%s' doesn't exist, creating it\n", Datadir)
			os.Mkdir(Datadir, 0777)
		}
	}
}

func InitLogging(Datadir string, LogFile string, LogLevel int, DebugFile string) {
	var writer io.Writer
	if LogFile == "" {
		writer = os.Stdout
	} else {
		writer = openLogFile(Datadir, LogFile)
	}
	logger.AddLogSystem(logger.NewStdLogSystem(writer, log.LstdFlags, logger.LogLevel(LogLevel)))
	if DebugFile != "" {
		writer = openLogFile(Datadir, DebugFile)
		logger.AddLogSystem(logger.NewStdLogSystem(writer, log.LstdFlags, logger.DebugLevel))
	}
}

func InitConfig(ConfigFile string, Datadir string, EnvPrefix string) *ethutil.ConfigManager {
	InitDataDir(Datadir)
	return ethutil.ReadConfig(ConfigFile, Datadir, EnvPrefix)
}

func exit(err error) {
	status := 0
	if err != nil {
		fmt.Println(err)
		ethlogger.Errorln("Fatal: ", err)
		status = 1
	}
	logger.Flush()
	os.Exit(status)
}

func NewDatabase(dbName string) ethutil.Database {
	db, err := ethdb.NewLDBDatabase(dbName)
	if err != nil {
		exit(err)
	}
	return db
}

func NewClientIdentity(clientIdentifier, version, customIdentifier string) *wire.SimpleClientIdentity {
	ethlogger.Infoln("identity created")
	return wire.NewSimpleClientIdentity(clientIdentifier, version, customIdentifier)
}

/*
func NewEthereum(db ethutil.Database, clientIdentity wire.ClientIdentity, keyManager *crypto.KeyManager, usePnp bool, OutboundPort string, MaxPeer int) *eth.Thelonious {
	ethereum, err := eth.New(db, clientIdentity, keyManager, eth.CapDefault, usePnp)
	if err != nil {
		logger.Fatalln("eth start err:", err)
	}
	ethereum.Port = OutboundPort
	ethereum.MaxPeers = MaxPeer
	return ethereum
}*/

func StartEthereum(ethereum *eth.Ethereum, UseSeed bool) {
	ethlogger.Infof("Starting %s", ethereum.ClientIdentity())
	ethereum.Start(UseSeed)
	RegisterInterrupt(func(sig os.Signal) {
		ethereum.Stop()
		logger.Flush()
	})
}

func ShowGenesis(ethereum *eth.Ethereum) {
	ethlogger.Infoln(ethereum.ChainManager().Genesis())
	exit(nil)
}

func NewKeyManager(KeyStore string, Datadir string, db ethutil.Database) *crypto.KeyManager {
	var keyManager *crypto.KeyManager
	switch {
	case KeyStore == "db":
		keyManager = crypto.NewDBKeyManager(db)
	case KeyStore == "file":
		keyManager = crypto.NewFileKeyManager(Datadir)
	default:
		exit(fmt.Errorf("unknown keystore type: %s", KeyStore))
	}
	return keyManager
}

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

func KeyTasks(keyManager *crypto.KeyManager, KeyRing string, GenAddr bool, SecretFile string, ExportDir string, NonInteractive bool) {

	var err error
	switch {
	case GenAddr:
		if NonInteractive || confirm("This action overwrites your old private key.") {
			err = keyManager.Init(KeyRing, 0, true)
		}
		exit(err)
	case len(SecretFile) > 0:
		SecretFile = ethutil.ExpandHomePath(SecretFile)

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

func StartRpc(ethereum *eth.Ethereum, RpcPort int) {
	//var err error
	/*
		ethereum.RpcServer, err = monkrpc.NewJsonRpcServer(xeth.NewJSPipe(ethereum), RpcPort)
		if err != nil {
			ethlogger.Errorf("Could not start RPC interface (port %v): %v", RpcPort, err)
		} else {
			go ethereum.RpcServer.Start()
		}*/
}

var myMiner *miner.Miner

func GetMiner() *miner.Miner {
	return myMiner
}

func StartMining(ethereum *eth.Ethereum) bool {

	if !ethereum.Mining {
		ethereum.Mining = true
		addr := ethereum.KeyManager().Address()

		go func() {
			ethlogger.Infoln("Start mining")
			if myMiner == nil {
				myMiner = miner.New(addr, ethereum)
			}
			// Give it some time to connect with peers
			time.Sleep(3 * time.Second)
			for !ethereum.IsUpToDate() {
				time.Sleep(5 * time.Second)
			}
			myMiner.Start()
		}()
		RegisterInterrupt(func(os.Signal) {
			StopMining(ethereum)
		})
		return true
	}
	return false
}

func FormatTransactionData(data string) []byte {
	d := ethutil.StringToByteFunc(data, func(s string) (ret []byte) {
		slice := regexp.MustCompile("\\n|\\s").Split(s, 1000000000)
		for _, dataItem := range slice {
			d := ethutil.FormatData(dataItem)
			ret = append(ret, d...)
		}
		return
	})

	return d
}

func StopMining(ethereum *eth.Ethereum) bool {
	if ethereum.Mining && myMiner != nil {
		myMiner.Stop()
		ethlogger.Infoln("Stopped mining")
		ethereum.Mining = false
		myMiner = nil
		return true
	}

	return false
}

// Replay block
func BlockDo(ethereum *eth.Ethereum, hash []byte) error {
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
/*
func CheckZeroBalance(pipe *xeth.Xeth, keyMang *crypto.KeyManager) {
	keys := keyMang.KeyRing()
	masterPair := keys.GetKeyPair(0)
	ethlogger.Infoln("master has ", pipe.Balance(keys.GetKeyPair(keys.Len()-1).Address()))
	for i := 0; i < keys.Len(); i++ {
		k := keys.GetKeyPair(i).Address()
		val := pipe.Balance(k)
		ethlogger.Infoln("key ", i, " ", ethutil.Bytes2Hex(k), " ", val)
		v := val.Int()
		if v < 100 {
			_, err := pipe.Transact(masterPair, k, ethutil.NewValue(ethutil.Big("10000000000000000000")), ethutil.NewValue(ethutil.Big("1000")), ethutil.NewValue(ethutil.Big("1000")), "")
			if err != nil {
				ethlogger.Infoln("Error transfering funds to ", ethutil.Bytes2Hex(k))
			}
		}
	}
}*/
