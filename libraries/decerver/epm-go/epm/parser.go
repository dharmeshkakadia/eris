package epm

import (
	"fmt"
	"log"
)

type parseStateFunc func(p *parser) parseStateFunc

type parser struct {
	l    *lexer
	last token
	jobs []Job // jobs to execute

	peekCount int // 1 if we've peeked

	inJob bool // are we in a job

	arg  []*tree // current arg
	argI int     // index of current arg

	tree  *tree // top of current tree
	treeP *tree // a pointer into current tree
	job   *Job  // current job
	jobI  int   // job counter

	diffsched map[int][]string
}

type Job struct {
	cmd  string
	args [][]*tree
}

type tree struct {
	token    token // this must be a vm op or a value
	parent   *tree
	children []*tree

	identifier bool // is the token a variable reference
}

func Parse(input string) *parser {
	l := Lex(input)
	p := &parser{
		l:         l,
		jobs:      []Job{},
		tree:      new(tree),
		job:       new(Job),
		diffsched: make(map[int][]string),
	}
	return p
}

func ParseArgs(cmd string, args string) *Job {
	args += "\n"
	p := Parse(args)
	parseStateArg(p)
	return NewJob(cmd, p.arg)
}

func (p *parser) next() token {
	if p.peekCount == 1 {
		p.peekCount = 0
		return p.last

	}
	p.last = <-p.l.Chan()
	return p.last
}

func (p *parser) peek() token {
	if p.peekCount == 1 {
		return p.last
	}
	p.next()
	p.peekCount = 1
	return p.last
}

func (p *parser) backup() {
	p.peekCount = 1
}

func (p *parser) run() error {
	for state := parseStateStart; state != nil; state = state(p) {
	}
	if p.last.typ == tokenErrTy {
		// return  err
	}
	return p.l.err
}

// return a parseStateFunc that prints the error and triggers exit (returns nil)
// closures++
func (p *parser) Error(s string) parseStateFunc {
	return func(pp *parser) parseStateFunc {
		// TODO: print location too
		log.Println("Error:", s)
		return nil
	}

}

func parseStateStart(p *parser) parseStateFunc {
	t := p.next()
	// scan past spaces, new lines, and comments
	switch t.typ {
	case tokenErrTy:
		return nil
	case tokenNewLineTy, tokenTabTy, tokenSpaceTy:
		return parseStateStart
	case tokenPoundTy:
		return parseStateComment
	case tokenLeftDiffTy:
		t = p.next()
		if t.typ != tokenStringTy {
			return p.Error("Diff braces must be followed by a string")
		}
		pj := p.jobI
		if _, ok := p.diffsched[pj]; !ok {
			p.diffsched[pj] = []string{}
		}
		p.diffsched[pj] = append(p.diffsched[pj], t.val)
		return parseStateStart
	case tokenCmdTy:
		cmd := t.val
		t = p.next()
		if t.typ != tokenColonTy {
			return p.Error("Commands must be followed by a colon")
		}
		j := &Job{
			cmd:  cmd,
			args: [][]*tree{},
		}
		p.job = j
		p.argI = 0
		p.jobI += 1
		return parseStateCommand
	}

	return p.Error(fmt.Sprintf("Unknown expression while looking for command: %s", t.val))
}

func parseStateComment(p *parser) parseStateFunc {
	p.next()
	if p.inJob {
		return parseStateCommand
	} else {
		return parseStateStart
	}
}

func parseStateCommand(p *parser) parseStateFunc {
	p.inJob = true
	t := p.next()
	switch t.typ {
	case tokenErrTy, tokenEOFTy:
		p.jobs = append(p.jobs, *p.job)
		return nil
	case tokenPoundTy:
		return parseStateComment
	case tokenNewLineTy:
		return parseStateCommand
	case tokenTabTy, tokenArrowTy:
		return parseStateArg
	case tokenCmdTy:
		// and we're done. onto the next command//
		p.jobs = append(p.jobs, *p.job)
		p.backup()
		return parseStateStart
	case tokenLeftDiffTy, tokenRightDiffTy:
		t = p.next()
		if t.typ != tokenStringTy {
			return p.Error("Diff braces must be followed by a string")
		}
		pj := p.jobI
		if _, ok := p.diffsched[pj]; !ok {
			p.diffsched[pj] = []string{}
		}
		p.diffsched[pj] = append(p.diffsched[pj], t.val)
		return parseStateCommand
	}

	return p.Error("Command args must be indented")
}

