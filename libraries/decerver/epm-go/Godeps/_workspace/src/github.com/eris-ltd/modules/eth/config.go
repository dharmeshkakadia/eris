package eth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/ethutil"
	"github.com/eris-ltd/epm-go/utils"
	"io/ioutil"
	"os"
	"path"
	"reflect"
)

var ErisLtd = utils.ErisLtd

type ChainConfig struct {
	Port             int    `json:"port"`
	Mining           bool   `json:"mining"`
	MaxPeers         int    `json:"max_peers"`
	ConfigFile       string `json:"config_file"`
	RootDir          string `json:"root_dir"`
	LogFile          string `json:"log_file"`
	DbName           string `json:"db_name"`
	LLLPath          string `json:"lll_path"`
	ContractPath     string `json:"contract_path"`
	ClientIdentifier string `json:"client"`
	Version          string `json:"version"`
	Identifier       string `json:"id"`
	KeySession       string `json:"key_session"`
	KeyStore         string `json:"key_store"`
	KeyCursor        int    `json:"key_cursor"`
	KeyFile          string `json:"key_file"`
	Difficulty       string `json:"difficulty"`
	LogLevel         int    `json:"log_level"`
	Adversary        int    `json:"adversary"`
}

// set default config object
var DefaultConfig = &ChainConfig{
	Port:       30303,
	Mining:     false,
	MaxPeers:   10,
	ConfigFile: "config",
	RootDir:    path.Join(usr.HomeDir, ".ethchain"),
	DbName:     "database",
	KeySession: "generous",
	LogFile:    "",
	//LLLPath: path.Join(homeDir(), "cpp-ethereum/build/lllc/lllc"),
	LLLPath:          "NETCALL",
	ContractPath:     path.Join(ErisLtd, "eris-std-lib"),
	ClientIdentifier: "EthGlue",
	Version:          "2.7.1",
	Identifier:       "chainId",
	KeyStore:         "file",
	KeyCursor:        0,
	KeyFile:          path.Join(ErisLtd, "decerver-interfaces", "glue", "eth", "keys.txt"),
	LogLevel:         5,
	Adversary:        0,
}

// can these methods be functions in decerver that take the modules as argument?
func (mod *EthModule) WriteConfig(config_file string) error {
	b, err := json.Marshal(mod.eth.config)
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
func (mod *EthModule) ReadConfig(config_file string) error {
	b, err := ioutil.ReadFile(config_file)
	if err != nil {
		fmt.Println("could not read config", err)
		fmt.Println("resorting to defaults")
		return err
	}
	var config ChainConfig
	err = json.Unmarshal(b, &config)
	if err != nil {
		fmt.Println("error unmarshalling config from file:", err)
		fmt.Println("resorting to defaults")
		//mod.eth.config = DefaultConfig
		return err
	}
	*(mod.Config) = config
	return nil
}

// this will probably never be used
func (mod *EthModule) SetConfigObj(config interface{}) error {
	if c, ok := config.(*ChainConfig); ok {
		mod.eth.config = c
	} else {
		return fmt.Errorf("Invalid config object")
	}
	return nil
}

// Set the package global variables, create the root data dir,
//  copy keys if they are available, and setup logging
func (eth *Eth) ethConfig() {
	cfg := eth.config
	// set lll path
	if cfg.LLLPath != "" {
		//ethutil.PathToLLL = cfg.LLLPath
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
	// eth-go uses a global ethutil.Config object. This will set it up for us, but we do our config of course our way
	// it also uses rakyl/globalconf, but fuck that for now
	ethutil.Config = &ethutil.ConfigManager{ExecPath: cfg.RootDir, Debug: true, Paranoia: true}
	// data dir, logfile, log level, debug file
	// TODO: enhance this with more pkg level control
	InitLogging(cfg.RootDir, cfg.LogFile, cfg.LogLevel, "")
}

// Set a field in the config struct.
func (mod *EthModule) SetProperty(field string, value interface{}) error {
	cv := reflect.ValueOf(mod.Config).Elem()
	return utils.SetProperty(cv, field, value)
}

func (mod *EthModule) Property(field string) interface{} {
	cv := reflect.ValueOf(mod.Config).Elem()
	f := cv.FieldByName(field)
	return f.Interface()
}
