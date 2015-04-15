package runtimemanager

import (
	"encoding/hex"
	"fmt"
	"github.com/eris-ltd/decerver/interfaces/logging"
	"github.com/obscuren/sha3"
	"github.com/robertkrimen/otto"
	"log"
	"math/big"
	"time"
	"os/user"
)

var BZERO *big.Int = big.NewInt(0)

func isZero(i *big.Int) bool {
	return i.Cmp(BZERO) == 0
}

var ottoLog *log.Logger = logging.NewLogger("Runtime")

// TODO clean up the scripts. Make proper function objects.
func BindDefaults(runtime *Runtime) {
	vm := runtime.vm

	bindGo(vm)

	bindCore(vm)
	bindNetworking(vm)
	bindEvents(vm)
	bindUtilities(vm)

}

func bindGo(vm *otto.Otto) {

	vm.Set("Add", func(call otto.FunctionCall) otto.Value {
		p0, p1, errP := parseBin(call)
		if errP != nil {
			return otto.UndefinedValue()
		}
		result, _ := vm.ToValue("0x" + hex.EncodeToString(p0.Add(p0, p1).Bytes()))
		return result
	})

	vm.Set("Sub", func(call otto.FunctionCall) otto.Value {
		p0, p1, errP := parseBin(call)
		if errP != nil {
			return otto.UndefinedValue()
		}
		p0.Sub(p0, p1)
		if p0.Sign() < 0 {
			otto.NaNValue() // TODO
		}
		result, _ := vm.ToValue("0x" + hex.EncodeToString(p0.Bytes()))
		return result
	})

	vm.Set("Mul", func(call otto.FunctionCall) otto.Value {
		p0, p1, errP := parseBin(call)
		if errP != nil {
			return otto.UndefinedValue()
		}
		result, _ := vm.ToValue("0x" + hex.EncodeToString(p0.Mul(p0, p1).Bytes()))
		return result
	})

	vm.Set("Div", func(call otto.FunctionCall) otto.Value {
		p0, p1, errP := parseBin(call)
		if errP != nil {
			return otto.UndefinedValue()
		}
		if isZero(p1) {
			return otto.NaNValue()
		}
		result, _ := vm.ToValue("0x" + hex.EncodeToString(p0.Div(p0, p1).Bytes()))
		return result
	})

	vm.Set("Mod", func(call otto.FunctionCall) otto.Value {
		p0, p1, errP := parseBin(call)
		if errP != nil {
			return otto.UndefinedValue()
		}
		if isZero(p1) {
			return otto.NaNValue()
		}
		result, _ := vm.ToValue("0x" + hex.EncodeToString(p0.Mod(p0, p1).Bytes()))
		return result
	})

	vm.Set("Equals", func(call otto.FunctionCall) otto.Value {
		p0, p1, errP := parseBin(call)
		if errP != nil {
			return otto.UndefinedValue()
		}
		ret := false
		if p0.Cmp(p1) == 0 {
			ret = true
		}
		result, _ := vm.ToValue(ret)
		return result
	})

	vm.Set("LT", func(call otto.FunctionCall) otto.Value {
		p0, p1, errP := parseBin(call)
		if errP != nil {
			return otto.UndefinedValue()
		}
		ret := false
		if p0.Cmp(p1) < 0 {
			ret = true
		}
		result, _ := vm.ToValue(ret)
		return result
	})

	vm.Set("LEQ", func(call otto.FunctionCall) otto.Value {
		p0, p1, errP := parseBin(call)
		if errP != nil {
			return otto.UndefinedValue()
		}
		ret := false
		if p0.Cmp(p1) <= 0 {
			ret = true
		}
		result, _ := vm.ToValue(ret)
		return result
	})

	vm.Set("GT", func(call otto.FunctionCall) otto.Value {
		p0, p1, errP := parseBin(call)
		if errP != nil {
			return otto.UndefinedValue()
		}
		ret := false
		if p0.Cmp(p1) > 0 {
			ret = true
		}
		result, _ := vm.ToValue(ret)
		return result
	})

	vm.Set("GEQ", func(call otto.FunctionCall) otto.Value {
		p0, p1, errP := parseBin(call)
		if errP != nil {
			return otto.UndefinedValue()
		}
		ret := false
		if p0.Cmp(p1) >= 0 {
			ret = true
		}
		result, _ := vm.ToValue(ret)
		return result
	})

	vm.Set("Exp", func(call otto.FunctionCall) otto.Value {
		p0, p1, errP := parseBin(call)
		if errP != nil {
			return otto.UndefinedValue()
		}
		result, _ := vm.ToValue("0x" + hex.EncodeToString(p0.Exp(p0, p1, nil).Bytes()))
		// fmt.Println("[OTTOTOTOOTT] " + )
		return result
	})

	vm.Set("IsZero", func(call otto.FunctionCall) otto.Value {
		prm, err0 := call.Argument(0).ToString()
		if err0 != nil {
			return otto.UndefinedValue()
		}
		isZero := prm == "0" || prm == "0x" || prm == "0x0"
		result, _ := vm.ToValue(isZero)

		return result
	})

	vm.Set("HexToString", func(call otto.FunctionCall) otto.Value {
		prm, err0 := call.Argument(0).ToString()
		if err0 != nil {
			fmt.Println(err0)
			return otto.UndefinedValue()
		}
		if prm == "" || prm == "0x0" || prm == "0x" || prm == "0" {
			logger.Println("Getting zero hex string from otto, returning empty string")
			r, _ := vm.ToValue("")
			return r
		}
		if prm[1] == 'x' {
			prm = prm[2:]
		}
		bts, err1 := hex.DecodeString(prm)
		if err1 != nil {
			fmt.Println(err1)
			return otto.UndefinedValue()
		}
		result, _ := vm.ToValue(string(bts))

		return result
	})

	vm.Set("StringToHex", func(call otto.FunctionCall) otto.Value {
		prm, err0 := call.Argument(0).ToString()
		if err0 != nil {
			return otto.UndefinedValue()
		}
		bts := []byte(prm)

		if 32 > len(bts) {
			zeros := make([]byte, 32-len(bts))
			bts = append(zeros, bts...)
		}
		res := "0x" + hex.EncodeToString(bts)
		result, _ := vm.ToValue(res)

		return result
	})

	// Millisecond time.
	vm.Set("TimeMS", func(call otto.FunctionCall) otto.Value {
		ts := time.Now().UnixNano() >> 9
		result, _ := vm.ToValue(ts)
		return result
	})

	// Millisecond time.
	vm.Set("GetUserHome", func(call otto.FunctionCall) otto.Value {
		fmt.Println("Getting user home.");
		usr, err := user.Current();
		if err != nil {
			vv, _ := vm.ToValue("")
			return vv
		}
		result, _ := vm.ToValue(usr.HomeDir)
		return result
	})

	// Crypto
	vm.Set("SHA3", func(call otto.FunctionCall) otto.Value {
		prm, err0 := call.Argument(0).ToString()
		if err0 != nil {
			return otto.UndefinedValue()
		}
		if len(prm) == 0 {
			logger.Printf("Trying to hash an empty string.")
			return otto.UndefinedValue()
		}
		if prm[1] == 'x' {
			prm = prm[2:]
		}
		h, err := hex.DecodeString(prm)
		if err != nil {
			logger.Printf("Error hashing: %s. Val: %s, Len: %d\n ", err.Error(), prm, len(prm))
			return otto.UndefinedValue()
		}
		d := sha3.NewKeccak256()
		d.Write(h)
		v := hex.EncodeToString(d.Sum(nil))
		//		fmt.Println("SHA3: " + v)
		result, _ := vm.ToValue("0x" + v)

		return result
	})

	vm.Set("Print", func(call otto.FunctionCall) otto.Value {
		output := make([]interface{}, 0)
		// TODO error
		for _, argument := range call.ArgumentList {
			arg, _ := argument.Export()
			output = append(output, arg)
		}
		ottoLog.Print(output...)
		return otto.Value{}
	})

	vm.Set("Println", func(call otto.FunctionCall) otto.Value {
		output := make([]interface{}, 0)
		// TODO error
		for _, argument := range call.ArgumentList {
			arg, _ := argument.Export()
			output = append(output, arg)
		}
		ottoLog.Println(output...)
		return otto.Value{}
	})

	vm.Set("Printf", func(call otto.FunctionCall) otto.Value {
		args := call.ArgumentList
		if args == nil || len(args) == 0 {
			ottoLog.Println("")
			return otto.Value{}
		}
		fmtStr, _ := args[0].Export()
		fs, ok := fmtStr.(string)
		if !ok {
			ottoLog.Println("")
			return otto.Value{}
		}

		if len(args) == 1 {
			ottoLog.Printf(fs)
		} else {
			output := make([]interface{}, 0)
			// TODO error
			for _, argument := range call.ArgumentList[1:] {
				arg, _ := argument.Export()
				output = append(output, arg)
			}
			ottoLog.Printf(fs, output...)
		}
		return otto.Value{}
	})
}