// An argument is a list of trees
// Most will be length one and depth 0 (eg. a string, number, variable)
// Others will be list of string/number/var/expression
func parseStateArg(p *parser) parseStateFunc {
	p.arg = []*tree{}
	var t = p.next()

	// TODO: switch what kind of parsing we do here based on arg number of the current job
	/* Args
	Deploy
		- string (no quotes)
		- set var
	Modify Deploy
		- string (no quotes)
		- set var
		- string (no quotes)
		- string (no quotes), use var
	Transact
		- use var
		- list of strings/num/var/expr
	Query
		- use var
		- string/var/num/exp
		- set var
	*/

	// a single arg may have multiple elements, and is terminated by => or \n or #comment
	for ; t.typ != tokenArrowTy && t.typ != tokenNewLineTy; t = p.next() {
		switch t.typ {
		case tokenNumberTy:
			// numbers are easy
			tr := &tree{token: t}
			p.arg = append(p.arg, tr)
		case tokenQuoteTy:
			// catch a quote delineated string
			t2 := p.next()
			if t2.typ != tokenStringTy {
				return p.Error(fmt.Sprintf("Invalid token following quote: %s", t2.val))
			}
			q := p.next()
			if q.typ != tokenQuoteTy {
				return p.Error(fmt.Sprintf("Missing ending quote"))
			}

			tr := &tree{token: t2}
			p.arg = append(p.arg, tr)
		case tokenStringTy:
			// new variable (string without quotes)
			tr := &tree{token: t}
			p.arg = append(p.arg, tr)
		case tokenBlingTy:
			// XXX: not in use
			// known variable
			v := p.next()
			if v.typ != tokenStringTy {
				return p.Error(fmt.Sprintf("Invalid variable name: %s", v.val))
			}
			// setting identifier means epm will
			// look it up in symbols table
			tr := &tree{
				token:      v,
				identifier: true,
			}
			p.arg = append(p.arg, tr)
		case tokenLeftBracesTy:
			v := p.next()
			if v.typ != tokenStringTy {
				return p.Error(fmt.Sprintf("Invalid variable name: %s", v.val))
			}
			cl := p.next()
			if cl.typ != tokenRightBracesTy {
				return p.Error("Must close left braces")
			}
			// setting identifier means epm will
			// look it up in symbols table
			// but if its in position to be a setter, resolve arg will ignore it
			tr := &tree{
				token:      v,
				identifier: true,
			}
			p.arg = append(p.arg, tr)
		case tokenLeftBraceTy:
			// we're entering an expression
			tr := new(tree)
			if err := p.parseExpression(tr); err != nil {
				return p.Error(err.Error())
			}
			p.arg = append(p.arg, tr)
		case tokenPoundTy:
			// consume the comment
			p.next()
		case tokenUnderscoreTy:
			p.arg = append(p.arg, &tree{token: t})
		case tokenEOFTy:
			p.job.args = append(p.job.args, p.arg)
			p.argI += 1
			p.backup()
			return parseStateCommand
		}
	}

	// add the arg to the job
	p.job.args = append(p.job.args, p.arg)

	if t.typ == tokenArrowTy {
		p.backup()
	}
	p.argI += 1
	return parseStateCommand
}

func PrintTree(tr *tree) {
	printTree(tr, "")
}

func printTree(tr *tree, prefix string) {
	fmt.Println(prefix + tr.token.val)
	for _, trc := range tr.children {
		printTree(trc, prefix+"\t")
	}
}

// called after a left brace token
func (p *parser) parseExpression(tr *tree) error {
	t := p.next()
	// this is the op
	tr.token = t
	// grab the args
	for t = p.next(); t.typ != tokenRightBraceTy; t = p.next() {
		switch t.typ {
		case tokenErrTy:
			return fmt.Errorf(t.val)
		case tokenLeftBraceTy:
			tr2 := new(tree)
			if err := p.parseExpression(tr2); err != nil {
				return err
			}
			tr.children = append(tr.children, tr2)
		case tokenStringTy, tokenNumberTy:
			tr2 := &tree{token: t}
			tr.children = append(tr.children, tr2)
		case tokenBlingTy:
			t = p.next()
			if t.typ != tokenStringTy {
				return fmt.Errorf("Invalid variable name: %s", t.val)
			}
			tr2 := &tree{token: t, identifier: true}
			tr.children = append(tr.children, tr2)
		case tokenLeftBracesTy:
			v := p.next()
			if v.typ != tokenStringTy {
				return fmt.Errorf("Invalid variable name: %s", v.val)
			}
			cl := p.next()
			if cl.typ != tokenRightBracesTy {
				return fmt.Errorf("Must close left braces")
			}
			tr2 := &tree{token: v, identifier: true}
			tr.children = append(tr.children, tr2)
		default:
		}
	}
	return nil
}

func parseStateBrace(p *parser) parseStateFunc {
	return nil
}

func parseStateString(p *parser) parseStateFunc {
	s := ""
	for t := p.next(); t.typ == tokenStringTy; t = p.peek() {
		s += t.val + " "
		p.next() // consume the lookahead
	}
	//job := len(p.jobs) - 1
	//p.jobs.args = append(p.jobs.args, s)
	t := p.peek()
	switch t.typ {
	case tokenArrowTy:
		p.next() // consume the arrow
		return parseStateCommand
	case tokenStringTy:
		return parseStateCommand
	case tokenNewLineTy:
		return parseStateCommand
	}

	return nil
}
