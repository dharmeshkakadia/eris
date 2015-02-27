package main

import (
	"encoding/hex"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/eris-ltd/epm-go/utils"
	"github.com/eris-ltd/lllc-server"
	"os"
	"runtime"
	"strconv"
)

var logger = lllcserver.Logger{}

// simple lllc-server and cli
func main() {

	app := cli.NewApp()
	app.Name = "lllc-server"
	app.Usage = ""
	app.Version = "0.1.0"
	app.Author = "Ethan Buchman"
	app.Email = "ethan@erisindustries.com"

	app.Action = cliServer
	app.Before = before

	app.Flags = []cli.Flag{
		portFlag,
		internalFlag,
		logFlag,
	}

	app.Commands = []cli.Command{
		cli.Command{
			Name:   "compile",
			Usage:  "compile a contract",
			Action: cliClient,
			Flags: []cli.Flag{
				hostFlag,
				localFlag,
				langFlag,
				//logFlag,
			},
		},
		cli.Command{
			Name:   "proxy",
			Usage:  "run a proxy server for out of process access",
			Action: cliProxy,
			Flags: []cli.Flag{
				portFlag,
			},
		},
	}

	run(app)
}

func before(c *cli.Context) error {
	lllcserver.DebugMode = c.GlobalInt("log")
	return nil
}

func cliClient(c *cli.Context) {
	tocompile := c.Args()[0]

	var err error
	lang := c.String("language")
	if lang == "" {
		lang, err = lllcserver.LangFromFile(tocompile)
		ifExit(err)
	}

	host := c.String("host")
	if host != "" {
		url := host + "/" + "compile"
		lllcserver.SetLanguageURL(lang, url)
	}
	logger.Debugln("language config:", lllcserver.Languages[lang])

	utils.InitDataDir(lllcserver.ClientCache)
	logger.Infoln("compiling", tocompile)
	if c.Bool("local") {
		lllcserver.SetLanguageNet(lang, false)
		//b, err := lllcserver.CompileWrapper(tocompile, lang)
		// force it through the compile pipeline so we get caching
		b, err := lllcserver.Compile(tocompile)
		ifExit(err)
		logger.Warnln("bytecode:", hex.EncodeToString(b))
	} else {
		code, err := lllcserver.Compile(tocompile)
		if err != nil {
			fmt.Println(err)
		}
		logger.Warnln("bytecode:", hex.EncodeToString(code))
	}
}

func cliProxy(c *cli.Context) {
	addr := "localhost:" + strconv.Itoa(c.Int("port"))
	lllcserver.StartProxy(addr)
}

func cliServer(c *cli.Context) {
	utils.InitDataDir(lllcserver.ServerCache)
	addr := ""
	if c.Bool("internal") {
		addr = "localhost"
	}
	addr += ":" + strconv.Itoa(c.Int("port"))
	lllcserver.StartServer(addr)
}

// so we can catch panics
func run(app *cli.App) {
	defer func() {
		if r := recover(); r != nil {
			trace := make([]byte, 2048)
			count := runtime.Stack(trace, true)
			fmt.Println("Panic: ", r)
			fmt.Printf("Stack of %d bytes: %s", count, trace)
		}
	}()

	app.Run(os.Args)
}

var (
	localFlag = cli.BoolFlag{
		Name:  "local",
		Usage: "use local compilers",
	}

	langFlag = cli.StringFlag{
		Name:  "language, l",
		Usage: "language the script is written in",
	}

	logFlag = cli.IntFlag{
		Name:  "log",
		Usage: "set the log level",
		Value: 5,
	}

	portFlag = cli.IntFlag{
		Name:  "port, p",
		Usage: "set the listening port",
		Value: 9099,
	}

	internalFlag = cli.BoolFlag{
		Name:  "internal, i",
		Usage: "only bind localhost (don't expose to internet)",
	}

	hostFlag = cli.StringFlag{
		Name:  "host",
		Usage: "set the server host (inlucde http://)",
		Value: "",
	}
)

func ifExit(err error) {
	if err != nil {
		logger.Errorln(err)
		os.Exit(0)
	}
}
