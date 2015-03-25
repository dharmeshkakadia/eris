package main

import (
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/codegangsta/cli"
)

var (

	//
	// INFORMATIONAL COMMANDS
	//
	headCmd = cli.Command{
		Name:   "head",
		Usage:  "display the current working blockchain",
		Action: cliHead,
	}

	plopCmd = cli.Command{
		Name:   "plop",
		Usage:  "machine readable variable display: epm plop <addr | chainid | config | genesis | key | pid | vars>",
		Action: cliPlop,
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
		Action: cliRefs,
		Subcommands: []cli.Command{
			lsRefCmd,
			addRefCmd,
			rmRefCmd,
		},
	}

	lsRefCmd = cli.Command{
		Name:   "ls",
		Usage:  "list the available blockchains which epm knows about",
		Action: cliRefs,
	}

	addRefCmd = cli.Command{
		Name:   "add",
		Usage:  "add a new reference to a blockchain id: epm refs add <thelonious/f858469a00e80e4f5eb536501eb7d98c7e1cc432>",
		Action: cliAddRef,
		Flags: []cli.Flag{
			multiFlag,
		},
	}

	rmRefCmd = cli.Command{
		Name:   "rm",
		Usage:  "remove a reference to a blockchain, but leave the data in the blockchains tree",
		Action: cliRmRef,
		Flags: []cli.Flag{
			multiFlag,
		},
	}

	checkoutCmd = cli.Command{
		Name:   "checkout",
		Usage:  "change the currently used blockchain",
		Action: cliCheckout,
	}

	//
	// MAKE && GET BLOCKCHAINS
	//
	initCmd = cli.Command{
		Name:   "init",
		Usage:  "initialize the epm tree in ~/.decerver",
		Action: cliInit,
	}

	newCmd = cli.Command{
		Name:   "new",
		Usage:  "create a new blockchain and install into the blockchains tree",
		Action: cliNew,
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
		Action: cliFetch,
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
		Usage:  "wipes out the contents of the specified directory in the decerver tree",
		Action: cliClean,
	}

	cpCmd = cli.Command{
		Name:   "cp",
		Usage:  "make a copy of a blockchain",
		Action: cliCp,
		Flags: []cli.Flag{
			bareFlag,
		},
	}

	configCmd = cli.Command{
		Name:   "config",
		Usage:  "configure epm variables in the blockchain's config.json: epm config <config key 1>:<config value 1> <config key 2>:<config value 2> ...",
		Action: cliConfig,
		Flags: []cli.Flag{
			chainFlag,
			multiFlag,
			viFlag,
		},
	}

	commandCmd = cli.Command{
		Name:   "cmd",
		Usage:  "run a command (useful when combined with RPC): epm cmd <deploy contract.lll>",
		Action: cliCommand,
		Flags: []cli.Flag{
			chainFlag,
			multiFlag,
			contractPathFlag,
		},
	}

	removeCmd = cli.Command{
		Name:   "rm",
		Usage:  "remove a blockchain reference as well as its data from the blockchains tree",
		Action: cliRemove,
		Flags: []cli.Flag{
			multiFlag,
		},
	}

	//
	// BLOCKCHAIN OPERATION COMMANDS
	//
	runCmd = cli.Command{
		Name:   "run",
		Usage:  "run a blockchain by reference or id",
		Action: cliRun,
		Flags: []cli.Flag{
			mineFlag,
			chainFlag,
			multiFlag,
		},
	}

	runDappCmd = cli.Command{
		Name:   "run-dapp",
		Usage:  "run a blockchain by dapp name",
		Action: cliRunDapp,
		Flags: []cli.Flag{
			mineFlag,
			multiFlag,
		},
	}

	//
	// SMART CONTRACT COMMANDS
	//
	deployCmd = cli.Command{
		Name:   "deploy",
		Usage:  "deploy a .pdx file onto a blockchain",
		Action: cliDeploy,
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
		Action: cliConsole,
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
		Usage:  "install a dapp into the decerver working tree and add a new blockchain with the same reference",
		Action: cliInstall,
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
		Action: cliLsKeys,
		Subcommands: []cli.Command{
			keygenCmd,
			keyLsCmd,
			keyUseCmd,
			keyExportCmd,
			keyImportCmd,
		},
	}

	keyLsCmd = cli.Command{
		Name:   "ls",
		Usage:  "list the keys and their associated names and addresses",
		Action: cliLsKeys,
	}

	keygenCmd = cli.Command{
		Name:   "gen",
		Usage:  "generate secp256k1 keys",
		Action: cliKeygen,
		Flags: []cli.Flag{
			noImportFlag,
		},
	}

	keyUseCmd = cli.Command{
		Name:   "use",
		Usage:  "use a particular key with the currently checked out blockchain",
		Action: cliKeyUse,
	}

	keyExportCmd = cli.Command{
		Name:   "export",
		Usage:  "export a key file",
		Action: cliKeyExport,
		Flags:  []cli.Flag{},
	}

	keyImportCmd = cli.Command{
		Name:   "import",
		Usage:  "import a key file",
		Action: cliKeyImport,
		Flags:  []cli.Flag{},
	}

	testCmd = cli.Command{
		Name:   "test",
		Usage:  "run all pdx/pdt in the directory",
		Action: cliTest,
		Flags: []cli.Flag{
			chainFlag,
			contractPathFlag,
		},
	}

	accountsCmd = cli.Command{
		Name:   "accounts",
		Usage:  "List all accounts, or dump the storage of a specified one",
		Action: cliAccounts,
		Flags:  []cli.Flag{},
	}
)
