package main

import (
	"bytes"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/eris-ltd/epm-go/chains"
	"github.com/eris-ltd/epm-go/epm"
	"github.com/eris-ltd/epm-go/utils"
	"github.com/eris-ltd/thelonious/monklog"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

// TODO: needs work..
func cleanupEPM() {
	dirs := []string{epm.EpmDir}
	for _, d := range dirs {
		err := os.RemoveAll(d)
		if err != nil {
			logger.Errorln("Error removing dir", d, err)
		}
	}
}

func installEPM() {
	cur, _ := os.Getwd()
	os.Chdir(path.Join(utils.ErisLtd, "epm-go", "cmd", "epm"))
	cmd := exec.Command("go", "install")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Run()
	logger.Infoln(out.String())
	os.Chdir(cur)
}

func pullErisRepo(repo, branch string) {
	// pull changes
	os.Chdir(path.Join(utils.ErisLtd, repo))
	cmd := exec.Command("git", "pull", "origin", branch)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Run()
	res := out.String()
	logger.Infoln(res)
}

func updateEPM() {
	cur, _ := os.Getwd()

	pullErisRepo("epm-go", "master")
	pullErisRepo("decerver-interfaces", "master")

	// install
	installEPM()

	// return to original dir
	os.Chdir(cur)
}

func cleanPullUpdate(clean, pull, update bool) {
	if clean && pull {
		cleanupEPM()
		updateEPM()
	} else if clean {
		cleanupEPM()
		if update {
			installEPM()
		}
	} else if pull {
		updateEPM()
	} else if update {
		installEPM()
	}
}

// looks for pkg-def file
// exits if error (none or more than 1)
// returns dir of pkg, name of pkg (no extension) and whether or not there's a test file
func getPkgDefFile(pkgPath string) (string, string, bool) {
	logger.Infoln("Pkg path:", pkgPath)
	var pkgName string
	var test_ bool

	// if its not a directory, look for a corresponding test file
	f, err := os.Stat(pkgPath)
	ifExit(err)

	if !f.IsDir() {
		dir, fil := path.Split(pkgPath)
		spl := strings.Split(fil, ".")
		pkgName = spl[0]
		ext := spl[1]
		if ext != PkgExt {
			exit(fmt.Errorf("Did not understand extension. Got %s, expected %s\n", ext, PkgExt))
		}

		_, err := os.Stat(path.Join(dir, pkgName) + "." + TestExt)
		if err != nil {
			logger.Errorf("There was no test found for package-definition %s. Deploying without test ...\n", pkgName)
			test_ = false
		} else {
			test_ = true
		}
		return dir, pkgName, test_
	}

	// read dir for files
	files, err := ioutil.ReadDir(pkgPath)
	ifExit(err)

	// find all package-defintion and package-definition-test files
	candidates := make(map[string]int)
	candidates_test := make(map[string]int)
	for _, f := range files {
		name := f.Name()
		spl := strings.Split(name, ".")
		if len(spl) < 2 {
			continue
		}
		name = spl[0]
		ext := spl[1]
		if ext == PkgExt {
			candidates[name] = 1
		} else if ext == TestExt {
			candidates_test[name] = 1
		}
	}
	// exit if too many or no options
	if len(candidates) > 1 {
		exit(fmt.Errorf("More than one package-definition file available. Please select with the '-p' flag"))
	} else if len(candidates) == 0 {
		exit(fmt.Errorf("No package-definition files found for extensions", PkgExt, TestExt))
	}
	// this should run once (there's only one candidate)
	for k, _ := range candidates {
		pkgName = k
		if candidates_test[pkgName] == 1 {
			test_ = true
		} else {
			logger.Infoln("There was no test found for package-definition %s. Deploying without test ...\n", pkgName)
			test_ = false
		}
	}
	return pkgPath, pkgName, test_
}

func checkInit() error {
	if _, err := os.Stat(path.Join(utils.Blockchains, "config.json")); err != nil {
		return fmt.Errorf("Could not find default config. Did you run `epm init` ?")
	}
	if _, err := os.Stat(path.Join(utils.Blockchains, "genesis.json")); err != nil {
		return fmt.Errorf("Could not find default genesis.json. Did you run `epm init` ?")
	}
	return nil
}

func editor(file string) error {
	editr := os.Getenv("EDITOR")
	switch editr {
	case "", "vim", "vi":
		return vi(file)
	case "emacs":
		return emacs(file)
	}
	return fmt.Errorf("Unknown editor %s", editr)
}

func emacs(file string) error {
	cmd := exec.Command("emacs", file)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func vi(file string) error {
	cmd := exec.Command("vim", file)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func exit(err error) {
	if err != nil {
		logger.Errorln(err)
	}
	monklog.Flush()
	os.Exit(0)
}

func ifExit(err error) {
	if err != nil {
		logger.Errorln(err)
		monklog.Flush()
		os.Exit(0)
	}
}

func confirm(message string) bool {
	fmt.Println(message, "Are you sure? (y/n)")
	var r string
	fmt.Scanln(&r)
	for ; ; fmt.Scanln(&r) {
		if r == "n" || r == "y" {
			break
		} else {
			fmt.Printf("Yes or no?", r)
		}
	}
	return r == "y"
}

// Read config, set deployment root, config gen block (if relevant)
// init, return chainId
func DeployChain(chain epm.Blockchain, root, config, deployGen string, novi bool) (string, error) {
	chain.ReadConfig(config)
	chain.SetProperty("RootDir", root)

	// TODO: nice way to handle multiple gen blocks on other chains
	// set genesis config file
	if th, ok := isThelonious(chain); ok {
		tempGen := copyEditGenesisConfig(deployGen, root, novi)
		setGenesisConfig(th, tempGen)
	} else if deployGen != "" {
		logger.Warnln("Genesis configuration only possible with thelonious (for now - https://github.com/eris-ltd/epm-go/issues/53)")
	}

	if err := chain.Init(); err != nil {
		return "", err
	}

	// get chain ID!
	return chain.ChainId()
}

// Copy files and deploy directory into global tree. Set configuration values for root dir and chain id.
func InstallChain(chain epm.Blockchain, root, chainType, tempConf, chainId string, rpc bool) error {
	// chain.Shutdown() and New again (so we dont move db while its open - does this even matter though?) !
	home := chains.ComposeRoot(chainType, chainId)
	if rpc {
		home = path.Join(home, "rpc")
	}
	logger.Infoln("Install directory ", home)
	// move datastore and configs
	// be sure to copy paths into config
	if err := utils.InitDataDir(home); err != nil {
		return err
	}
	if err := os.Rename(root, home); err != nil {
		return err
	}

	logger.Infoln("Loading and setting chainId ", tempConf)
	// set chainId and rootdir values in config file
	chain.ReadConfig(tempConf)

	chain.SetProperty("ChainId", chainId)
	//chain.SetProperty("ChainName", name)
	chain.SetProperty("RootDir", home)
	if chainType == "thelonious" {
		chain.SetProperty("GenesisConfig", path.Join(home, "genesis.json"))
	}
	chain.WriteConfig(tempConf)

	if err := os.Rename(tempConf, path.Join(home, "config.json")); err != nil {
		return err
	}

	return nil
}

func resolveRootFlag(c *cli.Context) (string, string, string, error) {
	ref := c.String("chain")
	rpc := c.GlobalBool("rpc")
	multi := c.String("multi")
	return resolveRoot(ref, rpc, multi)
}

func resolveRootArg(c *cli.Context) (string, string, string, error) {
	args := c.Args()
	ref := ""
	if len(args) > 0 {
		ref = args[0]
	}
	rpc := c.GlobalBool("rpc")
	multi := c.String("multi")
	return resolveRoot(ref, rpc, multi)
}

func resolveRoot(ref string, rpc bool, multi string) (string, string, string, error) {
	chainType, chainId, err := chains.ResolveChain(ref)
	if err != nil {
		return "", "", "", err
	}
	root := chains.ComposeRootMulti(chainType, chainId, multi)
	if rpc {
		root = path.Join(root, "rpc")
	}
	return root, chainType, chainId, nil
}
