package modules

import (
	"github.com/eris-ltd/decerver/interfaces/files"
	"github.com/eris-ltd/modules/types"
)

type (
	ModuleInfo struct {
		Name       string      `json:"name"`
		Version    string      `json:"version"`
		Author     *AuthorInfo `json:"author"`
		Licence    string      `json:"licence"`
		Repository string      `json:"repository"`
	}

	AuthorInfo struct {
		Name  string `json:"name"`
		EMail string `json:"e-mail"`
	}
)

type (
	Module interface {
		// For registering with decerver.
		Register(dc DecerverModuleApi) error
		Init() error
		Start() error
		Restart() error
		Shutdown() error
		Name() string
		Subscribe(name, event, target string) chan types.Event
		UnSubscribe(name string)

		SetProperty(name string, data interface{})
		Property(name string) interface{}
	}

	// Interface for the module manager.
	ModuleManager interface {
		Modules() map[string]Module
		ModuleNames() []string
		Add(m Module) error
		Init() error
		Start() error
		Shutdown() error
	}
	
	// This is the functionality that decerver exports to modules
	// when they register.
	DecerverModuleApi interface {
		// register an object with the script runtime manager (Atë).
		RegisterRuntimeObject(string,interface{})
		// Register script in the form of a string
		RegisterRuntimeScript(string)
		// File and folder management tool.
		FileIO() files.FileIO
	}
)