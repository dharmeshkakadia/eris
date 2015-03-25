package monkpipe

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

type JSBlock struct // Block interface exposed to QML
{
	//Transactions string `json:"transactions"`
	ref          *monkchain.Block
	Size         string         `json:"size"`
	Number       int            `json:"number"`
	Hash         string         `json:"hash"`
	Transactions *monkutil.List `json:"transactions"`
	Time         int64          `json:"time"`
	Coinbase     string         `json:"coinbase"`
	Name         string         `json:"name"`
	GasLimit     string         `json:"gasLimit"`
	GasUsed      string         `json:"gasUsed"`
	PrevHash     string         `json:"prevHash"`
}

// Creates a new QML Block from a chain block
func NewJSBlock(block *monkchain.Block) *JSBlock {
	if block == nil {
		return &JSBlock{}
	}

	var ptxs []JSTransaction
	for _, tx := range block.Transactions() {
		ptxs = append(ptxs, *NewJSTx(tx))
	}

	list := monkutil.NewList(ptxs)

	return &JSBlock{
		ref: block, Size: block.Size().String(),
		Number: int(block.Number.Uint64()), GasUsed: block.GasUsed.String(),
		GasLimit: block.GasLimit.String(), Hash: monkutil.Bytes2Hex(block.Hash()),
		Transactions: list, Time: block.Time,
		Coinbase: monkutil.Bytes2Hex(block.Coinbase),
		PrevHash: monkutil.Bytes2Hex(block.PrevHash),
	}
}

func (self *JSBlock) ToString() string {
	if self.ref != nil {
		return self.ref.String()
	}

	return ""
}

func (self *JSBlock) GetTransaction(hash string) *JSTransaction {
	tx := self.ref.GetTransaction(monkutil.Hex2Bytes(hash))
	if tx == nil {
		return nil
	}

	return NewJSTx(tx)
}

type JSTransaction struct {
	ref *monkchain.Transaction

	Value           string `json:"value"`
	Gas             string `json:"gas"`
	GasPrice        string `json:"gasPrice"`
	Hash            string `json:"hash"`
	Address         string `json:"address"`
	Sender          string `json:"sender"`
	RawData         string `json:"rawData"`
	Data            string `json:"data"`
	Contract        bool   `json:"isContract"`
	CreatesContract bool   `json:"createsContract"`
	Confirmations   int    `json:"confirmations"`
}

func NewJSTx(tx *monkchain.Transaction) *JSTransaction {
	hash := monkutil.Bytes2Hex(tx.Hash())
	receiver := monkutil.Bytes2Hex(tx.Recipient)
	if receiver == "0000000000000000000000000000000000000000" {
		receiver = monkutil.Bytes2Hex(tx.CreationAddress())
	}
	sender := monkutil.Bytes2Hex(tx.Sender())
	createsContract := tx.CreatesContract()

	var data string
	if tx.CreatesContract() {
		data = strings.Join(monkchain.Disassemble(tx.Data), "\n")
	} else {
		data = monkutil.Bytes2Hex(tx.Data)
	}

	return &JSTransaction{ref: tx, Hash: hash, Value: monkutil.CurrencyToString(tx.Value), Address: receiver, Contract: tx.CreatesContract(), Gas: tx.Gas.String(), GasPrice: tx.GasPrice.String(), Data: data, Sender: sender, CreatesContract: createsContract, RawData: monkutil.Bytes2Hex(tx.Data)}
}

func (self *JSTransaction) ToString() string {
	return self.ref.String()
}

type JSKey struct {
	Address    string `json:"address"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

func NewJSKey(key *monkcrypto.KeyPair) *JSKey {
	return &JSKey{monkutil.Bytes2Hex(key.Address()), monkutil.Bytes2Hex(key.PrivateKey), monkutil.Bytes2Hex(key.PublicKey)}
}

type JSObject struct {
	*Object
}

func NewJSObject(object *Object) *JSObject {
	return &JSObject{object}
}

type PReceipt struct {
	CreatedContract bool   `json:"createdContract"`
	Address         string `json:"address"`
	Hash            string `json:"hash"`
	Sender          string `json:"sender"`
}

func NewPReciept(contractCreation bool, creationAddress, hash, address []byte) *PReceipt {
	return &PReceipt{
		contractCreation,
		monkutil.Bytes2Hex(creationAddress),
		monkutil.Bytes2Hex(hash),
		monkutil.Bytes2Hex(address),
	}
}

// Peer interface exposed to QML

type JSPeer struct {
	ref          *monkchain.Peer
	Inbound      bool   `json:"isInbound"`
	LastSend     int64  `json:"lastSend"`
	LastPong     int64  `json:"lastPong"`
	Ip           string `json:"ip"`
	Port         int    `json:"port"`
	Version      string `json:"version"`
	LastResponse string `json:"lastResponse"`
	Latency      string `json:"latency"`
	Caps         string `json:"caps"`
}

func NewJSPeer(peer monkchain.Peer) *JSPeer {
	if peer == nil {
		return nil
	}

	var ip []string
	for _, i := range peer.Host() {
		ip = append(ip, strconv.Itoa(int(i)))
	}
	ipAddress := strings.Join(ip, ".")

	var caps []string
	capsIt := peer.Caps().NewIterator()
	for capsIt.Next() {
		caps = append(caps, capsIt.Value().Str())
	}

	return &JSPeer{ref: &peer, Inbound: peer.Inbound(), LastSend: peer.LastSend().Unix(), LastPong: peer.LastPong(), Version: peer.Version(), Ip: ipAddress, Port: int(peer.Port()), Latency: peer.PingTime(), Caps: fmt.Sprintf("%v", caps)}
}

type JSReceipt struct {
	CreatedContract bool   `json:"createdContract"`
	Address         string `json:"address"`
	Hash            string `json:"hash"`
	Sender          string `json:"sender"`
}

func NewJSReciept(contractCreation bool, creationAddress, hash, address []byte) *JSReceipt {
	return &JSReceipt{
		contractCreation,
		monkutil.Bytes2Hex(creationAddress),
		monkutil.Bytes2Hex(hash),
		monkutil.Bytes2Hex(address),
	}
}

type JSMessage struct {
	To        string `json:"to"`
	From      string `json:"from"`
	Input     string `json:"input"`
	Output    string `json:"output"`
	Path      int32  `json:"path"`
	Origin    string `json:"origin"`
	Timestamp int32  `json:"timestamp"`
	Coinbase  string `json:"coinbase"`
	Block     string `json:"block"`
	Number    int32  `json:"number"`
	Value     string `json:"value"`
}

func NewJSMessage(message *monkstate.Message) JSMessage {
	return JSMessage{
		To:        monkutil.Bytes2Hex(message.To),
		From:      monkutil.Bytes2Hex(message.From),
		Input:     monkutil.Bytes2Hex(message.Input),
		Output:    monkutil.Bytes2Hex(message.Output),
		Path:      int32(message.Path),
		Origin:    monkutil.Bytes2Hex(message.Origin),
		Timestamp: int32(message.Timestamp),
		Coinbase:  monkutil.Bytes2Hex(message.Origin),
		Block:     monkutil.Bytes2Hex(message.Block),
		Number:    int32(message.Number.Int64()),
		Value:     message.Value.String(),
	}
}
