package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/eris-ltd/thelonious/monklog"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"reflect"
	"strconv"
	"strings"
)

func resolveDecerverRoot() string {
	DecerverEnv := os.Getenv("DECERVER")
	if DecerverEnv == "" {
		return path.Join(usr.HomeDir, ".decerver")
	}
	return DecerverEnv
}

var (
	GoPath  = os.Getenv("GOPATH")
	ErisLtd = path.Join(GoPath, "src", "github.com", "eris-ltd")

	usr, _      = user.Current()        // error?!
	Decerver    = resolveDecerverRoot() //path.Join(usr.HomeDir, ".decerver")
	Dapps       = path.Join(Decerver, "dapps")
	Blockchains = path.Join(Decerver, "blockchains")
	Filesystems = path.Join(Decerver, "filesystems")
	Languages   = path.Join(Decerver, "languages")
	Logs        = path.Join(Decerver, "logs")
	Modules     = path.Join(Decerver, "modules")
	Scratch     = path.Join(Decerver, "scratch")
	HEAD        = path.Join(Blockchains, "HEAD")
	Refs        = path.Join(Blockchains, "refs")
	Epm         = path.Join(Scratch, "epm")
	Lllc        = path.Join(Scratch, "lllc")
	Keys        = path.Join(Decerver, "keys") // temporary solution to an age old problem
)

var MajorDirs = []string{
	Decerver, Dapps, Blockchains, Filesystems, Languages, Logs, Modules, Scratch, Refs, Epm, Lllc, Keys,
}

func exit(err error) {
	status := 0
	if err != nil {
		fmt.Println(err)
		status = 1
	}
	os.Exit(status)
}

func AbsolutePath(Datadir string, filename string) string {
	if path.IsAbs(filename) {
		return filename
	}
	return path.Join(Datadir, filename)
}

func Copy(src, dst string) error {
	f, err := os.Stat(src)
	if err != nil {
		return err
	}
	if f.IsDir() {
		if _, err := os.Stat(dst); err == nil {
			return fmt.Errorf("destination already exists")
		}
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

// assumes we've done our checking
func copyDir(src, dst string) error {
	fi, _ := os.Stat(src)
	if err := os.MkdirAll(dst, fi.Mode()); err != nil {
		return err
	}
	fs, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, f := range fs {
		s := path.Join(src, f.Name())
		d := path.Join(dst, f.Name())
		if f.IsDir() {
			if err := copyDir(s, d); err != nil {
				return err
			}
		} else {
			if err := copyFile(s, d); err != nil {
				return err
			}
		}
	}
	return nil
}

// common golang, really?
func copyFile(src, dst string) error {
	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}
	return nil
}

func InitDataDir(Datadir string) error {
	_, err := os.Stat(Datadir)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(Datadir, 0777)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func WriteJson(config interface{}, config_file string) error {
	b, err := json.Marshal(config)
	if err != nil {
		return err
	}
	var out bytes.Buffer
	err = json.Indent(&out, b, "", "\t")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(config_file, out.Bytes(), 0600)
	return err
}

// keeps N bytes of the conversion
func NumberToBytes(num interface{}, N int) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, num)
	if err != nil {
		// TODO: get this guy a return error?
		// logger.Errorln("NumberToBytes failed:", err)
	}
	//fmt.Println("btyes!", buf.Bytes())
	if buf.Len() > N {
		return buf.Bytes()[buf.Len()-N:]
	}
	return buf.Bytes()
}

// s can be string, hex, or int.
// returns properly formatted 32byte hex value
func Coerce2Hex(s string) string {
	//fmt.Println("coercing to hex:", s)
	// is int?
	i, err := strconv.Atoi(s)
	if err == nil {
		return "0x" + hex.EncodeToString(NumberToBytes(int32(i), i/256+1))
	}
	// is already prefixed hex?
	if len(s) > 1 && s[:2] == "0x" {
		if len(s)%2 == 0 {
			return s
		}
		return "0x0" + s[2:]
	}
	// is unprefixed hex?
	if len(s) > 32 {
		return "0x" + s
	}
	pad := strings.Repeat("\x00", (32-len(s))) + s
	ret := "0x" + hex.EncodeToString([]byte(pad))
	//fmt.Println("result:", ret)
	return ret
}

