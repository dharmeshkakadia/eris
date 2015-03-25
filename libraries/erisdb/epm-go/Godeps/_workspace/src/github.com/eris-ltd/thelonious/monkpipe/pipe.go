package monkpipe

import (
	//"strings"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkvm"
)

var logger = monklog.NewLogger("PIPE")

type VmVars struct {
	State *monkstate.State
}

type Pipe struct {
	obj          monkchain.NodeManager
	stateManager *monkchain.BlockManager
	blockChain   *monkchain.ChainManager
	world        *World

	Vm VmVars
}

func New(obj monkchain.NodeManager) *Pipe {
	pipe := &Pipe{
		obj:          obj,
		stateManager: obj.BlockManager(),
		blockChain:   obj.ChainManager(),
	}
	pipe.world = NewWorld(pipe)

	return pipe
}

func (self *Pipe) Balance(addr []byte) *monkutil.Value {
	return monkutil.NewValue(self.World().safeGet(addr).Balance)
}

func (self *Pipe) Nonce(addr []byte) uint64 {
	return self.World().safeGet(addr).Nonce
}

func (self *Pipe) Execute(addr []byte, data []byte, value, gas, price *monkutil.Value) ([]byte, error) {
	return self.ExecuteObject(&Object{self.World().safeGet(addr)}, data, value, gas, price)
}

func (self *Pipe) ExecuteObject(object *Object, data []byte, value, gas, price *monkutil.Value) ([]byte, error) {
	var (
		initiator = monkstate.NewStateObject(self.obj.KeyManager().KeyPair().Address())
		block     = self.blockChain.CurrentBlock()
	)

	self.Vm.State = self.World().State().Copy()

	vm := monkvm.New(NewEnv(self.Vm.State, block, value.BigInt(), initiator.Address()))
	vm.Verbose = true

	msg := monkvm.NewMessage(vm, object.Address(), data, gas.BigInt(), price.BigInt(), value.BigInt())
	ret, err := msg.Exec(object.Address(), initiator)

	return ret, err
}

func (self *Pipe) Block(hash []byte) *monkchain.Block {
	return self.blockChain.GetBlock(hash)
}

func (self *Pipe) Storage(addr, storageAddr []byte) *monkutil.Value {
	return self.World().safeGet(addr).GetStorage(monkutil.BigD(storageAddr))
}

func (self *Pipe) ToAddress(priv []byte) []byte {
	pair, err := monkcrypto.NewKeyPairFromSec(priv)
	if err != nil {
		return nil
	}

	return pair.Address()
}

func (self *Pipe) Exists(addr []byte) bool {
	return self.World().Get(addr) != nil
}

func (self *Pipe) TransactString(key *monkcrypto.KeyPair, rec string, value, gas, price *monkutil.Value, data string) ([]byte, error) {
	// Check if an address is stored by this address
	var hash []byte
	addr := self.World().Config().Get("NameReg").StorageString(rec).Bytes()
	if len(addr) > 0 {
		hash = addr
	} else if monkutil.IsHex(rec) {
		hash = monkutil.Hex2Bytes(rec[2:])
	} else {
		hash = monkutil.Hex2Bytes(rec)
	}

	return self.Transact(key, hash, value, gas, price, data)
}

/*
   Notes on data string
   if this creates a contract:
       data is either
           - hex - compiled script
           - text - filename
   if this is a regular transaction
       data is either
           - hex - packed input bytes
           - ascii version of packed bytes
*/
func (self *Pipe) Transact(key *monkcrypto.KeyPair, rec []byte, value, gas, price *monkutil.Value, data string) ([]byte, error) {
	//var hash []byte
	var contractCreation bool
	if rec == nil {
		contractCreation = true
	}

	var tx *monkchain.Transaction
	// Compile and assemble the given data
	if contractCreation {
		if monkutil.IsHex(data) {
			script := monkutil.Hex2Bytes(data[2:])
			tx = monkchain.NewContractCreationTx(value.BigInt(), gas.BigInt(), price.BigInt(), script)
		} else {
			/*
				script, err := monkutil.Compile(data, false)
				if err != nil {
					return nil, err
				}*/
			script := monkutil.Hex2Bytes(data)
			tx = monkchain.NewContractCreationTx(value.BigInt(), gas.BigInt(), price.BigInt(), script)
		}
	} else {
		/*
			data := monkutil.StringToByteFunc(data, func(s string) (ret []byte) {
				slice := strings.Split(s, "\n")
				for _, dataItem := range slice {
					d := monkutil.FormatData(dataItem)
					ret = append(ret, d...)
				}
				return
			})
		*/
		var d []byte
		if len(data) > 0 && data[:2] == "0x" {
			d = monkutil.Hex2Bytes(data[2:])
		} else {
			d = []byte(data)
		}
		tx = monkchain.NewTransactionMessage(rec, value.BigInt(), gas.BigInt(), price.BigInt(), d)
	}

	acc := self.stateManager.TransState().GetOrNewStateObject(key.Address())
	tx.Nonce = acc.Nonce
	acc.Nonce += 1
	self.stateManager.TransState().UpdateStateObject(acc)
	tx.Sign(key.PrivateKey)
	self.obj.TxPool().QueueTransaction(tx)

	if contractCreation {
		logger.Infof("Contract addr %x", tx.CreationAddress())
		//logger.Infoln(tx.String())
		return tx.CreationAddress(), nil
	}

	return tx.Hash(), nil
}

func (self *Pipe) PushTx(tx *monkchain.Transaction) ([]byte, error) {
	self.obj.TxPool().QueueTransaction(tx)
	if tx.Recipient == nil {
		logger.Infof("Contract addr %x", tx.CreationAddress())
		return tx.CreationAddress(), nil
	}
	return tx.Hash(), nil
}
