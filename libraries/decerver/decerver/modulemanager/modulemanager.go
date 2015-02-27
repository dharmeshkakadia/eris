package modulemanager

import (
	"errors"
	"fmt"
	"github.com/eris-ltd/decerver/interfaces/modules"
)

// The modulemanager is where the different modules are kept. Currently, modules has
// to be loaded upon startup, and cannot be unloaded.
type ModuleManager struct {
	modules     map[string]modules.Module
	moduleNames []string
}

func NewModuleManager() modules.ModuleManager {
	mm := &ModuleManager{}
	mm.modules = make(map[string]modules.Module, 1)
	mm.moduleNames = make([]string, 1)
	return mm
}

func (mm *ModuleManager) Modules() map[string]modules.Module {
	return mm.modules
}

func (mm *ModuleManager) ModuleNames() []string {
	return mm.moduleNames
}

func (mm *ModuleManager) Add(m modules.Module) error {
	// The name cannot already be taken.
	mod := mm.modules[m.Name()]
	if mod != nil {
		str := "Module '" + m.Name() + "' has already been registered."
		return errors.New(str)
	}
	mm.moduleNames = append(mm.moduleNames, m.Name())
	mm.modules[m.Name()] = m
	return nil
}

func (mm *ModuleManager) Init() error {
	for _, md := range mm.modules {
		err := md.Init()
		if err != nil {
			return err
		}
	}
	return nil
}

func (mm *ModuleManager) Start() error {
	for _, mod := range mm.modules {
		go func(module modules.Module) {
			fmt.Println("Loading module: " + module.Name())
			module.Start()
		}(mod)
	}
	return nil
}

func (mm *ModuleManager) Shutdown() error {
	for _, mod := range mm.modules {
		mod.Shutdown()
	}
	return nil
}
