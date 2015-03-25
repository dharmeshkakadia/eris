package monkchain

import (
	"fmt"
	"math/big"
	"runtime"

	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monktrie"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkvm"
)

/*
 * The State transitioning model
 *
 * A state transition is a change made when a transaction is applied to the current world state
 * The state transitioning model does all all the necessary work to work out a valid new state root.
 * 1) Nonce handling
 * 2) Pre pay / buy gas of the coinbase (miner)
 * 3) Create a new state object if the recipient is \0*32
 * 4) Value transfer
 * == If contract creation ==
 * 4a) Attempt to run transaction data
 * 4b) If valid, use result as code for the new state object
 * == end ==
 * 5) Run Script section
 * 6) Derive new state root
 */
type StateTransition struct {
	coinbase, receiver []byte
	tx                 *Transaction
	gas, gasPrice      *big.Int
	value              *big.Int
	data               []byte
	state              *monkstate.State
	block              *Block
	genesis            *Block

	msg *monkstate.Message

	cb, rec, sen *monkstate.StateObject
}

func NewStateTransition(coinbase *monkstate.StateObject, tx *Transaction, state *monkstate.State, block *Block) *StateTransition {
	return &StateTransition{coinbase.Address(), tx.Recipient, tx, new(big.Int), new(big.Int).Set(tx.GasPrice), tx.Value, tx.Data, state, block, nil, nil, coinbase, nil, nil}
}

func NewStateTransitionEris(coinbase *monkstate.StateObject, tx *Transaction, state *monkstate.State, block *Block, gen *Block) *StateTransition {
	return &StateTransition{coinbase.Address(), tx.Recipient, tx, new(big.Int), new(big.Int).Set(tx.GasPrice), tx.Value, tx.Data, state, block, gen, nil, coinbase, nil, nil}
}

func (self *StateTransition) Coinbase() *monkstate.StateObject {
	if self.cb != nil {
		return self.cb
	}

	self.cb = self.state.GetOrNewStateObject(self.coinbase)
	return self.cb
}
func (self *StateTransition) Sender() *monkstate.StateObject {
	if self.sen != nil {
		return self.sen
	}

	self.sen = self.state.GetOrNewStateObject(self.tx.Sender())

	return self.sen
}
func (self *StateTransition) Receiver() *monkstate.StateObject {
	if self.tx != nil && self.tx.CreatesContract() {
		return nil
	}

	if self.rec != nil {
		return self.rec
	}

	self.rec = self.state.GetOrNewStateObject(self.tx.Recipient)
	return self.rec
}

func (self *StateTransition) MakeStateObject(state *monkstate.State, tx *Transaction) *monkstate.StateObject {
	contract := MakeContract(tx, state)

	return contract
}

func (self *StateTransition) UseGas(amount *big.Int) error {
	if self.gas.Cmp(amount) < 0 {
		return OutOfGasError()
	}
	self.gas.Sub(self.gas, amount)

	return nil
}

func (self *StateTransition) AddGas(amount *big.Int) {
	self.gas.Add(self.gas, amount)
}

// Move this to monkdoug?
func (self *StateTransition) BuyGas() error {
	var err error

	sender := self.Sender()
	if sender.Balance.Cmp(self.tx.GasValue()) < 0 {
		return fmt.Errorf("Insufficient funds to pre-pay gas. Req %v, has %v", self.tx.GasValue(), sender.Balance)
	}

	coinbase := self.Coinbase()
	err = coinbase.BuyGas(self.tx.Gas, self.tx.GasPrice)
	if err != nil {
		return err
	}

	self.AddGas(self.tx.Gas)
	sender.SubAmount(self.tx.GasValue())

	return nil
}

func (self *StateTransition) RefundGas() {
	coinbase, sender := self.Coinbase(), self.Sender()
	coinbase.RefundGas(self.gas, self.tx.GasPrice)

	// Return remaining gas
	remaining := new(big.Int).Mul(self.gas, self.tx.GasPrice)
	sender.AddAmount(remaining)
}

func (self *StateTransition) preCheck() (err error) {
	// preCheck() should be a proxy for calling a doug permissions model
	// the permissions model will check all the things
	if err := genDoug.ValidateTx(self.tx, self.state); err != nil {
		return err
	}
	// Pre-pay gas / Buy gas off the coinbase account
	// TODO: reconfigure this to run from gendoug
	//  probably by moving the BuyGas function over
	if err = self.BuyGas(); err != nil {
		return err
	}
	return nil
}

