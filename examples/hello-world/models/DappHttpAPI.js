
/* DappHttpApi handles incoming http requests. Super basic URL parsing.
 * It assumes the url to be on the common form 'http://localhost:3000/html/helloworld/method?param1=val1&param2=val2&...'
 *
 */
function DappHttpAPI() {
	
	var F_RES = "files";

	var handlers = {};

	// Called on incoming messages.
	this.handle = function(httpReq) {
		
		// Get an url object.
		var urlObj = network.parseUrl(httpReq);
		Printf("URLOBJ: %v\n", urlObj);
		if(urlObj.error !== ""){
			network.getHttpResponse(400,{},urlObj.error);
		}
		
		if(urlObj.path[0] !== F_RES){
			network.getHttpResponse(400,{},"No resource with name: " + urlObj.path[0]);
		}

		// This is where the result will be stored.
		var result;
		var method = httpReq.Method;
		// Now check if the person wants to add a file.

		if(method === "POST"){
			Println("Working");
			if(urlObj.path.length !== 1){
				return network.getHttpResponse(400,{},"Malformed request: Bad url.");
			}
			if (httpReq.Body === ""){
				return network.getHttpResponse(400,{},"Malformed request: No filename provided.");
			}
			var bodyObj = JSON.parse(httpReq.Body);
			Printf("BODYOBJ: %v\n",bodyObj);
			if(bodyObj === undefined || bodyObj.name === undefined || bodyObj.data === undefined){
				return network.getHttpResponse(400,{},"Malformed request: No filename provided.");
			}
			// Now send the filename and data to the add method.
			result = handlers.add(bodyObj.name,bodyObj.data);
			
		} else if (method === "GET") {
			if(urlObj.path.length !== 2){
				return network.getHttpResponse(400,{},"Malformed request: bad url.");
			}
			// Run the 'get' method.
			result = handlers.get(urlObj.path[1]);
		} else {
			return network.getHttpResponse(400,{},"Illegal request: " + method);
		}
		// Generate a new http response.
		return result;
	}

	// Add a file with name 'filename' and the data 'data'.
	handlers.add = function(filename,data){
		var fName = sutil.stringToHex(filename);
		var fHash = writeFile(data);
		if(fHash === ""){
			return network.getHttpResponse(500,{},"Internal error: failed to read file");
		}
		var txData = [];
		txData.push(fName);
		txData.push(fHash);
		msg(txData);
		commit();
		return network.getHttpResponse(200,{},"");
	}

	// Get a file with name 'filename'
	handlers.get = function(name){
		var nameHex = sutil.stringToHex(name);
		var fHash = storageAt(nameHex);
		Println("Getting the storage for filename:" + nameHex);
		Println("Hash: " + fHash);
		var fileData = readFile(fHash);
		Printf("File data: %v\n",fileData);		
		if(fileData === ""){
			return network.getHttpResponse(404,{},"File not found: " + name);
		}
		
		return network.getHttpResponse(200,{},'{ "data" : "' + fileData + '"}');
	};

	// These methods are part of the DappCore UI, but are copied here since we don't need the entire thing.
	function commit(){
		monk.Commit();
	};
	
	// Send a message
	function msg(txData){
		Printf("Pushing stuff to monk. TxData: %v\n", txData);
		Printf("Root contract: " + RootContract);
		var msgRecipe = {
			"Success" : false,
			"Hash" : "",
			"Error" : ""
		};

		var m = monk.Msg(RootContract,txData);
		if (m.Error !== ""){
			msgRecipe.Error = m.Error;
		} else {
			msgRecipe.Success = true;
			Printf(m);
			msgRecipe.Hash = m.Data.Hash;
		}
		return msgRecipe;
	};

	// Gets you the value stored at address 'storageAddress' in the
	// 'RootContract' (which is the contract address specified in the package.json file).
	function storageAt(storageAddress){
		var sa = monk.StorageAt(RootContract, storageAddress);
		if (sa.Error !== ""){
			return "0x0";
		} else {
			return sa.Data;
		}
	};

		// Gets you the value stored at address 'storageAddress' in the
	// 'RootContract' (which is the contract address specified in the package.json file).
	function storageRoot(){
		var sa = monk.Storage(RootContract);
		if (sa.Error !== ""){
			return null;
		} else {
			return sa.Data.Storage;
		}
	};

	// Writes a file to the ipfs file system and returns the hash
	// as a hex string. 
	//
	// NOTE: The hash is stripped of its first two bytes, in order to 
	// get a 32 byte value. The first byte is the hashing algorithm
	// used (it's always 0x12), and the second is the length of the
	// hash (it is always 0x20). See DappCore.ipfsHeader.
	function writeFile(data) {
		var hashObj = ipfs.PushFileData(data);
		if(hashObj.Error !== "") {
			return "";
		} else {
			// This would be the 32 byte hash (omitting the initial "1220").
			return "0x" + hashObj.Data.slice(6);
		}
	};
	
	// Takes the 32 byte hash. Prepends "1220" to create the full hash.
	function readFile(hash) {
		if(hash[1] === 'x'){
			hash = hash.slice(2);
		}
		var fullHash = "1220" + hash;
		var fileObj = ipfs.GetFile(fullHash,false);
		
		if(fileObj.Error !== "") {
			return "";
		} else {
			// This would be the file data as a string.
			return fileObj.Data;
		}
	}

};
