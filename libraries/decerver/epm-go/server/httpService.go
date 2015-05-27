package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/go-martini/martini"
	"github.com/eris-ltd/epm-go/chains"
	"github.com/eris-ltd/epm-go/commands"
	"github.com/eris-ltd/epm-go/epm"
	"io"
	"io/ioutil"
	"net/http"
	// "net/rpc"
	// "net/rpc/jsonrpc"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"time"
)

// The default return when a requested URL does not match one of the handlers
const EPM_HELP = "That API endpoint does not exist. Please see the epm documentation."

// The HttpService object.
type HttpService struct {
	Router               *martini.Router
	ChainIsRunning       bool
	chainIsRestarting    bool
	ChainRunningName     string
	ChainRunningConfig   ChainConfig
	ChainShutDownChannel chan bool
	ChainIsShutDown      chan bool
	Chain                epm.Blockchain
}

type ChainConfig struct {
	ServeRPC  bool   `json:"serve_rpc"`
	RPCIp     string `json:"rpc_host"`
	RPCPort   int    `json:"rpc_port"`
	FetchPort int    `json:"fetch_port"`
}

// Create a new http service
func NewHttpService(cm martini.Router) *HttpService {
	h := &HttpService{}

	h.Router = &cm
	h.ChainIsRunning = false
	h.chainIsRestarting = false

	h.ChainShutDownChannel = make(chan bool, 1)
	h.ChainIsShutDown = make(chan bool, 1)

	chainShutDownViaOS := make(chan os.Signal, 1)

	signal.Notify(chainShutDownViaOS, os.Interrupt, os.Kill)
	go func() {
		<-chainShutDownViaOS
		h.CleanUpAndExit()
	}()

	return h
}

// -----------------------------------------------------------------
// ------------------- INFORMATIONAL HANDLERS ----------------------
// -----------------------------------------------------------------

// This API endpoint is equivalent to `epm plop` command. Note that
// keys are not returnable for obvious reasons.
func (this *HttpService) handlePlop(params martini.Params, w http.ResponseWriter, r *http.Request) {
	this.logIncoming("Plopping")

	if params["toPlop"] == "key" {
		this.writeMsg(w, 401, "Key is not exportable")
		return
	}

	cmdRaw := []string{"--chain", params["chainName"], "plop", params["toPlop"]}
	this.executeCommand(cmdRaw, w)
}

// This API endpoint is equivalent to `epm refs ls` command.
func (this *HttpService) handleLsRefs(params martini.Params, w http.ResponseWriter, r *http.Request) {
	this.logIncoming("List References")
	cmdRaw := []string{"refs", "ls"}
	this.executeCommand(cmdRaw, w)
}

// This API endpoint is equivalent to `epm refs add` command.
func (this *HttpService) handleAddRefs(params martini.Params, w http.ResponseWriter, r *http.Request) {
	this.logIncoming("Add a Reference")
	toAdd := params["chainType"] + "/" + params["chainId"]
	cmdRaw := []string{"refs", "add", toAdd, params["chainName"]}
	this.executeCommand(cmdRaw, w)
}

// This API endpoint is equivalent to `epm refs rm` command.
func (this *HttpService) handleRmRefs(params martini.Params, w http.ResponseWriter, r *http.Request) {
	this.logIncoming("Remove a Reference")
	cmdRaw := []string{"refs", "rm", params["chainName"]}
	this.executeCommand(cmdRaw, w)
}

// -----------------------------------------------------------------
// ------------------- CHAIN MANAGEMENT HANDLERS -------------------
// -----------------------------------------------------------------

// This API endpoint is equivalent to `epm config` command.
// It will parse the parameters passed via standard URL syntax.
func (this *HttpService) handleConfig(params martini.Params, w http.ResponseWriter, r *http.Request) {
	this.logIncoming("Save Config Values")

	configs, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		this.logError(w, 400, err)
		return
	}

	cmdRaw := []string{"config", "--chain", params["chainName"]}
	for k, v := range configs {
		toAdd := k + ":" + v[0]
		cmdRaw = append(cmdRaw, toAdd)
	}

	this.executeCommand(cmdRaw, w)
}

// This API endpoint is equivalent to `epm checkout`.
func (this *HttpService) handleCheckout(params martini.Params, w http.ResponseWriter, r *http.Request) {
	this.logIncoming("Checkout a Chain")
	cmdRaw := []string{"checkout", params["chainName"]}
	this.executeCommand(cmdRaw, w)
}

