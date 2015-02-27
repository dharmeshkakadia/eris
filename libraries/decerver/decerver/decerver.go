package decerver

import (
	"github.com/eris-ltd/decerver/dappmanager"
	"github.com/eris-ltd/decerver/eventprocessor"
	"github.com/eris-ltd/decerver/fileio"
	"github.com/eris-ltd/decerver/interfaces/dapps"
	"github.com/eris-ltd/decerver/interfaces/decerver"
	"github.com/eris-ltd/decerver/interfaces/events"
	"github.com/eris-ltd/decerver/interfaces/files"
	"github.com/eris-ltd/decerver/interfaces/logging"
	"github.com/eris-ltd/decerver/interfaces/modules"
	"github.com/eris-ltd/decerver/interfaces/network"
	"github.com/eris-ltd/decerver/interfaces/scripting"
	"github.com/eris-ltd/decerver/modulemanager"
	"github.com/eris-ltd/decerver/runtimemanager"
	"github.com/eris-ltd/decerver/server"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path"
)

const version = "1.0.0"

var logger *log.Logger = logging.NewLogger("Decerver Core")

// default config object
var DefaultConfig = &decerver.DCConfig{
	LogFile:       "",
	MaxClients:    10,
	Hostname:      "localhost",
	Port:          3000,
	DebugMode:     true,
}

type DeCerver struct {
	config        *decerver.DCConfig
	modApi        modules.DecerverModuleApi
	fileIO        files.FileIO
	ep            events.EventProcessor
	rm            scripting.RuntimeManager
	webServer     network.Server
	moduleManager modules.ModuleManager
	dappManager   dapps.DappManager
	isStarted     bool
}

func NewDeCerver() *DeCerver {
	dc := &DeCerver{}
	logger.Println("Starting decerver bootstrapping sequence.")
	dc.createFileIO()
	dc.loadConfig()
	dc.createModuleManager()
	dc.createEventProcessor()
	dc.createRuntimeManager()
	dc.createServer()
	dc.createDappManager()
	dc.modApi = newDecerverModuleApi(dc)
	return dc
}

func (dc *DeCerver) createFileIO() {
	usr, err := user.Current()
	if err != nil {
		panic("User error: " + err.Error())
	}
	root := path.Join(usr.HomeDir, ".decerver")
	dc.fileIO = fileio.NewFileIO(root)
	dc.fileIO.InitPaths()
}

func (dc *DeCerver) loadConfig(){
	fio := dc.fileIO
	config := &decerver.DCConfig{} 
	err := fio.UnmarshalJsonFromFile(fio.Root(),"config",config)
	if err != nil {
		logger.Println("Failed to load config: " + err.Error() )
		logger.Println("Generating...")
		config = DefaultConfig
		fio.MarshalJsonToFile(fio.Root(),"config",config)
	}
	dc.config = config
}

func (dc *DeCerver) Init() error {
	err := dc.moduleManager.Init()
	if err != nil {
		return err
	}
	dc.initDapps()
	return nil
}

func (dc *DeCerver) Start() error {
	dc.webServer.Start()
	logger.Println("Server started.")

	err := dc.moduleManager.Start()
	if err != nil {
		return err
	}

	// Now everything is registered.
	dc.isStarted = true

	logger.Println("Running...")
	// Just block for now.
	ss := block()

	logger.Println("Shutting down: " + ss)
	return nil
}

// TODO stuff
func (dc *DeCerver) Shutdown() error {
	dc.moduleManager.Shutdown()
	logger.Println("Bye.")
	return nil
}

func block() string {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	sig := <-c
	return sig.String()
}

func (dc *DeCerver) createServer() {
	dc.webServer = server.NewWebServer(dc)
}

func (dc *DeCerver) createEventProcessor() {
	dc.ep = eventprocessor.NewEventProcessor(dc)
}

func (dc *DeCerver) createRuntimeManager() {
	dc.rm = runtimemanager.NewRuntimeManager(dc)
}

func (dc *DeCerver) IsStarted() bool {
	return dc.isStarted
}

func (dc *DeCerver) createModuleManager() {
	dc.moduleManager = modulemanager.NewModuleManager()
}

func (dc *DeCerver) createDappManager() {
	dc.dappManager = dappmanager.NewDappManager(dc)
	dc.webServer.AddDappManager(dc.dappManager)
}

func (dc *DeCerver) LoadModule(md modules.Module) {
	// TODO re-add
	dc.FileIO().CreateModuleDirectory(md.Name())
	md.Register(dc.modApi)
	dc.moduleManager.Add(md)
	logger.Printf("Registering module '%s'.\n", md.Name())
}

func (dc *DeCerver) initDapps() {
	err := dc.dappManager.RegisterDapps(dc.FileIO().Dapps(), dc.FileIO().System())

	if err != nil {
		logger.Println("Error loading dapps: " + err.Error())
		os.Exit(0)
	}
}

func (dc *DeCerver) Config() *decerver.DCConfig {
	return dc.config
}

func (dc *DeCerver) FileIO() files.FileIO {
	return dc.fileIO
}

func (dc *DeCerver) DappManager() dapps.DappManager {
	return dc.dappManager
}

func (dc *DeCerver) EventProcessor() events.EventProcessor {
	return dc.ep
}

func (dc *DeCerver) ModuleManager() modules.ModuleManager {
	return dc.moduleManager
}

func (dc *DeCerver) RuntimeManager() scripting.RuntimeManager {
	return dc.rm
}

func (dc *DeCerver) Server() network.Server {
	return dc.webServer
}

// Satisfies the DecerverModuleAPI interface
type DecerverModuleApi struct {
	fileIO files.FileIO
	ep     events.EventProcessor
	rm     scripting.RuntimeManager
}

func newDecerverModuleApi(dc *DeCerver) modules.DecerverModuleApi {
	dma := &DecerverModuleApi{}
	dma.ep = dc.ep
	dma.fileIO = dc.fileIO
	dma.rm = dc.rm
	return dma
}

func (dma *DecerverModuleApi) RegisterRuntimeObject(name string, obj interface{}) {
	dma.rm.RegisterApiObject(name, obj)
}

func (dma *DecerverModuleApi) RegisterRuntimeScript(script string) {
	dma.rm.RegisterApiScript(script)
}

func (dma *DecerverModuleApi) FileIO() files.FileIO {
	return dma.fileIO
}
