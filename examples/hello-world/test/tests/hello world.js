module.exports = {
  "Hello World" : function (browser) {
    browser
      .useXpath()
      .url('http://helloworldwrite:3000/helloworld/')
      .waitForElementVisible('//input[@id="filenameAdd"]', 10000)

      .execute(function () {
        // This is a hackash workaround to the fact that there's no easy way to
        // wait for the alert.
        addFile = function (){
          var fName = document.getElementById('filenameAdd').value;
          var body = document.getElementById('input').value;

          if(body === ""){
            window.alert("There is nothing in the file input text area!");
            return;
          }

          // We send a POST request to the base url that ends with '/apis/helloworld/files'
          // We don't worry about headers and stuff now (although we should).
          var txt = sender.send("POST", baseUrl + "/files", JSON.stringify({name:fName,data:body}));
          document.body.innerHTML += '<div id="alertPresent">File sent! You can now get it by its name.</div>';
        };
      })

      .setValue('//input[@id="filenameAdd"]', "greeting")
      .setValue('//textarea[@id="input"]', "Hello, world!")
      .click('(//button)[1]')
      .waitForElementPresent('//div[@id="alertPresent"]', 1000)
      .pause(5000) // blocks take a bit of time to propagate and be processed by read nodes
      .url('http://helloworldread:3000/helloworld/')
      .waitForElementVisible('//input[@id="filenameGet"]', 15000)
      .pause(3000) // blocks take a bit of time to propagate and be processed by read nodes
      .setValue('//input[@id="filenameGet"]', "greeting")
      .click('(//button)[2]')
      .assert.value('//textarea[@id="output"]', "Hello, world!")
      .end();
  }
};
