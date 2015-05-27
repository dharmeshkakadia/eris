package server

import (
	"bytes"
	// "encoding/json"
	"fmt"
	// "github.com/eris-ltd/epm-go/Godeps/_workspace/src/golang.org/x/net/websocket"
	"io/ioutil"
	"net/http"
	// "os"
	// "os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const TEST_NUM = 10

var srvr *Server
var serverHost string = "localhost"
var serverPort int = 3000
var chainId string
var chainName string = "testchain1"
var chainNameOther string = "testchain2"
var chainNameThird string = "testchain3"
var fetchIP string = "104.236.146.58"
var fetchPort string = "15258"

func init() {
	rootPath, _ := filepath.Abs("/public")
	srvr = NewServer(serverHost, uint16(serverPort), TEST_NUM, rootPath)
	go func() {
		srvr.Start()
	}()

	// Can run tests with the below (instead of the above) for debugging purposes.
	// go func() {
	// 	cmd := exec.Command("epm", "--log", "5", "serve")
	// 	cmd.Stdout = os.Stdout
	// 	cmd.Stderr = os.Stderr
	// 	cmd.Run()
	// }()
	time.Sleep(1 * time.Second)
}

func doHttpCall(method string, endpoint string, expectedCode int, t *testing.T) string {
	client := &http.Client{}
	compiledEndPoint := "http://" + serverHost + ":" + strconv.Itoa(serverPort) + "/" + endpoint
	req, err := http.NewRequest(method, compiledEndPoint, bytes.NewBuffer([]byte{}))

	if err != nil {
		panic(err)
	}
	resp, err2 := client.Do(req)

	if err2 != nil {
		panic(err2)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != expectedCode {
		fmt.Println("Output:\n" + string(body))
		expectString := strconv.Itoa(expectedCode)
		resultString := strconv.Itoa(resp.StatusCode)
		t.Error("Expected: " + expectString + ", Got: " + resultString)
		t.FailNow()
	}
	return string(body)
}

func TestFourOhFour(t *testing.T) {
	fmt.Println("Begin 404 test.")

	_ = doHttpCall("GET", "1234", 404, t)

	fmt.Println("404 test: PASSED")
}

func TestHttpEcho(t *testing.T) {
	fmt.Println("Begin echo test.")

	ret := doHttpCall("GET", "echo/testmessage", 200, t)
	if ret != "testmessage" {
		t.Error("Expected: testmessage, Got: " + ret)
		t.FailNow()
	}

	fmt.Println("Echo test: PASSED")
}

func TestHttpNewChain(t *testing.T) {
	fmt.Println("Begin new chain test.")

	endPoint := "eris/new/" + chainName
	_ = doHttpCall("POST", endPoint, 200, t)

	fmt.Println("New chain test: PASSED")
}

func TestHttpPlop(t *testing.T) {
	fmt.Println("Begin plop test.")

	endPoint := "eris/plop/" + chainName + "/chainid"
	ret := doHttpCall("GET", endPoint, 200, t)
	chainId = strings.TrimSpace(ret)

	endPoint = "eris/plop/" + chainName + "/addr"
	_ = doHttpCall("GET", endPoint, 200, t)

	endPoint = "eris/plop/" + chainName + "/config"
	_ = doHttpCall("GET", endPoint, 200, t)

	endPoint = "eris/plop/" + chainName + "/genesis"
	_ = doHttpCall("GET", endPoint, 200, t)

	// no chain is running, this should 500 but we
  // aren't properly capturing exits at this point
	endPoint = "eris/plop/" + chainName + "/pid"
	_ = doHttpCall("GET", endPoint, 200, t)

	// no contracts are deployed, this should 500
  // but we aren't properly capturing exits at
  // this point
	endPoint = "eris/plop/" + chainName + "/vars"
	_ = doHttpCall("GET", endPoint, 200, t)

	endPoint = "eris/plop/" + chainName + "/key"
	_ = doHttpCall("GET", endPoint, 401, t)

	fmt.Println("FYI, the ChainID is: " + chainId)
	fmt.Println("Plop test: PASSED")
}

func TestRefsLs(t *testing.T) {
	fmt.Println("Begin refs ls test.")

	endPoint := "eris/refs/ls"
	_ = doHttpCall("GET", endPoint, 200, t)

	fmt.Println("Refs ls test: PASSED")
}

func TestRefsRm(t *testing.T) {
	fmt.Println("Begin refs rm test.")

	endPoint := "eris/refs/rm/" + chainName
	_ = doHttpCall("POST", endPoint, 200, t)

	fmt.Println("Refs rm test: PASSED")
}

func TestRefsAdd(t *testing.T) {
	fmt.Println("Begin refs add test.")

	endPoint := "eris/refs/add/" + chainName + "/thelonious/" + chainId
	_ = doHttpCall("POST", endPoint, 200, t)

	fmt.Println("Refs add test: PASSED")
}

func TestCheckOut(t *testing.T) {
	fmt.Println("Begin checkout test.")

	endPoint := "eris/new/" + chainNameOther
	_ = doHttpCall("POST", endPoint, 200, t)

	endPoint = "eris/checkout/" + chainName
	_ = doHttpCall("POST", endPoint, 200, t)

	fmt.Println("Refs add test: PASSED")
}

func TestConfig(t *testing.T) {
	fmt.Println("Begin config test.")

	endPoint := "eris/config/" + chainName + "?log_level=5"
	_ = doHttpCall("POST", endPoint, 200, t)

	endPoint = "eris/config/" + chainName + "?log_level=4&local_host=localhost&local_port=15254"
	_ = doHttpCall("POST", endPoint, 200, t)

	fmt.Println("Config test: PASSED")
}

func TestFetch(t *testing.T) {
	fmt.Println("Begin chain fetch test.")

	endPoint := "eris/fetch/" + chainNameThird + "/" + fetchIP + "/" + fetchPort + "?checkout=false"
	_ = doHttpCall("POST", endPoint, 200, t)

	endPoint = "eris/plop/" + chainName + "/chainid"
	ret := strings.TrimSpace(doHttpCall("GET", endPoint, 200, t))
	if ret != chainId {
		t.Error("Expected: " + chainId + ", Got: " + ret)
		t.FailNow()
	}

	fmt.Println("Chain fetch test: PASSED")
}

func TestChainStart(t *testing.T) {
	fmt.Println("Begin chain start test.")

	endPoint := "eris/start/" + chainName + "?log=5"
	_ = doHttpCall("POST", endPoint, 200, t)

	time.Sleep(5 * time.Second)

	fmt.Println("Chain start test: PASSED")
}

func TestChainStatus(t *testing.T) {
	fmt.Println("Begin chain status test.")

	endPoint := "eris/status/" + chainName
	ret := doHttpCall("GET", endPoint, 200, t)
	if ret != "true" {
		t.Error("Expected: true, Got: " + ret)
		t.FailNow()
	}

	fmt.Println("Chain status test: PASSED")
}

func TestRestartChain(t *testing.T) {
	fmt.Println("Begin chain restart test.")

	endPoint := "eris/restart/" + chainName
	_ = doHttpCall("POST", endPoint, 200, t)

	time.Sleep(5 * time.Second)

	fmt.Println("Chain restart test: PASSED")
}

func TestStopChain(t *testing.T) {
	fmt.Println("Begin chain stop test.")

	endPoint := "eris/stop/" + chainName
	_ = doHttpCall("POST", endPoint, 200, t)

	time.Sleep(10 * time.Second)

	fmt.Println("Chain stop test: PASSED")
}

func TestChainStartWithMining(t *testing.T) {
	fmt.Println("Begin chain start with options test.")

	endPoint := "eris/start/" + chainName + "?commit=true&log=5"
	_ = doHttpCall("POST", endPoint, 200, t)

	time.Sleep(5 * time.Second)

	endPoint = "eris/restart/" + chainName + "?commit=true&log=5"
	_ = doHttpCall("POST", endPoint, 200, t)

	time.Sleep(5 * time.Second)

	endPoint = "eris/stop/" + chainName
	_ = doHttpCall("POST", endPoint, 200, t)

	fmt.Println("Chain start with options test: PASSED")
}

func TestClean(t *testing.T) {
	fmt.Println("Begin clean test.")

	endPoint := "eris/clean/" + chainName
	_ = doHttpCall("POST", endPoint, 200, t)

	endPoint = "eris/clean/" + chainNameOther
	_ = doHttpCall("POST", endPoint, 200, t)

	endPoint = "eris/clean/" + chainNameThird
	_ = doHttpCall("POST", endPoint, 200, t)

	fmt.Println("Clean test: PASSED")
}

// Establish websocket connection and rpc to 'echo'
// func TestWsEcho(t *testing.T) {
// 	fmt.Println("Begin websocket echo test.")
// 	origin := "http://localhost/"
// 	url := "ws://localhost:3000/websocket"
// 	ws, err := websocket.Dial(url, "", origin)
// 	if err != nil {
// 		panic(err)
// 	}
// 	req := &Request{}
// 	req.ID = 1
// 	req.JsonRpc = "2.0"
// 	req.Method = "echo"

// 	sVal := &StringValue{"testmessage"}
// 	bts, _ := json.Marshal(sVal)
// 	raw := json.RawMessage(bts)
// 	req.Params = &raw

// 	bts, errJson := json.Marshal(req)
// 	if errJson != nil {
// 		panic(errJson)
// 	}
// 	if _, err := ws.Write(bts); err != nil {
// 		panic(err)
// 	}
// 	var msg = make([]byte, 512)
// 	var n int
// 	if n, err = ws.Read(msg); err != nil {
// 		panic(err)
// 	}

// 	resp := &Response{}

// 	respErr := json.Unmarshal(msg[:n], resp)

// 	if respErr != nil {
// 		panic(respErr)
// 	}

// 	respR, ok := resp.Result.(map[string]interface{})
// 	if !ok {
// 		t.Error("Response result cannot be cast to map")
//    t.FailNow()
// 	}
// 	retStr := respR["value"].(string)
// 	if retStr != "testmessage" {
// 		t.Error("Expected: testmessage, Got: " + retStr)
// 	} else {
// 		fmt.Println("Websocket echo test: PASSED")
// 	}
// 	ws.Close()

// 	time.Sleep(1 * time.Second)
// }
