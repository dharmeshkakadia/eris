package server

import (
	"fmt"
	"github.com/eris-ltd/decerver/interfaces/logging"
	"github.com/eris-ltd/decerver/interfaces/decerver"
	"github.com/eris-ltd/decerver/interfaces/dapps"
	"github.com/go-martini/martini"
	"log"
)

const DEFAULT_PORT = 3000  // For communicating with dapps (the atom browser).
// const DECERVER_PORT = 3100 // For communication with the atom client back-end.

var logger *log.Logger = logging.NewLogger("Webserver")

type WebServer struct {
	webServer      *martini.ClassicMartini
	maxConnections uint32
	host		   string
	port           int
	dc       	   decerver.Decerver
	has            *HttpAPIServer
	das            *DecerverAPIServer
	dm             dapps.DappManager
}

func NewWebServer(dc decerver.Decerver) *WebServer {
	ws := &WebServer{}
	
	ws.maxConnections = uint32(dc.Config().MaxClients)
	port := dc.Config().Port
	if port <= 0 {
		port = DEFAULT_PORT
	}
	ws.port = port
	ws.host = dc.Config().Hostname
	ws.dc = dc
	rm := dc.RuntimeManager()
	ws.has = NewHttpAPIServer(rm,ws.maxConnections)

	ws.webServer = martini.Classic()
	// TODO remember to change to martini.Prod
	martini.Env = martini.Dev

	return ws
}

func (ws *WebServer) RegisterDapp(dappId string) {
	fmt.Println("Registering path: " + dappId + "/(.*)")
	ws.webServer.Any("/apis/" + dappId + "/(.*)", ws.has.handleHttp)
}

func (ws *WebServer) AddDappManager(dm dapps.DappManager) {
	ws.dm = dm
}

func (ws *WebServer) Start() error {

	ws.webServer.Use(martini.Static(ws.dc.FileIO().Dapps()))

	das := NewDecerverAPIServer(ws.dc, ws.dm)

	// Decerver ready
	ws.webServer.Get("/admin/ready", das.handleReadyGET)

	// Decerver configuration
	ws.webServer.Get("/admin/decerver", das.handleDecerverGET)
	ws.webServer.Post("/admin/decerver", das.handleDecerverPOST)

	// Module configuration
	ws.webServer.Get("/admin/modules/(.*)", das.handleModuleGET)
	ws.webServer.Post("/admin/modules/(.*)", das.handleModulePOST)

	// Decerver configuration
	ws.webServer.Get("/admin/switch/(.*)", das.handleDappSwitch)

	// TODO Close down properly. Removed that third party stuff since 
	// it was a mess.
	go func() {
		ws.webServer.RunOnAddr(ws.host + ":" + fmt.Sprintf("%d", ws.port))
	}()
	
	return nil
}
