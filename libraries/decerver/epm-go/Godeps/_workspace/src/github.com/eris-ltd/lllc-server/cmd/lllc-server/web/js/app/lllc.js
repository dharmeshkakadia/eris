var hasStorage = false;
	
// Code mirror object
var editor;

var frSupport;

// Code mirror theme selector
var input = document.getElementById("ThemeSelector");
	
$( document ).ready(function() {
				
	// Check if localstorage exists.
	if(typeof(window.localStorage) !== "undefined") {
	    hasStorage = true;
	} else {
		console.log("No local storage. Theme selection will not carry over to the next session.");
	}
	
	// Check for file reading support.
	frSupport = window.File && window.FileReader && window.FileList;
	
	// Define the editor window.
	editor = CodeMirror.fromTextArea(document.getElementById("DataTextArea"),{
        matchBrackets: true,
        indentUnit: 4,
        tabSize: 4,
        indentWithTabs: true,
        lineNumbers: true,
        autoCloseBrackets: true,
        matchBrackets: true,
        styleActiveLine: true,
        mode: "text/x-common-lisp",
		extraKeys : {
			"F11" : function(cm) {
				cm.setOption("fullScreen", !cm.getOption("fullScreen"));
			},
			"Esc" : function(cm) {
				if (cm.getOption("fullScreen"))
					cm.setOption("fullScreen", false);
			},
			"Ctrl-S" : function(){
				editor.save();
			} 
		}
    });
	
	editor.save = function(){
		var str = editor.getValue();
		if(str === null || str.length === ""){
			return;
		}
		saveTextAs(str, "temp.lll");
	}
	
	editor.open = function(){
		
	}
	
	var choice = "rubyblue";
	// Load settings.
	if(hasStorage && localStorage.editorTheme){
		choice = localStorage.editorTheme;
	} else if (hasStorage){
		// Init
		localStorage.editorTheme = choice;
	}
	$('#ThemeSelectorText').html(choice);
	editor.setOption("theme", choice);
	
	prepEditor();
	
	$('#CompilerOutput').enscroll({
	    showOnHover: true,
	    verticalTrackClass: 'trackV',
	    verticalHandleClass: 'handleV',
	    easingDuration : 100
	});
	
	$('#CompilerOutput').html("<samp>*bzzz* Com-pi-ler rea-dy. *bzzz*</samp>");
	
});

function prepEditor(){
	
	sList = "";
	
	sList += '<li><a href="#" >default</a></li>';
	sList += '<li><a href="#" >3024-day</a></li>';
	sList += '<li><a href="#" >3024-night</a></li>';
	sList += '<li><a href="#" >ambiance</a></li>';
	sList += '<li><a href="#" >base16-dark</a></li>';
	sList += '<li><a href="#" >base16-light</a></li>';
	sList += '<li><a href="#" >blackboard</a></li>';
	sList += '<li><a href="#" >cobalt</a></li>';
	sList += '<li><a href="#" >eclipse</a></li>';
	sList += '<li><a href="#" >elegant</a></li>';
	sList += '<li><a href="#" >erlang-dark</a></li>';
	sList += '<li><a href="#" >lesser-dark</a></li>';
	sList += '<li><a href="#" >mbo</a></li>';
	sList += '<li><a href="#" >mdn-like</a></li>';
	sList += '<li><a href="#" >midnight</a></li>';
	sList += '<li><a href="#" >monokai</a></li>';
	sList += '<li><a href="#" >neat</a></li>';
	sList += '<li><a href="#" >neo</a></li>';
	sList += '<li><a href="#" >night</a></li>';
	sList += '<li><a href="#" >paraiso-dark</a></li>';
	sList += '<li><a href="#" >paraiso-light</a></li>';
	sList += '<li><a href="#" >pastel-on-dark</a></li>';
	sList += '<li><a href="#" >rubyblue</a></li>';
	sList += '<li><a href="#" >solarized dark</a></li>';
	sList += '<li><a href="#" >solarized light</a></li>';
	sList += '<li><a href="#" >the-matrix</a></li>';
	sList += '<li><a href="#" >tomorrow-night-eighties</a></li>';
	sList += '<li><a href="#" >twilight</a></li>';
	sList += '<li><a href="#" >vibrant-ink</a></li>';
	sList += '<li><a href="#" >xq-dark</a></li>';
	sList += '<li><a href="#" >xq-light</a></li>';
	
	$('#ThemeSelectorDropDown').html(sList);
	
}

