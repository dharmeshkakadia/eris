package monkdoug

import (
	"bytes"
	"fmt"
	"math/big"
	"time"
	//"log"
	vars "github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/eris-std-lib/go-tests"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
)

var Adversary = 0

type Protocol struct {
	g         *GenesisConfig
	consensus monkchain.Consensus
}

func (p *Protocol) Doug() []byte {
	return p.g.byteAddr
}

func (p *Protocol) Deploy(block *monkchain.Block) ([]byte, error) {
	// TODO: try deployer, fall back to default deployer
	return p.g.Deployer(block)
}

func (p *Protocol) ValidateChainID(chainId []byte, genesisBlock *monkchain.Block) error {
	return nil
}

// Determine whether to accept a new checkpoint
func (p *Protocol) Participate(coinbase []byte, parent *monkchain.Block) bool {
	return p.consensus.Participate(coinbase, parent)
}

func (p *Protocol) Difficulty(block, parent *monkchain.Block) *big.Int {
	return p.consensus.Difficulty(block, parent)
}

func (p *Protocol) ValidatePerm(addr []byte, role string, state *monkstate.State) error {
	return p.consensus.ValidatePerm(addr, role, state)
}

func (p *Protocol) ValidateBlock(block *monkchain.Block, bc *monkchain.ChainManager) error {
	return p.consensus.ValidateBlock(block, bc)
}

func (p *Protocol) ValidateTx(tx *monkchain.Transaction, state *monkstate.State) error {
	return p.consensus.ValidateTx(tx, state)
}

func (p *Protocol) CheckPoint(proposed []byte, bc *monkchain.ChainManager) bool {
	return p.consensus.CheckPoint(proposed, bc)
}

// The yes model grants all permissions
type YesModel struct {
	g *GenesisConfig
}

func NewYesModel(g *GenesisConfig) monkchain.Consensus {
	return &YesModel{g}
}

func (m *YesModel) Participate(coinbase []byte, parent *monkchain.Block) bool {
	return true
}

func (m *YesModel) Difficulty(block, parent *monkchain.Block) *big.Int {
	return monkutil.BigPow(2, m.g.Difficulty)
}

