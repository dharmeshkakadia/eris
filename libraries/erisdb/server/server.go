/*
Package server provides the main API for EPM.

When EPM is started with epm serve this API becomes
accessible on the the IP:Port combination which is
passed as flags to the epm serve command or via env
variables.

The API for EPM is encapsulated under the eris namespace.

There are roughly two major sets of API actions which can
be asked of the API:

	- eris admin actions; and
	- running blockchain RPC commands

Eris admin actions which are passed to the API are basic
epm administrative and informational functions; whereas RPC
commands are simply passed to whatever blockchain epm has
been asked to run.

Note. there is, at this time, no authentication which
is performed before epm serve executes the administrative
or informational commands. If you are running epm serve on
a remote location, you should put Nginx, Apache or another
reverse proxy in front of the application which will perform
some sort of authentication. At this point authentication is
on our work plan: https://github.com/eris-ltd/epm-go/issues/145
but it has not currently been finalized.

API Structure

The Eris API for administrative actions is meant to provide
a remote-like functionality. Responses are sent as plain-text
and most responses are simply a method of piping cli information
from a remote to a local client.

--------------------------------------------------------------

Informational Handlers

--------------------------------------------------------------

	GET http://IP:PORT/eris/plop/:chainName/:toPlop

Will return the variable passed in :toPlop. Namely, one of the
ploppable information commands: addr, chainid, config,
genesis, pid, or vars.

Note that the epm cli will be able to plop the private key of the
blockchain client. This function has purposefully not been implemented
in the API for fairly obvious reasons. Attempts to plop the private
key will fail.

	GET http://IP:PORT/eris/refs/ls

Will return the currently known references. Mirrors: epm refs ls.
Output formated exactly how the cli output is formated (including
coloring).

	POST http://IP:PORT/eris/refs/add/:chainName/:chainType/:chainType

Will add a reference to the blockchain tree. Mirrors; epm refs add.

	POST http://IP:PORT/eris/refs/rm/:chainName

Will remove a reference from a blockchain tree but will not remove
the blockchain directories.

--------------------------------------------------------------

Chain management handlers

--------------------------------------------------------------

	POST http://IP:PORT/eris/config/:chainName

Will add a configuration variable to the referenced chain's config.json.
Configuration variables must be sent in the same format as the
config.json's variables. Variables are sent as URL parameters in standard
format, namely:

	- http://IP:PORT/eris/config/:chainName?key=val

Any number of variables can be sent in the same POST call.

	POST http://IP:PORT/eris/checkout/:chainName

Will checkout the named blockchain. Mirrors epm checkout chainName.

	POST http://IP:PORT/eris/clean/:chainName

Will completely remove the named blockchain folders. Mirrors
epm clean --force chainName

--------------------------------------------------------------

Blockchain admin handlers

--------------------------------------------------------------

	POST http://IP:PORT/eris/fetch/:chainName/:fetchIP/:fetchPort

Will fetch the genesis block from the named fetchIP:fetchPort
combination, name the chain according to the :chainName which
is passed via the URL and will checkout the chain. Mirrors
epm fetch --name chainName --checkout fetchIP:fetchPort

Optional Parameters:

	- checkout = if checkout=false then the fetched blockchain will not be checked out.

Note, fetch will not begin the operation of the blockchain
it will simply retrieve the genesis block.

	POST http://IP:PORT/eris/new/:chainName

Will instatiate a blockchain. If the body of the post request
is blank then epm will instatiate the blockchain using the
default genesis.json. If the body of the post request is not
an empty string then the server will use the request body
string as the genesis.json.

Optional Parameters:

	- checkout = if checkout=false then the fetched blockchain will not be checked out.
	- type = instatiate a non-thelonious chain which is part of EPM's capabilities.

Note, there is no check that the passed string is valid json
before the server will seek to instatiate the chain.

	POST http://IP:PORT/eris/start/:chainName

Will start running the currently checked out blockchain.

Optional Parameters:

	- commit = if commit=true is passed via the URL, then the blockchain will be started with mining/committing turned on;
	- log = set the log level to a log level;
	- no-rpc = by default the server will turn on the RPC server and reverse proxy to it unless the no-rpc=true is set;
	- rpc-host = will set the rpc host to something other than localhost;
	- rpc-port = manually set an rpc port;

The default log level which is set is 2.

	POST http://IP:PORT/eris/stop/:chainName

Will stop a running blockchain.

	POST http://IP:PORT/eris/restart/:chainName

Will restart a running blockchain. The same optional parameters
as for the start API endpoint may be passed to restart.

	GET http://IP:PORT/eris/status

Will query whether a blockchain is running or not. Returns a plain
text string of true if a blockchain is running or false if a
blockchain is not running.

--------------------------------------------------------------

Keys handlers

--------------------------------------------------------------

	POST http://IP:PORT/eris/importkey/:keyName

Will import a key. Keys passed must have been generated
by epm keys gen NAME. The keyName passed to the URL must
match the local keyname.

Will save the POSTed key to the keychain and then import the key
into the config.json for the checked out blockchain.


*/
package server

