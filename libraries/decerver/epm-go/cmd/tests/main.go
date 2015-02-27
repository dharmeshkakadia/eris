package main

import (
	"fmt"
	"github.com/eris-ltd/epm-go"
	"github.com/eris-ltd/thelonious/monk"
	//"github.com/eris-ltd/thelonious/monkutil"
	"os"
	"path"
)

var GoPath = os.Getenv("GOPATH")

func NewMonkModule() *monk.MonkModule {
	epm.ErrMode = epm.ReturnOnErr
	m := monk.NewMonk(nil)
	m.Config.RootDir = ".ethchain"
	m.Config.LogLevel = 5
	m.Config.GenesisConfig = "genesis.json"
	m.Config.LLLLocal = true
	g := m.LoadGenesis(m.Config.GenesisConfig)
	g.Difficulty = 14
	m.SetGenesis(g)
	m.Init()
	m.Config.Mining = false
	m.Start()
	return m
}

// test the epm test file mechanism
func main() {
	// Startup the EthChain
	m := NewMonkModule()
	// Create ChainInterface instance
	//ethD := epm.NewEthD(eth)
	// setup EPM object with ChainInterface
	e, err := epm.NewEPM(m, ".epm-log-deploy-test")
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	// subscribe to new blocks..
	// epm parse the package definition file
	err = e.Parse(path.Join(epm.TestPath, "test_parse.epm"))
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	// epm execute jobs
	e.ExecuteJobs()
	e.Commit()
	e.Test(path.Join(epm.TestPath, "test_parse.epm-check"))

	//epm.PrintDiff(e.PrevState(), e.CurrentState())
	//eth.GetStorage()
}
