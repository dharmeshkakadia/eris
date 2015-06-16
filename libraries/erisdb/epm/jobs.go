package epm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/lllc-server"
	//"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/lllc-server/abi"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/epm/abi"
	"github.com/eris-ltd/epm-go/utils"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// What to do if a job errs
const (
	PersistOnErr int = iota
	ReturnOnErr
	FailOnErr
)

// Default to fail on error
var ErrMode = FailOnErr

// Commit changes to the db (ie. mine a block)
func (e *EPM) Commit() {
	e.chain.Commit()
}

// Execute parsed jobs
func (e *EPM) ExecuteJobs() error {
	if e.Diff {
		e.checkTakeStateDiff(0)
	}
	if e.existsModifyJob() {
		// temp dir
		if err := CopyContractPath(); err != nil {
			return err
		}
	}

	uncommited := false
	for i, j := range e.jobs {
		err := e.ExecuteJob(j)

		if j.cmd == "transact" || j.cmd == "deploy" || j.cmd == "modify-deploy" {
			uncommited = true
		}
		if j.cmd == "commit" {
			uncommited = false
		}

		if e.Diff {
			e.checkTakeStateDiff(i + 1)
		}

		if err != nil {
			switch ErrMode {
			case ReturnOnErr:
				return err
			case FailOnErr:
				monklog.Flush()
				log.Fatal(err)
			case PersistOnErr:
				continue
			}
		}
	}
	if e.Diff {
		e.checkTakeStateDiff(len(e.jobs))
	}
	if (uncommited && e.chain != nil) {
		e.chain.Commit()
	}
	return nil
}

func requireErr(args [][]*tree, n int, cmd string) error {
	if !require(args, n) {
		return fmt.Errorf("%s requires at least %d arguments. Provided %d", cmd, n, len(args))
	}
	return nil
}

func require(args [][]*tree, n int) bool {
	if len(args) >= n {
		return true
	}
	return false
}

func (e *EPM) resolveFunc(name string) (func([]string) error, int) {
	var f func([]string) error
	n := CommandArgs[name]
	switch name {
	case "deploy":
		f = e.Deploy
	case "modify-deploy":
		f = e.ModifyDeploy
	case "transact":
		f = e.Transact
	case "call":
		f = e.Call
	case "query":
		f = e.Query
	case "log":
		f = e.Log
	case "set":
		f = e.Set
	case "endow":
		f = e.Endow
	case "test":
		f = func(a []string) error {
			e.Commit()
			err := e.ExecuteTest(a[0], 0)
			if err != nil {
				logger.Errorln(err)
				return err
			}
			return nil
		}
	case "epm":
		f = e.EPMx
	case "include":
		f = e.Include
	case "assert":
		f = e.Assert
	case "commit":
		f = func([]string) error {
			e.Commit()
			return nil
		}
	default:
		f = func([]string) error { return fmt.Errorf("Unknown command: %s", name) }
		n = 0
	}
	return f, n
}

var NoChainErr = fmt.Errorf("Chain is nil")

// Job switch
// Args are still raw input from user (but only 2 or 3)
func (e *EPM) ExecuteJob(job Job) error {
	logger.Warnln("Executing job: ", job.cmd)
	f, n := e.resolveFunc(job.cmd)
	if err := requireErr(job.args, n, job.cmd); err != nil {
		return err
	}
	args, err := e.ResolveArgs(job.cmd, job.args)
	if err != nil {
		return err
	}
	logger.Infoln("ResolvedArgs:", args)
	if e.chain == nil {
		return NoChainErr
	}
	return f(args)
}

// Deploy a pdx from a pdx
func (e *EPM) EPMx(args []string) error {
	filename := args[0]
	// save the old jobs, empty the job list
	oldjobs := e.jobs
	e.jobs = []Job{}

	if err := e.Parse(filename); err != nil {
		logger.Errorln("failed to parse pdx file:", filename, err)
		return err
	}

	if len(args) > 1 {
		e.varsPrefix = args[1]
	}
	err := e.ExecuteJobs()
	if err != nil {
		return err
	}
	e.varsPrefix = ""

	// return to old jobs
	e.jobs = oldjobs
	return nil
}

// assert a variable equals some value
func (e *EPM) Assert(args []string) error {
	got, expected := args[0], args[1]
	got = strings.ToLower(utils.StripZeros(utils.StripHex(got)))
	expected = strings.ToLower(utils.StripZeros(utils.StripHex(expected)))
	if got != expected {
		return fmt.Errorf("assertion error. Got %s, expected %s", got, expected)
	}
	logger.Warnf("correct assertion: %s\n", got)
	return nil
}

