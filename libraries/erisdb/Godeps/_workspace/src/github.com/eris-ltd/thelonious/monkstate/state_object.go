package monkstate

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkcrypto"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monktrie"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

type Code []byte

func (self Code) String() string {
	return string(self) //strings.Join(Disassemble(self), " ")
}

type Storage map[string]*monkutil.Value

func (self Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range self {
		// XXX Do we need a 'value' copy or is this sufficient?
		cpy[key] = value
	}

	return cpy
}

func (s *StateObject) String() string {
	return fmt.Sprintf("Address: %x\nBalance: %d\nNonce %x\nRoot %x\nCode %x\n", s.address, s.Balance.Bytes(), s.Nonce, s.State.Root(), s.CodeHash())
}

type StateObject struct {
	// Address of the object
	address []byte
	// Shared attributes
	Balance  *big.Int
	codeHash []byte
	Nonce    uint64
	// Contract related attributes
	State    *State
	Code     Code
	InitCode Code

	storage Storage

	// Total gas pool is the total amount of gas currently
	// left if this object is the coinbase. Gas is directly
	// purchased of the coinbase.
	gasPool *big.Int

	// Mark for deletion
	// When an object is marked for deletion it will be delete from the trie
	// during the "update" phase of the state transition
	remove bool

	mut sync.Mutex
}

func (self *StateObject) Reset() {
	self.mut.Lock()
	defer self.mut.Unlock()
	self.storage = make(Storage)
	self.State.Reset()
}

func NewStateObject(addr []byte) *StateObject {
	// This to ensure that it has 20 bytes (and not 0 bytes), thus left or right pad doesn't matter.
	address := monkutil.Address(addr)

	object := &StateObject{address: address, Balance: new(big.Int), gasPool: new(big.Int)}
	object.State = New(monktrie.New(monkutil.Config.Db, ""))
	object.storage = make(Storage)
	object.gasPool = new(big.Int)

	return object
}

func NewContract(address []byte, balance *big.Int, root []byte) *StateObject {
	contract := NewStateObject(address)
	contract.Balance = balance
	contract.State = New(monktrie.New(monkutil.Config.Db, string(root)))

	return contract
}

func NewStateObjectFromBytes(address, data []byte) *StateObject {
	object := &StateObject{address: address}
	object.RlpDecode(data)

	return object
}

func (self *StateObject) MarkForDeletion() {
	self.remove = true
	statelogger.DebugDetailf("%x: #%d %v (deletion)\n", self.Address(), self.Nonce, self.Balance)
}

func (c *StateObject) GetAddr(addr []byte) *monkutil.Value {
	return monkutil.NewValueFromBytes([]byte(c.State.Trie.Get(string(addr))))
}

func (c *StateObject) SetAddr(addr []byte, value interface{}) {
	c.State.Trie.Update(string(addr), string(monkutil.NewValue(value).Encode()))
}

func (self *StateObject) GetStorage(key *big.Int) *monkutil.Value {
	return self.getStorage(key.Bytes())
}
func (self *StateObject) SetStorage(key *big.Int, value *monkutil.Value) {
	self.setStorage(key.Bytes(), value)
}

func (self *StateObject) getStorage(k []byte) *monkutil.Value {
	self.mut.Lock()
	defer self.mut.Unlock()

	key := monkutil.LeftPadBytes(k, 32)

	value := self.storage[string(key)]
	if value == nil {
		value = self.GetAddr(key)

		if !value.IsNil() {
			self.storage[string(key)] = value
		}
	}

	return value

	//return self.GetAddr(key)
}

func (self *StateObject) setStorage(k []byte, value *monkutil.Value) {
	self.mut.Lock()
	defer self.mut.Unlock()
	key := monkutil.LeftPadBytes(k, 32)
	self.storage[string(key)] = value.Copy()
}

// Iterate over each storage address and yield callback
func (self *StateObject) EachStorage(cb monktrie.EachCallback) {
	self.mut.Lock()
	defer self.mut.Unlock()
	// First loop over the uncommit/cached values in storage
	for key, value := range self.storage {
		// XXX Most iterators Fns as it stands require encoded values
		encoded := monkutil.NewValue(value.Encode())
		cb(key, encoded)
	}

	it := self.State.Trie.NewIterator()
	it.Each(func(key string, value *monkutil.Value) {
		// If it's cached don't call the callback.
		if self.storage[key] == nil {
			cb(key, value)
		}
	})
}

func (self *StateObject) Sync() {
	self.mut.Lock()
	defer self.mut.Unlock()

	for key, value := range self.storage {
		if value.Len() == 0 { // value.BigInt().Cmp(monkutil.Big0) == 0 {
			//data := self.getStorage([]byte(key))
			//fmt.Printf("deleting %x %x 0x%x\n", self.Address(), []byte(key), data)
			self.State.Trie.Delete(string(key))
			continue
		}

		self.SetAddr([]byte(key), value)
	}

	valid, t2 := monktrie.ParanoiaCheck(self.State.Trie)
	if !valid {
		statelogger.Infof("Warn: PARANOIA: Different state storage root during copy %x vs %x\n", self.State.Trie.Root, t2.Root)

		self.State.Trie = t2
	}
}

