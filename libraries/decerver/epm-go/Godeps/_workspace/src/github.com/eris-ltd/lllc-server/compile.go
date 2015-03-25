package lllcserver

import (
	"encoding/json"
	"fmt"
	"github.com/eris-ltd/epm-go/utils"
	"io/ioutil"
	"os"
	"path"
)

var DefaultUrl = "http://ps.erisindustries.com:8090/compile"

// Language configuration struct
// New language capabilities can be added to the server simply by
// providing a LangConfig
// Each element in IncludeReplaces is a pair of strings, between which is placed the filename
// CompileCmd is a list of what would be white-space separated tokens on the
// command line, with a `_` to denote the place of the filename
type LangConfig struct {
	URL             string     `json:"url"`
	Path            string     `json:"path"`
	Net             bool       `json:"net"`
	Extensions      []string   `json:"extensions"`
	IncludeRegexes  []string   `json:"regexes"`
	IncludeReplaces [][]string `json:"replaces"`
	CompileCmd      []string   `json:"cmd"`
}

// Append the language extension to the filename
func (l LangConfig) Ext(h string) string {
	return h + "." + l.Extensions[0]
}

// Fill in the filename and return the command line args
func (l LangConfig) Cmd(file string) (prgrm string, args []string) {
	prgrm = l.CompileCmd[0]
	for _, s := range l.CompileCmd[1:] {
		if s == "_" {
			args = append(args, file)
		} else {
			args = append(args, s)
		}
	}
	return
}

// Global variable mapping languages to their configs
var Languages = map[string]LangConfig{
	"lll": LangConfig{
		URL:        DefaultUrl,
		Path:       path.Join(homeDir(), "cpp-ethereum/build/lllc/lllc"),
		Net:        true,
		Extensions: []string{"lll", "def"},
		IncludeRegexes: []string{
			`\(include "(.+?)"\)`,
		},
		IncludeReplaces: [][]string{
			[]string{`(include "`, `.lll")`},
		},
		CompileCmd: []string{
			path.Join(homeDir(), "cpp-ethereum/build/lllc/lllc"),
			"_",
		},
	},

	"se": LangConfig{
		URL:        DefaultUrl,
		Path:       "/usr/local/bin/serpent",
		Net:        true,
		Extensions: []string{"se"},
		IncludeRegexes: []string{
			// because I'm not that good with regex and this
			// demonstrates how to have multiple expressions to match :)
			`create\("(.+?)"\)`,
			`create\('(.+?)'\)`,
		},
		IncludeReplaces: [][]string{
			[]string{`create("`, `.se")`},
			[]string{`create('`, `.se')`},
		},
		CompileCmd: []string{
			"/usr/local/bin/serpent",
			"compile",
			"_",
		},
	},
}

func init() {
	utils.InitDecerverDir()
	utils.InitDataDir(ClientCache)
	utils.InitDataDir(ServerCache)

	f := path.Join(utils.Languages, "config.json")
	err := checkConfig(f)
	if err != nil {
		logger.Errorln(err)
		logger.Errorln("resorting to default language settings")
		return
	}
}

func checkConfig(f string) error {
	if _, err := os.Stat(f); err != nil {
		err := utils.WriteJson(&Languages, f)
		if err != nil {
			return fmt.Errorf("Could not write default config to %s: %s", f, err.Error())
		}
	}

	b, err := ioutil.ReadFile(f)
	if err != nil {
		return err
	}

	c := new(map[string]LangConfig)
	err = json.Unmarshal(b, c)
	if err != nil {
		return err
	}

	Languages = *c
	return nil
}

// Set the languages compiler path
func SetLanguagePath(lang, path string) error {
	l, ok := Languages[lang]
	if !ok {
		return UnknownLang(lang)
	}
	l.Path = path
	Languages[lang] = l
	return nil
}

// Set the languages url
func SetLanguageURL(lang, url string) error {
	l, ok := Languages[lang]
	if !ok {
		return UnknownLang(lang)
	}
	l.URL = url
	Languages[lang] = l
	return nil
}

// Set whether the language should use the remote server or compile locally
func SetLanguageNet(lang string, net bool) error {
	l, ok := Languages[lang]
	if !ok {
		return UnknownLang(lang)
	}
	l.Net = net
	Languages[lang] = l
	return nil

}

// Main client struct to wrap a compiler interface and its configuration data
type CompileClient struct {
	config LangConfig
	lang   string
}

// Return the language name
func (c *CompileClient) Lang() string {
	return c.lang //c.Lang()
}

// Return the language's main extension
func (c *CompileClient) Ext(h string) string {
	return c.config.Ext(h)
}

// Return the regex string to match include statements
func (c *CompileClient) IncludeRegexes() []string {
	return c.config.IncludeRegexes
}

// Return the string to replace matched regex expressions
func (c *CompileClient) IncludeReplace(h string, i int) string {
	s := c.config.IncludeReplaces[i]
	return s[0] + h + s[1]
}

// Unknown language error
func UnknownLang(lang string) error {
	return fmt.Errorf("Unknown language %s", lang)
}

// Create a new compile client
func NewCompileClient(lang string) (*CompileClient, error) {
	l, ok := Languages[lang]
	if !ok {
		return nil, UnknownLang(lang)
	}
	cc := &CompileClient{
		config: l,
		lang:   lang,
	}
	return cc, nil
}
