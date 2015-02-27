package server

import (
	"encoding/json"
	"fmt"
	"github.com/eris-ltd/decerver/interfaces/scripting"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type HttpReqProxy struct {
	URL *url.URL
	Method string
	Host string
	Header http.Header
	Body string
}

func ProxyFromHttpReq(r *http.Request) (*HttpReqProxy, error) {
	
	bts, err := ioutil.ReadAll(r.Body);
	if err != nil {
		return nil, err;
	} else {
		// Make a runtime compatible object
		p := &HttpReqProxy{}
		p.Method = r.Method
		p.Host = r.Host
		p.URL = r.URL
		p.Header = r.Header
		p.Body = string(bts)
		return p, nil
	} 
}

type HttpResp struct {
	Status int               `json:"status"`
	Header map[string]string `json:"header"`
	Body   string            `json:"body"`
}

type HttpAPIServer struct {
	was *WsAPIServer
	rm scripting.RuntimeManager
}

func NewHttpAPIServer(rm scripting.RuntimeManager, maxConnections uint32) *HttpAPIServer {
	was := NewWsAPIServer(rm,maxConnections)
	return &HttpAPIServer{was, rm}
}

// This is our basic http receiver that takes the request and passes it into the js runtime.
func (has *HttpAPIServer) handleHttp(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Stuff")
	u := r.URL
	fmt.Printf("URL %v\n",u)
	if u.Scheme == "ws" {
		has.was.handleWs(w,r)
		return
	}
	
	p := u.Path 
	caller := strings.Split(strings.TrimLeft(p,"/"),"/")[1];
	
	rt := has.rm.GetRuntime(caller)
	// TODO Update this. It's basically how we check if dapp is ready now.
	if rt == nil {
		w.WriteHeader(400)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "Dapp not in focus")
		return
	}
	
	// logger.Println("Incoming: %v\n", r)
	
	prx, errpr := ProxyFromHttpReq(r)
	// Shouldn't happen.
	if errpr != nil {
		has.writeError(w, 400, errpr.Error())
		return
	}
	// TODO this is a bad solution. It should be possible to pass objects (at least maps) right in.
	bts, _ := json.Marshal(prx)
	// DEBUG
	fmt.Println("REQUEST: " + string(bts))
	ret, err := rt.CallFuncOnObj("network", "handleIncomingHttp", string(bts))

	if err != nil {
		has.writeError(w, 500, err.Error())
		return
	}
	
	rStr, sOk := ret.(string)
	if !sOk {
		has.writeError(w, 500, "Passing non string as return value from otto.")
		return
	}
	hr := &HttpResp{}
	errJson := json.Unmarshal([]byte(rStr), hr)

	if errJson != nil {
		has.writeError(w, 500, errJson.Error())
		return
	}
	
	has.writeReq(hr, w)
}

func (has *HttpAPIServer) writeReq(resp *HttpResp, w http.ResponseWriter) {
	logger.Printf("Response status message: %d\n", resp.Status)
	logger.Printf("Response header stuff: %v\n", resp.Header)
	w.WriteHeader(resp.Status)
	for k, v := range resp.Header {
		w.Header().Set(k, v)
	}
	w.Write([]byte(resp.Body))
}

func (has *HttpAPIServer) writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, msg)
}
