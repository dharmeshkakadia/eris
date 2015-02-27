 

Atë javascript Reference (draft)

Atë is the scripting component of decerver. On top of the regular javascript methods such as .toString(), and .strim(), Atë also expose a number of helper functions/api to make dapp writing more convenient.

Note: These functions and objects can also be found in the file: https://github.com/eris-ltd/decerver/blob/master/ate/jsbindings.go

There is however some methods there that is not mentioned here, that you should stay clear of. Those are marked.
Also, keep in mind that Atë will undergo a revision when the javascript runtime library has been modified and 
better fitted with the decerver (although it is unlikely that the apis will change much).

String math

the 'smath' api is a small math library for doing arithmetic operations on number-strings. It understands hex and decimal strings. Hex strings should always be prepended with a '0x', or the result is undefined. If a number-string is
returned from any function, it will be in hex, and start with an '0x'

``` javascript
// Add the two numbers A and B
//
// Params: A and B (as strings)
// Returns: The sum of A and B (as a string)
smath.add = function(A,B)

// Subtract number B from the number A
//
// Params: A and B (as strings)
// Returns: The difference between A and B (as a string)
smath.sub = function(A,B)

// Multiply the two numbers A and B
//
// Params: A and B (as strings)
// Returns: The product of A and B (as a string)
smath.mul = function(A,B)

// Divide the two numbers A and B
//
// Params: A and B (as strings)
// Returns: The quota of A and B (as a string)
//          Division by 0 is undefined.
smath.div = function(A,B)

// Calculates A mod B
//
// Params: A and B (as strings)
// Returns: A mod B (as a string)
//          A mod 0 is undefined
smath.mod = function(A,B)

// Calculates A ^ B (A raised to the power of B)
//
// Params: A and B (as strings)
// Returns: A ^ B
smath.exp = function(A,B)

// Calculates whether the input is equal to zero or not.
// This is true if the input is "0", "0x0", or "0x" (eth quirk)
//
// Params: The number (as a string) to try
// Returns: true if equal to zero, otehrwise false
smath.isZero = function(sNum)

// Calculates whether the two input number-strings are equal.
// This is true if the two numbers evaluate to the same
// Go-lang big integer value, meaning that regardless of
// base or format, tey must represent the same numeric value.
//
// Params: The two number-strings to compare
// Returns: true if equal, otehrwise false
smath.equals = function(A,B)




// Takes a string and converts it into a 32 byte left-padded 
// hex string. This is useful when passing strings as arguments
// to blockchain transactions.
//
// Note: Don't attempt UTF strings. That's not fully supported (yet).
sutil.stringToHex = function(stringVal){
	return StringToHex(stringVal);
}
```

String utilities

sutil has a few methods that makes it easier to translate between a string, and the 
hex representation of the bytes in that string. Doing sutil.stringToHex(string) will 
left-pad a string that is less then 32 bytes long so that the resulting hex string is 
32 bytes long. This is to make them blockchain compatible.

``` javascript
// Takes a string and converts it into a 32 byte left-padded 
// hex string. This is useful when passing strings as arguments
// to blockchain transactions.
//
// Note: Don't attempt UTF strings. That's not fully supported (yet).
sutil.stringToHex = function(stringVal)


// Takes a hex string and converts it into a normal string. It does
// so by converting the hex string into bytes, then converts the
// bytes into a string.
//
// Example:
// The hex string "0x4142" is converted to the byte array [0x41,0x42],
// which is the string "AB". 
//
// Note: Don't attempt UTF strings. That's not fully supported (yet).
sutil.hexToString = function(stringVal)
```

Crypto

A very meager api at this point. 'scrypto' gives you the sha3 (32 byte) digest of a string.

``` javascript

// Takes the sha3 digest of the argument string and returns it as a hex string.
scrypto.sha3 = function(stringVal)
```

Misc.

These are functions used to perform various special tasks, and to give out certain important information.
They always start with an upper case letter, and are global.