func bindCore(vm *otto.Otto) {
	_, err := vm.Run(`

		// (Integer) math done on strings. The strings can be
		// either hex or decimal. Hex strings should be prepended
		// by '0x'. All arithmetic operations has integer inputs and
		// outputs (no floating point numbers).
		//
		// If the returned value is a number, as in 'add' or 'mul',
		// then it is always a hex-string.
		var smath = {};

		// Add the two numbers A and B
		//
		// Params: A and B (as strings)
		// Returns: The sum of A and B (as a string)
		smath.add = function(A,B){
			return Add(A,B);
		}

		// Subtract number B from the number A
		//
		// Params: A and B (as strings)
		// Returns: The difference between A and B (as a string)
		smath.sub = function(A,B){
			return Sub(A,B);
		}

		// Multiply the two numbers A and B
		//
		// Params: A and B (as strings)
		// Returns: The product of A and B (as a string)
		smath.mul = function(A,B){
			return Mul(A,B);
		}

		// Divide the two numbers A and B
		//
		// Params: A and B (as strings)
		// Returns: The quota of A and B (as a string)
		//          Division by 0 is undefined.
		smath.div = function(A,B){
			return Div(A,B);
		}

		// Calculates A mod B
		//
		// Params: A and B (as strings)
		// Returns: A mod B (as a string)
		//          A mod 0 is undefined
		smath.mod = function(A,B){
			return Mod(A,B);
		}

		// Calculates A ^ B (A raised to the power of B)
		//
		// Params: A and B (as strings)
		// Returns: A ^ B
		smath.exp = function(A,B){
			return Exp(A,B);
		}

		// Calculates whether the input is equal to zero or not.
		// This is true if the input is "0", "0x0", or "0x" (eth quirk)
		//
		// Params: The number (as a string) to try
		// Returns: true if equal to zero, otehrwise false
		smath.isZero = function(sNum){
			return IsZero(sNum);
		}

		// Calculates whether the two input number-strings are equal.
		// This is true if the two numbers evaluate to the same
		// Go-lang big integer value, meaning that regardless of
		// base or format, tey must represent the same numeric value.
		//
		// Params: The two number-strings to compare
		// Returns: true if equal, otehrwise false
		smath.equals = function(A,B){
			return Equals(A,B);
		}

		// A few easy-to-use string utility functions, such as converting
		// between a string value and a hex representation of that string.
		var sutil = {};

		// Takes a string and converts it into a 32 byte left-padded
		// hex string. This is useful when passing strings as arguments
		// to blockchain transactions.
		//
		// Note: Don't attempt UTF strings. That's not fully supported (yet).
		sutil.stringToHex = function(stringVal){
			return StringToHex(stringVal);
		}

		// Takes a hex string and converts it into a normal string. It does
		// so by converting the hex string into bytes, then converts the
		// bytes into a string.
		//
		// Example:
		// The hex string "0x4142" is converted to the byte array [0x41,0x42],
		// which is the string "AB".
		//
		// Note: Don't attempt UTF strings. That's not fully supported (yet).
		sutil.hexToString = function(stringVal){
			return HexToString(stringVal);
		}

		// Crypto utility
		var scrypto = {};

		// Takes the sha3 digest of the argument string.
		scrypto.sha3 = function(stringVal){
			return SHA3(stringVal);
		}

	`)

	if err != nil {
		logger.Println("Error while bootstrapping js core: " + err.Error())
	} else {
		logger.Println("Core script loaded.")
	}
}

