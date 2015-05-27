package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/tendermint/tendermint/account"
	"github.com/eris-ltd/epm-go/utils"
)

// XXX: The only output (for now) should be the keyFile
func Keygen(c *cli.Context) {
	name := c.Args()[0]
	keyType := c.String("type")

	var (
		prv  []byte
		addr []byte
	)

	switch keyType {
	case "secp256k1", "bitcoin", "ethereum", "thelonious":
		// create a new ecdsa key
		key := monkcrypto.GenerateNewKeyPair()
		prv = key.PrivateKey
		addr = key.Address()
	case "ed25519", "tendermint":
		privAccount := account.GenPrivAccount()
		prv = []byte(privAccount.PrivKey.(account.PrivKeyEd25519))
		addr = privAccount.Address
	default:
		fmt.Println("Unknown key type: %s", keyType)
		os.Exit(1)
	}
	a := hex.EncodeToString(addr)
	if name != "" {
		name += "-"
	}
	name += a
	prvHex := hex.EncodeToString(prv)

	// write key to file
	keyFile := path.Join(utils.Keys, name)
	err := ioutil.WriteFile(keyFile, []byte(prvHex), 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err.Error())
		os.Exit(1)
	}
	fmt.Println(keyFile)
}

func main() {
	app := cli.NewApp()
	app.Name = "epm-keygen"
	app.Usage = "Generate keys for various cryptographic purposes"
	app.Version = "0.0.1"
	app.Author = "Ethan Buchman"
	app.Email = "ethan@erisindustries.com"
	app.Flags = []cli.Flag{
		typeFlag,
	}
	app.Action = Keygen
	fmt.Println(os.Args)
	app.Run(os.Args)

}

var (
	typeFlag = cli.StringFlag{
		Name:  "type",
		Value: "secp256k1",
		Usage: "specify the type of key to create",
	}
)
