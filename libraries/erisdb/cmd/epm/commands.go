package main

import (
	"os"
	"os/exec"
	"path"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/eris-ltd/epm-go/chains"
	"github.com/eris-ltd/epm-go/commands"
	"github.com/eris-ltd/epm-go/utils"
)

var standAlones = map[string]struct{}{
	"checkout": struct{}{},
	"clean":    struct{}{},
	"head":     struct{}{},
	"init":     struct{}{},
	"keys":     struct{}{}, // codegangsta/cli doesnt let you reference the super command :(
	"ls":       struct{}{},
	"gen":      struct{}{},
	"pub":      struct{}{},
	"":         struct{}{},
	"rm":       struct{}{},
	"refs":     struct{}{},
}

// wraps a epm-go/commands function in a closure that accepts cli.Context
func cliCall(f func(*commands.Context)) func(*cli.Context) {
	return func(c *cli.Context) {
		c2 := commands.TransformContext(c)
		if _, ok := standAlones[c.Command.Name]; ok {
			f(c2)
		} else {
			var err error
			var typ string
			if c.Command.Name == "new" {
				typ, err = chains.ResolveChainType(c2.String("type"))
				ifExit(err)
			} else if c.Command.Name == "fetch" {
				//
			} else {
				// ensure we are using the correct binary
				_, typ, _, err = commands.ResolveRootFlag(c2)
				if err != nil {
					exit(err)
				}
			}
			if typ != commands.CHAIN {
				// run the proper binary
				// if it does not exist, install it
				bin := path.Join(utils.GoPath, "bin", "epm-"+typ)
				if _, err := os.Stat(bin); err != nil {
					cur, _ := os.Getwd()
					ifExit(os.Chdir(path.Join(utils.ErisLtd, "epm-go")))
					cmd := exec.Command("go", "install", "./cmd/epm-binary-generator")
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					ifExit(cmd.Run())
					cmd = exec.Command(path.Join(utils.GoPath, "bin", "epm-binary-generator"), "./cmd/epm", "commands", typ)
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					ifExit(cmd.Run())
					ifExit(os.Chdir(cur))
				}

				cmd := exec.Command(bin, os.Args[1:]...)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Run()
			} else {
				// go for it
				f(c2)
			}
		}
	}
}