func bindNetworking(vm *otto.Otto) {

	// Networking.
	_, err := vm.Run(`

		// Network is an object that encapsulates all networking activity.
		var network = {};

		// Http

		// Returns a default response object. Status is 0, header and body is empty.
		network.getHttpResponse = function(status,header,body){
			return {
				"Status" : status || 400,
				"Header" : header || {},
				"Body" : body || "Not supported."
			};
		}

		// Returns a response object with status 500, empty header, and body set to: "Internal error"
		network.getHttpResponse500 = function(){
			return {
				"Status" : 500,
				"Header" : {},
				"Body" : "Internal error"
			};
		}

		// Returns a http request with status 200, empty header, and (what should be) a json formatted
		// string as body.
		network.getHttpResponseJSON = function(obj){
			return {
				"Status" : 200,
				"Header" : {"Content-Type": "application/json"},
				"Body" : JSON.stringify(obj)
			};
		}

		// This should be overridden by dapps. It is called whenever a new http request arrives, and
		// will pass the request object as argument to the function.
		//
		// The default funciton will return a 200 with header set to plain-text.
		network.incomingHttpCallback = function(httpReq){
			return {
				"Status" : 200,
				"Header" : {"Content-Type" : "text/plain; charset=utf-8"},
				"Body" : ""
			};
		}

		// Used internally. Do not call this from javascript.
		network.handleIncomingHttp = function(httpReqAsJson){
			var httpReq = JSON.parse(httpReqAsJson);
			var ret = this.incomingHttpCallback(httpReq);
			var rets;
			try {
				rets = JSON.stringify(ret);
				if (rets.length < 10000) {
					Println("Json string of resp obj:\n" + rets);
				} else {
					retsPrint = rets.substr(0, 10000) + " ...{truncated}";
					Println("Json string of resp obj:\n" + retsPrint);
				}
				return rets;
			} catch(err) {
				return network.getHttpResponse500();
			}
		};

		network.registerIncomingHttpCallback = function(callback){
			if(typeof callback !== "function"){
				throw Error("Attempting to register a non-function as incoming http callback");
			}
			Println("New http request callback registered");
			network.incomingHttpCallback = callback;
		}

		// Websockets

		// Error codes for ESRPC
		var E_PARSE = -32700;
		var E_INVALID_REQ = -32600;
		var	E_NO_METHOD = -32601;
		var	E_BAD_PARAMS = -32602;
		var	E_INTERNAL = -32603;
		var	E_SERVER = -32000;

		// Convenience method for creating an ESPRC response.
		network.getWsResponse = function(){
			return {
				"Protocol" : "EWSMP1",
				"Method" : "",
				"Result" : "",
				"Error" : "",
				"Time" : "",
				"Id" : ""
			};
		}

		// Convenience method for creating an ESPRC response from
		// an error.
		network.getWsError = function(error){
			if (typeof(error) !== "string") {
				error = "Server passed non string to error function (bad server-side javascript).";
			}
			return {
				"Protocol" : "EWSMP1",
				"Method" : "",
				"Result" : "",
				"Timestamp" : "",
				"Id" : "",
				"Error" : {
					"Code" : E_INTERNAL,
					"Message" : error,
					"Data" : null
				  }
			};
		}

		// Convenience method for creating an ESPRC response from
		// an error. This allows you to fill in more details then
		// network.getWsError
		network.getWsErrorDetailed = function(code, message, data){
			return {
				"Protocol" : "ESRPC",
				"Method" : "",
				"Result" : "",
				"Time" : "",
				"Id" : "",
				"Error" : {
					"Code" : code,
					"Message" : message,
					"Data" : data
				  }
			};
		}

		// Convenience method for creating an ESPRC response from
		// a E_BAD_PARAMS error.
		network.getWsBPError = function(msg){

			if(typeof(msg) !== "string") {
				if(typeof(msg) !== "undefined") {
					msg = "Server passed non string to error function (bad server-side javascript).";
				} else {
					msg = "Invalid params to method call.";
				}
			} else if(msg === ""){
				msg = "Invalid params to method call.";
			}

			return {
				"Protocol" : "EWSMP1",
				"Method" : "",
				"Result" : "",
				"Timestamp" : "",
				"Id" : "",
				"Error" : {
					"Code" : E_BAD_PARAMS,
					"Message" : msg,
					"Data" : null
				  }
			};
		}


		// handlers for websockets.
		network.wsHandlers = {};
		// the websocket sessions themselves.
		network.wsSessions = {};

		// This is used to set a callback for each new session.
		// the default function does nothing, and should be
		// overriden in dapp backend javascript.
		network.newWsCallback = function(sessionObj){
			return function (){
				Println("No callback registered for new websocket connections.");
			};
		};

		// This is called from go code as a response to newly negotiated
		// websocket connections. It is used to bind the session object
		// to the runtime.
		// WARNING: Should not be used.
		network.newWsSession = function(sessionObj){
			var sId = sessionObj.SessionId();
			Println("Adding new session: " + sId);
			network.wsHandlers[sId] = network.newWsCallback(sessionObj);
			network.wsSessions[sId] = sessionObj;
		}

		// This is called whenever a session is deleted.
		network.deleteWsCallback = function(sessionObj){
			return function (){
				Println("No callback registered for delete websocket connections.");
			};
		};

		// Called from go code to delete a session.
		// WARNING: Should not be used.
		network.deleteWsSession = function(sessionId){
			var sId = sessionId;
			var sessionObj = network.wsSessions[sId];
			if(typeof network.wsSessions[sId] === "undefined" || network.wsSessions[sId] === null){
				Println("No session with id " + sId + ". Cannot delete.");
				return;
			}
			Println("Deleting session: " + sId);
			network.wsSessions[sId] = null;
			network.deleteWsCallback(sessionObj);
		}

		// This is called from go code when new messages arrive.
		// WARNING: Should not be used.
		network.incomingWsMsg = function(sessionId, reqJson) {
			Println("Incoming websocket message.");
			try {
				var request = JSON.parse(reqJson);
				if (typeof(request.Method) === "undefined" || request.Method === ""){
					return JSON.stringify(network.getWsError(E_NO_METHOD, "No method in request."));
				} else {
					var handler = network.wsHandlers[sessionId];
					if (typeof handler !== "function"){
						return JSON.stringify(network.getWsError(E_SERVER, "Handler not registered for websocket session: " + sessionId.toString()) );
					}
					var response = handler(request);
					if(response === null){
						return null;
					}
					var respStr;
					try {
						response.Time = TimeMS();
						respStr = JSON.stringify(response);
					} catch (err) {
						return JSON.stringify(network.getWsError(E_INTERNAL, "Failed to marshal response: " + err));
					}
					return respStr;
				}
			} catch (err){
				response = JSON.stringify(network.getWsError(E_PARSE, err));
			}
		}

		// Parse the url of a request into a formatted URL object.
		// TODO document and handle errors.
		network.parseUrl = function(httpReq){
			// This would become ["apis", "dappname", "whatever", ... ]
			var p = httpReq.URL.Path;
			var pSplit = p.slice(1).split('/');

			// We cut out "apis" and "dappname".
			if (pSplit.length < 2){
				return network.newUrlObj([],{},"Invalid URL");
			}
			pSplit = pSplit.slice(2);

			var opts = {};

			var rawQuery = httpReq.URL.RawQuery.split('&');

			for(var i = 0; i < rawQuery.length; i++) {
				var q = rawQuery[i].split('=');
				// TODO error
				opts[q[0]] = q[1];
			}

			var urlObj = network.newUrlObj(pSplit,opts,"");
			return urlObj;
		}

		// Called by network.parseUrl()
		// 'path' is an array of strings
		// 'opts' is a string->string map
		// 'error' is a string
		// TODO document and handle errors
		network.newUrlObj = function(path,opts,error){
			if(arguments.length === 1 && typeof(arguments[0]) === "string"){
				return network.newUrlObj([],{},arguments[0]);
			} else {
				return {"path" : path || [], "options" : opts || {}, "error" : error || ""};
			}
		}

	`)

	if err != nil {
		logger.Println("Error while bootstrapping js networking: " + err.Error())
	} else {
		logger.Println("Networking script loaded.")
	}
}