func (c *StateObject) GetInstr(pc *big.Int) *monkutil.Value {
	if int64(len(c.Code)-1) < pc.Int64() {
		return monkutil.NewValue(0)
	}

	return monkutil.NewValueFromBytes([]byte{c.Code[pc.Int64()]})
}

func (c *StateObject) AddAmount(amount *big.Int) {
	c.SetBalance(new(big.Int).Add(c.Balance, amount))

	statelogger.Debugf("%x: #%d %v (+ %v)\n", c.Address(), c.Nonce, c.Balance, amount)
}

func (c *StateObject) SubAmount(amount *big.Int) {
	c.SetBalance(new(big.Int).Sub(c.Balance, amount))

	statelogger.Debugf("%x: #%d %v (- %v)\n", c.Address(), c.Nonce, c.Balance, amount)
}

func (c *StateObject) SetBalance(amount *big.Int) {
	c.Balance = amount
}

//
// Gas setters and getters
//

// Return the gas back to the origin. Used by the Virtual machine or Closures
func (c *StateObject) ReturnGas(gas, price *big.Int) {}
func (c *StateObject) ConvertGas(gas, price *big.Int) error {
	total := new(big.Int).Mul(gas, price)
	if total.Cmp(c.Balance) > 0 {
		return fmt.Errorf("insufficient amount: %v, %v", c.Balance, total)
	}

	c.SubAmount(total)

	return nil
}

func (self *StateObject) SetGasPool(gasLimit *big.Int) {
	self.gasPool = new(big.Int).Set(gasLimit)

	statelogger.DebugDetailf("%x: fuel (+ %v)", self.Address(), self.gasPool)
}

func (self *StateObject) BuyGas(gas, price *big.Int) error {
	if self.gasPool.Cmp(gas) < 0 {
		return GasLimitError(self.gasPool, gas)
	}

	rGas := new(big.Int).Set(gas)
	rGas.Mul(rGas, price)

	self.AddAmount(rGas)

	return nil
}

func (self *StateObject) RefundGas(gas, price *big.Int) {
	self.gasPool.Add(self.gasPool, gas)

	rGas := new(big.Int).Set(gas)
	rGas.Mul(rGas, price)

	self.Balance.Sub(self.Balance, rGas)
}

func (self *StateObject) Copy() *StateObject {
	self.mut.Lock()
	defer self.mut.Unlock()
	stateObject := NewStateObject(self.Address())
	stateObject.Balance.Set(self.Balance)
	stateObject.codeHash = monkutil.CopyBytes(self.codeHash)
	stateObject.Nonce = self.Nonce
	if self.State != nil {
		stateObject.State = self.State.Copy()
	}
	stateObject.Code = monkutil.CopyBytes(self.Code)
	stateObject.InitCode = monkutil.CopyBytes(self.InitCode)
	stateObject.storage = self.storage.Copy()
	stateObject.gasPool.Set(self.gasPool)
	stateObject.remove = self.remove

	return stateObject
}

func (self *StateObject) Set(stateObject *StateObject) {
	*self = *stateObject
}

//
// Attribute accessors
//

func (c *StateObject) N() *big.Int {
	return big.NewInt(int64(c.Nonce))
}

// Returns the address of the contract/account
func (c *StateObject) Address() []byte {
	return c.address
}

// Returns the initialization Code
func (c *StateObject) Init() Code {
	return c.InitCode
}

// To satisfy ClosureRef
func (self *StateObject) Object() *StateObject {
	return self
}

// Debug stuff
func (self *StateObject) CreateOutputForDiff() {
	fmt.Printf("%x %x %x %x\n", self.Address(), self.State.Root(), self.Balance.Bytes(), self.Nonce)
	self.EachStorage(func(addr string, value *monkutil.Value) {
		fmt.Printf("%x %x\n", addr, value.Bytes())
	})
}

//
// Encoding
//

// State object encoding methods
func (c *StateObject) RlpEncode() []byte {
	var root interface{}
	if c.State != nil {
		root = c.State.Trie.Root
	} else {
		root = ""
	}

	return monkutil.Encode([]interface{}{c.Nonce, c.Balance, root, c.CodeHash()})
}

func (c *StateObject) GetCodeHash() monkutil.Bytes {
	return c.codeHash
}

func (c *StateObject) CodeHash() monkutil.Bytes {
	var codeHash []byte
	if len(c.Code) > 0 {
		codeHash = monkcrypto.Sha3Bin(c.Code)
	}

	return codeHash
}

func (c *StateObject) RlpDecode(data []byte) {
	c.mut.Lock()
	defer c.mut.Unlock()
	decoder := monkutil.NewValueFromBytes(data)

	c.Nonce = decoder.Get(0).Uint()
	c.Balance = decoder.Get(1).BigInt()
	root := decoder.Get(2).Interface()
	c.State = New(monktrie.New(monkutil.Config.Db, root))
	c.storage = make(map[string]*monkutil.Value)
	c.gasPool = new(big.Int)

	c.codeHash = decoder.Get(3).Bytes()

	c.Code, _ = monkutil.Config.Db.Get(c.codeHash)
}

// Storage change object. Used by the manifest for notifying changes to
// the sub channels.
type StorageState struct {
	StateAddress []byte
	Address      []byte
	Value        *big.Int
}
