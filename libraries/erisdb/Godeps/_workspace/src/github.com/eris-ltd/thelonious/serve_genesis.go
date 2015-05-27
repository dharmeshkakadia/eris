package thelonious

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/braintree/manners"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkchain"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkstate"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monktrie"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monkutil"
	"io/ioutil"
	"net/http"
	"strconv"
)

/*
	Thel process runs a little http server to serve genesis block and state
*/

func (s *Thelonious) ServeGenesis(fetchPort int) {
	if fetchPort < 0 {
		return
	}
	fetchPortString := strconv.Itoa(fetchPort)
	mux := http.NewServeMux()
	mux.HandleFunc("/chainid", s.serveChainId)     // serve the chain id
	mux.HandleFunc("/genesis", s.serveGenesisJson) // serve genesis.json
	mux.HandleFunc("/port", s.servePort)           // serve p2p listening port (for bootstrapping)
	mux.HandleFunc("/block", s.serveGenesisBlock)  // serve genesis block
	mux.HandleFunc("/state/", s.serveGenesisState) // serve serialized trie given root hash
	mux.HandleFunc("/hash/", s.serveHash)          // serve value given key hash
	server := manners.NewServer()
	go server.ListenAndServe(":"+fetchPortString, mux)
	<-s.quit
	server.Shutdown <- true
	return
}

// serve the peers p2p listening port
func (s *Thelonious) servePort(w http.ResponseWriter, r *http.Request) {
	port := s.Port
	w.Write([]byte(port))
}

// serve the genesis.json
func (s *Thelonious) serveGenesisJson(w http.ResponseWriter, r *http.Request) {
	g := s.genConfig
	b, err := json.MarshalIndent(*g, "", "\t")
	if err != nil {
		fmt.Println("json marshal error:", err)
	}
	w.Write(b)
}

// serve the current chain's id
func (s *Thelonious) serveChainId(w http.ResponseWriter, r *http.Request) {
	chainId := s.blockChain.ChainID()
	w.Write(chainId)
}

// serve the value in the leveldb store corresponding to provided hash
func (s *Thelonious) serveHash(w http.ResponseWriter, r *http.Request) {
	hashHex := r.URL.Path[len("/hash/"):]
	hash := monkutil.Hex2Bytes(hashHex)
	b, _ := monkutil.Config.Db.Get(hash)
	w.Write(b)
}

// serve the current chain's rlp encoded genesis block
func (s *Thelonious) serveGenesisBlock(w http.ResponseWriter, r *http.Request) {
	b := s.blockChain.Genesis()
	encoded := b.RlpEncode()
	b = monkchain.NewBlockFromBytes(encoded)
	w.Write(encoded)
}

// serve the serialized state trie corresponding to the provided root hash
func (s *Thelonious) serveGenesisState(w http.ResponseWriter, r *http.Request) {
	hashHex := r.URL.Path[len("/state/"):]
	hash := monkutil.Hex2Bytes(hashHex)
	monklogger.Debugf("Fetching state for hash %x\n", hash)
	tr := monktrie.New(s.db, hash)
	response := serializeTrie(tr)
	b := monkutil.NewValue(response)
	w.Write(b.Encode())
}

/*
	Client functions (called by epm fetch)
*/

func GetFetchPeerPort(addr string) (int, error) {
	resp, err := http.Get(addr + "/port")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	port := string(body)
	return strconv.Atoi(port)
}

func GetGenesisJson(addr string) ([]byte, error) {
	resp, err := http.Get(addr + "/genesis")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return body, err
}

func GetChainId(addr string) ([]byte, error) {
	resp, err := http.Get(addr + "/chainid")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return body, err
}

func GetGenesisBlock(addr string) (*monkchain.Block, error) {
	resp, err := http.Get(addr + "/block")
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	block := monkchain.NewBlockFromBytes(body)
	monklogger.Debugln("Genesis Block:", block)
	return block, nil
}

// takes a fetchserver addr, trie state root
func GetGenesisState(addr, hash string, db monkutil.Database) error {
	// grab a list of key/val pairs for accounts in the state
	resp, err := http.Get(addr + "/state/" + hash)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	data := monkutil.NewValueFromBytes(body)

	tr := monktrie.New(db, "")
	state := monkstate.New(tr)
	if err := updateState(db, addr, state, data); err != nil {
		return err
	}

	correct, _ := monktrie.ParanoiaCheck(tr)
	if !correct {
		return fmt.Errorf("Failed paranoia check")
	}
	return nil
}

/*
	Utility functions for getting and setting state
*/

// serialize trie into list of pairs of (k, v)
func serializeTrie(tr *monktrie.Trie) []interface{} {
	trIt := tr.NewIterator()
	response := []interface{}{}
	trIt.Each(func(key string, val *monkutil.Value) {
		pair := []interface{}{[]byte(key), val.Bytes()}
		response = append(response, pair)
	})
	return response
}

// update state with serialized trie (updates all state objects too)
func updateState(db monkutil.Database, addr string, state *monkstate.State, data *monkutil.Value) error {
	for i := 0; i < data.Len(); i++ {
		l := data.Get(i)
		k := l.Get(0).Bytes()
		v := l.Get(1).Bytes()
		monklogger.Debugf("Key %x Value %x\n", k, v)
		// this value is hash of a stateobject
		if err := getStateObject(db, addr, state, k, v); err != nil {
			return err
		}
	}
	state.Update()
	state.Sync()
	return nil
}

// get code corresponding to a code hash and put in db
func getCode(db monkutil.Database, addr, hash string) ([]byte, error) {
	resp, err := http.Get(addr + "/hash/" + hash)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	monklogger.Debugf("CodeHash %s Bytes %x\n", hash, body)
	db.Put(monkutil.Hex2Bytes(hash), body)
	return body, nil
}

// get state object's code and trie
func getStateObject(db monkutil.Database, addr string, state *monkstate.State, k []byte, v []byte) error {
	// state obejct may have a trie hash, but our db
	// has none of the entries yet
	stateObj := monkstate.NewStateObjectFromBytes(k, v)

	codeHash := stateObj.GetCodeHash()
	if len(codeHash) > 0 {
		code, err := getCode(db, addr, monkutil.Bytes2Hex(codeHash))
		if err != nil {
			return err
		}
		stateObj.Code = code

		if bytes.Compare(stateObj.CodeHash(), stateObj.GetCodeHash()) != 0 {
			return fmt.Errorf("Code hash err %x vs. %x", stateObj.CodeHash(), stateObj.GetCodeHash())
		}
	}

	hash := stateObj.State.Trie.Root
	hashB, ok := hash.([]byte)
	if !ok {
		return fmt.Errorf("State hash not bytes: %x", hash)
	}

	resp, err := http.Get(addr + "/state/" + monkutil.Bytes2Hex(hashB))
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	data := monkutil.NewValueFromBytes(body)

	// apply to new trie
	tr := monktrie.New(db, "")
	updateTrie(tr, data)
	correct, _ := monktrie.ParanoiaCheck(tr)
	if !correct {
		return fmt.Errorf("Failed paranoia check for subtrie")
	}

	stateObj.State.Trie = tr
	state.UpdateStateObject(stateObj)

	return nil
}

func updateTrie(tr *monktrie.Trie, data *monkutil.Value) {
	for i := 0; i < data.Len(); i++ {
		l := data.Get(i)
		k := l.Get(0).Bytes()
		v := l.Get(1).Bytes()

		tr.Update(string(k), string(v))
	}
	tr.Sync()
}
