package decerver

import (
	"github.com/eris-ltd/decerver/interfaces/scripting"
	"github.com/eris-ltd/decerver/interfaces/dapps"
	"github.com/eris-ltd/decerver/interfaces/events"
	"github.com/eris-ltd/decerver/interfaces/files"
	"github.com/eris-ltd/decerver/interfaces/modules"
	"github.com/eris-ltd/decerver/interfaces/network"
)

// The decerver configuration file.
type DCConfig struct {
	LogFile    string `json:"logfile"`
	MaxClients int    `json:"max_clients"`
	Hostname   string `json:"hostname"`
	Port       int    `json:"port"`
	DebugMode  bool   `json:"debug_mode"`
}



// The decerver interface.
type Decerver interface {
	// Get the config file.
	Config() *DCConfig
	// Is the decerver started?
	IsStarted() bool
	// Get the runtime manager.
	RuntimeManager() scripting.RuntimeManager
	// Get the dapp registry.
	DappManager() dapps.DappManager
	// Get the event processor.
	EventProcessor() events.EventProcessor
	// Get the fileIO file and folder management tool.
	FileIO() files.FileIO
	// Get the module manager.
	ModuleManager() modules.ModuleManager
	// Get the webserver.
	Server() network.Server
	// Initialize
	Init() error
	// Start
	Start() error
	// Shutdown
	Shutdown() error
}