func AddHex(s string) string {
	if len(s) < 2 {
		return "0x" + s
	}

	if s[:2] != "0x" {
		return "0x" + s
	}

	return s
}

func StripHex(s string) string {
	if len(s) > 1 {
		if s[:2] == "0x" {
			return s[2:]
		}
	}
	return s
}

func Usr() string {
	u, _ := user.Current()
	return u.HomeDir
}

func InitLogging(Datadir string, LogFile string, LogLevel int, DebugFile string) {
	if !monklog.IsNil() {
		return
	}
	var writer io.Writer
	if LogFile == "" {
		writer = os.Stdout
	} else {
		writer = openLogFile(Datadir, LogFile)
	}
	monklog.AddLogSystem(monklog.NewStdLogSystem(writer, log.LstdFlags, monklog.LogLevel(LogLevel)))
	if DebugFile != "" {
		writer = openLogFile(Datadir, DebugFile)
		monklog.AddLogSystem(monklog.NewStdLogSystem(writer, log.LstdFlags, monklog.DebugLevel))
	}
}

func openLogFile(Datadir string, filename string) *os.File {
	path := AbsolutePath(Datadir, filename)
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("error opening log file '%s': %v", filename, err))
	}
	return file
}

// Create the default decerver tree
func InitDecerverDir() (err error) {
	for _, d := range MajorDirs {
		err := InitDataDir(d)
		if err != nil {
			return err
		}
	}
	if _, err = os.Stat(HEAD); err != nil {
		_, err = os.Create(HEAD)
	}
	return
}

func NewInvalidKindErr(kind, k reflect.Kind) error {
	return fmt.Errorf("Invalid kind. Expected %s, received %s", kind, k)
}

func FieldFromTag(v reflect.Value, field string) (string, error) {
	iv := v.Interface()
	st := reflect.TypeOf(iv)
	for i := 0; i < v.NumField(); i++ {
		tag := st.Field(i).Tag.Get("json")
		if tag == field {
			return st.Field(i).Name, nil
		}
	}
	return "", fmt.Errorf("Invalid field name")
}

// Set a field in a struct value
// Field can be field name or json tag name
// Values can be strings that can be cast to int or bool
//  only handles strings, ints, bool
func SetProperty(cv reflect.Value, field string, value interface{}) error {
	f := cv.FieldByName(field)
	if !f.IsValid() {
		name, err := FieldFromTag(cv, field)
		if err != nil {
			return err
		}
		f = cv.FieldByName(name)
	}
	kind := f.Kind()

	k := reflect.ValueOf(value).Kind()
	if k != kind && k != reflect.String {
		return NewInvalidKindErr(kind, k)
	}

	if kind == reflect.String {
		f.SetString(value.(string))
	} else if kind == reflect.Int {
		if k != kind {
			v, err := strconv.Atoi(value.(string))
			if err != nil {
				return err
			}
			f.SetInt(int64(v))
		} else {
			f.SetInt(int64(value.(int)))
		}
	} else if kind == reflect.Bool {
		if k != kind {
			v, err := strconv.ParseBool(value.(string))
			if err != nil {
				return err
			}
			f.SetBool(v)
		} else {
			f.SetBool(value.(bool))
		}
	}
	return nil
}

func ClearDir(dir string) error {
	fs, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, f := range fs {
		n := f.Name()
		if f.IsDir() {
			if err := os.RemoveAll(path.Join(dir, f.Name())); err != nil {
				return err
			}
		} else {
			if err := os.Remove(path.Join(dir, n)); err != nil {
				return err
			}
		}
	}
	return nil
}