func bindEvents(vm *otto.Otto) {
	_, err := vm.Run(`

		// This is the events object. It handles events that comes
		// in from the event processor.
		var events = {};

		// These are callbacks that are used for events.
		events.callbacks = {};

		/*  Called to subscribe on an event.
		 *
		 *  Params:
		 *  eventSource - the source module, ipfs, monk, etc. (string)
		 *  eventType   - the type of event. Could be 'newBlock' for monk. (string)
		 *  eventTarget - optional (string)
		 *  callbackFn  - the callback function to use when the event (string).
		 *                comes in.
		 *  uid         - usually the session id as a string. Used to make the id unique.
		 *                Uid needs to be a string.
		 */
		events.subscribe = function(eventSource, eventType, eventTarget, callbackFn, uid){
			Println("Subscribing");
			if(typeof(callbackFn) !== "function"){
				throw new Error("Trying to register a non callback function as callback.");
			}
			var eventId = events.generateId(eventSource,eventType, uid);
			Println("Adding sub: " + eventId);
			// The jsr_events object has the go bindings to actually subscribe.
			events_subscribe(eventSource, eventType, eventTarget, eventId);
			this.callbacks[eventId] = callbackFn;
		}

		events.unsubscribe = function(eventSource,eventName, uid){
			var subId = events.generateId(eventSource,eventName, uid);
			events_unsubscribe(subId);
			events.callbacks[subId] = null;
		}

		// Called by the go event processor.
		events.post = function(subId, eventJson){
			var event = JSON.parse(eventJson);
			var cfn = this.callbacks[subId];
			if (typeof(cfn) === "function"){
				cfn(event);
			} else {
				Println("No callback for event: " + eventId);
			}
			return;
		}

		// used by events to generate unique subscriber Ids based on
		// the event source and name.
		events.generateId = function(evtSource,evtName, uid){
			return RuntimeId + "_" + evtSource + "_" + evtName + "_" + uid;
		}
	`)

	if err != nil {
		logger.Println("Error while bootstrapping js event-processing: " + err.Error())
	} else {
		logger.Println("Event processing script loaded.")
	}
}