// This API endpoint is equivalent to `epm clean --force`.
func (this *HttpService) handleClean(params martini.Params, w http.ResponseWriter, r *http.Request) {
	this.logIncoming("Removing a Chain from the Tree")
	cmdRaw := []string{"rm", "--force", params["chainName"]}
	this.executeCommand(cmdRaw, w)
}

// -----------------------------------------------------------------
// ------------------- BLOCKCHAIN ADMIN HANDLERS -------------------
// -----------------------------------------------------------------

// This API endpoint is equivalent to `epm fetch`.
func (this *HttpService) handleFetchChain(params martini.Params, w http.ResponseWriter, r *http.Request) {
	this.logIncoming("Fetchin a Blockchain")

	toAdd := params["fetchIP"] + ":" + params["fetchPort"]
	chainName := params["chainName"]
	toCheckout := r.URL.Query().Get("checkout")

	cmdRaw := []string{"fetch", toAdd, "--checkout"}
	if toCheckout == "false" {
		cmdRaw = cmdRaw[:2]
	}
	if chainName != "" {
		cmdRaw = append(cmdRaw, "--force-name", chainName)
	}

	this.executeCommand(cmdRaw, w)
}

// This API endpoint is equivalent to `epm new`.
func (this *HttpService) handleNewChain(params martini.Params, w http.ResponseWriter, r *http.Request) {
	this.logIncoming("Making a new Blockchain")

	// Read the genesis.json passed in to a temp file
	var hasGenesis bool
	genesis, err := ioutil.ReadAll(r.Body)
	if err != nil {
		this.logError(w, 400, err)
		return
	}
	defer r.Body.Close()

	var genesisFile *os.File
	defer genesisFile.Close()

	if len(genesis) != 0 {
		hasGenesis = true

		genesisFile, err = ioutil.TempFile(os.TempDir(), "epm-serve-")
		if err != nil {
			this.logError(w, 500, err)
			return
		}

		err = ioutil.WriteFile(genesisFile.Name(), genesis, 644)
		if err != nil {
			this.logError(w, 500, err)
			return
		}
	} else {
		hasGenesis = false
	}

	chainName := params["chainName"]
	chainType := r.URL.Query().Get("type")
	toCheckout := r.URL.Query().Get("checkout")
	cmdRaw := []string{"new", "--no-edit", "--checkout"}
	if toCheckout == "false" {
		cmdRaw = cmdRaw[:2]
	}
	if chainName != "" {
		cmdRaw = append(cmdRaw, "--force-name", chainName)
	}
	if chainType == "" {
		cmdRaw = append(cmdRaw, "--type", "thelonious")
	} else {
		cmdRaw = append(cmdRaw, "--type", chainType)
	}
	if hasGenesis {
		cmdRaw = append(cmdRaw, "--genesis", genesisFile.Name())
	}

	this.executeCommand(cmdRaw, w)
}

// This API endpoint is equivalent to `epm run`.
func (this *HttpService) handleStartChain(params martini.Params, w http.ResponseWriter, r *http.Request) {
	this.logIncoming("Starting Chain Runner")

	if !this.ChainIsRunning {

		c := &commands.Context{
			Arguments: []string{},
			Strings:   make(map[string]string),
			Integers:  make(map[string]int),
			Booleans:  make(map[string]bool),
			HasSet:    make(map[string]struct{}),
		}

		root, chainType, _, err := commands.ResolveRootFlag(c)
		if err != nil {
			this.logError(w, 500, err)
			return
		}

		if !this.chainIsRestarting {
			this.setupRPC(root, w, r)
		}

		logLevel := r.URL.Query().Get("log")
		if logLevel == "" {
			logLevel = "2"
		}
		toCommit := r.URL.Query().Get("commit")

		go func() {
			c.Integers["log"], _ = strconv.Atoi(logLevel)
			c.Set("log")
			if toCommit == "true" {
				c.Booleans["mine"] = true
				c.Set("mine")
			}
			this.ChainRunningName = params["chainName"]

			this.logInfo("Starting Blockchain with log level: " + logLevel)
			var err error
			this.Chain, err = commands.LoadChain(c, chainType, root)

			if err == nil {
				<-this.ChainShutDownChannel

				this.logInfo("Shutting Down Chain")
				this.Chain.Shutdown()
				this.Chain.WaitForShutdown()
			} else {
				this.logInfo("Chain Could Not Be Started.")
			}

			this.ChainIsShutDown <- true
		}()

		this.ChainIsRunning = true

	} else {

		this.writeMsg(w, 500, "A blockchain is already running.")
		return

	}

	if !this.chainIsRestarting {
		this.writeMsg(w, 200, "Blockchain started.")
	}
}

