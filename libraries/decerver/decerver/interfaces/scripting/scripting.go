package scripting

import (
	"github.com/eris-ltd/decerver/interfaces/types"	
)

const (
	STATUS_NORMAL = iota
	STATUS_WARNING
	STATUS_ERROR
	STATUS_CRITICAL
)

// typedef for javascript objects.
type SObject map[string]interface{}

type(
	// This is the interface for the javascript runtime manager, or 'Atë'.
	RuntimeManager interface {
		GetRuntime(string) Runtime
		CreateRuntime(string) Runtime
		RemoveRuntime(string)
		RegisterApiObject(string, interface{})
		RegisterApiScript(string)
		ShutdownRuntimes()
	}

	// This is the interface for a javascript runtime.
	Runtime interface {
		Init(string)
		Shutdown()
		// This is normally the same as the dapp id when running decerver.
		Id() string
		BindScriptObject(name string, val interface{}) error
		LoadScriptFile(fileName string) error
		LoadScriptFiles(fileName ...string) error
		AddScript(script string) error
		CallFunc(funcName string, param ...interface{}) (interface{}, error)
		CallFuncOnObj(objName, funcName string, param ...interface{}) (interface{}, error)
	}
)

// Converts a data and an error values into a javascript ready object. If an error occurs, 
// the status will be set as such:
// STATUS_NORMAL - if data is non-nil and error is nil, or if both are nil.
// STATUS_ERROR - if data is nil and error is non-nil
// STATUS_WARNING - if both data and error is non-nil
func JsReturnVal(data interface{}, err error) SObject {
	ret := make(SObject)
	
	if data == nil && err == nil {
		ret["Data"] = nil
		ret["Error"] = nil
		ret["Status"] = STATUS_NORMAL
	} else if data != nil && err == nil {
		ret["Data"] = types.ToJsValue(data)
		ret["Error"] = ""
		ret["Status"] = STATUS_NORMAL
	} else if data != nil && err != nil {
		ret["Data"] = types.ToJsValue(data)
		ret["Error"] = err.Error()
		ret["Status"] = STATUS_WARNING
	} else {
		ret["Data"] = nil
		ret["Error"] = err.Error()
		ret["Status"] = STATUS_ERROR
	}
	return ret
}

// Same as JsReturnVal, but lets you set the status code.
func JsReturnValStat(data interface{}, err error, status int) SObject {
	ret := make(SObject)
	ret["Data"] = types.ToJsValue(data)
	if err != nil {
		ret["Error"] = err.Error()
	} else {
		ret["Error"] = ""
	}
	ret["Status"] = status
	
	return ret
}

// If there is no error
func JsReturnValNoErr(data interface{}) SObject {
	ret := make(SObject)
	ret["Data"] = types.ToJsValue(data)
	ret["Error"] = ""
	ret["Status"] = STATUS_NORMAL
	return ret
}

// If there is an error. The error code is set to STATUS_ERROR
func JsReturnValErr(err error) SObject {
	return JsReturnValErrStat(err,STATUS_ERROR)
}

// Same as JsReturnValErr but allows you to set the error code.
func JsReturnValErrStat(err error, statusCode int) SObject {
	ret := make(SObject)
	ret["Data"] = nil
	ret["Error"] = err.Error()
	ret["Status"] = statusCode
	return ret
}