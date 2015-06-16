package main

import (
	"encoding/hex"
	"fmt"
	"github.com/eris-ltd/lllc-server/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/eris-ltd/lllc-server/Godeps/_workspace/src/github.com/eris-ltd/epm-go/utils"
	"github.com/eris-ltd/lllc-server"
	"log"
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
	app.Version = "0.9.0"
	app.Author = "Ethan Buchman"
	app.Email = "ethan@erisindustries.com"

	app.Action = cliServer
	app.Before = before

	app.Flags = []cli.Flag{
		securePortFlag,
		unsecurePortFlag,
		unsecureOnlyFlag,
		secureOnlyFlag,
		certFlag,
		keyFlag,
		internalFlag,
		logFlag,
		hostFlag,
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
	if len(c.Args()) == 0 {
		log.Fatal("Specify a contract to compile")
	}
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
		b, abi, err := lllcserver.Compile(tocompile)
		ifExit(err)
		logger.Warnln("bytecode:", hex.EncodeToString(b))
		logger.Warnln("abi:", abi)
	} else {
		code, abi, err := lllcserver.Compile(tocompile)
		if err != nil {
			fmt.Println(err)
		}
		logger.Warnln("bytecode:", hex.EncodeToString(code))
		logger.Warnln("abi:", abi)
	}
}

func cliProxy(c *cli.Context) {
	addr := "localhost:" + strconv.Itoa(c.Int("port"))
	lllcserver.StartProxy(addr)
}

func cliServer(c *cli.Context) {

	utils.InitDataDir(lllcserver.ServerCache)
	addrUnsecure := ""
	addrSecure := ""

	if c.Bool("internal") {
		addrUnsecure = "localhost"
		addrSecure = "localhost"
	}

	addrUnsecure += ":" + strconv.Itoa(c.Int("unsecure-port"))
	addrSecure += ":" + strconv.Itoa(c.Int("secure-port"))

	if c.Bool("secure-only") {
		addrUnsecure = ""
	}
	if c.Bool("no-ssl") {
		addrSecure = ""
	}

	key := c.String("key")
	cert := c.String("cert")

	if !c.Bool("no-ssl") {

		if _, err := os.Stat(key); os.IsNotExist(err) {
			ifExit(err)
		}
		if _, err := os.Stat(cert); os.IsNotExist(err) {
			ifExit(err)
		}

	}

	lllcserver.StartServer(addrUnsecure, addrSecure, key, cert)
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
		Name:  "port",
		Usage: "set the proxy port",
		Value: 9097,
	}

	unsecurePortFlag = cli.IntFlag{
		Name:  "unsecure-port, p",
		Usage: "set the listening port",
		Value: 9099,
	}

	securePortFlag = cli.IntFlag{
		Name:  "secure-port, P",
		Usage: "set the listening port",
		Value: 9098,
	}

	secureOnlyFlag = cli.BoolFlag{
		Name:  "secure-only, s",
		Usage: "only use https",
	}

	unsecureOnlyFlag = cli.BoolFlag{
		Name:  "no-ssl",
		Usage: "do not use ssl",
		EnvVar: "NO_SSL",
	}

	certFlag = cli.StringFlag{
		Name:  "cert",
		Usage: "set the https certificate",
		Value: "",
	}

	keyFlag = cli.StringFlag{
		Name:  "key",
		Usage: "set the https certificate",
		Value: "",
	}

	internalFlag = cli.BoolFlag{
		Name:  "internal, i",
		Usage: "only bind localhost (don't expose to internet)",
	}

	hostFlag = cli.StringFlag{
		Name:  "host",
		Usage: "set the server host (include http(s)://)",
		Value: "",
		EnvVar: "HOST",
	}
)

func ifExit(err error) {
	if err != nil {
		logger.Errorln(err)
		os.Exit(0)
	}
}
