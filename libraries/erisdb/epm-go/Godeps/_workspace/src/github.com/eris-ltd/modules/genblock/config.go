package genblock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/eris-ltd/epm-go/utils"
	"io/ioutil"
	"os"
	"path"
	"reflect"
)

var ErisLtd = utils.ErisLtd

type ChainConfig struct {
	ConfigFile   string `json:"config_file"`
	RootDir      string `json:"root_dir"`
	LogFile      string `json:"log_file"`
	DbName       string `json:"db_name"`
	LLLPath      string `json:"lll_path"`
	ContractPath string `json:"contract_path"`
	KeySession   string `json:"key_session"`
	KeyStore     string `json:"key_store"`
	KeyCursor    int    `json:"key_cursor"`
	KeyFile      string `json:"key_file"`
	LogLevel     int    `json:"log_level"`
	Unique       bool   `json:"unique"`
	PrivateKey   string `json:"private_key"`
}

// set default config object
var DefaultConfig = &ChainConfig{
	ConfigFile: "config",
	RootDir:    path.Join(usr.HomeDir, ".monkchain2"),
	DbName:     "database",
	KeySession: "generous",
	LogFile:    "",
	//LLLPath: path.Join(homeDir(), "cpp-ethereum/build/lllc/lllc"),
	LLLPath:      "NETCALL",
	ContractPath: path.Join(ErisLtd, "eris-std-lib"),
	KeyStore:     "file",
	KeyCursor:    0,
	KeyFile:      path.Join(ErisLtd, "thelonious", "monk", "keys.txt"),
	LogLevel:     5,
}

// can these methods be functions in decerver that take the modules as argument?
func (mod *GenBlockModule) WriteConfig(config_file string) error {
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
func (mod *GenBlockModule) ReadConfig(config_file string) error {
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
		return err
	}
	*(mod.Config) = config
	return nil
}

// this will probably never be used
func (mod *GenBlockModule) SetConfigObj(config interface{}) error {
	if c, ok := config.(*ChainConfig); ok {
		mod.Config = c
	} else {
		return fmt.Errorf("Invalid config object")
	}
	return nil
}

// Set a field in the config struct.
func (mod *GenBlockModule) SetProperty(field string, value interface{}) error {
	cv := reflect.ValueOf(mod.Config).Elem()
	return utils.SetProperty(cv, field, value)
}

func (mod *GenBlockModule) Property(field string) interface{} {
	cv := reflect.ValueOf(mod.Config).Elem()
	f := cv.FieldByName(field)
	return f.Interface()
}

// Set the package global variables, create the root data dir,
//  copy keys if they are available, and setup logging
func (mod *GenBlockModule) gConfig() {
	cfg := mod.Config
	// set lll path
	if cfg.LLLPath != "" {
		//monkutil.PathToLLL = cfg.LLLPath
	}

	// check on data dir
	// create keys
	utils.InitDataDir(cfg.RootDir)
	_, err := os.Stat(path.Join(cfg.RootDir, cfg.KeySession) + ".prv")
	if err != nil {
		utils.Copy(cfg.KeyFile, path.Join(cfg.RootDir, cfg.KeySession)+".prv")
	}
	// TODO: logging ... ?
}
