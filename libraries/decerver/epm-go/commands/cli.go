package commands

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	color "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/daviddengcn/go-colortext"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/decerver/interfaces/dapps"
	"github.com/eris-ltd/epm-go/chains"
	"github.com/eris-ltd/epm-go/epm" // ed25519 key generation
	"github.com/eris-ltd/epm-go/utils"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	//epm-binary-generator:IMPORT
	mod "github.com/eris-ltd/epm-go/commands/modules/thelonious"
)

func Clean(c *Context) {
	toclean := c.Args()[0]
	if toclean == "" {
		exit(fmt.Errorf("You must enter a directory or file to wipe"))
	}
	dir := path.Join(utils.Decerver, toclean)
	exit(utils.ClearDir(dir))
}

// plop the config or genesis defaults into current dir
func Plop(c *Context) {
	root, chainType, chainId, err := ResolveRootFlag(c)
	ifExit(err)
	var toPlop string
	if len(c.Args()) > 0 {
		toPlop = c.Args()[0]
	} else {
		toPlop = ""
	}
	switch toPlop {
	case "genesis":
		b, err := ioutil.ReadFile(path.Join(root, "genesis.json"))
		ifExit(err)
		fmt.Println(string(b))
	case "config":
		b, err := ioutil.ReadFile(path.Join(root, "config.json"))
		ifExit(err)
		fmt.Println(string(b))
	case "chainid":
		fmt.Println(chainId)
	case "vars":
		b, err := ioutil.ReadFile(path.Join(root, EPMVars))
		ifExit(err)
		if len(c.Args()) > 1 {
			// plop only a single var
			spl := strings.Split(string(b), "\n")
			for _, s := range spl {
				ss := strings.Split(s, ":")
				if ss[0] == c.Args()[1] {
					fmt.Println(ss[1])
				}
			}
		} else {
			fmt.Println(string(b))
		}
	case "pid":
		b, err := ioutil.ReadFile(path.Join(root, "pid"))
		ifExit(err)
		fmt.Println(string(b))
	case "key", "addr":
		rpc := c.Bool("rpc")
		configPath := path.Join(root, "config.json")
		m := mod.NewChain(chainType, rpc)
		err := m.ReadConfig(configPath)
		ifExit(err)
		keyname := m.Property("KeySession").(string)
		var b []byte
		switch toPlop {
		case "key":
			b, err = ioutil.ReadFile(path.Join(root, keyname+".prv"))
		case "addr":
			b, err = ioutil.ReadFile(path.Join(root, keyname+".addr"))
		}
		ifExit(err)
		fmt.Println(string(b))
	case "abi":
		if len(c.Args()) == 1 {
			ifExit(fmt.Errorf("Specify a contract to see its abi"))
		}
		e, err := epm.NewEPM(nil, epm.LogFile)
		ifExit(err)
		e.ReadVars(path.Join(root, EPMVars))
		addr := c.Args()[1]
		if epm.IsVar(addr) {
			addr, err = e.VarSub(addr)
			ifExit(err)
		}
		b, err := ioutil.ReadFile(path.Join(root, "abi", utils.StripHex(addr)))
		ifExit(err)
		fmt.Println(string(b))
	default:
		logger.Errorln("Plop options: addr, chainid, config, genesis, key, pid, vars")
	}
	exit(nil)
}

// list the refs
func Refs(c *Context) {
	r, err := chains.GetRefs()
	_, h, _ := chains.GetHead()
	fmt.Printf("%-20s%-60s%-20s\n", "Name:", "Blockchain:", "Address:")
	for rk, rv := range r {
		// loop through the known blockchains
		chainType, chainId, e := chains.ResolveChain(rv)
		ifExit(e)
		chainDir, er := chains.ResolveChainDir(chainType, rk, chainId)
		ifExit(er)
		configPath := path.Join(chainDir, "config.json")
		cfg := struct {
			KeySession string `json:"key_session"`
		}{}
		err := utils.ReadJson(&cfg, configPath)
		ifExit(err)

		// now find the keysession and addresses
		keyname := cfg.KeySession
		var key []byte
		var kn string
		key, err = ioutil.ReadFile(path.Join(chainDir, keyname+".addr"))
		if err != nil {
			if strings.Contains(keyname, "-") {
				key = []byte(strings.Split(keyname, "-")[1])
			} else {
				key = []byte("unset")
			}
		}
		if string(key) != "unset" {
			kn = "0x" + string(key)
		} else {
			kn = string(key)
		}

		// display the results
		if strings.Contains(rv, h) {
			color.ChangeColor(color.Green, true, color.None, false)
			fmt.Printf("%-20s%-60s%-20s\n", rk, rv, kn)
			color.ResetColor()
		} else {
			fmt.Printf("%-20s%-60s%-20s\n", rk, rv, kn)
		}
	}
	exit(err)
}