// TODO: return the message!!!
func (self *StateTransition) TransitionState() (err error) {
	statelogger.Debugf("(~) %x\n", self.tx.Hash())

	defer func() {
		if r := recover(); r != nil {
			trace := make([]byte, 1024)
			runtime.Stack(trace, true)
			statelogger.Infoln(r)
			err = fmt.Errorf("state transition err %v", r)
			fmt.Println(string(trace))
		}
	}()

	// XXX Transactions after this point are considered valid.
	if err = self.preCheck(); err != nil {
		return
	}

	var (
		tx       = self.tx
		sender   = self.Sender()
		receiver *monkstate.StateObject
	)

	defer self.RefundGas()

	// Increment the nonce for the next transaction
	sender.Nonce += 1

	// Transaction gas
	if err = self.UseGas(monkvm.GasTx); err != nil {
		return
	}

	// Pay data gas
	dataPrice := big.NewInt(int64(len(self.data)))
	dataPrice.Mul(dataPrice, monkvm.GasData)
	if err = self.UseGas(dataPrice); err != nil {
		return
	}

	if sender.Balance.Cmp(self.value) < 0 {
		return fmt.Errorf("Insufficient funds to transfer value. Req %v, has %v", self.value, sender.Balance)
	}

	var snapshot *monkstate.State
	// If the receiver is nil it's a contract (\0*32).
	if tx.CreatesContract() {
		// Subtract the (irreversible) amount from the senders account
		sender.SubAmount(self.value)

		snapshot = self.state.Copy()

		// Create a new state object for the contract
		receiver = self.MakeStateObject(self.state, tx)
		self.rec = receiver
		if receiver == nil {
			return fmt.Errorf("Unable to create contract")
		}

		// Add the amount to receivers account which should conclude this transaction
		receiver.AddAmount(self.value)
	} else {
		receiver = self.Receiver()

		// Subtract the amount from the senders account
		sender.SubAmount(self.value)
		// Add the amount to receivers account which should conclude this transaction
		receiver.AddAmount(self.value)

		snapshot = self.state.Copy()
	}

	msg := self.state.Manifest().AddMessage(&monkstate.Message{
		To: receiver.Address(), From: sender.Address(),
		Output: nil,
		Input:  self.tx.Data,
		Origin: sender.Address(),
		Block:  self.block.Hash(), Timestamp: self.block.Time, Coinbase: self.block.Coinbase, Number: self.block.Number,
		Value: self.value,
	})

	// Process the init code and create 'valid' contract
	if IsContractAddr(self.receiver) {
		// Evaluate the initialization script
		// and use the return value as the
		// script section for the state object.
		self.data = nil

		code, err := self.Eval(msg, receiver.Init(), receiver, "init")
		if err != nil {
			self.state.Set(snapshot)

			return fmt.Errorf("Error during init execution %v", err)
		}

		receiver.Code = code
		msg.Output = code
	} else {
		if len(receiver.Code) > 0 {
			ret, err := self.Eval(msg, receiver.Code, receiver, "code")
			if err != nil {
				self.state.Set(snapshot)

				return fmt.Errorf("Error during code execution %v", err)
			}

			msg.Output = ret
		}
	}

	// so we can retrieve return values
	self.msg = msg

	return
}

func (self *StateTransition) transferValue(sender, receiver *monkstate.StateObject) error {
	if sender.Balance.Cmp(self.value) < 0 {
		return fmt.Errorf("Insufficient funds to transfer value. Req %v, has %v", self.value, sender.Balance)
	}

	// Subtract the amount from the senders account
	sender.SubAmount(self.value)
	// Add the amount to receivers account which should conclude this transaction
	receiver.AddAmount(self.value)

	return nil
}

func (self *StateTransition) Eval(msg *monkstate.Message, script []byte, context *monkstate.StateObject, typ string) (ret []byte, err error) {
	var (
		transactor    = self.Sender()
		state         = self.state
		env           = NewEnv(state, self.tx, self.block)
		callerClosure = monkvm.NewClosure(msg, transactor, context, script, self.gas, self.gasPrice)
	)

	vm := monkvm.New(env)
	vm.Verbose = true
	vm.Fn = typ

	ret, _, err = callerClosure.Call(vm, self.tx.Data)

	if err == nil {
		// Execute POSTs
		for e := vm.Queue().Front(); e != nil; e = e.Next() {
			msg := e.Value.(*monkvm.Message)

			msg.Exec(msg.Addr(), transactor)
		}
	}

	return
}

// Converts an transaction in to a state object
func MakeContract(tx *Transaction, state *monkstate.State) *monkstate.StateObject {
	// Create contract if there's no recipient
	if tx.IsContract() {
		addr := tx.CreationAddress()

		contract := state.NewStateObject(addr)
		contract.InitCode = tx.Data
		contract.State = monkstate.New(monktrie.New(monkutil.Config.Db, ""))

		return contract
	}

	return nil
}
