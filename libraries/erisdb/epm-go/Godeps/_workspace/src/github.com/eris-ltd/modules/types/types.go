package types

import (
	"time"
)

type (

	// A default object that implements 'Event'
	Event struct {
		Event     string
		Target    string
		Resource  interface{}
		Source    string
		TimeStamp time.Time
	}

	Addresses struct {
		ActiveAddress string
		AddressList   []string
	}

	Account struct {
		Address string
		Balance string
		Nonce   string
		Script  string
		Storage *Storage

		IsScript bool
	}

	// Ordered map for storage in an account or generalized table
	Storage struct {
		// hex strings for eth, arrays of strings (cols) for sql dbs
		Storage map[string]string
		Order   []string
	}

	// Ordered map for all accounts
	State struct {
		State map[string]*Storage // map addrs to map of storage to value
		Order []string            // ordered addrs and ordered storage inside
	}

	WorldState struct {
		Accounts map[string]*Account
		Order    []string
	}

	BlockMini struct {
		Number           string
		Hash             string
		Transactions     int
		PrevHash         string
		AccountsAffected []*AccountMini
	}

	StateAtArgs struct {
		Address string
		Storage string
	}

	TransactionArgs struct {
		BlockHash string
		TxHash    string
	}

	Block struct {
		Number       string
		Time         int
		Nonce        string
		Hash         string
		PrevHash     string
		Difficulty   string
		Coinbase     string
		Transactions []*Transaction
		Uncles       []string
		GasLimit     string
		GasUsed      string
		MinGasPrice  string
		TxRoot       string
		UncleRoot    string
	}

	Transaction struct {
		ContractCreation bool
		Nonce            string
		Hash             string
		Sender           string
		Recipient        string
		Value            string
		Gas              string
		GasCost          string
		BlockHash        string
		Inputs           []*Input
		Outputs          []*Output
		Error            string
	}

	Input struct {
		PrevOut struct {
			Address string
			Number  int64
			Type    int64
			Value   int64
		}
		Script string
	}

	Output struct {
		Address string
		Number  int64
		Type    int64
		Value   int64
	}

	TxIndata struct {
		Recipient string
		Gas       string
		GasCost   string
		Value     string
		// endline is the separator for tx data. Each string is padded with 0's to become 32 bytes.
		Data string
	}

	TxReceipt struct {
		Success  bool   // If transaction hash was created basically.
		Compiled bool   // If a contract was created, and the txdata was successfully compiled.
		Address  string // If a contract was created.
		Hash     string // Transaction hash
		Error    string
	}

	AccountMini struct {
		// Modified (0), Added (1), Deleted(2)
		Flag     int
		Contract bool
		Address  string
		Nonce    string
		Balance  string
	}
)

func ToMap(obj interface{}) map[string]interface{} {
	mp := make(map[string]interface{})
	switch o := obj.(type) {
	case *Addresses:
		mp["ActiveAddress"] = o.ActiveAddress
		mp["AddressList"] = o.AddressList
		break
	case *Account:
		mp["Address"] = o.Address
		mp["Balance"] = o.Balance
		mp["IsScript"] = o.IsScript
		mp["Nonce"] = o.Nonce
		mp["Script"] = o.Script
		mp["Storage"] = ToMap(o.Storage)
		break
	case *Storage:
		mp["Order"] = o.Order
		mp["Storage"] = o.Storage
		break
	case *State:
		mp["Order"] = o.Order
		stmp := make(map[string]map[string]interface{})
		for k, v := range o.State {
			stmp[k] = ToMap(v)
		}
		mp["Storage"] = stmp
		break
	case *WorldState:
		mp["Order"] = o.Order
		stmp := make(map[string]map[string]interface{})
		for k, v := range o.Accounts {
			stmp[k] = ToMap(v)
		}
		mp["Accounts"] = stmp
		break
	case *AccountMini:
		mp["Address"] = o.Address
		mp["Balance"] = o.Balance
		mp["Contract"] = o.Contract
		mp["Flag"] = o.Flag
		mp["Nonce"] = o.Nonce
		break
	case *BlockMini:
		mp["Hash"] = o.Hash
		mp["Number"] = o.Number
		mp["PrevHash"] = o.PrevHash
		mp["Transactions"] = o.Transactions
		aamp := make([]map[string]interface{}, len(o.AccountsAffected))
		for k, v := range o.AccountsAffected {
			aamp[k] = ToMap(v)
		}
		break
	case *Block:
		mp["Coinbase"] = o.Coinbase
		mp["Difficulty"] = o.Difficulty
		mp["GasLimit"] = o.GasLimit
		mp["GasUsed"] = o.GasUsed
		mp["Hash"] = o.Hash
		mp["MinGasPrice"] = o.MinGasPrice
		mp["Nonce"] = o.Nonce
		mp["Number"] = o.Number
		mp["PrevHash"] = o.PrevHash
		mp["Time"] = o.Time
		mp["TxRoot"] = o.TxRoot
		mp["UncleRoot"] = o.UncleRoot
		mp["Uncles"] = o.Uncles

		mp["Transactions"] = o.Transactions
		txmp := make([]map[string]interface{}, len(o.Transactions))
		for k, v := range o.Transactions {
			txmp[k] = ToMap(v)
		}
		break
	case *Transaction:
		mp["BlockHash"] = o.BlockHash
		mp["ContractCreation"] = o.ContractCreation
		mp["Error"] = o.Error
		mp["Gas"] = o.Gas
		mp["GasCost"] = o.GasCost
		mp["Hash"] = o.Hash
		mp["Nonce"] = o.Nonce
		mp["Recipient"] = o.Recipient
		mp["Sender"] = o.Sender
		mp["Value"] = o.Value
		break
	}

	return mp
}