// list the keyfiles
func LsKeys(c *Context) {
	keys, err := ioutil.ReadDir(utils.Keys)
	ifExit(err)
	fmt.Printf("%-20s%-60s%-20s\n", "Name:", "Address:", "Key Value:")
	for i := range keys {
		k := strings.Split(keys[i].Name(), "-")
		k[1] = "0x" + k[1]
		kv, err := ioutil.ReadFile(path.Join(utils.Keys, keys[i].Name()))
		ifExit(err)
		fmt.Printf("%-20s%-60s%-20s\n", k[0], k[1], kv)
	}
	exit(err)
}

// print current head
func Head(c *Context) {
	typ, id, err := chains.GetHead()
	if err == nil {
		fmt.Println(path.Join(typ, id))
	}
	exit(err)
}

// duplicate a chain
func Cp(c *Context) {
	args := c.Args()
	var (
		root  string
		typ   string
		id    string
		err   error
		multi string
	)
	if len(args) == 0 {
		log.Fatal(`To copy a chain, specify a chain and a new name, \n eg. "cp thel/14c32 chaincopy"`)

	} else if len(args) == 1 {
		multi = args[0]
		// copy the checked out chain
		typ, id, err = chains.GetHead()
		ifExit(err)
		if id == "" {
			log.Fatal(`No chain is checked out. To copy a chain, specify a chainId and an new name, \n eg. "cp thel/14c32 chaincopy"`)
		}
		root = chains.ComposeRoot(typ, id)
	} else {
		ref := args[0]
		multi = args[1]
		root, typ, id, err = resolveRoot(ref, false, "")
		ifExit(err)
	}
	newRoot := chains.ComposeRootMulti(typ, id, multi)
	if c.Bool("bare") {
		err = utils.InitDataDir(newRoot)
		ifExit(err)
		err = utils.Copy(path.Join(root, "config.json"), path.Join(newRoot, "config.json"))
		ifExit(err)
	} else {
		err = utils.Copy(root, newRoot)
		ifExit(err)
	}
	chain := mod.NewChain(typ, false)
	configureRootDir(c, chain, newRoot)
	chain.WriteConfig(path.Join(newRoot, "config.json"))
}

// create ~/.eris tree and drop default monk/gen configs in there
func Init(c *Context) {
	exit(utils.InitDecerverDir())
}

// fetch a genesis block and state from a peer server
func Fetch(c *Context) {
	if len(c.Args()) == 0 {
		ifExit(fmt.Errorf("Must specify a peerserver address"))
	}

	chainType := "thelonious"
	peerserver := c.Args()[0]
	chainId, err := mod.Fetch(chainType, peerserver)
	ifExit(err)
	logger.Warnf("Fetched genesis block for chain %x", chainId)

	chainID := hex.EncodeToString(chainId)
	if c.Bool("checkout") {
		ifExit(chains.ChangeHead(chainType, chainID))
		logger.Warnf("Checked out chain: %s/%s", chainType, chainID)
	}

	// update refs
	updateRefs(chainType, chainID, c.String("force-name"), c.String("name"))
}

// deploy the genblock into a random folder in scratch
// and install into the global tree (must compute chainId before we know where to put it)
// possibly checkout the newly deployed
// chain agnostic!
func New(c *Context) {
	fmt.Println(c.String("type"))
	chainType, err := chains.ResolveChainType(c.String("type"))
	ifExit(err)
	name := c.String("name")
	forceName := c.String("force-name")
	rpc := c.Bool("rpc")

	r := make([]byte, 8)
	rand.Read(r)
	tmpRoot := path.Join(utils.Scratch, "epm", hex.EncodeToString(r))

	// if genesis or config are not specified
	// use defaults set by `epm init`
	deployConf := c.String("config")
	deployGen := c.String("genesis")
	tempConf := ".config.json"
	editCfg := c.Bool("edit-config")
	noEdit := c.Bool("no-edit")
	editGen := c.Bool("edit")
	// if we provide genesis, dont open editor for genesis
	noEditor := c.IsSet("genesis")
	// but maybe the user wants different behaviour
	if noEdit {
		noEditor = true
	} else if editGen {
		noEditor = false
	}

	chainId := deployInstallChain(tmpRoot, deployConf, deployGen, tempConf, chainType, rpc, editCfg, noEditor)

	if c.Bool("checkout") {
		ifExit(chains.ChangeHead(chainType, chainId))
		logger.Warnf("Checked out chain: %s/%s", chainType, chainId)
	}

	// update refs
	updateRefs(chainType, chainId, forceName, name)
}

