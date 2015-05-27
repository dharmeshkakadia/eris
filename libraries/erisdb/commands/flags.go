package commands

import (
	"fmt"
	"github.com/eris-ltd/epm-go/epm"
	"os"
	"path/filepath"
)

func setLogLevel(c *Context, m epm.Blockchain) {
	logLevel := c.Int("log")
	if c.IsSet("log") {
		m.SetProperty("LogLevel", logLevel)
	}
}

func setKeysFile(c *Context, m epm.Blockchain) {
	keys := c.String("keys")
	if c.IsSet("k") {
		//if keyfile != defaultKeys {
		keysAbs, err := filepath.Abs(keys)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
		m.SetProperty("KeyFile", keysAbs)
	}
}

func setGenesisPath(c *Context, m epm.Blockchain) {
	genesis := c.String("genesis")
	if c.IsSet("genesis") {
		//if *config != defaultGenesis && genfile != "" {
		genAbs, err := filepath.Abs(genesis)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
		m.SetProperty("GenesisPath", genAbs)
	}
}

func setContractPath(c *Context, m epm.Blockchain) {
	contractPath := c.String("c")
	if c.IsSet("c") {
		//if cpath != defaultContractPath {
		cPathAbs, err := filepath.Abs(contractPath)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
		m.SetProperty("ContractPath", cPathAbs)
	}
}

func setMining(c *Context, m epm.Blockchain) {
	tomine := c.Bool("mine")
	if c.IsSet("mine") {
		m.SetProperty("Mining", tomine)
	}
}

func setRpc(c *Context, m epm.Blockchain) {
	if !c.Bool("rpc") {
		return
	}

	if c.IsSet("host") {
		m.SetProperty("rpc_host", c.String("host"))
	}
	if c.IsSet("port") {
		fmt.Println(c.Int("port"))
		m.SetProperty("rpc_port", c.Int("port"))
	}
	if c.IsSet("local") {
		m.SetProperty("local", c.Bool("local"))
	}
}

func setDb(c *Context, config *string, dbpath string) {
	var err error
	if c.IsSet("db") {
		*config, err = filepath.Abs(dbpath)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
	}
}
