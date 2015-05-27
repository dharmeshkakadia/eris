package monkchain

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"
	_ "strconv"
	"time"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/go-ethereum/crypto/secp256k1"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monktrie"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

type BlockInfo struct {
	Number uint64
	Hash   []byte
	Parent []byte
	TD     *big.Int
}

func (bi *BlockInfo) RlpDecode(data []byte) {
	decoder := monkutil.NewValueFromBytes(data)

	bi.Number = decoder.Get(0).Uint()
	bi.Hash = decoder.Get(1).Bytes()
	bi.Parent = decoder.Get(2).Bytes()
	bi.TD = decoder.Get(3).BigInt()
}

func (bi *BlockInfo) RlpEncode() []byte {
	return monkutil.Encode([]interface{}{bi.Number, bi.Hash, bi.Parent, bi.TD})
}

type Blocks []*Block

func (self Blocks) AsSet() monkutil.UniqueSet {
	set := make(monkutil.UniqueSet)
	for _, block := range self {
		set.Insert(block.Hash())
	}

	return set
}

type BlockBy func(b1, b2 *Block) bool

func (self BlockBy) Sort(blocks Blocks) {
	bs := blockSorter{
		blocks: blocks,
		by:     self,
	}
	sort.Sort(bs)
}

type blockSorter struct {
	blocks Blocks
	by     func(b1, b2 *Block) bool
}

func (self blockSorter) Len() int { return len(self.blocks) }
func (self blockSorter) Swap(i, j int) {
	self.blocks[i], self.blocks[j] = self.blocks[j], self.blocks[i]
}
func (self blockSorter) Less(i, j int) bool { return self.by(self.blocks[i], self.blocks[j]) }

func Number(b1, b2 *Block) bool { return b1.Number.Cmp(b2.Number) < 0 }

type Block struct {
	// Hash to the previous block
	PrevHash monkutil.Bytes
	// Uncles of this block
	Uncles   Blocks
	UncleSha []byte
	// The coin base address
	Coinbase []byte
	// Block Trie state
	//state *monkutil.Trie
	state *monkstate.State
	// Difficulty for the current block
	Difficulty *big.Int
	// Creation time
	Time int64
	// The block number
	Number *big.Int
	// Minimum Gas Price
	MinGasPrice *big.Int
	// Gas limit
	GasLimit *big.Int
	// Gas used
	GasUsed *big.Int
	// Extra data
	Extra string
	// Block Nonce for verification
	Nonce monkutil.Bytes
	// List of transactions and/or contracts
	transactions []*Transaction
	receipts     []*Receipt
	TxSha        []byte

	// signature for verified miners
	v    byte
	r, s []byte
}

func NewBlockFromBytes(raw []byte) *Block {
	block := &Block{}
	block.RlpDecode(raw)

	return block
}

// New block takes a raw encoded string
func NewBlockFromRlpValue(rlpValue *monkutil.Value) *Block {
	block := &Block{}
	block.RlpValueDecode(rlpValue)

	return block
}

func CreateBlock(root interface{},
	prevHash []byte,
	base []byte,
	Difficulty *big.Int,
	Nonce []byte,
	extra string) *Block {

	block := &Block{
		PrevHash:    prevHash,
		Coinbase:    base,
		Difficulty:  Difficulty,
		Nonce:       Nonce,
		Time:        time.Now().Unix(),
		Extra:       extra,
		UncleSha:    EmptyShaList,
		GasUsed:     new(big.Int),
		MinGasPrice: new(big.Int),
		GasLimit:    new(big.Int),
	}
	block.SetUncles([]*Block{})

	block.state = monkstate.New(monktrie.New(monkutil.Config.Db, root))

	return block
}

// Returns a hash of the block
func (block *Block) Hash() monkutil.Bytes {
	return monkcrypto.Sha3Bin(monkutil.NewValue(block.header()).Encode())
	//return monkcrypto.Sha3Bin(block.Value().Encode())
}

func (block *Block) HashNoNonce() []byte {
	return monkcrypto.Sha3Bin(monkutil.Encode([]interface{}{block.PrevHash,
		block.UncleSha, block.Coinbase, block.state.Trie.Root,
		block.TxSha, block.Difficulty, block.Number, block.MinGasPrice,
		block.GasLimit, block.GasUsed, block.Time, block.Extra}))
}

func (block *Block) State() *monkstate.State {
	return block.state
}

func (block *Block) Transactions() []*Transaction {
	return block.transactions
}

