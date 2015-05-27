package main

import (
	"fmt"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/epm"
	"github.com/eris-ltd/epm-go/server"
	"github.com/eris-ltd/epm-go/utils"
	"os"
	"path"
	"runtime"
)

func main() {

	app := cli.NewApp()
	app.Name = "epm"
	app.Usage = "The Eris Package Manager Builds, Tests, Operates, and Manages Blockchains and Smart Contract Systems"
	app.Version = "0.9.0"
	app.Author = "Ethan Buchman"
	app.Email = "support@erisindustries.com"
	//	app.EnableBashCompletion = true // this is buggy

	app.Before = before
	app.Flags = []cli.Flag{
		// which chain
		chainFlag,

		// log
		logLevelFlag,

		// rpc
		rpcFlag,
		rpcHostFlag,
		rpcPortFlag,
		rpcLocalFlag,

		// languages
		compilerFlag,

		// runtime configuration
		runConfigFlag,
	}

	app.Commands = []cli.Command{
		accountsCmd,
		checkoutCmd,
		cleanCmd,
		commandCmd,
		configCmd,
		consoleCmd,
		cpCmd,
		deployCmd,
		fetchCmd,
		headCmd,
		initCmd,
		installCmd,
		keysCmd,
		newCmd,
		plopCmd,
		refsCmd,
		removeCmd,
		runCmd,
		runDappCmd,
		serveCmd,
		testCmd,
	}

	run(app)

	monklog.Flush()
}

func before(c *cli.Context) error {
	utils.InitLogging(path.Join(utils.Logs, "epm"), "", c.Int("log"), "")
	if _, err := os.Stat(utils.Decerver); err != nil {
		exit(fmt.Errorf("Could not find eris directory tree. Did you run `epm init`?"))
	}
	if c.GlobalIsSet("compiler") {
		epm.SetCompilerServer(c.GlobalString("compiler"))
	}

	return nil
}

// so we can catch panics
func run(app *cli.App) {
	defer func() {
		if r := recover(); r != nil {
			trace := make([]byte, 2048)
			count := runtime.Stack(trace, true)
			fmt.Printf("Panic: ", r)
			fmt.Printf("Stack of %d bytes: %s", count, trace)
		}
		monklog.Flush()
	}()

	app.Run(os.Args)
}

func exit(err error) {
	monklog.Flush()
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(0)
}

func ifExit(err error) {
	if err != nil {
		monklog.Flush()
		fmt.Println(err)
		os.Exit(1)
	}
}

// this is here to resolve circular dependencies between commands and server packages
func serve(c *cli.Context) {
	srvr := server.NewServer("localhost", 3000, 1000, ".")
	srvr.Start()
}
