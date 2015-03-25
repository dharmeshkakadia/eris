package monkpipe

import (
	"bytes"
	"encoding/json"
	"sync/atomic"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

type JSPipe struct {
	*Pipe
}

func NewJSPipe(eth monkchain.NodeManager) *JSPipe {
	return &JSPipe{New(eth)}
}

func (self *JSPipe) BlockByHash(strHash string) *JSBlock {
	hash := monkutil.Hex2Bytes(strHash)
	block := self.obj.ChainManager().GetBlock(hash)

	return NewJSBlock(block)
}

func (self *JSPipe) BlockByNumber(num int32) *JSBlock {
	if num == -1 {
		return NewJSBlock(self.obj.ChainManager().CurrentBlock())
	}

	return NewJSBlock(self.obj.ChainManager().GetBlockByNumber(uint64(num)))
}

func (self *JSPipe) Block(v interface{}) *JSBlock {
	if n, ok := v.(int32); ok {
		return self.BlockByNumber(n)
	} else if str, ok := v.(string); ok {
		return self.BlockByHash(str)
	} else if f, ok := v.(float64); ok { // Don't ask ...
		return self.BlockByNumber(int32(f))
	}

	return nil
}

func (self *JSPipe) Key() *JSKey {
	return NewJSKey(self.obj.KeyManager().KeyPair())
}

func (self *JSPipe) StateObject(addr string) *JSObject {
	object := &Object{self.World().safeGet(monkutil.Hex2Bytes(addr))}

	return NewJSObject(object)
}

func (self *JSPipe) PeerCount() int {
	return self.obj.PeerCount()
}

func (self *JSPipe) Peers() []JSPeer {
	var peers []JSPeer
	for peer := self.obj.Peers().Front(); peer != nil; peer = peer.Next() {
		p := peer.Value.(monkchain.Peer)
		// we only want connected peers
		if atomic.LoadInt32(p.Connected()) != 0 {
			peers = append(peers, *NewJSPeer(p))
		}
	}

	return peers
}

func (self *JSPipe) IsMining() bool {
	return self.obj.IsMining()
}

func (self *JSPipe) IsListening() bool {
	return self.obj.IsListening()
}

func (self *JSPipe) CoinBase() string {
	return monkutil.Bytes2Hex(self.obj.KeyManager().Address())
}

func (self *JSPipe) NumberToHuman(balance string) string {
	b := monkutil.Big(balance)

	return monkutil.CurrencyToString(b)
}

func (self *JSPipe) StorageAt(addr, storageAddr string) string {
	storage := self.World().SafeGet(monkutil.Hex2Bytes(addr)).Storage(monkutil.Hex2Bytes(storageAddr))

	return monkutil.Bytes2Hex(storage.Bytes())
}

func (self *JSPipe) BalanceAt(addr string) string {
	return self.World().SafeGet(monkutil.Hex2Bytes(addr)).Balance.String()
}

func (self *JSPipe) TxCountAt(address string) int {
	// return transitional state nonce
	return int(self.obj.BlockManager().TransState().GetOrNewStateObject(monkutil.Hex2Bytes(address)).Nonce)
}

func (self *JSPipe) CodeAt(address string) string {
	return monkutil.Bytes2Hex(self.World().SafeGet(monkutil.Hex2Bytes(address)).Code)
}

func (self *JSPipe) IsContract(address string) bool {
	return len(self.World().SafeGet(monkutil.Hex2Bytes(address)).Code) > 0
}

func (self *JSPipe) SecretToAddress(key string) string {
	pair, err := monkcrypto.NewKeyPairFromSec(monkutil.Hex2Bytes(key))
	if err != nil {
		return ""
	}

	return monkutil.Bytes2Hex(pair.Address())
}

func (self *JSPipe) Execute(addr, value, gas, price, data string) (string, error) {
	ret, err := self.ExecuteObject(&Object{
		self.World().safeGet(monkutil.Hex2Bytes(addr))},
		monkutil.Hex2Bytes(data),
		monkutil.NewValue(value),
		monkutil.NewValue(gas),
		monkutil.NewValue(price),
	)

	return monkutil.Bytes2Hex(ret), err
}

type KeyVal struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (self *JSPipe) EachStorage(addr string) string {
	var values []KeyVal
	object := self.World().SafeGet(monkutil.Hex2Bytes(addr))
	object.EachStorage(func(name string, value *monkutil.Value) {
		value.Decode()
		values = append(values, KeyVal{monkutil.Bytes2Hex([]byte(name)), monkutil.Bytes2Hex(value.Bytes())})
	})

	valuesJson, err := json.Marshal(values)
	if err != nil {
		return ""
	}

	return string(valuesJson)
}

func (self *JSPipe) ToAscii(str string) string {
	padded := monkutil.RightPadBytes([]byte(str), 32)

	return "0x" + monkutil.Bytes2Hex(padded)
}

func (self *JSPipe) FromAscii(str string) string {
	if monkutil.IsHex(str) {
		str = str[2:]
	}

	return string(bytes.Trim(monkutil.Hex2Bytes(str), "\x00"))
}

func (self *JSPipe) FromNumber(str string) string {
	if monkutil.IsHex(str) {
		str = str[2:]
	}

	return monkutil.BigD(monkutil.Hex2Bytes(str)).String()
}

func (self *JSPipe) Transact(key, toStr, valueStr, gasStr, gasPriceStr, codeStr string) (*JSReceipt, error) {
	var hash []byte
	var contractCreation bool
	if len(toStr) == 0 {
		contractCreation = true
	} else {
		// Check if an address is stored by this address
		addr := self.World().Config().Get("NameReg").StorageString(toStr).Bytes()
		if len(addr) > 0 {
			hash = addr
		} else {
			hash = monkutil.Hex2Bytes(toStr)
		}
	}

	var keyPair *monkcrypto.KeyPair
	var err error
	if monkutil.IsHex(key) {
		keyPair, err = monkcrypto.NewKeyPairFromSec([]byte(monkutil.Hex2Bytes(key[2:])))
	} else {
		keyPair, err = monkcrypto.NewKeyPairFromSec([]byte(monkutil.Hex2Bytes(key)))
	}

	if err != nil {
		return nil, err
	}

	var (
		value    = monkutil.Big(valueStr)
		gas      = monkutil.Big(gasStr)
		gasPrice = monkutil.Big(gasPriceStr)
		data     []byte
		tx       *monkchain.Transaction
	)

	if monkutil.IsHex(codeStr) {
		data = monkutil.Hex2Bytes(codeStr[2:])
	} else {
		data = monkutil.Hex2Bytes(codeStr)
	}

	if contractCreation {
		tx = monkchain.NewContractCreationTx(value, gas, gasPrice, data)
	} else {
		tx = monkchain.NewTransactionMessage(hash, value, gas, gasPrice, data)
	}

	acc := self.obj.BlockManager().TransState().GetOrNewStateObject(keyPair.Address())
	tx.Nonce = acc.Nonce
	acc.Nonce += 1
	self.obj.BlockManager().TransState().UpdateStateObject(acc)

	tx.Sign(keyPair.PrivateKey)
	self.obj.TxPool().QueueTransaction(tx)

	if contractCreation {
		logger.Infof("Contract addr %x", tx.CreationAddress())
	}

	return NewJSReciept(contractCreation, tx.CreationAddress(), tx.Hash(), keyPair.Address()), nil
}

func (self *JSPipe) PushTx(txStr string) (*JSReceipt, error) {
	tx := monkchain.NewTransactionFromBytes(monkutil.Hex2Bytes(txStr))
	self.obj.TxPool().QueueTransaction(tx)
	return NewJSReciept(tx.CreatesContract(), tx.CreationAddress(), tx.Hash(), tx.Sender()), nil
}

func ToJSMessages(messages monkstate.Messages) *monkutil.List {
	var msgs []JSMessage
	for _, m := range messages {
		msgs = append(msgs, NewJSMessage(m))
	}

	return monkutil.NewList(msgs)
}