var (

	//
	// INFORMATIONAL COMMANDS
	//
	headCmd = cli.Command{
		Name:   "head",
		Usage:  "display the current working blockchain",
		Action: cliCall(commands.Head),
	}

	plopCmd = cli.Command{
		Name:   "plop",
		Usage:  "machine readable variable display: epm plop <addr | chainid | config | genesis | key | pid | vars>",
		Action: cliCall(commands.Plop),
		Flags: []cli.Flag{
			chainFlag,
		},
	}

	//
	// BLOCKCHAIN WORKSPACE COMMANDS
	//
	refsCmd = cli.Command{
		Name:   "refs",
		Usage:  "display and manage blockchain names (references)",
		Action: cliCall(commands.Refs),
		Subcommands: []cli.Command{
			lsRefCmd,
			addRefCmd,
			rmRefCmd,
		},
	}

	lsRefCmd = cli.Command{
		Name:   "ls",
		Usage:  "list the available blockchains which epm knows about",
		Action: cliCall(commands.Refs),
	}

	addRefCmd = cli.Command{
		Name:   "add",
		Usage:  "add a new reference to a blockchain id: epm refs add <thelonious/f858469a00e80e4f5eb536501eb7d98c7e1cc432>",
		Action: cliCall(commands.AddRef),
		Flags: []cli.Flag{
			multiFlag,
		},
	}

	rmRefCmd = cli.Command{
		Name:   "rm",
		Usage:  "remove a reference to a blockchain, but leave the data in the blockchains tree",
		Action: cliCall(commands.RmRef),
		Flags: []cli.Flag{
			multiFlag,
		},
	}

	checkoutCmd = cli.Command{
		Name:   "checkout",
		Usage:  "change the currently used blockchain",
		Action: cliCall(commands.Checkout),
	}

	//
	// MAKE && GET BLOCKCHAINS
	//
	initCmd = cli.Command{
		Name:   "init",
		Usage:  "initialize the epm tree in ~/.eris",
		Action: cliCall(commands.Init),
	}

	newCmd = cli.Command{
		Name:   "new",
		Usage:  "create a new blockchain and install into the blockchains tree",
		Action: cliCall(commands.New),
		Flags: []cli.Flag{
			newCheckoutFlag,
			newConfigFlag,
			newGenesisFlag,
			nameFlag,
			forceNameFlag,
			typeFlag,
			editConfigFlag,
			noEditFlag,
			editGenesisFlag,
		},
	}

	fetchCmd = cli.Command{
		Name:   "fetch",
		Usage:  "fetch a blockchain from a given peer server",
		Action: cliCall(commands.Fetch),
		Flags: []cli.Flag{
			nameFlag,
			forceNameFlag,
			newCheckoutFlag,
		},
	}

	//
	// OTHER BLOCKCHAIN WORKING COMMANDS
	//
	cleanCmd = cli.Command{
		Name:   "clean",
		Usage:  "wipes out the contents of the specified directory in the eris directory tree",
		Action: cliCall(commands.Clean),
	}

	cpCmd = cli.Command{
		Name:   "cp",
		Usage:  "make a copy of a blockchain",
		Action: cliCall(commands.Cp),
		Flags: []cli.Flag{
			bareFlag,
		},
	}

	configCmd = cli.Command{
		Name:   "config",
		Usage:  "configure epm variables in the blockchain's config.json: epm config <config key 1>:<config value 1> <config key 2>:<config value 2> ...",
		Action: cliCall(commands.Config),
		Flags: []cli.Flag{
			chainFlag,
			multiFlag,
			viFlag,
		},
	}

	commandCmd = cli.Command{
		Name:   "cmd",
		Usage:  "run a command (useful when combined with RPC): epm cmd <deploy contract.lll>",
		Action: cliCall(commands.Command),
		Flags: []cli.Flag{
			chainFlag,
			multiFlag,
			contractPathFlag,
		},
	}

	removeCmd = cli.Command{
		Name:   "rm",
		Usage:  "remove a blockchain reference as well as its data from the blockchains tree",
		Action: cliCall(commands.Remove),
		Flags: []cli.Flag{
			multiFlag,
			forceRmFlag,
		},
	}

	//
	// BLOCKCHAIN OPERATION COMMANDS
	//
	runCmd = cli.Command{
		Name:   "run",
		Usage:  "run a blockchain by reference or id",
		Action: cliCall(commands.Run),
		Flags: []cli.Flag{
			mineFlag,
			chainFlag,
			multiFlag,
		},
	}

	runDappCmd = cli.Command{
		Name:   "run-dapp",
		Usage:  "run a blockchain by dapp name",
		Action: cliCall(commands.RunDapp),
		Flags: []cli.Flag{
			mineFlag,
			multiFlag,
		},
	}

	serveCmd = cli.Command{
		Name:   "serve",
		Usage:  "run and serve a blockchain",
		Action: serve,
		Flags:  []cli.Flag{},
	}

	//
	// SMART CONTRACT COMMANDS
	//
	deployCmd = cli.Command{
		Name:   "deploy",
		Usage:  "deploy a .pdx file onto a blockchain",
		Action: cliCall(commands.Deploy),
		Flags: []cli.Flag{
			chainFlag,
			multiFlag,
			diffFlag,
			dontClearFlag,
			contractPathFlag,
		},
	}

	consoleCmd = cli.Command{
		Name:   "console",
		Usage:  "run epm in interactive mode",
		Action: cliCall(commands.Console),
		Flags: []cli.Flag{
			chainFlag,
			multiFlag,
			diffFlag,
			dontClearFlag,
			contractPathFlag,
		},
	}

	installCmd = cli.Command{
		Name:   "install",
		Usage:  "install a dapp into the eris working tree and add a new blockchain with the same reference",
		Action: cliCall(commands.Install),
		Flags: []cli.Flag{
			newConfigFlag,
			newGenesisFlag,
			nameFlag,
			forceNameFlag,
			editConfigFlag,
			noNewChainFlag,
		},
	}

	//
	// KEYS -- RELUCTANTLY
	//
	keysCmd = cli.Command{
		Name:   "keys",
		Usage:  "generate, import, and export keys for your blockchains",
		Action: cliCall(commands.LsKeys),
		Subcommands: []cli.Command{
			keygenCmd,
			keyLsCmd,
			keyUseCmd,
			keyExportCmd,
			keyImportCmd,
			keyPubCmd,
		},
	}

	keyLsCmd = cli.Command{
		Name:   "ls",
		Usage:  "list the keys and their associated names and addresses",
		Action: cliCall(commands.LsKeys),
	}

	keygenCmd = cli.Command{
		Name:   "gen",
		Usage:  "generate secp256k1 keys",
		Action: cliCall(commands.Keygen),
		Flags: []cli.Flag{
			importFlag,
			keyTypeFlag,
		},
	}

	keyUseCmd = cli.Command{
		Name:   "use",
		Usage:  "use a particular key with the currently checked out blockchain",
		Action: cliCall(commands.KeyUse),
	}

	keyExportCmd = cli.Command{
		Name:   "export",
		Usage:  "export a key file",
		Action: cliCall(commands.KeyExport),
		Flags:  []cli.Flag{},
	}

	keyImportCmd = cli.Command{
		Name:   "import",
		Usage:  "import a key file",
		Action: cliCall(commands.KeyImport),
		Flags:  []cli.Flag{},
	}

	keyPubCmd = cli.Command{
		Name:   "pub",
		Usage:  "print the public key associated with some address",
		Action: cliCall(commands.KeyPublic),
		Flags: []cli.Flag{
			keyTypeFlag,
		},
	}

	testCmd = cli.Command{
		Name:   "test",
		Usage:  "run all pdx/pdt in the directory",
		Action: cliCall(commands.Test),
		Flags: []cli.Flag{
			chainFlag,
			contractPathFlag,
		},
	}

	accountsCmd = cli.Command{
		Name:   "accounts",
		Usage:  "List all accounts, or dump the storage of a specified one",
		Action: cliCall(commands.Accounts),
		Flags:  []cli.Flag{},
	}
)
