/*  This can be instantiated and run in order to start the hello world dapp. The startup 
 *  sequence is mostly a formality (it's the same in every single dapp, and does not really 
 *  require any thinking - just follow this script basically.
 */
function HelloWorldDapp(){

	Println("Creating helloworld.");

	// var dappCore = new DappCore();
	// var api = new DappHttpAPI(dappCore);
	var api = new DappHttpAPI();

	this.run = function(){
		Println("Starting helloworld.");
		// We overwrite the new incoming http callback with this function.
		network.incomingHttpCallback = function(request) {
			return api.handle(request);
		}
	}
}

// Start it up
new HelloWorldDapp().run();
