package commands

import (
	"github.com/eris-ltd/epm-go/chains"
	"github.com/eris-ltd/epm-go/epm"
	"github.com/eris-ltd/epm-go/utils"
	"os"
	"path"
	"path/filepath"

	//epm-binary-generator:IMPORT
	mod "github.com/eris-ltd/epm-go/commands/modules/thelonious"
)

// this needs to match the type of the chain we're trying to run
// it should be blank for the base epm (even though it includes thel ...)
//epm-binary-generator:CHAIN
const CHAIN = ""

// chainroot is a full path to the dir
func LoadChain(c *Context, chainType, chainRoot string) (epm.Blockchain, error) {
	rpc := c.Bool("rpc")
	logger.Debugln("Loading chain ", c.String("type"))

	chain := mod.NewChain(chainType, rpc)
	err := setupModule(c, chain, chainRoot)
	return chain, err
}

func configureRootDir(c *Context, m epm.Blockchain, chainRoot string) error {
	// we need to overwrite the default monk config with our defaults
	root, _ := filepath.Abs(defaultDatabase)
	m.SetProperty("RootDir", root)

	// if the HEAD is set, it overrides the default
	if typ, c, err := chains.GetHead(); err == nil && c != "" {
		root, _ = chains.ResolveChainDir(typ, c, c)
		m.SetProperty("RootDir", root)
	}

	// if the chainRoot is set, it overwrites the head
	if chainRoot != "" {
		m.SetProperty("RootDir", chainRoot)
	}

	// make rpc dir and copy config
	r := m.Property("RootDir").(string)
	if err := makeRPCDir(r); err != nil {
		return err
	}

	if c.Bool("rpc") {
		last := filepath.Base(r)
		if last != "rpc" {
			r = path.Join(r, "rpc")
		}
	}

	m.SetProperty("RootDir", r)
	return nil
}

func readConfigFile(c *Context, m epm.Blockchain) {
	// if there's a config file in the root dir, use that
	// else fall back on default or flag
	// TODO: switch those priorities around!
	configFlag := c.String("config")
	s := path.Join(m.Property("RootDir").(string), "config.json")
	if _, err := os.Stat(s); err == nil {
		m.ReadConfig(s)
	} else {
		m.ReadConfig(configFlag)
	}
}

func applyFlags(c *Context, m epm.Blockchain) {
	// then apply flags
	setLogLevel(c, m)
	setKeysFile(c, m)
	setGenesisPath(c, m)
	setContractPath(c, m)
	setMining(c, m)
	setRpc(c, m)
}

func setupModule(c *Context, m epm.Blockchain, chainRoot string) error {
	// TODO: kinda bullshit and useless since we set log level at epm
	// m.Config.LogLevel = defaultLogLevel

	if err := configureRootDir(c, m, chainRoot); err != nil {
		return err
	}

	readConfigFile(c, m)
	applyFlags(c, m)
	if c.Bool("config") {
		// write the config to a temp file, open in editor, reload
		tempConfig := path.Join(utils.Epm, "tempconfig.json")
		ifExit(m.WriteConfig(tempConfig))
		ifExit(utils.Editor(tempConfig))
		ifExit(m.ReadConfig(tempConfig))
	}

	rootDir := m.Property("RootDir").(string)
	logger.Infoln("Root directory: ", rootDir)

	// initialize and start
	if err := m.Init(); err != nil {
		return err
	}
	if err := m.Start(); err != nil {
		return err
	}

	return nil
}
