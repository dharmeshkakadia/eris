package commands

import (
	"fmt"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/eris-ltd/epm-go/Godeps/_workspace/src/github.com/eris-ltd/thelonious/monklog"
	"os"
	"reflect"
)

var (
	GoPath = os.Getenv("GOPATH")

	// epm extensions
	PkgExt  = "pdx"
	TestExt = "pdt"

	EPMVars = "epm.vars"

	DefaultContractPath = "." //path.Join(utils.ErisLtd, "eris-std-lib")
	defaultDatabase     = ".chain"

	logger *monklog.Logger = monklog.NewLogger("EPM-CLI")
)

type Context struct {
	Arguments []string
	Strings   map[string]string
	Integers  map[string]int
	Booleans  map[string]bool

	HasSet map[string]struct{}
}

func (c *Context) String(s string) string {
	return c.Strings[s]
}

func (c *Context) Int(s string) int {
	return c.Integers[s]
}

func (c *Context) Bool(s string) bool {
	return c.Booleans[s]
}

func (c *Context) Args() []string {
	return c.Arguments
}

func (c *Context) IsSet(s string) bool {
	_, ok := c.HasSet[s]
	return ok
}

func (c *Context) Set(s string) {
	c.HasSet[s] = struct{}{}
}

func setFields(c *Context, flags []string, generic func(s string) interface{}, HasSet func(s string) bool) {
	for _, f := range flags {
		t := generic(f)
		elem := reflect.ValueOf(t).Elem()
		ty := elem.Kind()
		switch ty {
		case reflect.String:
			c.Strings[f] = elem.String()
		case reflect.Int:
			c.Integers[f] = int(elem.Int())
		case reflect.Bool:
			c.Booleans[f] = elem.Bool()
		default:
			panic(fmt.Sprintf("Unknown type! %v", ty))
		}
		if HasSet(f) {
			c.Set(f)
		}
	}
}

func TransformContext(c *cli.Context) *Context {
	c2 := &Context{
		Arguments: []string{},
		Strings:   make(map[string]string),
		Integers:  make(map[string]int),
		Booleans:  make(map[string]bool),
		HasSet:    make(map[string]struct{}),
	}
	for _, a := range c.Args() {
		c2.Arguments = append(c2.Arguments, string(a))
	}
	flags := c.FlagNames()
	setFields(c2, flags, c.Generic, c.IsSet)
	flags = c.GlobalFlagNames()
	setFields(c2, flags, c.GlobalGeneric, c.GlobalIsSet)
	return c2
}