// This API endpoint is equivalent to `kill -SIGTERM $(epm plop chainid)`.
func (this *HttpService) handleStopChain(params martini.Params, w http.ResponseWriter, r *http.Request) {
	this.logIncoming("Stopping Chain Runner")

	// First check if there is a running chain via in process check.
	if this.ChainIsRunning && this.ChainRunningName == params["chainName"] {

		this.ChainShutDownChannel <- true
		<-this.ChainIsShutDown
		this.ChainIsRunning = false

		if !this.chainIsRestarting {
			this.writeMsg(w, 200, "Blockchain stopped.")
		}

		return

	} else if this.ChainIsRunning {

		this.writeMsg(w, 400, "Improper chain name")
		return

	}

	// If `epm serve` did not start a blockchain, check if there
	// is a pid file in the checked out blockchain's folder
	// which would mean that there is a running blockchain which
	// was started by the cli.
	cmdRaw := []string{"plop", "chainid"}
	toTrim, err := this.executeCommandRaw(cmdRaw, w)
	if err != nil {
		this.logError(w, 400, err)
		return
	}
	chainId := strings.TrimSpace(toTrim)

	chainType := r.URL.Query().Get("type")
	if chainType == "" {
		chainType = "thelonious"
	}

	chainDir := chains.ComposeRoot(chainType, chainId)
	pidFile := path.Join(chainDir, "pid")
	if _, err := os.Stat(pidFile); err != nil {
		err := fmt.Errorf("There was no blockchain running.")
		this.logError(w, 500, err)
		return
	}

	var pidInt int
	var chainProcess *os.Process

	pid, err := ioutil.ReadFile(pidFile)
	if err != nil {
		this.logError(w, 500, err)
		return
	}

	pidInt, err = strconv.Atoi(string(pid))
	if err != nil {
		this.logError(w, 500, err)
		return
	}

	chainProcess, err = os.FindProcess(pidInt)
	if err != nil {
		this.logError(w, 500, err)
		return
	}
	chainProcess.Signal(os.Interrupt)

	if !this.chainIsRestarting {
		this.writeMsg(w, 200, "Blockchain stopped.")
	}
}

// This API endpoint is equivalent to `kill -SIGTERM $(epm plop chainid) && sleep 5 && epm run`
func (this *HttpService) handleRestartChain(params martini.Params, w http.ResponseWriter, r *http.Request) {
	this.logIncoming("Restarting Chain Runner")

	if this.ChainIsRunning {

		this.chainIsRestarting = true
		this.handleStopChain(params, w, r)
		time.Sleep(5 * time.Second)
		this.handleStartChain(params, w, r)
		this.chainIsRestarting = false

	} else {

		err := fmt.Errorf("There was no blockchain running.")
		this.logError(w, 500, err)
		return

	}

	this.writeMsg(w, 200, "Blockchain restarted.")
}

// This API endpoint has no equivalent in the cli.
func (this *HttpService) handleChainStatus(params martini.Params, w http.ResponseWriter, r *http.Request) {
	this.logIncoming("Chain Running Status")

	if this.ChainIsRunning {
		if this.ChainRunningName == params["chainName"] {
			this.writeMsg(w, 200, "true")
			return
		}
	}

	this.writeMsg(w, 200, "false")
}

// -----------------------------------------------------------------
// ------------------- KEYS HANDLERS -------------------------------
// -----------------------------------------------------------------

// This API endpoint will save the POSTed key and import it to the
// checked out blockchain.
func (this *HttpService) handleKeyImport(params martini.Params, w http.ResponseWriter, r *http.Request) {
	this.logIncoming("Keys Import")

	keyFileRaw, _, err := r.FormFile("key")
	if err != nil {
		this.logError(w, 400, err)
		return
	}

	keyFile, _ := ioutil.TempFile("", "key-")
	defer keyFile.Close()
	_, err = io.Copy(keyFile, keyFileRaw)
	if err != nil {
		this.logError(w, 500, err)
		return
	}

	newName := path.Join(os.TempDir(), params["keyName"])
	os.Rename(keyFile.Name(), newName)

	cmdRaw := []string{"keys", "import", newName}
	this.executeCommand(cmdRaw, w)
}

// -----------------------------------------------------------------
// ------------------- HELPER FUNCTIONS ----------------------------
// -----------------------------------------------------------------

// Helper function to ensure if a chain is running that it has the time to shut
// down before the parent process exits.
func (this *HttpService) CleanUpAndExit() {
	logger.Errorln("Shutdown Signal Received")
	if this.ChainIsRunning {
		this.ChainShutDownChannel <- true
		<-this.ChainIsShutDown
	}
	os.Exit(0)
}