func updateRefs(chainType, chainId, forceName, name string) {
	// update refs
	if forceName != "" {
		err := chains.AddRefForce(chainType, chainId, forceName)
		if err != nil {
			ifExit(err)
		}
		logger.Warnf("Created ref %s to point to chain %s\n", forceName, chainId)
	} else if name != "" {
		err := chains.AddRef(chainType, chainId, name)
		if err != nil {
			ifExit(err)
		}
		logger.Warnf("Created ref %s to point to chain %s\n", name, chainId)
	}
}

func deployInstallChain(tmpRoot, deployConf, deployGen, tempConf, chainType string, rpc, editCfg, noEditor bool) string {
	if deployConf == "" {
		if rpc {
			deployConf = path.Join(utils.Blockchains, chainType, "rpc", "config.json")
		} else {
			deployConf = path.Join(utils.Blockchains, chainType, "config.json")
		}
	}

	chain := mod.NewChain(chainType, rpc)

	// if config doesnt exist, lay it
	if _, err := os.Stat(deployConf); err != nil {
		utils.InitDataDir(path.Join(utils.Blockchains, chainType))
		if rpc {
			utils.InitDataDir(path.Join(utils.Blockchains, chainType, "rpc"))
		}
		ifExit(chain.WriteConfig(deployConf))
	}
	// copy and edit temp
	ifExit(utils.Copy(deployConf, tempConf))
	if editCfg {
		ifExit(utils.Editor(tempConf))
	}

	// deploy and install chain
	chainId, err := DeployChain(chain, tmpRoot, tempConf, deployGen, noEditor)
	ifExit(err)
	if chainId == "" {
		exit(fmt.Errorf("ChainId must not be empty. How else would we ever find you?!"))
	}
	err = InstallChain(chain, tmpRoot, chainType, tempConf, chainId, rpc)
	ifExit(err)

	s := fmt.Sprintf("Deployed and installed chain: %s/%s", chainType, chainId)
	if rpc {
		s += " with rpc"
	}
	logger.Warnln(s)
	ifExit(chain.Shutdown())
	return chainId
}

// change the currently active chain
func Checkout(c *Context) {
	args := c.Args()
	if len(args) == 0 {
		exit(fmt.Errorf("Please specify the chain to checkout"))
	}
	head := args[0]

	typ, id, err := chains.ResolveChain(head)
	ifExit(err)

	if err := chains.ChangeHead(typ, id); err != nil {
		exit(err)
	}
	logger.Infoln("Checked out new head: ", path.Join(typ, id))
	exit(nil)
}

// remove a reference from a chainId
func RmRef(c *Context) {
	args := c.Args()
	if len(args) == 0 {
		exit(fmt.Errorf("Please specify the ref to remove"))
	}
	ref := args[0]

	_, _, err := chains.ResolveChain(ref)
	ifExit(err)
	err = os.Remove(path.Join(utils.Refs, ref))
	ifExit(err)
}

// add a new reference to a chainId
func AddRef(c *Context) {
	args := c.Args()
	var typ string
	var id string
	var err error
	var ref string
	if len(args) < 1 {
		log.Fatal("Must at least enter a ref name")
	} else if len(args) == 1 {
		ref = args[0]
		typ, id, err = chains.GetHead()
		ifExit(err)
		if id == "" {
			log.Fatal(`No chain is checked out. To add a ref, specify both a chainId and a name, \n eg. "epm add thel/14c32 mychain"`)
		}
	} else {
		chain := args[0]
		ref = args[1]
		typ, id, err = chains.SplitRef(chain)

		if err != nil {
			exit(fmt.Errorf(`Error: specify the type in the first
                argument as '<type>/<chainId>'`))
		}
	}
	exit(chains.AddRef(typ, id, ref))
}

// run a node on a chain
func Run(c *Context) {
	root, chainType, chainId, err := ResolveRootFlag(c)
	ifExit(err)

	pid := os.Getpid()
	pidFile := path.Join(root, "pid")
	err = ioutil.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0600)
	ifExit(err)

	logger.Infof("Running chain %s/%s\n", chainType, chainId)
	chain, err := LoadChain(c, chainType, root)
	ifExit(err)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		chain.Shutdown()
	}()

	chain.WaitForShutdown()
	err = os.Remove(pidFile)
	ifExit(err)
}