// Deploy a contract and save its address
func (e *EPM) Deploy(args []string) error {
	contract := args[0]
	key := args[1]
	contract = strings.Trim(contract, "\"")
	logger.Debugln("Deploying contract:", contract)
	var p string
	// compile contract
	if filepath.IsAbs(contract) {
		p = contract
	} else {
		p = path.Join(ContractPath, contract)
	}
	logger.Debugln("Contract path:", p)
	// compile
	bytecode, abiSpec, err := lllcserver.Compile(p)
	if err != nil {
		return err
	}
	logger.Debugln("Abi spec:", string(abiSpec))
	// send transaction
	_, addr, err := e.chain.Script(hex.EncodeToString(bytecode))
	if err != nil {
		err = fmt.Errorf("Error deploying contract %s: %s", p, err.Error())
		logger.Infoln(err.Error())
		return err
	}
	logger.Warnf("Deployed %s as %s\n", addr, key)
	// write abi to file
	abiDir := path.Join(e.chain.Property("RootDir").(string), "abi")
	if _, err := os.Stat(abiDir); err != nil {
		if err := os.Mkdir(abiDir, 0700); err != nil {
			return err
		}
	}
	if err := ioutil.WriteFile(path.Join(abiDir, utils.StripHex(addr)), []byte(abiSpec), 0600); err != nil {
		return err
	}
	// save contract address
	e.StoreVar(key, addr)
	return nil
}

// Modify lines in the contract prior to deploy, and save its address
func (e *EPM) ModifyDeploy(args []string) error {
	contract := args[0]
	key := args[1]
	args = args[2:]

	contract = strings.Trim(contract, "\"")
	newName, err := e.Modify(path.Join(ContractPath, contract), args)
	if err != nil {
		return err
	}
	return e.Deploy([]string{newName, key})
}

func coerceHex(aa string, padright bool) string {
	if !utils.IsHex(aa) {
		//first try and convert to int
		n, err := strconv.Atoi(aa)
		if err != nil {
			// right pad strings
			if padright {
				aa = "0x" + fmt.Sprintf("%x", aa) + fmt.Sprintf("%0"+strconv.Itoa(64-len(aa)*2)+"s", "")
			} else {
				aa = "0x" + fmt.Sprintf("%x", aa)
			}
		} else {
			aa = "0x" + fmt.Sprintf("%x", n)
		}
	}
	return aa
}

func (e *EPM) packArgsABI(to string, data ...string) ([]string, error) {
	packed := []string{}
	// check for abi
	abiSpec, ok := ReadAbi(e.chain.Property("RootDir").(string), to)
	if ok {
		funcName := data[0]
		args := data[1:]

//		fmt.Println("ABI Spec", abiSpec)
		a := []interface{}{}
		for _, aa := range args {
			aa = coerceHex(aa, true)
			bb, _ := hex.DecodeString(utils.StripHex(aa))
			a = append(a, bb)
		}
		packedBytes, err := abiSpec.Pack(funcName, a...)
		if err != nil {
			return nil, err
		}
		packed = []string{hex.EncodeToString(packedBytes)}

	} else {
		for _, aa := range data {
			aa = coerceHex(aa, false)
			packed = append(packed, aa)
		}
	}
	return packed, nil
}

// Send a transaction with data to a contract
// Data should be list of strings/hex/numeric
// already resolved
func (e *EPM) Transact(args []string) (err error) {
	to := args[0]
	data := args[1:]

	packed, err := e.packArgsABI(to, data...)

	if err != nil {
		return
	}

	if _, err = e.chain.Msg(to, packed); err != nil {
		return
	}
	logger.Warnf("Sent %s to %s", data, to)
	return
}

// Simulate sending a transaction with data to a contract
// Data should be list of strings/hex/numeric
// already resolved
func (e *EPM) Call(args []string) (err error) {
	to := args[0]
	data := args[1 : len(args)-1]
	varName := args[len(args)-1]

	packed, err := e.packArgsABI(to, data...)
	if err != nil {
		return err
	}

	ret, err := e.chain.Call(to, packed)
	if err != nil {
		return
	}
	logger.Warnf("Sent %s to %s", data, to)
	e.StoreVar(varName, ret)
	logger.Warnf("Result: %s", ret)
	return
}

// Issue a query.
// XXX: Only works after a commit ...
func (e *EPM) Query(args []string) error {
	addr := args[0]
	storage := args[1]
	varName := args[2]

	v := e.chain.StorageAt(addr, storage)
	e.StoreVar(varName, v)
	logger.Warnf("result: %s = %s\n", varName, v)
	return nil
}

