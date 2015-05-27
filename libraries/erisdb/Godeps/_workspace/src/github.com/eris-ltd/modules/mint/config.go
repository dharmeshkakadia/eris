package mint

import (
	"fmt"
	"github.com/eris-ltd/epm-go/utils"
	"os"
	"os/user"
	"path"
	"reflect"
)

var (
	GoPath         = os.Getenv("GOPATH")
	usr, _         = user.Current() // error?!
	ErisLtd        = utils.ErisLtd
	TendermintUser = path.Join(GoPath, "src", "github.com", "tendermint", "tendermint")
	Decerver       = utils.Decerver
	Tendermint     = path.Join(utils.Blockchains, "tendermint")

	DefaultRoot          = path.Join(Tendermint, "default-chain")
	DefaultGenesisConfig = path.Join(TendermintUser, "defaults", "genesis.json")
	DefaultKeyFile       = "" //defaultKeyFile
	defaultKeyFile       = path.Join(TendermintUser, "defaults", "keys.txt")
)

// main configuration struct for
// starting a blockchain module
type ChainConfig struct {
	// Networking
	ListenHost string `json:"local_host"`
	ListenPort int    `json:"local_port"`
	Listen     bool   `json:"listen"`
	RemoteHost string `json:"remote_host"`
	RemotePort int    `json:"remote_port"`
	UseSeed    bool   `json:"use_seed"`
	RpcHost    string `json:"rpc_host"`
	RpcPort    int    `json:"rpc_port"`
	ServeRpc   bool   `json:"serve_rpc"`
	FetchPort  int    `json:"fetch_port"`

	// ChainId and Name
	ChainId   string `json:"chain_id"`
	ChainName string `json:"chain_name"`

	// Local Node
	FastSync   bool   `json:"fast_sync"`
	MaxPeers   int    `json:"max_peers"`
	Moniker    string `json:"moniker"`
	Version    string `json:"version"`
	Network    string `json:"network"`
	KeySession string `json:"key_session"`
	KeyStore   string `json:"key_store"`
	KeyCursor  int    `json:"key_cursor"`
	KeyFile    string `json:"key_file"`

	// Paths
	ConfigFile    string `json:"config_file"`
	RootDir       string `json:"root_dir"`
	DbName        string `json:"db_name"`
	DbMem         bool   `json:"db_mem"`
	ContractPath  string `json:"contract_path"`
	GenesisConfig string `json:"genesis_config"`

	// Logs
	LogFile   string `json:"log_file"`
	DebugFile string `json:"debug_file"`
	LogLevel  int    `json:"log_level"`
}

// set default config object
var DefaultConfig = &ChainConfig{
	// Network
	ListenHost: "0.0.0.0",
	ListenPort: 40404,
	Listen:     true,
	RemoteHost: "",
	RemotePort: 40404,
	UseSeed:    false,
	RpcHost:    "",
	RpcPort:    40403,
	ServeRpc:   false,
	FetchPort:  40405,

	// ChainId and Name
	ChainId:   "",
	ChainName: "",

	// Local Node
	MaxPeers:   10,
	Moniker:    "anon",
	Version:    "",
	Network:    "",
	KeySession: "generous",
	KeyStore:   "file",
	KeyCursor:  0,
	KeyFile:    DefaultKeyFile,

	// Paths
	RootDir:       "",
	DbName:        "database",
	DbMem:         false,
	ContractPath:  path.Join(ErisLtd, "eris-std-lib"),
	GenesisConfig: DefaultGenesisConfig,

	// Log
	LogFile:   "",
	DebugFile: "",
	LogLevel:  2,
}

func InitChain() error {
	err := utils.InitDecerverDir()
	if err != nil {
		return err
	}
	err = utils.InitDataDir(Tendermint)
	if err != nil {
		return err
	}
	return nil
	/*err = utils.WriteJson(DefaultConfig, path.Join(utils.Blockchains, "config.json"))
	if err != nil {
		return err
	}
	return utils.WriteJson(monkdoug.DefaultGenesis, path.Join(utils.Blockchains, "genesis.json"))
	*/
}