// TODO: GenDoug?
func (block *Block) CalcGasLimit(parent *Block) *big.Int {
	return monkutil.Big("100000000000000000000000")

	if block.Number.Cmp(big.NewInt(0)) == 0 {
		return monkutil.BigPow(10, 6)
	}

	// ((1024-1) * parent.gasLimit + (gasUsed * 6 / 5)) / 1024

	previous := new(big.Int).Mul(big.NewInt(1024-1), parent.GasLimit)
	current := new(big.Rat).Mul(new(big.Rat).SetInt(parent.GasUsed), big.NewRat(6, 5))
	curInt := new(big.Int).Div(current.Num(), current.Denom())

	result := new(big.Int).Add(previous, curInt)
	result.Div(result, big.NewInt(1024))

	min := big.NewInt(125000)

	return monkutil.BigMax(min, result)
}

func (block *Block) BlockInfo() BlockInfo {
	bi := BlockInfo{}
	data, _ := monkutil.Config.Db.Get(append(block.Hash(), []byte("Info")...))
	bi.RlpDecode(data)

	return bi
}

func (self *Block) GetTransaction(hash []byte) *Transaction {
	for _, receipt := range self.receipts {
		if bytes.Compare(receipt.Tx.Hash(), hash) == 0 {
			return receipt.Tx
		}
	}

	return nil
}

// Sync the block's state and contract respectively
func (block *Block) Sync() {
	block.state.Sync()
}

func (block *Block) Undo() {
	// Sync the block state itself
	block.state.Reset()
}

/////// Block Encoding
func (block *Block) rlpReceipts() interface{} {
	// Marshal the transactions of this block
	encR := make([]interface{}, len(block.receipts))
	for i, r := range block.receipts {
		// Cast it to a string (safe)
		encR[i] = r.RlpData()
	}

	return encR
}

func (block *Block) rlpUncles() interface{} {
	// Marshal the transactions of this block
	uncles := make([]interface{}, len(block.Uncles))
	for i, uncle := range block.Uncles {
		// Cast it to a string (safe)
		uncles[i] = uncle.header()
	}

	return uncles
}

func (block *Block) SetUncles(uncles []*Block) {
	block.Uncles = uncles

	// Sha of the concatenated uncles
	block.UncleSha = monkcrypto.Sha3Bin(monkutil.Encode(block.rlpUncles()))
}

func (self *Block) SetReceipts(receipts []*Receipt, txs []*Transaction) {
	self.receipts = receipts
	self.setTransactions(txs)
}

func (block *Block) setTransactions(txs []*Transaction) {
	block.transactions = txs
}

func CreateTxSha(receipts Receipts) (sha []byte) {
	trie := monktrie.New(monkutil.Config.Db, "")
	for i, receipt := range receipts {
		trie.Update(string(monkutil.NewValue(i).Encode()), string(monkutil.NewValue(receipt.RlpData()).Encode()))
	}

	switch trie.Root.(type) {
	case string:
		sha = []byte(trie.Root.(string))
	case []byte:
		sha = trie.Root.([]byte)
	default:
		panic(fmt.Sprintf("invalid root type %T", trie.Root))
	}

	return sha
}

func (self *Block) SetTxHash(receipts Receipts) {
	self.TxSha = CreateTxSha(receipts)
}

func (block *Block) Value() *monkutil.Value {
	return monkutil.NewValue([]interface{}{block.header(), block.rlpReceipts(), block.rlpUncles(), []interface{}{block.v, block.r, block.s}})
}

func (block *Block) RlpEncode() []byte {
	// Encode a slice interface which contains the header and the list of
	// transactions.
	return block.Value().Encode()
}

func (block *Block) RlpDecode(data []byte) {
	rlpValue := monkutil.NewValueFromBytes(data)
	block.RlpValueDecode(rlpValue)
}

func (block *Block) RlpValueDecode(decoder *monkutil.Value) {
	header := decoder.Get(0)

	block.PrevHash = header.Get(0).Bytes()
	block.UncleSha = header.Get(1).Bytes()
	block.Coinbase = header.Get(2).Bytes()
	block.state = monkstate.New(monktrie.New(monkutil.Config.Db, header.Get(3).Val))
	block.TxSha = header.Get(4).Bytes()
	block.Difficulty = header.Get(5).BigInt()
	block.Number = header.Get(6).BigInt()
	//fmt.Printf("#%v : %x\n", block.Number, block.Coinbase)
	block.MinGasPrice = header.Get(7).BigInt()
	block.GasLimit = header.Get(8).BigInt()
	block.GasUsed = header.Get(9).BigInt()
	block.Time = int64(header.Get(10).BigInt().Uint64())
	block.Extra = header.Get(11).Str()
	block.Nonce = header.Get(12).Bytes()

	// Tx list might be empty if this is an uncle. Uncles only have their
	// header set.
	if decoder.Get(1).IsNil() == false { // Yes explicitness
		receipts := decoder.Get(1)
		block.transactions = make([]*Transaction, receipts.Len())
		block.receipts = make([]*Receipt, receipts.Len())
		for i := 0; i < receipts.Len(); i++ {
			receipt := NewRecieptFromValue(receipts.Get(i))
			block.transactions[i] = receipt.Tx
			block.receipts[i] = receipt
		}

	}

	if decoder.Get(2).IsNil() == false { // Yes explicitness
		uncles := decoder.Get(2)
		block.Uncles = make([]*Block, uncles.Len())
		for i := 0; i < uncles.Len(); i++ {
			block.Uncles[i] = NewUncleBlockFromValue(uncles.Get(i))
		}
	}

	if decoder.Get(3).IsNil() == false { // Yes explicitness
		sig := decoder.Get(3)
		block.v = sig.Get(0).Byte()
		block.r = sig.Get(1).Bytes()
		block.s = sig.Get(2).Bytes()
	}

}

