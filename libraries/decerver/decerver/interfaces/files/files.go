package files

import ()

type FileIO interface {
	Root() string
	Log() string
	Dapps() string
	Blockchains() string
	Filesystems() string
	Modules() string
	System() string
	Tempfiles() string
	InitPaths() error
	// Useful when you want to load a file inside of a directory gotten by the
	// 'Paths' object. Reads and returns the bytes.
	ReadFile(string,string) ([]byte, error)
	// Useful when you want to save a file into a directory gotten by the 'Paths'
	// object.
	WriteFile(string, string, []byte) error
	// Useful when you want to load json encoded files into objects.
	UnmarshalJsonFromFile(string, string, interface{}) error
	// Useful when you want to store json encoding of objects in files.
	MarshalJsonToFile(string, string, interface{}) error
	// Convenience method for creating module directories.
	CreateModuleDirectory(string) error
	// Convenience method for creating directories.
	CreateDirectory(string) error
	// Convenience method for writing dapp and module tempfiles
	WriteDappTempFile(string, string, []byte) error
	ReadDappTempFile(string, string) ([]byte,error)
	WriteModuleTempFile(string, string, []byte) error
	ReadModuleTempFile(string, string) ([]byte,error)
}
