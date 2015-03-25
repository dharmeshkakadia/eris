package monkrpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	mutils "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/modules/monkutils"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
	"github.com/eris-ltd/epm-go/utils"
	"io/ioutil"
	"os"
	"path"
	"reflect"
)

var ErisLtd = utils.ErisLtd

type RpcConfig struct {
	// Networking
	RpcHost string `json:"rpc_host"`
	RpcPort int    `json:"rpc_port"`

	// If true, key management is handled
	// by the server (presumably on a local machine)
	// else, txs are signed by a key and rlp serialized
	Local bool `json:"local"`

	// Only relevant if Local is false
	KeySession string `json:"key_session"`
	KeyStore   string `json:"key_store"`
	KeyCursor  int    `json:"key_cursor"`
	KeyFile    string `json:"key_file"`

	// Paths
	RootDir      string `json:"root_dir"`
	DbName       string `json:"db_name"`
	LLLPath      string `json:"lll_path"`
	ContractPath string `json:"contract_path"`

	// Logs
	LogFile   string `json:"log_file"`
	DebugFile string `json:"debug_file"`
	LogLevel  int    `json:"log_level"`
}

// set default config object
var DefaultConfig = &RpcConfig{
	// Network
	RpcHost: "",
	RpcPort: 30304,

	Local: true,

	// Local Node
	KeySession: "generous",
	KeyStore:   "file",
	KeyCursor:  0,
	KeyFile:    path.Join(ErisLtd, "thelonious", "monk", "keys.txt"),

	// Paths
	RootDir:      path.Join(usr.HomeDir, ".monkchain2"),
	DbName:       "database",
	LLLPath:      "NETCALL", //path.Join(homeDir(), "cpp-ethereum/build/lllc/lllc"),
	ContractPath: path.Join(ErisLtd, "eris-std-lib"),

	// Log
	LogFile:   "",
	DebugFile: "",
	LogLevel:  5,
}

// Marshal the current configuration to file in pretty json.
func (mod *MonkRpcModule) WriteConfig(config_file string) error {
	b, err := json.Marshal(mod.Config)
	if err != nil {
		fmt.Println("error marshalling config:", err)
		return err
	}
	var out bytes.Buffer
	json.Indent(&out, b, "", "\t")
	err = ioutil.WriteFile(config_file, out.Bytes(), 0600)
	if err != nil {
		return err
	}
	return nil
}

// Unmarshal the configuration file into module's config struct.
func (mod *MonkRpcModule) ReadConfig(config_file string) error {
	b, err := ioutil.ReadFile(config_file)
	if err != nil {
		fmt.Println("could not read config", err)
		fmt.Println("resorting to defaults")
		return err
	}
	var config RpcConfig
	err = json.Unmarshal(b, &config)
	if err != nil {
		fmt.Println("error unmarshalling config from file:", err)
		fmt.Println("resorting to defaults")
		return err
	}
	*(mod.Config) = config
	return nil
}

// Set the config object directly
func (mod *MonkRpcModule) SetConfigObj(config interface{}) error {
	if c, ok := config.(*RpcConfig); ok {
		mod.Config = c
	} else {
		return fmt.Errorf("Invalid config object")
	}
	return nil
}

func (mod *MonkRpcModule) SetProperty(field string, value interface{}) error {
	cv := reflect.ValueOf(mod.Config).Elem()
	return utils.SetProperty(cv, field, value)
}

func (mod *MonkRpcModule) Property(field string) interface{} {
	cv := reflect.ValueOf(mod.Config).Elem()
	f := cv.FieldByName(field)
	return f.Interface()
}

// Set package global variables (LLLPath, monkutil.Config, logging).
// Create the root data dir if it doesn't exist, and copy keys if they are available
func (mod *MonkRpcModule) rConfig() {
	cfg := mod.Config
	// set lll path
	if cfg.LLLPath != "" {
		//monkutil.PathToLLL = cfg.LLLPath
	}

	// check on data dir
	// create keys
	_, err := os.Stat(cfg.RootDir)
	if err != nil {
		os.Mkdir(cfg.RootDir, 0777)
		_, err := os.Stat(path.Join(cfg.RootDir, cfg.KeySession) + ".prv")
		if err != nil {
			utils.Copy(cfg.KeyFile, path.Join(cfg.RootDir, cfg.KeySession)+".prv")
		}
	}
	// a global monkutil.Config object is used for shared global access to the db.
	// this also uses rakyl/globalconf, but we mostly ignore all that
	if monkutil.Config == nil {
		monkutil.Config = &monkutil.ConfigManager{ExecPath: cfg.RootDir, Debug: true, Paranoia: true}
	}

	if monkutil.Config.Db == nil {
		monkutil.Config.Db = mutils.NewDatabase(mod.Config.DbName, false)
	}

	// TODO: enhance this with more pkg level control
	utils.InitLogging(cfg.RootDir, cfg.LogFile, cfg.LogLevel, cfg.DebugFile)
}