// Log an incoming request
func (this *HttpService) logIncoming(incoming string) {
	logger.Warnln("Incoming Handle Request: " + incoming)
}

// Log some information
func (this *HttpService) logInfo(incoming string) {
	logger.Warnln(incoming)
}

// Log that an error has happened
func (this *HttpService) logError(w http.ResponseWriter, code int, err error) {
	errString := fmt.Sprintf("%v", err)
	logger.Errorf("ERROR :(\treturning http code: %v\tbecause: %s\n", code, errString)
	this.writeMsg(w, code, errString)
}

// Helper function to execute the cli commands and send back
// the result of the command to the caller.
func (this *HttpService) executeCommand(cmdRaw []string, w http.ResponseWriter) {
	product, err := this.executeCommandRaw(cmdRaw, w)
	if err != nil {
		return
	}
	this.writeMsg(w, 200, product)
}

// Assembles the command.
func (this *HttpService) executeCommandRaw(cmdRaw []string, w http.ResponseWriter) (string, error) {

	var cmd *exec.Cmd
	cmd = exec.Command("epm")

	for _, ele := range cmdRaw {
		cmd.Args = append(cmd.Args, ele)
	}

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	err := cmd.Run()
	if err != nil {
		this.logError(w, 500, err)
		return "", err
	}

	if errOut.String() != "" {
		err := fmt.Errorf(errOut.String())
		this.logError(w, 500, err)
		return "", err
	}

	return out.String(), nil
}

// loop throu headers
func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// Setup the rpc
func (this *HttpService) setupRPC(root string, w http.ResponseWriter, r *http.Request) {
	// rpc override?
	if r.URL.Query().Get("no-rpc") == "true" {
		var configParsed ChainConfig
		configParsed.ServeRPC = false
		this.ChainRunningConfig = configParsed
		return
	}

	// else turn it on
	configRaw, err := ioutil.ReadFile(path.Join(root, "config.json"))
	if err != nil {
		this.logError(w, 500, err)
		return
	}
	var configParsed ChainConfig
	err = json.Unmarshal(configRaw, &configParsed)
	if err != nil {
		this.logError(w, 500, err)
		return
	}

	// make sure the RPC server is turned on
	if !configParsed.ServeRPC {
		this.logInfo("Turning on RPC Server.")
		_, err = this.executeCommandRaw([]string{"config", "serve_rpc:true"}, w)
		if err != nil {
			this.logError(w, 500, err)
			return
		}
	}

	// set the RPC host
	if r.URL.Query().Get("rpc-host") != "" {
		this.logInfo("Making sure RPC Host is set.")
		rpcHost := "rpc_host:" + r.URL.Query().Get("rpc-host")
		_, err = this.executeCommandRaw([]string{"config", rpcHost}, w)
		if err != nil {
			this.logError(w, 500, err)
			return
		}
	} else if configParsed.RPCIp == "" {
		this.logInfo("Making sure RPC Host is set to localhost.")
		_, err = this.executeCommandRaw([]string{"config", "rpc_host:localhost"}, w)
		if err != nil {
			this.logError(w, 500, err)
			return
		}
	}

	// set the RPC port
	if r.URL.Query().Get("rpc-port") != "" {
		this.logInfo("Making sure RPC Port is set.")
		rpcPort := "rpc_port:" + r.URL.Query().Get("rpc-port")
		_, err = this.executeCommandRaw([]string{"config", rpcPort}, w)
		if err != nil {
			this.logError(w, 500, err)
			return
		}
	}

	// Now set the vars in the object
	configRaw, err = ioutil.ReadFile(root + "/config.json")
	if err != nil {
		this.logError(w, 500, err)
		return
	}
	err = json.Unmarshal(configRaw, &configParsed)
	if err != nil {
		this.logError(w, 500, err)
		return
	}

	this.ChainRunningConfig = configParsed
}

// Handler for not found.
func (this *HttpService) handleNotFound(w http.ResponseWriter, r *http.Request) {
	this.logIncoming("404! No handler found for that endpoint.")
	this.writeMsg(w, 404, EPM_HELP)
}

// Handler for echo. Useful for testing.
func (this *HttpService) handleEcho(params martini.Params, w http.ResponseWriter, r *http.Request) {
	this.logIncoming("Echo")

	// todo: remove this
	// var a string
	// ro := *this.Router
	// for _, r := range ro.All() {
	// 	a = a + "/n" + r.Pattern()
	// }
	// this.writeMsg(w, 200, a)

	this.writeMsg(w, 200, params["message"])
}

// Utility method for responding with an error.
func (this *HttpService) writeMsg(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, msg)
}