// Log something to the log file
func (e *EPM) Log(args []string) error {
	k := args[0]
	v := args[1]

	_, err := os.Stat(e.log)
	var f *os.File
	if err != nil {
		f, err = os.Create(e.log)
		if err != nil {
			return err
		}
	} else {
		f, err = os.OpenFile(e.log, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
	}
	defer f.Close()

	if _, err = f.WriteString(fmt.Sprintf("%s : %s", k, v)); err != nil {
		return err
	}
	return nil
}

// Set an epm variable
func (e *EPM) Set(args []string) error {
	k := args[0]
	v := args[1]
	e.StoreVar(k, v)
	return nil
}

// Send a basic transaction transfering value.
func (e *EPM) Endow(args []string) error {
	addr := args[0]
	value := args[1]
	e.chain.Tx(addr, value)
	logger.Warnf("Endowed %s with %s", addr, value)
	return nil
}

func (e *EPM) Include(args []string) error {
	if len(args)%2 != 0 {
		return fmt.Errorf("Each include statement must have two args (a path and a label)")
	}
	for i := 0; i < len(args)/2; i++ {
		includePath := args[2*i]
		varName := args[2*i+1]

		absIncludePath := path.Join(utils.GoPath, "src", includePath)
		if _, err := os.Stat(absIncludePath); err != nil {
			// attempt to git clone it
			logger.Debugf("Package %s does not exist. Attempting to clone it...\n", includePath)
			cur, _ := os.Getwd()
			os.Chdir(path.Join(utils.GoPath, "src"))
			dir := path.Dir(includePath)
			if _, err := os.Stat(dir); err != nil {
				os.MkdirAll(dir, 0700)
			}
			os.Chdir(dir)
			cmd := exec.Command("git", "clone", "https://"+includePath)
			err := cmd.Run()
			os.Chdir(cur)
			if err != nil {
				return fmt.Errorf("Included path %s does not exist.Error on clone: %s", err.Error())
			}
		}
		e.StoreVar(varName, absIncludePath)
	}
	return nil
}

// Apply substitution: replace pairs from args to contract
// and save in a temporary file
func (e *EPM) Modify(contract string, args []string) (string, error) {
	b, err := ioutil.ReadFile(contract)
	if err != nil {
		return "", fmt.Errorf("Could not open file %s: %s", contract, err.Error())
	}

	lll := string(b)
	fmt.Println("ORIGINAL LLL:", lll)

	// when we modify a contract, we save it in the .tmp dir in the same relative path as its original root.
	// eg. if ContractPath is ~/ponos and we modify ponos/projects/issue.lll then the modified version will be found in
	//  scratch/ponos/projects/somehash.lll
	dirC := path.Dir(contract) // absolute path
	l := len(ContractPath)
	var dir string
	if dirC != ContractPath {
		dir = dirC[l+1:] //this is relative to the contract root (ie. projects/issue.lll)
	} else {
		dir = ""
	}
	root := path.Base(ContractPath) // base of the ContractPath should be the root dir of the contracts
	dir = path.Join(root, dir)      // add in the root (ie. ponos/projects/issue.lll)

	for len(args) > 0 {
		sub := args[0]
		rep := args[1]

		// rep may have an epm var in it
		rep = e.RegVarSub(rep)

		lll = strings.Replace(lll, sub, rep, -1)
		args = args[2:]
	}

	hash := sha256.Sum256([]byte(lll))
	newPath := path.Join(EpmDir, dir, hex.EncodeToString(hash[:])+".lll")
	err = ioutil.WriteFile(newPath, []byte(lll), 0644)
	if err != nil {
		return "", fmt.Errorf("Could not write file %s: %s", newPath, err.Error())
	}
	return newPath, nil
}

func ReadAbi(root, to string) (abi.ABI, bool) {
	p := path.Join(root, "abi", utils.StripHex(to))
	if _, err := os.Stat(p); err != nil {
		log.Println("Abi doesn't exist for", p)
		return abi.NullABI, false
	}
	b, err := ioutil.ReadFile(p)

	if err != nil {
		log.Println("Failed to read abi file:", err)
		return abi.NullABI, false
	}

	if (len(b) == 0) {
		return abi.NullABI, false
	}

	a := new(abi.ABI)

	if err := a.UnmarshalJSON(b); err != nil {
		log.Println("failed to unmarshal", err)
		return abi.NullABI, false
	}
	return *a, true
}

func SetCompilerServer(hostPort string) {
	for lang, _ := range lllcserver.Languages {
		lllcserver.SetLanguageURL(lang, hostPort)
		lllcserver.SetLanguageNet(lang, true)
	}
}

// TODO: we should really only every copy what and when we need to
func (e *EPM) existsModifyJob() bool {
	for _, j := range e.jobs {
		if strings.Contains(j.cmd, "modify") {
			return true
		}
	}
	return false
}