func bindUtilities(vm *otto.Otto) {

	_, err := vm.Run(`

		var STATUS_NORMAL = 0;
		var STATUS_WARNING = 1;
		var STATUS_ERROR = 2;
		var STATUS_CRITICAL = 3;

		var validator = new function(){

			// TODO add
			this.validateBlockchain = function(blockchain){
				return true;
			}

			this.validateFilesystem = function(filesystem){
				return true;
			}

		}

		// This is a simple decerver API is for dapps. Gives access to monk, ipfs and legal markdown.
		function LightApi(){

			// *************************** Variables *************************

			// The blockchain we're using.
			var bc = monk;

			// This is the root contract of any given dapp, as provided by the
			// dapps package.json. It is bound to the runtime under the name
			// "RootContract".
			var rootContract;

			if (typeof(RootContract) !== "undefined"){
				rootContract = RootContract;
			} else {
				Println("No root contract address! The dapp will not work.");
				rootContract = "0x0";
			}

			// The first four hex-digits (or two bytes) of the ipfs hash.
			var ipfsHead = "1220";

			// Used internally to keep track of received events.
			var eventCallbacks = {};

			// ************************* Blockchain ************************

			// Gets you the balance of the account with address 'accountAddress'.
			this.balanceAt = function(accountAddress){
				var acc = this.account(accountAddress)
				if(acc === null){
					return "0x0";
				} else {
					return acc.Balance;
				}
			}

			// Gets you the value stored at address 'storageAddress' in the account
			// with address 'accountAddress'
			this.storageAt = function(accountAddress,storageAddress){
				var sa = bc.StorageAt(accountAddress, storageAddress);
				if (sa.Error !== ""){
					return "0x0";
				} else {
					return sa.Data;
				}
			}

			// Gets you the entire storage of the account with address 'accountAddress'.
			this.storage = function(accountAddress){
				var storage = bc.Storage(accountAddress);
				if (storage.Error !== ""){
					return null;
				} else {
					return storage.Data;
				}
			}

			// Gets you the account with address 'accountAddress'
			this.account = function(accountAddress){
				var a = bc.Account(accountAddress);
				if (a.Error !== ""){
					return null;
				} else {
					return a.Data;
				}
			}

			this.msg = function(recipient,txData){

				var msgRecipe = {
					"Success" : false,
					"Hash" : "",
					"Error" : ""
				};

				var m = bc.Msg(recipient,txData);
				if (m.Error !== ""){
					msgRecipe.Error = m.Error;
				} else {
					msgRecipe.Success = true;
					msgRecipe.Hash = m.Data.Hash;
				}
				return msgRecipe;
			}

			this.endow = function(recipient,sum){

				var edRecipe = {
					"Success" : false,
					"Hash" : "",
					"Error" : ""
				};

				var e = bc.Msg(recipient,txData);
				if (e.Error !== ""){
					edRecipe.Error = e.Error;
				} else {
					edRecipe.Success = true;
					edRecipe.Hash = e.Data.Hash;
				}
				return msgRecipe;
			}

			this.script = function(script,language){

				var scRecipe = {
					"Compiled" : false,
					"Hash" : "",
					"Address" : "",
					"Error" : ""
				};

				var s = bc.Script(recipient,txData);
				if (s.Error !== ""){
					edRecipe.Error = s.Error;
				} else {
					edRecipe.Success = true;
					edRecipe.Hash = s.Data.Hash;
				}
				return msgRecipe;
			}

			// Sends a "full" transaction to the blockchain.
			this.transact = function(recipientAddress,value,gas,gascost,data){

				var txRecipe = {
					"Compiled" : false,
					"Error"    : "",
		            "Success"  : false,
		            "Address"  : "",
					"Hash"     : ""
				};

				if (recipientAddress === "") {
					Println("Create transaction.");
					var addr = bc.Script(data, "lll-literal")
					if (addr.Error !== "") {
						return null;
					} else {
						txRecipe.Address = addr.Data;
						txRecipe.Compiled = true;
						txRecipe.Success = true;
					}
				} else if (data === "") {
					Println("Processing tx");
					var rData = bc.Tx(recipientAddress,value);
					if (rData.Error !== ""){
						return null;
					}
					txRecipe.Success = true;
		       		txRecipe.Hash = rData.Data.Hash;
				} else if (value === ""){
					Println("Processing message");
					var txData = data.split("\n");
					for (var i = 0; i < txData.length; i++){
						txData[i] = txData[i].trim();
					}
					var rData = bc.Msg(recipientAddress,txData);
					if (rData.Error !== ""){
						return null
					}
					txRecipe.Success = true;
					txRecipe.Hash = rData.Data.Hash;
				} else {
					// TODO general purpose tx
					Println("Processing transaction");
					var txData = txData.split("\n");
					for (var i = 0; i < txData.length; i++){
						txData[i] = txData[i].trim();
					}

					//var rData = bc.Transact(recipientAddress,value,gas,gascost,txData);
					//if (rData.Error !== ""){
					//	return null
					//}
					//txRecipe.Success = true;
					//txRecipe.Hash = rData.Data.Hash;
				}
				return txRecipe;
			}

			// ******************** FileSystem (IPFS) ********************

			// Writes a file to the ipfs file system and returns the hash
			// as a hex string.
			//
			// NOTE: The hash is stripped of its first two bytes, in order to
			// get a 32 byte value. The first byte is the hashing algorithm
			// used (it's always 0x12), and the second is the length of the
			// hash (it is always 0x20). See DappCore.ipfsHeader.
			this.writeFile = function(data) {
				var hashObj = ipfs.PushBlock(data);
				if(hashObj.Error !== "") {
					return "";
				} else {
					// This would be the 32 byte hash (omitting the initial "1220").
					return "0x" + hashObj.Data.slice(4);
				}
			}

			// Takes the 32 byte hash. Prepends "1220" to create the full hash.
			this.readFile = function(hash) {
				// Remove '0x' if it is there.
				if(hash[1] === 'x'){
					hash = hash.slice(2);
				}
				var fullHash = ipfsHead + hash;
				var fileObj = ipfs.GetBlock(fullHash);

				if(fileObj.Error !== "") {
					return "";
				} else {
					// This would be the file data as a string.
					return fileObj.Data;
				}
			}

			// ********************** Legal Markdown **********************

			// This function gives you a pdf representation of legal markdown.
			//
			// NOTE: It is mostly used for legal contract dual-integration, as
			// when you want to create a smart contract on the blockchain and
			// at the same time put an actual legal contract into the filesystem
			// as a complement. The hash of the legal contract would often be
			// part of the smart contract, as a guarantee.
			//
			// More info can be found here:
			//
			// https://github.com/eris-ltd/legalmarkdown
			// http://lmd.io
			//
			this.markdownToPDF = function(markdown,params){
				var retData = lmd.Compile(markdown,params);
				if(retData.Error !== ""){
					return "";
				} else {
					return retData.Data;
				}
			}

		}
	`)

	if err != nil {
		logger.Println("Error while bootstrapping js utilities: " + err.Error())
	} else {
		logger.Println("Js utilities script loaded.")
	}
}

func parseUn(call otto.FunctionCall) (*big.Int, error) {
	str, err0 := call.Argument(0).ToString()
	if err0 != nil {
		return nil, err0
	}
	val := atob(str)
	return val, nil
}

func parseBin(call otto.FunctionCall) (*big.Int, *big.Int, error) {
	left, err0 := call.Argument(0).ToString()
	if err0 != nil {
		return nil, nil, err0
	}
	right, err1 := call.Argument(1).ToString()

	if err1 != nil {
		return nil, nil, err1
	}
	p0 := atob(left)
	p1 := atob(right)
	return p0, p1, nil
}

func atob(str string) *big.Int {
	i := new(big.Int)
	fmt.Sscan(str, i)
	return i
}
