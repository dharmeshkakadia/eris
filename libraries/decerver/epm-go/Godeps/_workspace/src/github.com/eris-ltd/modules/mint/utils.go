package mint

import (
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/logger"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/confer"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/ed25519"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/account"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/binary"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/config"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/state"
	"io/ioutil"
	"os"
	"os/signal"
	"os/user"
	"path"
	"strconv"
)

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
			mintlogger.Errorf("Shutting down (%v) ... \n", sig)
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

func exit(err error) {
	status := 0
	if err != nil {
		fmt.Println(err)
		mintlogger.Errorln("Fatal: ", err)
		status = 1
	}
	logger.Flush()
	os.Exit(status)
}

func int2Level(i int) string {
	switch i {
	case 0:
		return "crit"
	case 1:
		return "error"
	case 2:
		return "warn"
	case 3:
		return "info"
	case 4:
		return "debug"
	case 5:
		return "debug"
	default:
		return "info"
	}
}

func Config2Config(c *ChainConfig) {
	app := confer.NewConfig()
	app.SetDefault("Network", c.Network)
	app.SetDefault("ListenAddr", c.ListenHost+":"+strconv.Itoa(c.ListenPort))
	app.SetDefault("DB.Backend", "leveldb")
	app.SetDefault("DB.Dir", path.Join(c.RootDir, c.DbName))
	app.SetDefault("Log.Stdout.Level", int2Level(c.LogLevel))
	app.SetDefault("Log.File.Dir", path.Join(c.RootDir, c.DebugFile))
	app.SetDefault("Log.File.Level", "debug")
	app.SetDefault("RPC.HTTP.ListenAddr", c.RpcHost+":"+strconv.Itoa(c.RpcPort))
	if c.UseSeed {
		app.SetDefault("SeedNode", c.RemoteHost+":"+strconv.Itoa(c.RemotePort))
	}
	app.SetDefault("GenesisFile", path.Join(c.RootDir, "genesis.json"))
	app.SetDefault("AddrBookFile", path.Join(c.RootDir, "addrbook.json"))
	app.SetDefault("PrivValidatorfile", path.Join(c.RootDir, "priv_validator.json"))
	app.SetDefault("FastSync", c.FastSync)
	config.SetApp(app)
}

func trapSignal(cb func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			mintlogger.Infoln(fmt.Sprintf("captured %v, exiting..", sig))
			cb()
			os.Exit(1)
		}
	}()
	select {}
}

func flip(b []byte) []byte {
	flipped := make([]byte, len(b))
	l2 := len(b) / 2
	if len(b) > 0 {
		flipped[l2] = b[l2]
	}
	for i := 0; i < l2; i++ {
		flipped[i] = b[len(b)-1-i]
		flipped[len(b)-1-i] = b[i]
	}
	return flipped
}

func writeValidatorFile(rootDir string, priv *account.PrivAccount) error {
	privB := []byte(priv.PrivKey.(account.PrivKeyEd25519))
	privKeyBytes := new([64]byte)
	copy(privKeyBytes[:32], privB)
	pubKeyBytes := ed25519.MakePublicKey(privKeyBytes)
	pubKey := account.PubKeyEd25519(pubKeyBytes[:])
	privKey := account.PrivKeyEd25519(privKeyBytes[:])
	validator := state.PrivValidator{
		Address:    pubKey.Address(),
		PubKey:     pubKey,
		PrivKey:    privKey,
		LastHeight: 0,
		LastRound:  0,
		LastStep:   0,
	}
	jsonBytes := binary.JSONBytes(validator)
	return ioutil.WriteFile(path.Join(rootDir, "priv_validator.json"), jsonBytes, 0700)
	return nil
}

func loadOrCreateKey(keySession, keyFile, rootDir string) (*account.PrivAccount, error) {
	if keySession == "" {
		return nil, fmt.Errorf("KeySession may not be empty")
	}
	var priv []byte
	var err error
	if _, err = os.Stat(path.Join(rootDir, keySession)); err != nil {
		if _, err := os.Stat(keyFile); err != nil {
			priv = createKey()
		} else {
			privB, err := ioutil.ReadFile(keyFile)
			if err != nil {
				return nil, err
			}
			priv, err = hex.DecodeString(string(privB))
			if err != nil {
				return nil, err
			}
		}
		if err = ioutil.WriteFile(path.Join(rootDir, keySession), []byte(hex.EncodeToString(priv)), 0600); err != nil {
			return nil, err
		}
	} else {
		privB, err := ioutil.ReadFile(path.Join(rootDir, keySession))
		if err != nil {
			return nil, err
		}
		priv, err = hex.DecodeString(string(privB))
		if err != nil {
			return nil, err
		}
	}

	privKeyBytes := new([64]byte)
	copy(privKeyBytes[:32], priv)
	pubKeyBytes := ed25519.MakePublicKey(privKeyBytes)
	pubKey := account.PubKeyEd25519(pubKeyBytes[:])
	privKey := account.PrivKeyEd25519(priv[:])
	return &account.PrivAccount{
		Address: pubKey.Address(),
		PubKey:  pubKey,
		PrivKey: privKey,
	}, nil
}

func createKey() []byte {
	// create key
	privKeyBytes := new([64]byte)
	b := make([]byte, 32)
	_, err := crand.Read(b)
	if err != nil {
		panic(err)
	}
	copy(privKeyBytes[:32], b)
	return privKeyBytes[:]
}

// get users home directory
func homeDir() string {
	usr, _ := user.Current()
	return usr.HomeDir
}
