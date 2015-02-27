package fileio

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"fmt"
	"path"
	"sync"
)

type FileIO struct {
	mutex       *sync.Mutex
	root        string
	modules     string
	log         string
	blockchains string
	filesystems string
	dapps       string
	system      string
	tempfiles	string
}

func NewFileIO(rootDir string) *FileIO {
	fio := &FileIO{}
	fio.mutex = &sync.Mutex{}
	fio.root = rootDir
	return fio
}

func (fio *FileIO) Root() string {
	return fio.root
}

func (fio *FileIO) Modules() string {
	return fio.modules
}

func (fio *FileIO) Log() string {
	return fio.log
}

func (fio *FileIO) Dapps() string {
	return fio.dapps
}

func (fio *FileIO) Blockchains() string {
	return fio.blockchains
}

func (fio *FileIO) Filesystems() string {
	return fio.filesystems
}

func (fio *FileIO) System() string {
	return fio.system
}

func (fio *FileIO) Tempfiles() string {
	return fio.tempfiles
}

// Thread safe read file function. Reads an entire file and returns the bytes.
func (fio *FileIO) ReadFile(directory, name string) ([]byte, error) {
	fio.mutex.Lock()
	defer fio.mutex.Unlock()
	return ioutil.ReadFile((path.Join(directory, name)))
}

// Thread safe read file function. It'll read the given file and attempt to
// unmarshal it into the provided object.
func (fio *FileIO) UnmarshalJsonFromFile(directory, name string, object interface{}) error {
	fio.mutex.Lock()
	defer fio.mutex.Unlock()
	bts, err := ioutil.ReadFile((path.Join(directory, name)))
	if err != nil {
		return err
	}
	return json.Unmarshal(bts, object)
}

// Thread safe write file function. Writes the provided byte slice into the file 'name'
// in directory 'directory'. Uses filemode 0600.
func (fio *FileIO) WriteFile(directory, name string, data []byte) error {
	fio.mutex.Lock()
	defer fio.mutex.Unlock()
	return ioutil.WriteFile((path.Join(directory, name)), data, 0600)
}

// Thread safe write file function. Writes the provided object into a file after
// marshaling it into json. Uses filemode 0600.
func (fio *FileIO) MarshalJsonToFile(directory, name string, object interface{}) error {
	fio.mutex.Lock()
	defer fio.mutex.Unlock()
	bts, err := json.MarshalIndent(object, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile((path.Join(directory, name)), bts, 0600)
}

// Creates a new directory for a module, and returns the path.
func (fio *FileIO) CreateModuleDirectory(moduleName string) error {
	fio.mutex.Lock()
	defer fio.mutex.Unlock()
	dir := path.Join(fio.modules,moduleName)
	return initDir(dir)
}

// Creates a new directory for a module, and returns the path.
func (fio *FileIO) WriteDappTempFile(dappName, fileName string, data []byte) error {
	fio.mutex.Lock()
	defer fio.mutex.Unlock()
	dir := path.Join(fio.tempfiles, "dapps", dappName)
	initDir(dir)
	return ioutil.WriteFile((path.Join(dir, fileName)), data, 0600)
}

// Creates a new directory for a module, and returns the path.
func (fio *FileIO) WriteModuleTempFile(moduleName, fileName string, data []byte) error {
	fio.mutex.Lock()
	defer fio.mutex.Unlock()
	dir := path.Join(fio.tempfiles, "modules", moduleName)
	initDir(dir)
	return ioutil.WriteFile((path.Join(dir, fileName)), data, 0600)
}

// Thread safe read file function. Reads an entire file and returns the bytes.
func (fio *FileIO) ReadDappTempFile(dappName, fileName string) ([]byte, error) {
	fio.mutex.Lock()
	defer fio.mutex.Unlock()
	return ioutil.ReadFile((path.Join(fio.tempfiles, "dapps", dappName, fileName)))
}

// Thread safe read file function. Reads an entire file and returns the bytes.
func (fio *FileIO) ReadModuleTempFile(moduleName, fileName string) ([]byte, error) {
	fio.mutex.Lock()
	defer fio.mutex.Unlock()
	return ioutil.ReadFile((path.Join(fio.tempfiles, "modules", moduleName, fileName)))
}

// Helper function to create directories.
func (fio *FileIO) CreateDirectory(dir string) error {
	fio.mutex.Lock()
	defer fio.mutex.Unlock()
	return initDir(dir)
}

func (fio *FileIO) InitPaths() error {
	fio.mutex.Lock()
	defer fio.mutex.Unlock()

	var err error
	err = initDir(fio.root)
	if err != nil {
		return err
	}
	
	fio.log = fio.root + "/logs"
	err = initDir(fio.log)
	if err != nil {
		return err
	}
	
	fio.modules = fio.root + "/modules"
	err = initDir(fio.modules)
	if err != nil {
		return err
	}
	
	fio.dapps = fio.root + "/dapps"
	err = initDir(fio.dapps)
	if err != nil {
		return err
	}
	
	fio.filesystems = fio.root + "/filesystems"
	err = initDir(fio.filesystems)
	if err != nil {
		return err
	}
	
	fio.blockchains = fio.root + "/blockchains"
	err = initDir(fio.blockchains)
	if err != nil {
		return err
	}
	
	fio.system = fio.root + "/system"
	err = initDir(fio.system)
	if err != nil {
		return err
	}
	
	fio.tempfiles = fio.root + "/tempfiles"
	err = initDir(fio.tempfiles)
	if err != nil {
		return err
	}
	
	err = initDir(path.Join(fio.tempfiles,"modules"))
	if err != nil {
		return err
	}
	
	err = initDir(path.Join(fio.tempfiles,"dapps"))
	if err != nil {
		return err
	}
	
	return nil
}

func initDir(Datadir string) error {
	_, err := os.Stat(Datadir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Data directory '%s' doesn't exist, creating it\n", Datadir)
			mdaErr := os.MkdirAll(Datadir, 0777)
			if mdaErr != nil {
				return mdaErr
			}
		}
	} else {
		return err
	}
	return nil
}