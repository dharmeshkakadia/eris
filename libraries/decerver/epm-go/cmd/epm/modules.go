package main

import (
	"github.com/eris-ltd/epm-go/chains"
	"github.com/eris-ltd/epm-go/epm"
	"github.com/eris-ltd/epm-go/utils"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/codegangsta/cli"

	// modules
	"github.com/eris-ltd/modules/eth"
	"github.com/eris-ltd/modules/genblock"
	"github.com/eris-ltd/modules/monkrpc"
	"github.com/eris-ltd/thelonious/monk"
	"github.com/eris-ltd/thelonious/monkdoug"
)

func newChain(chainType string, rpc bool) epm.Blockchain {
	switch chainType {
	case "thel", "thelonious", "monk":
		if rpc {
			return monkrpc.NewMonkRpcModule()
		} else {
			return monk.NewMonk(nil)
		}
	case "btc", "bitcoin":
		if rpc {
			log.Fatal("Bitcoin rpc not implemented yet")
		} else {
			log.Fatal("Bitcoin not implemented yet")
		}
	case "eth", "ethereum":
		if rpc {
			log.Fatal("Eth rpc not implemented yet")
		} else {
			return eth.NewEth(nil)
		}
	case "gen", "genesis":
		return genblock.NewGenBlockModule(nil)
	}
	return nil

}

// chainroot is a full path to the dir
func loadChain(c *cli.Context, chainType, chainRoot string) epm.Blockchain {
	rpc := c.GlobalBool("rpc")
	logger.Debugln("Loading chain ", c.String("type"))

	chain := newChain(chainType, rpc)
	setupModule(c, chain, chainRoot)
	return chain
}

// TODO: if we are passed a chainRoot but also db is set
//   we should copy from the chainroot to db
// For now, if a chainroot is provided, we use that dir directly

func configureRootDir(c *cli.Context, m epm.Blockchain, chainRoot string) {
	// we need to overwrite the default monk config with our defaults
	root, _ := filepath.Abs(defaultDatabase)
	m.SetProperty("RootDir", root)

	// if the HEAD is set, it overrides the default
	if typ, c, err := chains.GetHead(); err == nil && c != "" {
		root, _ = chains.ResolveChainDir(typ, c, c)
		m.SetProperty("RootDir", root)
		//path.Join(utils.Blockchains, "thelonious", c)
	}

	// if the chainRoot is set, it overwrites the head
	if chainRoot != "" {
		m.SetProperty("RootDir", chainRoot)
	}

	if c.GlobalBool("rpc") {
		r := m.Property("RootDir").(string)
		last := filepath.Base(r)
		if last != "rpc" {
			m.SetProperty("RootDir", path.Join(r, "rpc"))
		}
	}
}

func readConfigFile(c *cli.Context, m epm.Blockchain) {
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

func applyFlags(c *cli.Context, m epm.Blockchain) {
	// then apply cli flags
	setLogLevel(c, m)
	setKeysFile(c, m)
	setGenesisPath(c, m)
	setContractPath(c, m)
	setMining(c, m)
	setRpc(c, m)
}

func setupModule(c *cli.Context, m epm.Blockchain, chainRoot string) {
	// TODO: kinda bullshit and useless since we set log level at epm
	// m.Config.LogLevel = defaultLogLevel

	configureRootDir(c, m, chainRoot)
	readConfigFile(c, m)
	applyFlags(c, m)
	if c.GlobalBool("config") {
		// write the config to a temp file, open in editor, reload
		tempConfig := path.Join(utils.Epm, "tempconfig.json")
		ifExit(m.WriteConfig(tempConfig))
		ifExit(editor(tempConfig))
		ifExit(m.ReadConfig(tempConfig))
	}

	logger.Infoln("Root directory: ", m.Property("RootDir").(string))

	// initialize and start
	m.Init()
	m.Start()
}

func isThelonious(chain epm.Blockchain) (*monk.MonkModule, bool) {
	th, ok := chain.(*monk.MonkModule)
	return th, ok
}

func setGenesisConfig(m *monk.MonkModule, genesis string) {
	if strings.HasSuffix(genesis, ".pdx") || strings.HasSuffix(genesis, ".gdx") {
		m.GenesisConfig = &monkdoug.GenesisConfig{Address: "0000000000THISISDOUG", NoGenDoug: false, Pdx: genesis}
		m.GenesisConfig.Init()
	} else {
		m.Config.GenesisConfig = genesis
	}
}

func copyEditGenesisConfig(deployGen, tmpRoot string, novi bool) string {
	tempGen := path.Join(tmpRoot, "genesis.json")
	utils.InitDataDir(tmpRoot)

	if deployGen == "" {
		deployGen = path.Join(utils.Blockchains, "thelonious", "genesis.json")
	}
	if _, err := os.Stat(deployGen); err != nil {
		err := utils.WriteJson(monkdoug.DefaultGenesis, deployGen)
		ifExit(err)
	}
	ifExit(utils.Copy(deployGen, tempGen))
	if !novi {
		ifExit(editor(tempGen))
	}
	return tempGen
}
