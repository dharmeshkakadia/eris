package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/eris-ltd/epm-go/epm"
	"os"
	"path/filepath"
)

func setLogLevel(c *cli.Context, m epm.Blockchain) {
	logLevel := c.GlobalInt("log")
	if c.GlobalIsSet("log") {
		m.SetProperty("LogLevel", logLevel)
	}
}

func setKeysFile(c *cli.Context, m epm.Blockchain) {
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

func setGenesisPath(c *cli.Context, m epm.Blockchain) {
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

func setContractPath(c *cli.Context, m epm.Blockchain) {
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

func setMining(c *cli.Context, m epm.Blockchain) {
	tomine := c.Bool("mine")
	if c.IsSet("mine") {
		m.SetProperty("Mining", tomine)
	}
}

func setRpc(c *cli.Context, m epm.Blockchain) {
	if !c.GlobalBool("rpc") {
		return
	}

	if c.GlobalIsSet("host") {
		m.SetProperty("rpc_host", c.GlobalString("host"))
	}
	if c.GlobalIsSet("port") {
		m.SetProperty("rpc_port", c.GlobalString("port"))
	}
	if c.GlobalIsSet("local") {
		m.SetProperty("local", c.GlobalBool("local"))
	}
}

func setDb(c *cli.Context, config *string, dbpath string) {
	var err error
	if c.IsSet("db") {
		*config, err = filepath.Abs(dbpath)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
	}
}

var (
	nameFlag = cli.StringFlag{
		Name:   "name, n",
		Value:  "",
		Usage:  "specify a ref name",
		EnvVar: "",
	}

	forceNameFlag = cli.StringFlag{
		Name:   "force-name, N",
		Value:  "",
		Usage:  "Force a ref name (even if already taken)",
		EnvVar: "",
	}

	chainFlag = cli.StringFlag{
		Name:   "chain",
		Value:  "",
		Usage:  "set the chain by <ref name> or by <type>/<id>",
		EnvVar: "",
	}

	multiFlag = cli.StringFlag{
		Name:  "multi",
		Value: "",
		Usage: "use another version of a chain with the same id",
	}

	typeFlag = cli.StringFlag{
		Name:   "type",
		Value:  "thelonious",
		Usage:  "set the chain type (thelonious, genesis, bitcoin, ethereum)",
		EnvVar: "",
	}

	interactiveFlag = cli.BoolFlag{
		Name:   "i",
		Usage:  "Run epm in interactive mode",
		EnvVar: "",
	}

	diffFlag = cli.BoolFlag{
		Name:   "diff",
		Usage:  "Show a diff of all contract storage",
		EnvVar: "",
	}

	dontClearFlag = cli.BoolFlag{
		Name:   "dont-clear",
		Usage:  "Stop epm from clearing the epm cache on startup",
		EnvVar: "",
	}

	contractPathFlag = cli.StringFlag{
		Name:  "contracts, c",
		Value: defaultContractPath,
		Usage: "set the contract path",
	}

	pdxPathFlag = cli.StringFlag{
		Name:  "p",
		Value: ".",
		Usage: "specify a .pdx file to deploy",
	}

	logLevelFlag = cli.IntFlag{
		Name:   "log",
		Value:  2,
		Usage:  "set the log level",
		EnvVar: "EPM_LOG",
	}

	mineFlag = cli.BoolFlag{
		Name:  "mine, commit",
		Usage: "commit blocks",
	}

	bareFlag = cli.BoolFlag{
		Name:  "bare",
		Usage: "only copy the config",
	}

	rpcFlag = cli.BoolFlag{
		Name:   "rpc",
		Usage:  "run commands over rpc",
		EnvVar: "",
	}

	rpcHostFlag = cli.StringFlag{
		Name:  "host",
		Value: "localhost",
		Usage: "set the rpc host",
	}

	rpcPortFlag = cli.IntFlag{
		Name:  "port",
		Value: 5,
		Usage: "set the rpc port",
	}

	rpcLocalFlag = cli.BoolFlag{
		Name:  "local",
		Usage: "let the rpc server handle keys (sign txs)",
	}

	newCheckoutFlag = cli.BoolFlag{
		Name:  "checkout, o",
		Usage: "checkout the chain into head",
	}
	newConfigFlag = cli.StringFlag{
		Name:  "config, c",
		Usage: "specify config file",
	}
	newGenesisFlag = cli.StringFlag{
		Name:  "genesis, g",
		Usage: "specify genesis file",
	}

	viFlag = cli.BoolFlag{
		Name:  "vi",
		Usage: "edit the config in a vim window",
	}

	editConfigFlag = cli.BoolFlag{
		Name:  "edit-config",
		Usage: "open the config in an editor on epm new",
	}

	runConfigFlag = cli.BoolFlag{
		Name:  "config",
		Usage: "run time config edits",
	}

	noEditFlag = cli.BoolFlag{
		Name:  "no-edit",
		Usage: "prevent genesis.json from popping up (uses default)",
	}

	editGenesisFlag = cli.BoolFlag{
		Name:  "edit, e",
		Usage: "edit the genesis.json even if it is provided",
	}
)
