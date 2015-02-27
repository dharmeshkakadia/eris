package main

import (
	"fmt"
	"github.com/eris-ltd/decerver"
	"github.com/eris-ltd/decerver-modules/ipfs"
	"github.com/eris-ltd/decerver-modules/legalmarkdown"
	"github.com/eris-ltd/decerver-modules/monk"
	//"github.com/eris-ltd/modules/blockchaininfo"
	"os"
)

func main() {
	dc := decerver.NewDeCerver()
	fm := ipfs.NewIpfsModule()
	lmd := legalmarkdown.NewLmdModule()
	mjs := monkjs.NewMonkModule()
	//bci := blockchaininfo.NewBlkChainInfo()
	
	dc.LoadModule(fm)
	dc.LoadModule(lmd)
	//dc.LoadModule(bci)
	dc.LoadModule(mjs)
	
	errInit := dc.Init()
	if errInit != nil {
		fmt.Printf("Module failed to initialize: %s. Shutting down.\n", errInit.Error())
		os.Exit(1)
	}
	
	//Run decerver
	errStart := dc.Start()
	if errStart != nil {
		fmt.Printf("Module failed to start: %s. Shutting down.\n", errStart.Error())
		os.Exit(1)
	}
	
	dc.Shutdown()
}