``` javascript
// Returns a milisecond timestamp (Number).
// Deprecated: This will be deprecated in favor of the normal javascript Date object.
TimeMS()
```

Printing and logging

The print functions works like the normal Go functions with the same name.
``` javascript
Print(val...) 
Println(val...)
Printf(formattedString,val...)
```

Every message printed by these functions will be prepended with a '[JsRuntime] TIME', where TIME is on a date-time form. 

Example:
var a = "0x1";
var b = "0x2";
var c = smath.add(a,b);
Println("Result of adding: " + c);
Printf("%s + %s = %s\n",a,b,c);


You also have access to the regular console: `console.log(logMessage);`. This is not the recommended method,
however, because it writes the log message without formatting or time-stamping.

Network

The network api lets you handle http requests and websocket connections. It can only do inbound http at this point (no outgoing requests).

Objects:

These are the objects used for http requests and responses:

Request 

{
	URL : UrlObject
	Method : string
	Host : string
	Header : {"Field1" : string, "Field2" : string, ...}
	Body : string
}

UrlObject

{
	Scheme    : string
    Opaque    : string
    Host      : string
    Path      : string
    RawQuery  : string
    Fragment  : string
}

Response

{
	Status : number
	Header : {"Field1" : string, "Field2" : string, ...}
	Body : string
}

Q: Where is Cookies and UserInfo, and what's with the caps?
A: Caps is because these objects are exposed from Go. Go does not export objects or methods that begins with
   lower case letters. UserInfo and Cookies are not here because they haven't been added yet. It's a TODO.
   At this point, http is used mostly to send very basic commands via URLs, GET, POST some stuff, maybe json-rpc, 
   stuff like that.
   
These are the http methods:

```javascript
// Network is an object that encapsulates all networking activity.
		
// Returns a default response object. Status is 0, header and body is empty.
network.getHttpResponse = function(){
	return {
		"Status" : 0,
		"Header" : {},
		"Body" : ""
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
network.getHttpResponseJSON = function(jsonString){
	return {
		"Status" : 200,
		"Header" : {},
		"Body" : jsonString
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
```

Websockets

This is the websocket API.

Session

A websocket session can be used to send message from Atë. It has an id and a writeJson function.

Normally, the websocket connections uses the ESRPC protocol, which is currently a draft. (link)

Also, the function for registering a handler on incoming requests on a socket has been left out.
There are examples of how to do it, for example in monkadmin, and (the coming) basic tutorial 2:
 

```javascript

// Get the session Id. It is a number.
session.sessionId()
// Write a json formatted string.
session.writeJson(jsonString)
```

```javascript
// Error codes for ESRPC
var E_PARSE = -32700;
var E_INVALID_REQ = -32600;
var	E_NO_METHOD = -32601;
var	E_BAD_PARAMS = -32602;
var	E_INTERNAL = -32603;
var	E_SERVER = -32000;

// Convenience method for creating an ESPRC response.
network.getWsResponse = function()

// Convenience method for creating an ESPRC response from
// an error.
network.getWsError = function(errorString)

// Convenience method for creating an ESPRC response from
// an error. This allows you to fill in more details then 
// network.getWsError
network.getWsErrorDetailed = function(code, message, data)

// Convenience method for creating an ESPRC response from
// a E_BAD_PARAMS error.
network.getWsBPError = function(msg)

// This is used to set a callback for each new session. The function should take a session object as
// parameter, and return a function that will be called whenever a new request comes in over that socket. 
// An example of this can be found in the monkadmin:
//
// network.newWsCallback = function(sessionObj){
//	// Instantiate a new blockchain ws for each session.
//	var api = new BlockchainWs(sessionObj, monk);
//	// start listening to monk/eth events.
//	api.startListening();
//	// Attach the api to the session object, for convenience.
//	sessionObj.api = api;
//
//	return function(request){
//		return api.handle(request);
//	};
// }
network.newWsCallback = function(sessionObj)

// This is called whenever a session is deleted.
network.deleteWsCallback = function(sessionObj)
```