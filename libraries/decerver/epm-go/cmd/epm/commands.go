package main

import (
	"github.com/codegangsta/cli"
)

var (
	cleanCmd = cli.Command{
		Name:   "clean",
		Usage:  "wipes out the contents of the specified directory in the decerver tree",
		Action: cliClean,
	}

	plopCmd = cli.Command{
		Name:   "plop",
		Usage:  "epm plop <config | genesis | chainid | vars | pid>",
		Action: cliPlop,
		Flags: []cli.Flag{
			chainFlag,
		},
	}

	refsCmd = cli.Command{
		Name:   "refs",
		Usage:  "display and manage chain references",
		Action: cliRefs,
		Subcommands: []cli.Command{
			addRefCmd,
			rmRefCmd,
		},
	}

	cpCmd = cli.Command{
		Name:   "cp",
		Usage:  "copy a blockchain",
		Action: cliCp,
		Flags: []cli.Flag{
			bareFlag,
		},
	}

	headCmd = cli.Command{
		Name:   "head",
		Usage:  "display the current working chain",
		Action: cliHead,
	}

	initCmd = cli.Command{
		Name:   "init",
		Usage:  "initialize the epm tree in ~/.decerver",
		Action: cliInit,
	}

	fetchCmd = cli.Command{
		Name:   "fetch",
		Usage:  "fetch a chain from peer server",
		Action: cliFetch,
		Flags: []cli.Flag{
			nameFlag,
			forceNameFlag,
			newCheckoutFlag,
		},
	}

	newCmd = cli.Command{
		Name:   "new",
		Usage:  "create a new chain and install into the decerver tree",
		Action: cliNew,
		Flags: []cli.Flag{
			newCheckoutFlag,
			newConfigFlag,
			newGenesisFlag,
			nameFlag,
			forceNameFlag,
			typeFlag,
			editConfigFlag,
		},
	}

	checkoutCmd = cli.Command{
		Name:   "checkout",
		Usage:  "change the current working chain",
		Action: cliCheckout,
	}

	addRefCmd = cli.Command{
		Name:   "add",
		Usage:  "add a new reference to a chain id: `epm refs add thel/f8`",
		Action: cliAddRef,
		Flags: []cli.Flag{
			multiFlag,
		},
	}

	rmRefCmd = cli.Command{
		Name:   "rm",
		Usage:  "rm a reference from a chain id, but leave the data",
		Action: cliRmRef,
		Flags: []cli.Flag{
			multiFlag,
		},
	}

	runCmd = cli.Command{
		Name:   "run",
		Usage:  "run a chain by reference or id",
		Action: cliRun,
		Flags: []cli.Flag{
			mineFlag,
			chainFlag,
			multiFlag,
		},
	}

	runDappCmd = cli.Command{
		Name:   "run-dapp",
		Usage:  "run a chain by dapp name",
		Action: cliRunDapp,
		Flags: []cli.Flag{
			mineFlag,
			multiFlag,
		},
	}

	configCmd = cli.Command{
		Name:   "config",
		Usage:  "epm config <config key 1>:<config value 1> <config key 2>:<config value 2> ...",
		Action: cliConfig,
		Flags: []cli.Flag{
			chainFlag,
			multiFlag,
			viFlag,
		},
	}

	commandCmd = cli.Command{
		Name:   "cmd",
		Usage:  "epm cmd deploy contract.lll",
		Action: cliCommand,
		Flags: []cli.Flag{
			chainFlag,
			multiFlag,
			contractPathFlag,
		},
	}

	removeCmd = cli.Command{
		Name:   "rm",
		Usage:  "remove a chain from the global directory",
		Action: cliRemove,
		Flags: []cli.Flag{
			multiFlag,
		},
	}

	deployCmd = cli.Command{
		Name:   "deploy",
		Usage:  "deploy a .pdx file onto a chain",
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

	keygenCmd = cli.Command{
		Name:   "keygen",
		Usage:  "generate secp256k1 keys",
		Action: cliKeygen,
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

	installCmd = cli.Command{
		Name:   "install",
		Usage:  "install a dapp",
		Action: cliInstall,
		Flags: []cli.Flag{
			newConfigFlag,
			newGenesisFlag,
			nameFlag,
			forceNameFlag,
			editConfigFlag,
		},
	}
)