// Marshal the current configuration to file in pretty json.
func (mint *MintModule) WriteConfig(config_file string) error {
	return utils.WriteJson(mint.Config, config_file)
}

// Unmarshal the configuration file into module's config struct.
func (mint *MintModule) ReadConfig(config_file string) error {
	var config ChainConfig
	if err := utils.ReadJson(&config, config_file); err != nil {
		return err
	}
	*(mint.Config) = config
	return nil
}

// Set a field in the config struct.
func (mint *MintModule) SetProperty(field string, value interface{}) error {
	cv := reflect.ValueOf(mint.Config).Elem()
	return utils.SetProperty(cv, field, value)
}

func (mint *MintModule) Property(field string) interface{} {
	cv := reflect.ValueOf(mint.Config).Elem()
	f := cv.FieldByName(field)
	return f.Interface()
}

// Set the config object directly
func (mint *MintModule) SetConfigObj(config interface{}) error {
	if c, ok := config.(*ChainConfig); ok {
		mint.Config = c
	} else {
		return fmt.Errorf("Invalid config object")
	}
	return nil
}

// Create the root data dir if it doesn't exist, and copy keys if they are available
// TODO: copy a key file
func (mint *MintModule) tendermintConfig() {
	cfg := mint.Config
	// check on data dir
	// create keys
	utils.InitDataDir(cfg.RootDir)
	_, err := os.Stat(path.Join(cfg.RootDir, cfg.KeySession) + ".prv")
	if err != nil {
		utils.Copy(cfg.KeyFile, path.Join(cfg.RootDir, cfg.KeySession)+".prv")
	}
	// if the root dir is the default dir, make sure genesis.json's are available
	// mod.ConfigureGenesis()

	//TODO: logging?
}

// Marshal the current configuration to file in pretty json.
func (mint *MintRPCModule) WriteConfig(config_file string) error {
	return utils.WriteJson(mint.Config, config_file)
}

// Unmarshal the configuration file into module's config struct.
func (mint *MintRPCModule) ReadConfig(config_file string) error {
	var config ChainConfig
	if err := utils.ReadJson(&config, config_file); err != nil {
		return err
	}
	*(mint.Config) = config
	return nil
}

// Set a field in the config struct.
func (mint *MintRPCModule) SetProperty(field string, value interface{}) error {
	cv := reflect.ValueOf(mint.Config).Elem()
	return utils.SetProperty(cv, field, value)
}

func (mint *MintRPCModule) Property(field string) interface{} {
	cv := reflect.ValueOf(mint.Config).Elem()
	f := cv.FieldByName(field)
	return f.Interface()
}

// Set the config object directly
func (mint *MintRPCModule) SetConfigObj(config interface{}) error {
	if c, ok := config.(*ChainConfig); ok {
		mint.Config = c
	} else {
		return fmt.Errorf("Invalid config object")
	}
	return nil
}

// Create the root data dir if it doesn't exist, and copy keys if they are available
// TODO: copy a key file
func (mint *MintRPCModule) tendermintConfig() {
	cfg := mint.Config
	// check on data dir
	// create keys
	utils.InitDataDir(cfg.RootDir)
	_, err := os.Stat(path.Join(cfg.RootDir, cfg.KeySession) + ".prv")
	if err != nil {
		utils.Copy(cfg.KeyFile, path.Join(cfg.RootDir, cfg.KeySession)+".prv")
	}
	// if the root dir is the default dir, make sure genesis.json's are available
	// mod.ConfigureGenesis()

	//TODO: logging?
}

func (mod *MintRPCModule) rConfig() {
	cfg := mod.Config
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
	// TODO: enhance this with more pkg level control
	utils.InitLogging(cfg.RootDir, cfg.LogFile, cfg.LogLevel, cfg.DebugFile)
}
