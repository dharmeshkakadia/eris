package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/modules/mint"
	mintconfig "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/config"
	"github.com/eris-ltd/epm-go/epm"
	"github.com/eris-ltd/epm-go/utils"
)

func NewChain(chainType string, rpc bool) epm.Blockchain {
	switch chainType {
	case "tendermint", "mint":
		if rpc {
			return mint.NewMintRPC()
		} else {
			return mint.NewMint()
		}
	}
	return nil

}

func setGenesisConfigMint(m *mint.MintModule, genesis string) {
	if strings.HasSuffix(genesis, ".pdx") || strings.HasSuffix(genesis, ".gdx") {
		//m.GenesisConfig = &monkdoug.GenesisConfig{Address: "0000000000THISISDOUG", NoGenDoug: false, Pdx: genesis}
		//m.GenesisConfig.Init()
	} else {
		m.Config.GenesisConfig = genesis
	}
}

func ChainSpecificDeploy(chain epm.Blockchain, deployGen, root string, novi bool) error {
	tempGen := path.Join(root, "genesis.json")
	utils.InitDataDir(root)

	if deployGen == "" {
		deployGen = path.Join(utils.Blockchains, "tendermint", "genesis.json")
	}
	if _, err := os.Stat(deployGen); err != nil {
		err := ioutil.WriteFile(deployGen, []byte(mintconfig.DefaultGenesis), 0600)
		if err != nil {
			return err
		}
	}
	if err := utils.Copy(deployGen, tempGen); err != nil {
		return err
	}
	if !novi {
		if err := utils.Editor(tempGen); err != nil {
			return err
		}
	}

	tmint := chain.(*mint.MintModule)
	setGenesisConfigMint(tmint, tempGen)
	return nil
}

func Fetch(chainType, peerserver string) ([]byte, error) {
	return nil, fmt.Errorf("Fetch not supported for mint")
}