func NewUncleBlockFromValue(header *monkutil.Value) *Block {
	block := &Block{}

	block.PrevHash = header.Get(0).Bytes()
	block.UncleSha = header.Get(1).Bytes()
	block.Coinbase = header.Get(2).Bytes()
	block.state = monkstate.New(monktrie.New(monkutil.Config.Db, header.Get(3).Val))
	block.TxSha = header.Get(4).Bytes()
	block.Difficulty = header.Get(5).BigInt()
	block.Number = header.Get(6).BigInt()
	block.MinGasPrice = header.Get(7).BigInt()
	block.GasLimit = header.Get(8).BigInt()
	block.GasUsed = header.Get(9).BigInt()
	block.Time = int64(header.Get(10).BigInt().Uint64())
	block.Extra = header.Get(11).Str()
	block.Nonce = header.Get(12).Bytes()

	return block
}

func (block *Block) GetRoot() interface{} {
	return block.state.Trie.Root
}

func (self *Block) Receipts() []*Receipt {
	return self.receipts
}

// blocks need signatures that should match the coinbase
func (self *Block) Signature(key []byte) []byte {
	hash := self.Hash()
	sig, _ := secp256k1.Sign(hash, key)
	return sig
}

func (self *Block) Sign(privk []byte) []byte {
	sig := self.Signature(privk)
	sig[64] += 27
	self.r = sig[:32]
	self.s = sig[32:64]
	self.v = sig[64]
	return sig
}

func (self *Block) GetSig() []byte {
	if self.r != nil && self.s != nil {
		return append(self.r, append(self.s, self.v)...)
	}
	return nil
}

func (self *Block) PublicKey() []byte {
	hash := self.Hash()

	r := monkutil.LeftPadBytes(self.r, 32)
	s := monkutil.LeftPadBytes(self.s, 32)
	sig := append(r, s...)
	sig = append(sig, self.v-27)

	pubkey, _ := secp256k1.RecoverPubkey(hash, sig)

	return pubkey
}

func (self *Block) Signer() []byte {
	if len(self.r) == 0 || len(self.s) == 0 {
		return []byte("\x00")
	}

	pubkey := self.PublicKey()

	if pubkey[0] != 4 {
		return nil
	}

	return monkcrypto.Sha3Bin(pubkey[1:])[12:]
}

func (block *Block) Header() []interface{} {
	return block.header()
}

func (block *Block) header() []interface{} {
	return []interface{}{
		// Sha of the previous block
		block.PrevHash,
		// Sha of uncles
		block.UncleSha,
		// Coinbase address
		block.Coinbase,
		// root state
		block.state.Trie.Root,
		// Sha of tx
		block.TxSha,
		// Current block Difficulty
		block.Difficulty,
		// The block number
		block.Number,
		// Block minimum gas price
		block.MinGasPrice,
		// Block upper gas bound
		block.GasLimit,
		// Block gas used
		block.GasUsed,
		// Time the block was found?
		block.Time,
		// Extra data
		block.Extra,
		// Block's Nonce for validation
		block.Nonce,
	}
}

func (block *Block) String() string {
	return fmt.Sprintf(`
	BLOCK(%x): Size: %v
	PrevHash:   %x
	UncleSha:   %x
	Coinbase:   %x
	Root:       %x
	TxSha:      %x
	Difficulty: %v
	Number:     %v
	MinGas:     %v
	MaxLimit:   %v
	GasUsed:    %v
	Time:       %v
	Extra:      %v
	Nonce:      %x
	NumTx:      %v
`,
		block.Hash(),
		block.Size(),
		block.PrevHash,
		block.UncleSha,
		block.Coinbase,
		block.state.Trie.Root,
		block.TxSha,
		block.Difficulty,
		block.Number,
		block.MinGasPrice,
		block.GasLimit,
		block.GasUsed,
		block.Time,
		block.Extra,
		block.Nonce,
		len(block.transactions),
	)
}

func (self *Block) Size() monkutil.StorageSize {
	return monkutil.StorageSize(len(self.RlpEncode()))
}
