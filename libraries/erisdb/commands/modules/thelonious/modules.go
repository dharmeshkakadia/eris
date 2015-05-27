package commands

import (
	"fmt"
	"github.com/eris-ltd/epm-go/chains"
	"github.com/eris-ltd/epm-go/epm"
	"github.com/eris-ltd/epm-go/utils"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strings"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/modules/monkrpc"
	mutils "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/modules/monkutils"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monk"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkdoug"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

func NewChain(chainType string, rpc bool) epm.Blockchain {
	switch chainType {
	case "thel", "thelonious", "monk":
		if rpc {
			return monkrpc.NewMonkRpcModule()
		} else {
			return monk.NewMonk(nil)
		}
	}
	// TODO raise WrongChain error!
	return nil
}

func isThelonious(chain epm.Blockchain) (*monk.MonkModule, bool) {
	th, ok := chain.(*monk.MonkModule)
	return th, ok
}

func setGenesisConfig(m *monk.MonkModule, genesis string) {
	if strings.HasSuffix(genesis, ".pdx") || strings.HasSuffix(genesis, ".gdx") {
		m.GenesisConfig = &monkdoug.GenesisConfig{Address: "0000000000THISISDOUG", NoGenDoug: false, Pdx: genesis}
		m.GenesisConfig.Init()
	} else {
		m.Config.GenesisConfig = genesis
	}
}

func copyEditGenesisConfig(deployGen, tmpRoot string, novi bool) (string, error) {
	tempGen := path.Join(tmpRoot, "genesis.json")
	utils.InitDataDir(tmpRoot)

	if deployGen == "" {
		deployGen = path.Join(utils.Blockchains, "thelonious", "genesis.json")
	}
	if _, err := os.Stat(deployGen); err != nil {
		err := utils.WriteJson(monkdoug.DefaultGenesis, deployGen)
		return "", err
	}
	if err := utils.Copy(deployGen, tempGen); err != nil {
		return "", err
	}
	if !novi {
		if err := utils.Editor(tempGen); err != nil {
			return "", err
		}
	}
	return tempGen, nil
}

func ChainSpecificDeploy(chain epm.Blockchain, deployGen, root string, novi bool) error {
	tempGen, err := copyEditGenesisConfig(deployGen, root, novi)
	if err != nil {
		return err
	}
	th := chain.(*monk.MonkModule)
	setGenesisConfig(th, tempGen)
	return nil
}

func Fetch(chainType, peerserver string) ([]byte, error) {
	peerip, _, err := net.SplitHostPort(peerserver)
	if err != nil {
		return nil, err
	}
	peerserver = "http://" + peerserver

	chainId, err := thelonious.GetChainId(peerserver)
	if err != nil {
		return nil, err
	}

	rootDir := chains.ComposeRoot(chainType, monkutil.Bytes2Hex(chainId))
	monkutil.Config = &monkutil.ConfigManager{ExecPath: rootDir, Debug: true, Paranoia: true}
	utils.InitLogging(rootDir, "", 5, "")
	db := mutils.NewDatabase("database", false)
	monkutil.Config.Db = db

	genesisBlock, err := thelonious.GetGenesisBlock(peerserver)
	if err != nil {
		return nil, err
	}

	db.Put([]byte("GenesisBlock"), genesisBlock.RlpEncode())
	db.Put([]byte("ChainID"), chainId)

	hash := genesisBlock.GetRoot()
	hashB, ok := hash.([]byte)
	if !ok {
		return nil, fmt.Errorf("State root is not []byte:", hash)
	}
	err = thelonious.GetGenesisState(peerserver, monkutil.Bytes2Hex(hashB), db)
	if err != nil {
		return nil, err
	}
	db.Close()

	// get genesis.json
	g, err := thelonious.GetGenesisJson(peerserver)
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(path.Join(rootDir, "genesis.json"), g, 0600)
	if err != nil {
		return nil, err
	}

	peerport, err := thelonious.GetFetchPeerPort(peerserver)
	if err != nil {
		return nil, err
	}

	// drop config
	chain := NewChain(chainType, false)
	chain.SetProperty("RootDir", rootDir)
	chain.SetProperty("RemoteHost", peerip)
	chain.SetProperty("RemotePort", peerport)
	chain.SetProperty("UseSeed", true)
	err = chain.WriteConfig(path.Join(rootDir, "config.json"))
	if err != nil {
		return nil, err
	}

	return chainId, nil
}