$(function() {
	
	$('#ThemeSelectorDropDown li a').click(function(event){
		event.preventDefault();
		$('#ThemeSelectorText').html($(this).text());
		$(this).parent().parent().dropdown('toggle');
		editor.setOption("theme", $(this).text());
		if (hasStorage){
			localStorage.editorTheme = $(this).text();
		}
		return false;
	});
	
	$('#EditorUndoButton').click(function(event){
		event.preventDefault();
		editor.undo();
		return true;
	});
	
	$('#EditorRedoButton').click(function(event){
		event.preventDefault();
		editor.redo();
		return true;
	});
	
	$('#EditorSaveButton').click(function(event){
		event.preventDefault();
		editor.save();
		return false;
	});
	
	$('#EditorOpenButton').click(function(event){
		event.preventDefault();
		if (frSupport) {
			$('#FileOpenBox').click();
		} else {
		  alert('The File APIs are not fully supported by this browser. You must Copy/Paste yourself.');
		}
		return false;
	});
	
	$('#CompileButton').click(function(event){
		event.preventDefault();
		var lllStr = editor.getValue();
		if(lllStr === null || lllStr.length === 0){
			$('#CompilerOutput').html("<samp><b>There is nothing to compile.</b></samp>");
			return false;
		}
		compile(lllStr);
		return false;
	});
	
	
	
	document.getElementById('FileOpenBox').addEventListener('change', handleFileSelect, false);
});

function handleFileSelect(evt) {
	
	var files = evt.target.files; // FileList object
	var file = files[0];
	
	var reader = new FileReader();

	// Closure to capture the file information.
	reader.onload = (function(theFile) {
		return function(e) {
			editor.setValue(e.target.result);
		};
	})(file);

	// Read in the image file as a data URL.
	reader.readAsText(file);
}

// string to bytes
String.prototype.getBytes = function () {
  var bytes = [];
  for (var i = 0; i < this.length; ++i) {
    bytes.push(this.charCodeAt(i));
  }
  return bytes;
};

// general framework for ajax calls. 

function new_request_obj(){
    if (window.XMLHttpRequest)
        return new XMLHttpRequest();
    else
        return new ActiveXObject("Microsoft.XMLHTTP");
}

function register_callback(xmlhttp, _func, args){
    xmlhttp.onreadystatechange = function(){
        if (xmlhttp.readyState==4 && xmlhttp.status==200){
        	args.unshift(xmlhttp);
        	_func.apply(this, args);
        };
    }
}

function make_request(xmlhttp, method, path, async, params){
    xmlhttp.open(method, path, async);
    xmlhttp.setRequestHeader("Content-type", "application/json");
    xmlhttp.send(JSON.stringify(params));
}

function compile_callback(xmlhttp){
   response = JSON.parse(xmlhttp.responseText);
   console.log(response);
   bytecode = response['bytecode'];
   console.log(bytecode);
   $('#CompilerOutput').html("<samp>Compiled bytecode: 0x" + bytecode + "</samp>");
}

function compile(code){
    codebytes = code.getBytes();
    console.log(code);
    console.log(codebytes);
    c = [codebytes];
    console.log(c);
    xmlhttp = new_request_obj();
    register_callback(xmlhttp, compile_callback, []);
    make_request(xmlhttp, "POST", "/compile2", true, {"scripts":c});
    return false;
}
