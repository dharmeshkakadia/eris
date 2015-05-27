package commands

import (
	"fmt"
	"github.com/eris-ltd/epm-go/epm"
	"log"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/modules/eth"
)

func NewChain(chainType string, rpc bool) epm.Blockchain {
	switch chainType {
	case "eth", "ethereum":
		if rpc {
			log.Fatal("Eth rpc not implemented yet")
		} else {
			return eth.NewEth(nil)
		}
	}
	return nil

}

// This will be called and should do nothing.
func ChainSpecificDeploy(chain epm.Blockchain, deployGen, root string, novi bool) error {
	return nil
}

// This is invalid
func Fetch(chainType, peerserver string) ([]byte, error) {
	return nil, fmt.Errorf("Fetch not defined for eth")
}