func (m *YesModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error {
	return nil
}

func (m *YesModel) ValidateBlock(block *monkchain.Block, bc *monkchain.ChainManager) error {
	return nil
}

func (m *YesModel) ValidateTx(tx *monkchain.Transaction, state *monkstate.State) error {
	return nil
}

func (m *YesModel) CheckPoint(proposed []byte, bc *monkchain.ChainManager) bool {
	return true
}

// The no model grants no permissions
type NoModel struct {
	g *GenesisConfig
}

func NewNoModel(g *GenesisConfig) monkchain.Consensus {
	return &NoModel{g}
}

func (m *NoModel) Participate(coinbase []byte, parent *monkchain.Block) bool {
	// we tell it to start mining even though we know it will fail
	// because this model is mostly just used for testing...
	return true
}

func (m *NoModel) Difficulty(block, parent *monkchain.Block) *big.Int {
	return monkutil.BigPow(2, m.g.Difficulty)
}

func (m *NoModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error {
	return fmt.Errorf("No!")
}

func (m *NoModel) ValidateBlock(block *monkchain.Block, bc *monkchain.ChainManager) error {
	return fmt.Errorf("No!")
}

func (m *NoModel) ValidateTx(tx *monkchain.Transaction, state *monkstate.State) error {
	return fmt.Errorf("No!")
}

func (m *NoModel) CheckPoint(proposed []byte, bc *monkchain.ChainManager) bool {
	return false
}

// The VM Model runs all processing through the EVM
type VmModel struct {
	g    *GenesisConfig
	doug []byte

	// map of contract names to syscalls
	// names are json tags, addresses are
	// left-padded struct field names (VmConsensus struct)
	contract map[string]SysCall
}

func NewVmModel(g *GenesisConfig) monkchain.Consensus {
	contract := make(map[string]SysCall)
	return &VmModel{g, g.byteAddr, contract}
}

// TODO:
//  - enforce read-only option for vm (no SSTORE)

func (m *VmModel) Participate(coinbase []byte, parent *monkchain.Block) bool {
	state := parent.State()
	if scall, ok := m.getSysCall("compute-participate", state); ok {
		addr := scall.byteAddr
		obj, code := m.pickCallObjAndCode(addr, state)
		coinbaseHex := monkutil.Bytes2Hex(coinbase)
		data := monkutil.PackTxDataArgs2(coinbaseHex)
		ret := m.EvmCall(code, data, obj, state, nil, parent, true)
		// TODO: check not nil
		return monkutil.BigD(ret).Uint64() > 0
	}

	// get perm from doug
	doug := state.GetStateObject(m.doug)
	data := monkutil.PackTxDataArgs2("checkperm", "mine", "0x"+monkutil.Bytes2Hex(coinbase))
	douglogger.Infoln("Calling permision verify (GENDOUG) contract to check if we should participate")
	ret := m.EvmCall(doug.Code, data, doug, state, nil, nil, true)

	if monkutil.BigD(ret).Uint64() > 0 {
		return true
	}
	return false
}

func (m *VmModel) pickCallObjAndCode(addr []byte, state *monkstate.State) (obj *monkstate.StateObject, code []byte) {
	obj = state.GetStateObject(addr)
	code = obj.Code
	//if useDoug {
	//	obj = state.GetStateObject(m.doug)
	//}
	return
}

func (m *VmModel) getSysCall(name string, state *monkstate.State) (SysCall, bool) {
	if s, ok := m.contract[name]; ok {
		return s, ok
	}

	addr := GetValue(m.doug, name, state)
	if addr != nil {
		return SysCall{byteAddr: addr}, true
	}
	return SysCall{}, false
}

func (m *VmModel) Difficulty(block, parent *monkchain.Block) *big.Int {
	state := parent.State()
	if scall, ok := m.getSysCall("compute-difficulty", state); ok {
		addr := scall.byteAddr
		obj, code := m.pickCallObjAndCode(addr, state)
		data := packBlockParent(block, parent)
		douglogger.Infoln("Calling difficulty contract")
		ret := m.EvmCall(code, data, obj, state, nil, block, true)
		//fmt.Println("RETURN DIF:", ret)
		// TODO: check not nil
		return monkutil.BigD(ret)
	}
	r := EthDifficulty(5*60, block, parent)
	//fmt.Println("RETURN DIF:", r)
	return r
	//return monkutil.BigPow(2, m.g.Difficulty)
}

func packBlockParent(block, parent *monkchain.Block) []byte {
	block1rlp := monkutil.Encode(block.Header())
	l1 := len(block1rlp)
	l1bytes := big.NewInt(int64(l1)).Bytes()
	block2rlp := monkutil.Encode(parent.Header())
	l2 := len(block2rlp)
	l2bytes := big.NewInt(int64(l2)).Bytes()

	// data is
	// (len block 1), (block 1), (len block 2), (block 2), (len sig for block 1), (sig block 1)
	data := []byte{}
	data = append(data, monkutil.LeftPadBytes(l1bytes, 32)...)
	data = append(data, block1rlp...)
	data = append(data, monkutil.LeftPadBytes(l2bytes, 32)...)
	data = append(data, block2rlp...)
	return data
}

func (m *VmModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error {
	var ret []byte
	if scall, ok := m.getSysCall("permission-verify", state); ok {
		contract := scall.byteAddr
		obj, code := m.pickCallObjAndCode(contract, state)
		data := monkutil.PackTxDataArgs2(monkutil.Bytes2Hex(addr), role)
		douglogger.Infoln("Calling permision verify contract")
		ret = m.EvmCall(code, data, obj, state, nil, nil, true)
	} else {
		// get perm from doug
		doug := state.GetStateObject(m.doug)
		data := monkutil.PackTxDataArgs2("checkperm", role, "0x"+monkutil.Bytes2Hex(addr))
		douglogger.Infoln("Calling permision verify (GENDOUG) contract")
		ret = m.EvmCall(doug.Code, data, doug, state, nil, nil, true)
	}
	if monkutil.BigD(ret).Uint64() > 0 {
		return nil
	}
	return fmt.Errorf("Permission error")
}

func (m *VmModel) ValidateBlock(block *monkchain.Block, bc *monkchain.ChainManager) error {
	parent := bc.CurrentBlock()
	state := parent.State()

	if scall, ok := m.getSysCall("block-verify", state); ok {
		addr := scall.byteAddr
		obj, code := m.pickCallObjAndCode(addr, state)
		sig := block.GetSig()

		sigrlp := monkutil.Encode([]interface{}{sig[:32], sig[32:64], monkutil.RightPadBytes([]byte{sig[64] - 27}, 32)})
		lsig := len(sigrlp)
		lsigbytes := big.NewInt(int64(lsig)).Bytes()

		// data is
		// (len block 1), (block 1), (len block 2), (block 2), (len sig for block 1), (sig block 1)
		data := packBlockParent(block, parent)
		data = append(data, monkutil.LeftPadBytes(lsigbytes, 32)...)
		data = append(data, sigrlp...)

		douglogger.Infoln("Calling block verify contract")
		ret := m.EvmCall(code, data, obj, state, nil, block, true)
		if monkutil.BigD(ret).Uint64() > 0 {
			return nil
		}
		return fmt.Errorf("Permission error")
	}
	return m.ValidatePerm(block.Coinbase, "mine", state)
}

func (m *VmModel) ValidateTx(tx *monkchain.Transaction, state *monkstate.State) error {
	if scall, ok := m.getSysCall("tx-verify", state); ok {
		addr := scall.byteAddr
		obj, code := m.pickCallObjAndCode(addr, state)
		data := tx.RlpEncode()
		l := big.NewInt(int64(len(data))).Bytes()
		data = append(monkutil.LeftPadBytes(l, 32), data...)

		douglogger.Infoln("Calling tx verify contract")
		ret := m.EvmCall(code, data, obj, state, tx, nil, true)
		if monkutil.BigD(ret).Uint64() > 0 {
			return nil
		}
		return fmt.Errorf("Permission error")
	}
	var perm string
	if tx.IsContract() {
		perm = "create"
	} else {
		perm = "transact"
	}
	return m.ValidatePerm(tx.Sender(), perm, state)
}

func (m *VmModel) CheckPoint(proposed []byte, bc *monkchain.ChainManager) bool {
	// TODO: checkpoint validation contract
	return true
}

// The stdlib model grants permissions based on the state of the gendoug
// It depends on the eris-std-lib for its storage model
type StdLibModel struct {
	base *big.Int
	doug []byte
	g    *GenesisConfig
	pow  monkchain.PoW
}

func NewStdLibModel(g *GenesisConfig) monkchain.Consensus {
	return &StdLibModel{
		base: new(big.Int),
		doug: g.byteAddr,
		g:    g,
		pow:  &monkchain.EasyPow{},
	}
}

func (m *StdLibModel) GetPermission(addr []byte, perm string, state *monkstate.State) *monkutil.Value {
	public := vars.GetSingle(m.doug, "public:"+perm, state)
	// A stand-in for a one day more sophisticated system
	if len(public) > 0 {
		return monkutil.NewValue(1)
	}
	locator := vars.GetLinkedListElement(m.doug, "permnames", perm, state)
	locatorBig := monkutil.BigD(locator)
	locInt := locatorBig.Uint64()
	permStr := vars.GetKeyedArrayElement(m.doug, "perms", monkutil.Bytes2Hex(addr), int(locInt), state)
	return monkutil.NewValue(permStr)
}

func (m *StdLibModel) HasPermission(addr []byte, perm string, state *monkstate.State) bool {
	permBig := m.GetPermission(addr, perm, state).BigInt()
	return permBig.Int64() > 0
}

// Save energy in the round robin by not mining until close to your turn
// or too much time has gone by
func (m *StdLibModel) Participate(coinbase []byte, parent *monkchain.Block) bool {
	if Adversary != 0 {
		return true
	}

	consensus := m.consensus(parent.State())
	// if we're not in a round robin, always mine
	if consensus != "robin" {
		return true
	}
	// find out our distance from the current next miner
	next := m.nextCoinbase(parent)
	nMiners := vars.GetLinkedListLength(m.doug, "seq:name", parent.State())
	var i int
	for i = 0; i < nMiners; i++ {
		next, _ = vars.GetNextLinkedListElement(m.doug, "seq:name", string(next), parent.State())
		if bytes.Equal(next, coinbase) {
			break
		}
	}
	// if we're less than halfway from the current miner, we should mine
	if i <= int(nMiners/2) {
		return true
	}
	// if we're more than halfway, but enough time has gone by, we should mine
	mDiff := i - int(nMiners/2)
	t := parent.Time
	cur := time.Now().Unix()
	blocktime := m.blocktime(parent.State())
	tDiff := (cur - t) / blocktime
	if tDiff > int64(mDiff) {
		return true
	}
	// otherwise, we should not mine
	return false
}

// Difficulty of the current block for a given coinbase
func (m *StdLibModel) Difficulty(block, parent *monkchain.Block) *big.Int {
	var b *big.Int

	consensus := m.consensus(parent.State())

	// compute difficulty according to consensus model
	switch consensus {
	case "robin":
		b = m.RoundRobinDifficulty(block, parent)
	case "stake-weight":
		b = m.StakeDifficulty(block, parent)
	case "constant":
		b = m.baseDifficulty(parent.State())
	default:
		blockTime := m.blocktime(parent.State())
		b = EthDifficulty(blockTime, block, parent)
	}
	return b
}

func (m *StdLibModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error {
	if Adversary != 0 {
		return nil
	}
	if m.HasPermission(addr, role, state) {
		return nil
	}
	return monkchain.InvalidPermError(addr, role)
}

func (m *StdLibModel) ValidateBlock(block *monkchain.Block, bc *monkchain.ChainManager) error {
	if Adversary != 0 {
		return nil
	}

	// we have to verify using the state of the previous block!
	prevBlock := bc.GetBlock(block.PrevHash)

	// check that miner has permission to mine
	if !m.HasPermission(block.Coinbase, "mine", prevBlock.State()) {
		return monkchain.InvalidPermError(block.Coinbase, "mine")
	}

	// check that signature of block matches miners coinbase
	if !bytes.Equal(block.Signer(), block.Coinbase) {
		return monkchain.InvalidSigError(block.Signer(), block.Coinbase)
	}

	// check if the block difficulty is correct
	// it must be specified exactly
	newdiff := m.Difficulty(block, prevBlock)
	if block.Difficulty.Cmp(newdiff) != 0 {
		return monkchain.InvalidDifficultyError(block.Difficulty, newdiff, block.Coinbase)
	}

	// TODO: is there a time when some consensus element is
	// not specified in difficulty and must appear here?
	// Do we even budget for lists of signers/forgers and all
	// that nutty PoS stuff?

	// check block times
	if err := CheckBlockTimes(prevBlock, block); err != nil {
		return err
	}

	// Verify the nonce of the block. Return an error if it's not valid
	// TODO: for now we leave pow on everything
	// soon we will want to generalize/relieve
	// also, variable hashing algos
	if !m.pow.Verify(block.HashNoNonce(), block.Difficulty, block.Nonce) {
		return monkchain.ValidationError("Block's nonce is invalid (= %v)", monkutil.Bytes2Hex(block.Nonce))
	}

	return nil
}

func (m *StdLibModel) ValidateTx(tx *monkchain.Transaction, state *monkstate.State) error {
	if Adversary != 0 {
		return nil
	}

	// check that sender has permission to transact or create
	var perm string
	if tx.IsContract() {
		perm = "create"
	} else {
		perm = "transact"
	}
	if !m.HasPermission(tx.Sender(), perm, state) {
		return monkchain.InvalidPermError(tx.Sender(), perm)
	}
	// check that tx uses less than maxgas
	gas := tx.GasValue()
	max := vars.GetSingle(m.doug, "maxgastx", state)
	maxBig := monkutil.BigD(max)
	if max != nil && gas.Cmp(maxBig) > 0 {
		return monkchain.GasLimitTxError(gas, maxBig)
	}
	// Make sure this transaction's nonce is correct
	sender := state.GetOrNewStateObject(tx.Sender())
	if sender.Nonce != tx.Nonce {
		return monkchain.NonceError(tx.Nonce, sender.Nonce)
	}
	return nil
}

func (m *StdLibModel) CheckPoint(proposed []byte, bc *monkchain.ChainManager) bool {
	// TODO: something reasonable
	return true
}

type EthModel struct {
	pow monkchain.PoW
	g   *GenesisConfig
}

func NewEthModel(g *GenesisConfig) monkchain.Consensus {
	return &EthModel{&monkchain.EasyPow{}, g}
}

func (m *EthModel) Participate(coinbase []byte, parent *monkchain.Block) bool {
	return true
}

func (m *EthModel) Difficulty(block, parent *monkchain.Block) *big.Int {
	return EthDifficulty(int64(m.g.BlockTime), block, parent)
}

func (m *EthModel) ValidatePerm(addr []byte, role string, state *monkstate.State) error {
	return nil
}

func (m *EthModel) ValidateBlock(block *monkchain.Block, bc *monkchain.ChainManager) error {
	// we have to verify using the state of the previous block!
	prevBlock := bc.GetBlock(block.PrevHash)

	// check that signature of block matches miners coinbase
	// XXX: not strictly necessary for eth...
	if !bytes.Equal(block.Signer(), block.Coinbase) {
		return monkchain.InvalidSigError(block.Signer(), block.Coinbase)
	}

	// check if the difficulty is correct
	newdiff := m.Difficulty(block, prevBlock)
	if block.Difficulty.Cmp(newdiff) != 0 {
		return monkchain.InvalidDifficultyError(block.Difficulty, newdiff, block.Coinbase)
	}

	// check block times
	if err := CheckBlockTimes(prevBlock, block); err != nil {
		return err
	}

	// Verify the nonce of the block. Return an error if it's not valid
	if !m.pow.Verify(block.HashNoNonce(), block.Difficulty, block.Nonce) {
		return monkchain.ValidationError("Block's nonce is invalid (= %v)", monkutil.Bytes2Hex(block.Nonce))
	}

	return nil
}

func (m *EthModel) ValidateTx(tx *monkchain.Transaction, state *monkstate.State) error {
	// Make sure this transaction's nonce is correct
	sender := state.GetOrNewStateObject(tx.Sender())
	if sender.Nonce != tx.Nonce {
		return monkchain.NonceError(tx.Nonce, sender.Nonce)
	}
	return nil
}

func (m *EthModel) CheckPoint(proposed []byte, bc *monkchain.ChainManager) bool {
	// TODO: can we authenticate eth checkpoints?
	//   or just do something reasonable
	return false
}