// TODO: multi types
// TODO: deprecate in exchange for -dapp flag on run
func RunDapp(c *Context) {
	dapp := c.Args()[0]
	chainType := "thelonious"
	chainId, err := chains.ChainIdFromDapp(dapp)
	ifExit(err)
	logger.Infoln("Running chain ", chainId)
	chain, err := LoadChain(c, chainType, chains.ComposeRoot(chainType, chainId))
	ifExit(err)
	chain.WaitForShutdown()
}

// edit a config value
func Config(c *Context) {
	var (
		root      string
		chainType string
		err       error
	)
	rpc := c.Bool("rpc")
	root, chainType, _, err = ResolveRootFlag(c)
	ifExit(err)

	if rpc {
		ifExit(makeRPCDir(root))
	}

	configPath := path.Join(root, "config.json")
	if c.Bool("vi") {
		ifExit(utils.Editor(configPath))
	} else {
		m := mod.NewChain(chainType, rpc)
		if m == nil {
			ifExit(fmt.Errorf("Got nil chain. Is this the correct type: %s", chainType))
		}
		err = m.ReadConfig(configPath)
		ifExit(err)

		args := c.Args()
		for _, a := range args {
			sp := strings.Split(a, ":")
			if len(sp) != 2 {
				logger.Errorln("Invalid arg")
				continue
			}
			key := sp[0]
			value := sp[1]
			if err := m.SetProperty(key, value); err != nil {
				logger.Errorln(err)
			}
		}
		m.WriteConfig(path.Join(root, "config.json"))
	}
}

// remove a chain
func Remove(c *Context) {
	if len(c.Args()) < 1 {
		exit(fmt.Errorf("Error: specify the chain ref as an argument"))
	}
	root, _, _, err := ResolveRootArg(c)
	ifExit(err)

	if !c.IsSet("force") {
		if confirm("This will permanently delete the directory: " + root) {
			// remove the directory
			os.RemoveAll(root)
			// we only remove refs if its not a multi
			if !c.IsSet("multi") {
				// remove from head (if current head)
				_, h, _ := chains.GetHead()
				if strings.Contains(root, h) {
					chains.NullHead()
				}
				// remove refs
				refs, err := chains.GetRefs()
				ifExit(err)
				for k, v := range refs {
					if strings.Contains(root, v) {
						os.Remove(path.Join(utils.Blockchains, "refs", k))
					}
				}
			}
			// if there are no chains left, wipe the dir
			dir := path.Dir(root)
			fs, _ := ioutil.ReadDir(dir)
			if len(fs) == 0 {
				if confirm("Remove the directory " + dir + "?") {
					os.RemoveAll(dir)
				}
			}
		}
	} else {
		// remove the directory
		os.RemoveAll(root)
		// we only remove refs if its not a multi
		if !c.IsSet("multi") {
			// remove from head (if current head)
			_, h, _ := chains.GetHead()
			if strings.Contains(root, h) {
				chains.NullHead()
			}
			// remove refs
			refs, err := chains.GetRefs()
			ifExit(err)
			for k, v := range refs {
				if strings.Contains(root, v) {
					os.Remove(path.Join(utils.Blockchains, "refs", k))
				}
			}
		}
		// if there are no chains left, wipe the dir
		dir := path.Dir(root)
		fs, _ := ioutil.ReadDir(dir)
		if len(fs) == 0 {
			os.RemoveAll(dir)
		}
	}
}

// run a single epm on-chain command (endow, deploy, etc.)
func Command(c *Context) {
	root, chainType, _, err := ResolveRootFlag(c)
	ifExit(err)

	chain := mod.NewChain(chainType, c.Bool("rpc"))

	args := c.Args()
	if len(args) < 3 {
		exit(fmt.Errorf("You must specify a command and at least 2 arguments"))
	}
	cmd := args[0]
	args = args[1:]

	// put the args into a string and parse them
	argString := ""
	for _, a := range args {
		argString += a + " "
	}
	job := epm.ParseArgs(cmd, argString)

	// set contract path
	contractPath := c.String("c")
	if !c.IsSet("c") {
		contractPath = DefaultContractPath
	}
	epm.ContractPath, err = filepath.Abs(contractPath)
	ifExit(err)
	logger.Infoln("Contract path:", epm.ContractPath)

	epm.ErrMode = epm.ReturnOnErr
	// load epm
	e, err := epm.NewEPM(chain, epm.LogFile)
	ifExit(err)
	e.ReadVars(path.Join(root, EPMVars))

	// we don't need to turn anything on for "set"
	if cmd != "set" {
		ifExit(setupModule(c, chain, root))
	}

	// run job
	e.AddJob(job)
	err = e.ExecuteJobs()
	ifExit(err)
	e.WriteVars(path.Join(root, EPMVars))
}

