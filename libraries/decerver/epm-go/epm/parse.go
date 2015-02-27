package epm

import (
	"encoding/hex"
	"fmt"
	"github.com/eris-ltd/epm-go/utils"
	"log"
	"math/big"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

// make sure command is valid
func checkCommand(cmd string) bool {
	r := false
	for _, c := range CMDS {
		if c == cmd {
			r = true
		}
	}
	return r
}

// peel off the next command and its args
func peelCmd(lines *[]string, startLine int) (*Job, error) {
	job := Job{"", []string{}}
	for line, t := range *lines {
		// ignore comments and blank lines
		tt := strings.TrimSpace(t)
		if len(tt) == 0 || tt[0:1] == "#" {
			continue
		}
		// ignore comments at end of the line
		t = strings.Split(t, "#")[0]

		// if no cmd yet, this should be a cmd
		if job.cmd == "" {
			// cmd syntax check
			t = strings.TrimSpace(t)
			l := len(t)
			if t[l-1:] != ":" {
				return nil, fmt.Errorf("Syntax error: missing ':' on line %d", line+startLine)
			}
			cmd := t[:l-1]
			// ensure known cmd
			if !checkCommand(cmd) {
				return nil, fmt.Errorf("Invalid command '%s' on line %d", cmd, line+startLine)
			}
			job.cmd = cmd
			continue
		}

		// if the line does not begin with white space, we are done
		if !(t[0:1] == " " || t[0:1] == "\t") {
			// peel off lines we've read
			*lines = (*lines)[line:]
			return &job, nil
		}
		t = strings.TrimSpace(t)
		// if there is a diff statement, we are done
		if t[:2] == StateDiffOpen || t[:2] == StateDiffClose {
			*lines = (*lines)[line:]
			return &job, nil
		}

		// the line is args. parse them
		// first, eliminate prefix whitespace/tabs
		args := strings.Split(t, "=>")
		// should be 'arg1 => arg2'
		// TODO: tailor lengths to the job cmd
		if len(args) > 3 {
			return nil, fmt.Errorf("Syntax error: improper argument formatting on line %d", line+startLine)
		}
		for _, a := range args {
			shaven := strings.TrimSpace(a)

			job.args = append(job.args, shaven)
		}
	}
	// only gets here if we finish all the lines
	*lines = nil
	return &job, nil
}

func parseStateDiff(lines *[]string, startLine int) string {
	for n, t := range *lines {
		tt := strings.TrimSpace(t)
		if len(tt) == 0 || tt[0:1] == "#" {
			continue
		}
		t = strings.Split(t, "#")[0]
		t = strings.TrimSpace(t)
		if len(t) == 0 {
			continue
		} else if len(t) > 2 && (t[:2] == StateDiffOpen || t[:2] == StateDiffClose) {
			// we found a match
			// shave previous lines
			*lines = (*lines)[n:]
			// see if there are other diff statements on this line
			i := strings.IndexAny(t, " \t")
			if i != -1 {
				(*lines)[0] = (*lines)[0][i:]
			} else if len(*lines) >= 1 {
				*lines = (*lines)[1:]
			}
			return t[2:]
		} else {
			*lines = (*lines)[n:]
			return ""
		}
	}
	return ""
}

// takes a simple string of nums/hex and ops
// all strings should have been removed
func tokenize(s string) []string {
	tokens := []string{}
	r_opMatch := regexp.MustCompile(`\+|\-|\*`)
	m_inds := r_opMatch.FindAllStringSubmatchIndex(s, -1)
	// if no ops, just return the string
	if len(m_inds) == 0 {
		return []string{s}
	}
	// for each theres a symbol and hex/num after it
	l := 0
	for i, matchI := range m_inds {
		i0 := matchI[0]
		i1 := matchI[1]
		ss := s[l:i0]
		l = i1
		if len(ss) != 0 {
			tokens = append(tokens, ss)
		}
		tokens = append(tokens, s[i0:i1])
		if i == len(m_inds)-1 {
			tokens = append(tokens, s[i1:])
			break
		}
		//tokens = append(tokens, s[i1:m_inds[i+1][0]])
	}
	return tokens
}

// applies any math within an arg
// splits each arg into tokens first by pulling out strings, then by parsing ops/nums/hex between strings
// finally, run through all the tokens doing the math
func DoMath(args []string) []string {
	margs := []string{} // return
	//fmt.Println("domath:", args)
	r_stringMatch := regexp.MustCompile(`\"(.*?)\"`) //"

	for _, a := range args {
		//fmt.Println("time to tokenize:", a)
		tokens := []string{}
		// find all strings (between "")
		strMatches := r_stringMatch.FindAllStringSubmatchIndex(a, -1)
		// grab the expression before the first string
		if len(strMatches) > 0 {
			// loop through every interval between strMatches
			// tokenize, append to tokens
			l := 0
			for j, matchI := range strMatches {
				i0 := matchI[0]
				i1 := matchI[1]
				// get everything before this token
				s := a[l:i0]
				l = i1
				// if s is empty, add the string to tokens, move on
				if len(s) == 0 {
					tokens = append(tokens, a[i0+1:i1-1])
				} else {
					t := tokenize(s)
					tokens = append(tokens, t...)
					tokens = append(tokens, a[i0+1:i1-1])
				}
				// if we're on the last one, get anything that comes after
				if j == len(strMatches)-1 {
					s := a[l:]
					if len(s) > 0 {
						t := tokenize(s)
						tokens = append(tokens, t...)
					}
				}
			}
		} else {
			// just tokenize the args
			tokens = tokenize(a)
		}
		//fmt.Println("tokens:", tokens)

		// now run through the tokens doin the math
		// initialize the first value
		tokenBigBytes, err := hex.DecodeString(utils.StripHex(utils.Coerce2Hex(tokens[0])))
		if err != nil {
			log.Fatal(err)
		}
		result := new(big.Int).SetBytes(tokenBigBytes)
		// start with the second token, and go up in twos (should be odd num of tokens)
		for j := 0; j < (len(tokens)-1)/2; j++ {
			op := tokens[2*j+1]
			n := tokens[2*j+2]
			nBigBytes, err := hex.DecodeString(utils.StripHex(utils.Coerce2Hex(n)))
			if err != nil {
				log.Fatal(err)
			}
			tokenBigInt := new(big.Int).SetBytes(nBigBytes)
			switch op {
			case "+":
				result.Add(result, tokenBigInt)
			case "-":
				result.Sub(result, tokenBigInt)
			case "*":
				result.Mul(result, tokenBigInt)
			}
		}
		// TODO: deal with 32-byte overflow
		resultHex := "0x" + hex.EncodeToString(result.Bytes())
		//fmt.Println("resultHex:", resultHex)
		margs = append(margs, resultHex)
	}
	return margs
}

// split line and trim space
func parseLine(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimRight(line, ";")

	args := strings.Split(line, ";")
	for i, a := range args {
		args[i] = strings.TrimSpace(a)
	}
	return args
}

func CopyContractPath() error {
	// copy the current dir into scratch/epm. Necessary for finding include files after a modify. :sigh:
	root := path.Base(ContractPath)
	p := path.Join(EpmDir, root)
	// TODO: should we delete and copy even if it does exist?
	// we might miss changed otherwise
	if _, err := os.Stat(p); err != nil {
		cmd := exec.Command("cp", "-r", ContractPath, p)
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("error copying working dir into tmp: %s", err.Error())
		}
	}
	return nil
}