import (
	"fmt"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/go-martini/martini"
	"log"
	"net/http"
	"os"
	"time"
)

var logger *monklog.Logger = monklog.NewLogger("EPM_SERVER")

// The server object.
type Server struct {
	// The maximum number of active connections that the server allows.
	maxConnections uint32
	// The host.
	host string
	// The port.
	port uint16
	// The root, or serving directory.
	rootDir string
	// The classic martini instance.
	cMartini *martini.ClassicMartini
	// The http service.
	httpService *HttpService
	// The websocket service.
	wsService *WsService
}

// To override martini's logger we need to build
// a custom start function. In other words, we need
// to make our "own" martini and cannot just
// use martini classic out of the box as martini
// classic out of the box uses martini's ugly logger.
func epmClassic() *martini.ClassicMartini {
	m := martini.New()
	m.Use(ServeLogger())
	m.Use(martini.Recovery())
	r := martini.NewRouter()
	m.MapTo(r, (*martini.Routes)(nil))
	m.Action(r.Handle)
	return &martini.ClassicMartini{m, r}
}

// Create a new server object.
func NewServer(host string, port uint16, maxConnections uint32, rootDir string) *Server {

	cMartini := epmClassic()
	httpService := NewHttpService(cMartini.Router)
	wsService := NewWsService(maxConnections)

	return &Server{
		maxConnections,
		host,
		port,
		rootDir,
		cMartini,
		httpService,
		wsService,
	}
}

// Start running the server.
func (this *Server) Start() error {

	cm := this.cMartini

	// Static.
	cm.Use(martini.Static(this.rootDir))

	// Simple echo for testing http
	cm.Get("/echo/:message", this.httpService.handleEcho)

	// Informational handlers
	cm.Get("/eris/plop/:chainName/:toPlop", this.httpService.handlePlop)
	cm.Get("/eris/refs/ls", this.httpService.handleLsRefs)
	cm.Post("/eris/refs/add/:chainName/:chainType/:chainId", this.httpService.handleAddRefs)
	cm.Post("/eris/refs/rm/:chainName", this.httpService.handleRmRefs)

	// Chain management handlers
	cm.Post("/eris/config/:chainName", this.httpService.handleConfig)
	cm.Post("/eris/checkout/:chainName", this.httpService.handleCheckout)
	cm.Post("/eris/clean/:chainName", this.httpService.handleClean)

	// Blockchain admin handlers
	cm.Post("/eris/fetch/:chainName/:fetchIP/:fetchPort", this.httpService.handleFetchChain)
	cm.Post("/eris/new/:chainName", this.httpService.handleNewChain)
	cm.Post("/eris/start/:chainName", this.httpService.handleStartChain)
	cm.Post("/eris/stop/:chainName", this.httpService.handleStopChain)
	cm.Post("/eris/restart/:chainName", this.httpService.handleRestartChain)
	cm.Get("/eris/status/:chainName", this.httpService.handleChainStatus)

	// Keys handlers
	cm.Post("/eris/importkey/:keyName", this.httpService.handleKeyImport)

	// Handle websocket negotiation requests.
	cm.Get("/ws", this.wsService.handleWs)

	// Default 404 message.
	cm.NotFound(this.httpService.handleNotFound)

	cm.RunOnAddr(this.host + ":" + fmt.Sprintf("%d", this.port))

	return nil
}

// Get the maximum number of active connections/sessions that the server allows.
func (this *Server) MaxConnections() uint32 {
	return this.maxConnections
}

// Get the root, or served directory.
func (this *Server) RootDir() string {
	return this.rootDir
}

// Get the http service object.
func (this *Server) HttpService() *HttpService {
	return this.httpService
}

// Get the websocket service object.
func (this *Server) WsService() *WsService {
	return this.wsService
}

// Custom Logger to harmonize logging events with
// the rest of EPM logging. While this is non-standard
// it is necessary because martini's logging is
// really quite ugly.
func ServeLogger() martini.Handler {
	out := log.New(os.Stdout, "", 0)
	return func(res http.ResponseWriter, req *http.Request, c martini.Context, log *log.Logger) {
		start := time.Now()

		addr := req.Header.Get("X-Real-IP")
		if addr == "" {
			addr = req.Header.Get("X-Forwarded-For")
			if addr == "" {
				addr = req.RemoteAddr
			}
		}

		out.Printf("%d/%02d/%02d %02d:%02d:%02d [EPM_SERVER] Started %s %s for %s\n", start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute(), start.Second(), req.Method, req.URL.Path, addr)

		rw := res.(martini.ResponseWriter)
		c.Next()

		stop := time.Now()
		out.Printf("%d/%02d/%02d %02d:%02d:%02d [EPM_SERVER] Completed %v %s in %v\n", stop.Year(), stop.Month(), stop.Day(), start.Hour(), start.Minute(), start.Second(), rw.Status(), http.StatusText(rw.Status()), time.Since(start))
	}
}