func Test(c *Context) {
	packagePath := "."
	if len(c.Args()) > 0 {
		packagePath = c.Args()[0]
	}

	contractPath := c.String("contracts")
	dontClear := c.Bool("dont-clear")
	diffStorage := c.Bool("diff")

	chainRoot, chainType, _, err := ResolveRootFlag(c)
	ifExit(err)
	// hierarchy : name > chainId > db > config > HEAD > default

	if !c.IsSet("contracts") {
		contractPath = DefaultContractPath
	}
	epm.ContractPath, err = filepath.Abs(contractPath)
	ifExit(err)

	logger.Debugln("Contract root:", epm.ContractPath)

	// clear the cache
	if !dontClear {
		err := os.RemoveAll(utils.Epm)
		if err != nil {
			logger.Errorln("Error clearing cache: ", err)
		}
		utils.InitDataDir(utils.Epm)
	}

	// read all pdxs in the dir
	fs, err := ioutil.ReadDir(packagePath)
	ifExit(err)
	failed := make(map[string][]int)
	for _, f := range fs {
		fname := f.Name()
		if path.Ext(fname) != ".pdx" {
			continue
		}
		sp := strings.Split(fname, ".")
		pkg := sp[0]
		dir := packagePath
		if _, err := os.Stat(path.Join(dir, pkg+".pdt")); err != nil {
			continue
		}

		// setup EPM object with ChainInterface
		var chain epm.Blockchain
		chain, err = LoadChain(c, chainType, chainRoot)
		ifExit(err)
		e, err := epm.NewEPM(chain, epm.LogFile)
		ifExit(err)
		e.ReadVars(path.Join(chainRoot, EPMVars))

		// epm parse the package definition file
		err = e.Parse(path.Join(dir, fname))
		ifExit(err)

		if diffStorage {
			e.Diff = true
		}

		// epm execute jobs
		e.ExecuteJobs()
		// write epm variables to file
		e.WriteVars(path.Join(chainRoot, EPMVars))

		// run tests
		results, err := e.Test(path.Join(dir, pkg+"."+TestExt))
		if err != nil {
			logger.Errorln(err)
			if results != nil {
				logger.Errorln("Failed tests:", results.FailedTests)
			}
		}
		chain.Shutdown()
		if results.Err != "" {
			log.Fatal(results.Err)
		}
		if results.Failed > 0 {
			failed[pkg] = results.FailedTests
		}
	}
	if len(failed) == 0 {
		fmt.Println("All tests passed")
	} else {
		fmt.Println("Failed:")
		for p, ns := range failed {
			fmt.Println(p, ns)
		}
	}
}

// deploy a pdx file on a chain
func Deploy(c *Context) {
	packagePath := "."
	if len(c.Args()) > 0 {
		packagePath = c.Args()[0]
	}

	contractPath := c.String("c")
	dontClear := c.Bool("dont-clear")
	diffStorage := c.Bool("diff")

	chainRoot, chainType, _, err := ResolveRootFlag(c)
	ifExit(err)
	// hierarchy : name > chainId > db > config > HEAD > default

	// Startup the chain
	var chain epm.Blockchain
	chain, err = LoadChain(c, chainType, chainRoot)
	ifExit(err)

	if !c.IsSet("c") {
		contractPath = DefaultContractPath
	}
	epm.ContractPath, err = filepath.Abs(contractPath)
	ifExit(err)

	logger.Debugln("Contract root:", epm.ContractPath)

	// clear the cache
	if !dontClear {
		err := os.RemoveAll(utils.Epm)
		if err != nil {
			logger.Errorln("Error clearing cache: ", err)
		}
		utils.InitDataDir(utils.Epm)
	}

	// setup EPM object with ChainInterface
	e, err := epm.NewEPM(chain, epm.LogFile)
	ifExit(err)
	e.ReadVars(path.Join(chainRoot, EPMVars))

	// comb directory for package-definition file
	// exits on error
	dir, pkg, test_ := getPkgDefFile(packagePath)

	// epm parse the package definition file
	err = e.Parse(path.Join(dir, pkg+"."+PkgExt))
	ifExit(err)

	if diffStorage {
		e.Diff = true
	}

	// epm execute jobs
	e.ExecuteJobs()
	// write epm variables to file
	e.WriteVars(path.Join(chainRoot, EPMVars))

	if test_ {
		results, err := e.Test(path.Join(dir, pkg+"."+TestExt))
		if err != nil {
			logger.Errorln(err)
			if results != nil {
				logger.Errorln("Failed tests:", results.FailedTests)
			}
			e.Stop()
			fmt.Printf("Testing %s.pdt failed\n", pkg)
			os.Exit(1)
		}
	}
}

