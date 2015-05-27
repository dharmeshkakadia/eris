package epm

import (
	"fmt"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/modules/types"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/utils"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

var logger *monklog.Logger = monklog.NewLogger("EPM")

var (
	StateDiffOpen  = "!{"
	StateDiffClose = "!}"
)

var GOPATH = os.Getenv("GOPATH")

var (
	// XXX: this typically gets overwritten
	// TODO: safer/more explicit?
	// Default is "."
	ContractPath = path.Join(utils.ErisLtd, "epm-go", "cmd", "tests", "test_eris_lll")
	TestPath     = path.Join(utils.ErisLtd, "epm-go", "cmd", "tests")

	EpmDir  = utils.Epm
	LogFile = path.Join(utils.Logs, "epm", "log")
)

type KeyManager interface {
	ActiveAddress() string
	Address(n int) (string, error)
	SetAddress(addr string) error
	SetAddressN(n int) error
	NewAddress(set bool) string
	AddressCount() int
}

type DecerverModule interface {
	Init() error
	Start() error

	ReadConfig(string) error
	WriteConfig(string) error
	SetProperty(string, interface{}) error
	Property(string) interface{}
}

type Blockchain interface {
	KeyManager
	DecerverModule
	ChainId() (string, error)
	WorldState() *types.WorldState
	State() *types.State
	Storage(target string) *types.Storage
	Account(target string) *types.Account
	StorageAt(target, storage string) string
	BlockCount() int
	LatestBlock() string
	Block(hash string) *types.Block
	IsScript(target string) bool
	Tx(addr, amt string) (string, error)
	Msg(addr string, data []string) (string, error)
	Call(addr string, data []string) (string, error)
	Script(code string) (string, string, error)
	Subscribe(name, event, target string) chan types.Event
	UnSubscribe(name string)
	Commit()
	AutoCommit(toggle bool)
	IsAutocommit() bool

	Shutdown() error
	WaitForShutdown()
}

// EPM object. Maintains list of jobs and a symbols table
type EPM struct {
	chain Blockchain

	jobs       []Job
	vars       map[string]string
	varsPrefix string

	pkgdef string
	Diff   bool
	states map[string]types.State

	//map job numbers to names of diffs invoked before a job
	diffSched map[int][]string

	log string
}

// New empty EPM
func NewEPM(chain Blockchain, log string) (*EPM, error) {
	e := &EPM{
		chain:     chain,
		jobs:      []Job{},
		vars:      make(map[string]string),
		log:       ".epm-log",
		Diff:      false, // off by default
		states:    make(map[string]types.State),
		diffSched: make(map[int][]string),
	}
	return e, nil
}

func (e *EPM) Stop() {
	e.chain.Shutdown()
}

// Parse a pdx file into a series of EPM jobs
func (e *EPM) Parse(filename string) error {
	logger.Infoln("Parsing ", filename)
	// set current file to parse
	e.pkgdef = filename
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	p := Parse(string(b))
	if err := p.run(); err != nil {
		return err
	}
	e.jobs = p.jobs
	e.diffSched = p.diffsched
	return nil
}

// New EPM Job
func NewJob(cmd string, args []*tree) *Job {
	j := new(Job)
	j.cmd = cmd
	j.args = [][]*tree{}
	for _, a := range args {
		j.args = append(j.args, []*tree{a})
	}
	return j
}

// Add job to EPM jobs
func (e *EPM) AddJob(j *Job) {
	e.jobs = append(e.jobs, *j)
}

func (e *EPM) VarSub(id string) (string, error) {
	if strings.HasPrefix(id, "{{") && strings.HasSuffix(id, "}}") {
		id = id[2 : len(id)-2]
	}
	v, ok := e.vars[id]
	if !ok {
		return "", fmt.Errorf("Unknown variable %s", id)
	}
	return v, nil
}

// replaces any {{varname}} args with the variable value
func (e *EPM) RegVarSub(arg string) string {
	r, _ := regexp.Compile(`\{\{(.+?)\}\}`)
	// if its a known var, replace it
	// else, leave alone
	return r.ReplaceAllStringFunc(arg, func(s string) string {
		k := s[2 : len(s)-2] // shave the brackets
		v, ok := e.vars[k]
		if ok {
			return v
		} else {
			return s
		}
	})
}

// Read EPM variables in from a file
func (e *EPM) ReadVars(file string) error {
	f, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	sp := strings.Split(string(f), "\n")
	for _, kv := range sp {
		kvsp := strings.Split(kv, ":")
		if len(kvsp) != 2 {
			return fmt.Errorf("Invalid variable formatting in %s", file)
		}
		k := kvsp[0]
		v := kvsp[1]
		e.vars[k] = v
	}
	return nil
}

// Write EPM variables to file
func (e *EPM) WriteVars(file string) error {
	vars := e.Vars()
	s := ""
	for k, v := range vars {
		s += k + ":" + v + "\n"
	}
	if len(s) == 0 {
		return nil
	}
	// remove final new line
	s = s[:len(s)-1]
	err := ioutil.WriteFile(file, []byte(s), 0600)
	return err
}

// Return map of EPM variables.
func (e *EPM) Vars() map[string]string {
	return e.vars
}

func IsVar(v string) bool {
	if strings.HasPrefix(v, "{{") && strings.HasSuffix(v, "}}") {
		return true
	}
	return false
}

// Return list of jobs
func (e *EPM) Jobs() []Job {
	return e.jobs
}

// Store a variable (strips {{ }} from key if necessary)
func (e *EPM) StoreVar(key, val string) {
	fmt.Println("Storing:", key, val)
	if len(key) > 4 && key[:2] == "{{" && key[len(key)-2:] == "}}" {
		key = key[2 : len(key)-2]
	}
	if e.varsPrefix != "" {
		key = e.varsPrefix + "." + key
	}
	// if it's a path, don't coerce
	if strings.Contains(val, "/") {
		e.vars[key] = val
	} else {
		e.vars[key] = utils.Coerce2Hex(val)
	}
	logger.Infof("Stored var %s:%s\n", key, e.vars[key])
}

func CopyContractPath() error {
	// copy the current dir into scratch/epm. Necessary for finding include files after a modify. :sigh:
	root := path.Base(ContractPath)
	p := path.Join(EpmDir, root)
	// TODO: should we delete and copy even if it does exist?
	// we might miss changed otherwise
	if _, err := os.Stat(p); err != nil {
		cmd := exec.Command("cp", "-r", ContractPath, p)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("error copying working dir into tmp: %s", err.Error())
		}
	}
	return nil
}
