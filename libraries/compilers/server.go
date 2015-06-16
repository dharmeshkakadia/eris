package lllcserver

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/eris-ltd/lllc-server/Godeps/_workspace/src/github.com/ebuchman/go-shell-pipes"
	"github.com/eris-ltd/lllc-server/Godeps/_workspace/src/github.com/eris-ltd/epm-go/utils"
	"github.com/eris-ltd/lllc-server/Godeps/_workspace/src/github.com/go-martini/martini"
	"github.com/eris-ltd/lllc-server/Godeps/_workspace/src/github.com/martini-contrib/gorelic"
	"github.com/eris-ltd/lllc-server/Godeps/_workspace/src/github.com/martini-contrib/secure"
	segment "github.com/eris-ltd/lllc-server/Godeps/_workspace/src/github.com/segmentio/analytics-go"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strings"
)

var (
	//"" = abi.ABI{}
	NEWRELIC_KEY = os.Getenv("NEWRELIC_KEY")
	NEWRELIC_APP = os.Getenv("NEWRELIC_APP")
	SEGMENT_KEY  = os.Getenv("SEGMENT_KEY")
)

// must have compiler installed!
func homeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}

// Server cache location in eris tree
var ServerCache = path.Join(utils.Lllc, "server")

// Handler for proxy requests (ie. a compile request from langauge other than go)
func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	// read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Errorln("err on read http request body", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println("body:", string(body))
	req := new(ProxyReq)
	err = json.Unmarshal(body, req)
	if err != nil {
		logger.Errorln("err on read http request body", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var code []byte
	var abi string
	if req.Literal {
		code, abi, err = CompileLiteral(req.Source, req.Language)
	} else {
		code, abi, err = Compile(req.Source)
	}
	resp := NewProxyResponse(code, abi, err)

	respJ, err := json.Marshal(resp)
	if err != nil {
		logger.Errorln("failed to marshal", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write(respJ)
}

// Main http request handler
// Read request, compile, build response object, write
func CompileHandler(w http.ResponseWriter, r *http.Request) {
	resp := compileResponse(w, r)
	if resp == nil {
		return
	}
	respJ, err := json.Marshal(resp)
	if err != nil {
		logger.Errorln("failed to marshal", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Write(respJ)
}

// Convenience wrapper for javascript frontend
func CompileHandlerJs(w http.ResponseWriter, r *http.Request) {
	resp := compileResponse(w, r)
	if resp == nil {
		return
	}
	code := resp.Bytecode
	hexx := hex.EncodeToString(code)
	w.Write([]byte(fmt.Sprintf(`{"bytecode": "%s"}`, hexx)))
}

// read in the files from the request, compile them
func compileResponse(w http.ResponseWriter, r *http.Request) *Response {
	// read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Errorln("err on read http request body", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	// unmarshall body into req struct
	req := new(Request)
	err = json.Unmarshal(body, req)
	if err != nil {
		logger.Errorln("err on json unmarshal of request", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	resp := compileServerCore(req)

	// track
	if SEGMENT_KEY != "" {
		informSegment(req.Language, r)
	}

	return resp
}

// core compile functionality. used by the server and locally to mimic the server
func compileServerCore(req *Request) *Response {
	var name string
	lang := req.Language
	compiler := Languages[lang]

	c := req.Script
	if c == nil || len(c) == 0 {
		return NewResponse(nil, "", fmt.Errorf("No script provided"))
	}

	// take sha2 of request object to get tmp filename
	hash := sha256.Sum256(c)
	filename := path.Join(ServerCache, compiler.Ext(hex.EncodeToString(hash[:])))
	name = filename

	maybeCached := true

	// lllc requires a file to read
	// check if filename already exists. if not, write
	if _, err := os.Stat(filename); err != nil {
		ioutil.WriteFile(filename, c, 0644)
		logger.Debugln(filename, "does not exist. Writing")
		maybeCached = false
	}

	// loop through includes, also save to drive
	for k, v := range req.Includes {
		filename := path.Join(ServerCache, compiler.Ext(k))
		if _, err := os.Stat(filename); err != nil {
			maybeCached = false
		}
		ioutil.WriteFile(filename, v, 0644)
	}

	// check cache
	if maybeCached {
		r, err := checkCache(hash[:])
		if err == nil {
			return r
		}
	}

	var resp *Response
	//compile scripts, return bytecode and error
	compiled, docs, err := CompileWrapper(name, lang)

	// cache
	cacheResult(hash[:], compiled, docs)

	resp = NewResponse(compiled, docs, err)

	return resp
}

func informSegment(lang string, r *http.Request) {
	seg := segment.New(SEGMENT_KEY)

	con := make(map[string]interface{})
	ip  := strings.Split(r.RemoteAddr, ":")[0]
	con["ip"] = ip

	prp := make(map[string]interface{})
	prp["name"] = lang
	prp["path"] = "/compile/" + lang
	prp["url"]  = "http://compilers.eris.industries/compile/" + lang

	t   := &segment.Page{
		Context:     con,
		Traits:      prp,
		AnonymousId: ip,
		// Category:    lang,
		Name:        "Compile lang: " + lang,
	}

	logger.Debugln("Sending notification to Segment.")
	seg.Page(t)
}

func commandWrapper_(prgrm string, args []string) (string, error) {
	a := fmt.Sprint(args)
	logger.Infoln(fmt.Sprintf("Running command %s %s ", prgrm, a))
	cmd := exec.Command(prgrm, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	outstr := out.String()
	// get rid of new lines at the end
	outstr = strings.TrimSpace(outstr)
	return outstr, nil
}

func commandWrapper(tokens ...string) (string, error) {
	s, err := pipes.RunStrings(tokens...)
	s = strings.TrimSpace(s)
	return s, err
}

// wrapper to cli
func CompileWrapper(filename string, lang string) ([]byte, string, error) {
	// we need to be in the same dir as the files for sake of includes
	cur, _ := os.Getwd()
	dir := path.Dir(filename)
	dir, _ = filepath.Abs(dir)
	filename = path.Base(filename)

	if _, ok := Languages[lang]; !ok {
		return nil, "", UnknownLang(lang)
	}

	os.Chdir(dir)
	defer func() {
		os.Chdir(cur)
	}()

	tokens := Languages[lang].Cmd(filename)
	hexCode, err := commandWrapper(tokens...)
	if err != nil {
		logger.Errorln("Couldn't compile!!", err)
		return nil, "", err
	}

	tokens = Languages[lang].Abi(filename)
	jsonAbi, err := commandWrapper(tokens...)
	if err != nil {
		logger.Errorln("Couldn't produce abi doc!!", err)
		// we swallow this error, but maybe we shouldnt...
	}

	b, err := hex.DecodeString(hexCode)
	if err != nil {

		return nil, "", err
	}

	return b, jsonAbi, nil
}

// Start the compile server
func StartServer(addrUnsecure, addrSecure, key, cert string) {
	martini.Env = martini.Prod
	srv := martini.New()
	srv.Use(martini.Logger())
	srv.Use(martini.Recovery())

	// Static files
	srv.Use(martini.Static("./web"))

	// Routes
	r := martini.NewRouter()
	srv.MapTo(r, (*martini.Routes)(nil))
	srv.Action(r.Handle)

	r.Post("/compile", CompileHandler)
	r.Post("/compile2", CompileHandlerJs)

	// new relic for error reporting
	if NEWRELIC_KEY != "" {
		logger.Infoln("Starting new relic.")
		gorelic.InitNewrelicAgent(NEWRELIC_KEY, NEWRELIC_APP, false)
		srv.Use(gorelic.Handler)
	}

	// Use SSL ?
	if addrSecure == "" {

		srv.RunOnAddr(addrUnsecure)

	} else {

		srv.Use(secure.Secure(secure.Options{
			SSLRedirect: true,
			SSLHost:     addrSecure,
		}))

		// HTTP
		if addrUnsecure != "" {
			go func() {
				if err := http.ListenAndServe(addrUnsecure, srv); err != nil {
					logger.Errorln("Cannot serve on http port: ", err)
					os.Exit(1)
				}
			}()
		}

		// HTTPS
		if err := http.ListenAndServeTLS(addrSecure, cert, key, srv); err != nil {
			logger.Errorln("Cannot serve on https port: ", err)
			os.Exit(1)
		}
	}
}

// Start the proxy server
// Dead simple json-rpc so we can compile code from languages other than go
func StartProxy(addr string) {
	srv := martini.Classic()
	srv.Post("/", ProxyHandler)
	srv.RunOnAddr(addr)
}