func Console(c *Context) {

	contractPath := c.String("c")
	dontClear := c.Bool("dont-clear")
	diffStorage := c.Bool("diff")

	chainRoot, chainType, _, err := ResolveRootFlag(c)
	ifExit(err)
	// hierarchy : name > chainId > db > config > HEAD > default

	// Startup the chain
	var chain epm.Blockchain
	chain, err = LoadChain(c, chainType, chainRoot)
	ifExit(err)

	if !c.IsSet("c") {
		contractPath = DefaultContractPath
	}
	epm.ContractPath, err = filepath.Abs(contractPath)
	ifExit(err)

	logger.Debugln("Contract root:", epm.ContractPath)

	// clear the cache
	if !dontClear {
		err := os.RemoveAll(utils.Epm)
		if err != nil {
			logger.Errorln("Error clearing cache: ", err)
		}
		utils.InitDataDir(utils.Epm)
	}

	// setup EPM object with ChainInterface
	e, err := epm.NewEPM(chain, epm.LogFile)
	ifExit(err)
	e.ReadVars(path.Join(chainRoot, EPMVars))

	if diffStorage {
		e.Diff = true
	}
	//e.Repl()
}

func KeyImport(c *Context) {
	logger.Warnln("Note that the key will not be physically copied until the chain is started up again.")
	if len(c.Args()) == 0 {
		exit(fmt.Errorf("Please enter path to key to import"))
	}
	keyFile := c.Args()[0]
	useKey(keyFile, c)
}

func KeyUse(c *Context) {
	if len(c.Args()) == 0 {
		exit(fmt.Errorf("Please enter a key name to use."))
	}
	var keyFile string
	keyName := c.Args()[0]
	allKeys, err := filepath.Glob(path.Join(utils.Keys, keyName) + "*")
	ifExit(err)
	if len(allKeys) > 1 {
		var i int
		fmt.Println("More than one key found with that name. Please select the proper one.")
		for key := range allKeys {
			fmt.Printf("%v.\t%s\n", (key + 1), allKeys[key])
		}
		fmt.Printf(">>> ")
		fmt.Scan(&i)
		keyFile = allKeys[(i - 1)]
	} else if len(allKeys) == 1 {
		keyFile = allKeys[0]
	} else {
		exit(fmt.Errorf("No key found with that name."))
	}
	useKey(keyFile, c)
}

func useKey(keyFile string, c *Context) {

	name := path.Base(keyFile)

	// set key in chain's config
	rpc := c.Bool("rpc")
	root, chainType, _, err := ResolveRootFlag(c)
	ifExit(err)

	// ugly hack because cli loses the global flag with nested commands (sigh)
	if os.Args[1] == "--rpc" {
		root = path.Join(root, "rpc")
	}

	configPath := path.Join(root, "config.json")
	m := mod.NewChain(chainType, rpc)
	err = m.ReadConfig(configPath)
	ifExit(err)

	if err := m.SetProperty("KeyFile", keyFile); err != nil {
		logger.Errorln(err)
	}
	if err := m.SetProperty("KeySession", name); err != nil {
		logger.Errorln(err)
	}
	ifExit(m.WriteConfig(path.Join(root, "config.json")))
	logger.Warnln("Using key ", name)
}

func KeyExport(c *Context) {
	if len(c.Args()) == 0 {
		exit(fmt.Errorf("Please enter a location to export to"))
	}
	dst := c.Args()[0]
	logger.Warnln("Note that only keys which were generated by `epm keys gen` will be exported")
	if _, err := os.Stat(dst); err != nil {
		ifExit(os.MkdirAll(dst, 0700))
	}
	fs, err := ioutil.ReadDir(utils.Keys)
	ifExit(err)
	for _, f := range fs {
		s := path.Join(utils.Keys, f.Name())
		d := path.Join(dst, f.Name())
		ifExit(utils.Copy(s, d))
	}
	logger.Warnln("Done")
}

func KeyPublic(c *Context) {
	if len(c.Args()) == 0 {
		exit(fmt.Errorf("You must enter a key name/address whose public key we should print"))
	}
	keyType := c.String("type")
	name := c.Args()[0]

	key, err := ioutil.ReadFile(path.Join(utils.Keys, name))
	ifExit(err)
	switch keyType {
	case "secp256k1", "bitcoin", "ethereum", "thelonious":
		// TODO
	case "ed25519", "tendermint":
		// keys are saved appended to public key
		l := len(key)
		pubkey := key[l/2:]
		fmt.Println(string(pubkey))
	default:
		exit(fmt.Errorf("Unknown key type: %s", keyType))

	}

}

// TODO: move to its own binary
func Keygen(c *Context) {
	if len(c.Args()) == 0 {
		exit(fmt.Errorf("Please enter a name for your key"))
	}
	name := c.Args()[0]
	typ := c.String("type")

	// check for keygen binary
	bin := path.Join(GoPath, "bin", "epm-keygen")
	src := path.Join(utils.ErisLtd, "epm-go", "cmd", "epm-keygen")
	cur, _ := os.Getwd()
	ifExit(os.Chdir(src))
	if _, err := os.Stat(bin); err != nil {
		cmd := exec.Command("go", "install")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		err := cmd.Run()
		ifExit(os.Chdir(cur))
		if err != nil {
			exit(err)
		}
	}

	buf := new(bytes.Buffer)
	// run keygen
	// the only output is the name of the keyfile (for now)
	cmd := exec.Command(bin, "-type", typ, name)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	ifExit(cmd.Run())
	keyFile := string(buf.Bytes())
	fmt.Println(path.Base(strings.TrimSpace(keyFile)))

	if c.Bool("import") {
		// set key in chain's config
		rpc := c.Bool("rpc")
		root, chainType, _, err := ResolveRootFlag(c)
		ifExit(err)

		configPath := path.Join(root, "config.json")
		m := mod.NewChain(chainType, rpc)
		err = m.ReadConfig(configPath)
		ifExit(err)

		if err := m.SetProperty("KeyFile", keyFile); err != nil {
			logger.Errorln(err)
		}
		if err := m.SetProperty("KeySession", name); err != nil {
			logger.Errorln(err)
		}
		m.WriteConfig(path.Join(root, "config.json"))
	}
}

func Install(c *Context) {
	if len(c.Args()) == 0 {
		ifExit(fmt.Errorf("Please provide a path to the dapp to install"))
	}
	dappPath := c.Args()[0]
	dappName := path.Base(dappPath)
	if len(c.Args()) > 1 {
		dappName = c.Args()[1]
	}

	if strings.Contains(dappPath, "github.com") {
		// make sure the path doesn't exist before trying to clone
		if _, err := os.Stat(dappPath); err != nil {
			logger.Infoln("fetching dapp from", dappPath)
			cmd := exec.Command("git", "clone", "https://"+dappPath, path.Join(utils.Dapps, dappName))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			ifExit(cmd.Run())
			dappPath = path.Join(utils.Dapps, dappName)
		}
	}
	time.Sleep(time.Second) // TODO: remove

	pdxPath := path.Join(dappPath, "contracts")

	r := make([]byte, 8)
	rand.Read(r)
	tmpRoot := path.Join(utils.Scratch, "epm", hex.EncodeToString(r))

	var chainRoot string
	var chainId string
	// deploy a new chain for installation of the dapp
	if !c.Bool("no-new") {
		chainType := "thelonious"

		forceName := c.String("force-name")
		name := c.String("name")
		deployConf := c.String("config")
		deployGen := c.String("genesis")
		tempConf := ".config.json"
		editCfg := c.Bool("edit-config")
		rpc := c.Bool("rpc")
		// if we provide genesis, dont open editor for genesis
		noEditor := c.IsSet("genesis")

		// if deployConf not given, and the dapp has a config.json, use that
		if !c.IsSet("config") {
			if _, err := os.Stat(path.Join(dappPath, "config.json")); err == nil {
				deployConf = path.Join(dappPath, "config.json")
			}
		}

		// if deployGen not given, and the dapp has a genesis.json, use that
		if !c.IsSet("genesis") {
			if _, err := os.Stat(path.Join(dappPath, "genesis.json")); err == nil {
				deployGen = path.Join(dappPath, "genesis.json")
			}
		}

		// install chain
		chainId = deployInstallChain(tmpRoot, deployConf, deployGen, tempConf, chainType, rpc, editCfg, noEditor)

		ifExit(chains.ChangeHead(chainType, chainId))
		logger.Warnf("Checked out chain: %s/%s", chainType, chainId)

		updateRefs(chainType, chainId, forceName, name)
		chainRoot = chains.ComposeRootMulti("thelonious", chainId, "0")
	} else {
		var err error
		chainRoot, _, chainId, err = ResolveRootFlag(c)
		ifExit(err)
	}

	// deploy pdx
	contractPath := c.String("c")

	// Startup the chain
	logger.Warnln("Starting up chain:", chainRoot)
	var chain epm.Blockchain
	chain, err := LoadChain(c, "thelonious", chainRoot)
	ifExit(err)

	if !c.IsSet("c") {
		// contractPath = DefaultContractPath
		contractPath = pdxPath
	}
	epm.ContractPath, err = filepath.Abs(contractPath)
	ifExit(err)

	logger.Debugln("Contract root:", epm.ContractPath)

	// clear cache
	err = os.RemoveAll(utils.Epm)
	if err != nil {
		logger.Errorln("Error clearing cache: ", err)
	}
	utils.InitDataDir(utils.Epm)

	// setup EPM object with ChainInterface
	e, err := epm.NewEPM(chain, epm.LogFile)
	ifExit(err)
	e.ReadVars(path.Join(chainRoot, EPMVars))

	// comb directory for package-definition file
	// exits on error
	dir, pkg, test_ := getPkgDefFile(pdxPath)

	// epm parse the package definition file
	err = e.Parse(path.Join(dir, pkg+"."+PkgExt))
	ifExit(err)

	diffStorage := c.Bool("diff")
	if diffStorage {
		e.Diff = true
	}

	// epm execute jobs
	e.ExecuteJobs()
	// write epm variables to file
	e.WriteVars(path.Join(chainRoot, EPMVars))
	// wait for a block
	e.Commit()
	// run tests
	if test_ {
		results, err := e.Test(path.Join(dir, pkg+"."+TestExt))
		if err != nil {
			logger.Errorln(err)
			if results != nil {
				logger.Errorln("Failed tests:", results.FailedTests)
			}
		}
	}

	var rootContract string
	b, err := ioutil.ReadFile(path.Join(chainRoot, EPMVars))
	ifExit(err)
	sp := strings.Split(string(b), "\n")
	for _, s := range sp {
		sp := strings.Split(s, ":")
		name := sp[0]
		val := sp[1]
		if name == "ROOT" {
			rootContract = val
		}
	}
	// TODO: fetch root contract from vars...

	// install dapp into eris tree if not there yet
	abs, err := filepath.Abs(dappPath)
	ifExit(err) // this should never happen ...
	p := path.Join(utils.Dapps, dappName)
	if !strings.Contains(abs, p) {
		ifExit(utils.Copy(dappPath, p))
	}

	// update package.json with chainid and root contract
	p = path.Join(p, "package.json")
	b, err = ioutil.ReadFile(p)
	ifExit(err)
	var pkgFile dapps.PackageFile
	var monkData dapps.MonkData
	err = json.Unmarshal(b, &pkgFile)
	ifExit(err)
	deps := pkgFile.ModuleDependencies
	for i, d := range deps {
		if d.Name == "monk" {
			data := d.Data // json.RawMessage
			err := json.Unmarshal(*data, &monkData)
			ifExit(err)
			monkData.ChainId = "0x" + chainId
			monkData.RootContract = rootContract
			b, err := json.Marshal(monkData)
			ifExit(err)
			raw := json.RawMessage(b)
			deps[i].Data = &raw
			break
		}
	}
	pkgFile.ModuleDependencies = deps
	b, err = json.MarshalIndent(pkgFile, "", "\t")
	ifExit(err)
	err = ioutil.WriteFile(p, b, 0600)
	ifExit(err)
}

func Accounts(c *Context) {
	account := ""
	if len(c.Args()) > 0 {
		account = c.Args()[0]
	}

	root, chainType, _, err := ResolveRootFlag(c)
	ifExit(err)
	chain, err := LoadChain(c, chainType, root)
	ifExit(err)

	if account == "" {
		// dump list of all accounts
		world := chain.WorldState()
		for _, s := range world.Order {
			a := world.Accounts[s]
			p := "account"
			if a.IsScript {
				p = "contract"
			}
			fmt.Println(a.Address, p)
		}
	} else {
		account := chain.Account(account)
		if account == nil {
			fmt.Printf("Account %s does not exist\n", account)
		}
		fmt.Printf("Balance: %s\n", account.Balance)
		fmt.Printf("Nonce: %s\n", account.Nonce)
		if account.IsScript {
			storage := account.Storage
			fmt.Printf("Code: %s\n", account.Script)
			for _, s := range storage.Order {
				fmt.Printf("%s : %s\n", s, storage.Storage[s])
			}
		}

	}
}
