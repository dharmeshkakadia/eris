package epm

// This lexer is heavily inspired by Rob Pike's "Lexical Scanning in Go"

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

const (
	EOFSTRING = "EOFSTRING"
)

// when in a state, do an action, which brings us to another state
// so a state is really a state+action ie. stateFunc that returns the next
// stateFunc. Done when it returns nil
type lexStateFunc func(*lexer) lexStateFunc

// the lexer object
type lexer struct {
	input  string // input string to lex
	length int    // length of the input string
	pos    int    // current pos
	start  int    // start of current token

	line        int // current line number
	lastNewLine int // pos of last new line

	tokens chan token // channel to emit tokens over

	temp string // a place to hold eg. commands

	err error // if it erred
}

// a token
type token struct {
	typ tokenType
	val string

	loc location
}

// location for error reporting
type location struct {
	line int
	col  int
}

// Lex the input, returning the lexer
// Tokens can be fetched off the channel
func Lex(input string) *lexer {
	l := &lexer{
		input:  input,
		length: len(input),
		pos:    0,
		tokens: make(chan token, 2),
	}
	go l.run()
	return l
}

func (l *lexer) Error(s string) lexStateFunc {
	return func(l *lexer) lexStateFunc {
		// TODO: print location data too
		l.err = errors.New(s)
		log.Println(l.err)
		return nil
	}
}

// Return the tokens channel
func (l *lexer) Chan() chan token {
	return l.tokens
}

// Run the lexer
// This is the most beautiful function in the world
func (l *lexer) run() {
	for state := lexStateStart; state != nil; state = state(l) {
		// :D
	}
	l.emit(tokenEOFTy)
	close(l.tokens)
}

// Return next character in the string
// To hell with utf8 :p
func (l *lexer) next() string {
	if l.pos >= l.length {
		return EOFSTRING
	}
	b := l.input[l.pos : l.pos+1]
	l.pos += 1
	return b
}

// backup a step
func (l *lexer) backup() {
	l.pos -= 1
}

// peek ahead a character without consuming
func (l *lexer) peek() string {
	s := l.next()
	l.backup()
	return s
}

// consume a token and push out on the channel
func (l *lexer) emit(ty tokenType) {
	l.tokens <- token{
		typ: ty,
		val: l.input[l.start:l.pos],
		loc: location{
			line: l.line,
			col:  l.pos - l.lastNewLine,
		},
	}
	l.start = l.pos
}

func (l *lexer) accept(options string) bool {
	if strings.Contains(options, l.next()) {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) acceptRun(options string) bool {
	i := 0
	s := l.next()
	for ; l.pos < l.length && strings.Contains(options, s); s = l.next() {
		i += 1
	}
	if strings.Contains(options, s) {
		// if we are at the end
		// the loop never runs
		return true
	} else if l.pos < l.length {
		l.backup()
	} else if s != "" {
		l.backup()
	}

	return i > 0
}

// Starting state
func lexStateStart(l *lexer) lexStateFunc {
	// check the one character tokens
	t := l.next()
	switch t {
	case "", EOFSTRING:
		return nil
	case tokenNewLine:
		return lexStateNewLine
	case tokenTab:
		l.emit(tokenTabTy)
		return lexStateStart
	case tokenPound:
		l.emit(tokenPoundTy)
		return lexStateComment
	case tokenColon:
		l.emit(tokenColonTy)
		return lexStateStart
	case tokenQuote:
		l.emit(tokenQuoteTy)
		return lexStateQuote
	case tokenBling:
		l.emit(tokenBlingTy)
		return lexStateString
	case tokenLeftBrace:
		l.emit(tokenLeftBraceTy)
		return lexStateStart
	case tokenRightBrace:
		l.emit(tokenRightBraceTy)
		return lexStateStart
	case tokenUnderscore:
		l.emit(tokenUnderscoreTy)
		return lexStateStart
	}
	l.backup()

	remains := l.input[l.pos:]

	// check for tabs (four spaces)
	if strings.HasPrefix(remains, tokenFourSpaces) {
		// if its more than four spaces, ignore it all
		if strings.HasPrefix(remains, tokenFourSpaces+" ") {
			return lexStateSpace
		}
		return lexStateFourSpaces
	}

	// skip spaces
	if isSpace(l.peek()) {
		return lexStateSpace
	}

	// check for left braces
	if strings.HasPrefix(remains, tokenLeftBraces) {
		return lexStateLeftBraces
	}

	// check for right braces
	if strings.HasPrefix(remains, tokenRightBraces) {
		return lexStateRightBraces
	}

	// check for left diff
	if strings.HasPrefix(remains, tokenLeftDiff) {
		return lexStateLeftDiff
	}

	// check for right diff
	if strings.HasPrefix(remains, tokenRightDiff) {
		return lexStateRightDiff
	}

	// check for arrow
	if strings.HasPrefix(remains, tokenArrow) {
		return lexStateArrow
	}

	// check for command
	for t, _ := range tokenCmds {
		if strings.HasPrefix(remains, t+":") {
			l.temp = t
			return lexStateCmd
		}
	}

	return lexStateExpressions

	return nil
}

func isSpace(s string) bool {
	return s == " " || s == "\t"
}

func lexStateExpressions(l *lexer) lexStateFunc {
	s := l.next()

	// check for number
	if strings.Contains(tokenNumbers, s) {
		l.backup()
		return lexStateNumber
	}

	// check for ops
	if strings.Contains(tokenOps, s) {
		l.emit(tokenOpTy)
		return lexStateStart
	}

	// check for chars
	if strings.Contains(tokenChars, s) {
		l.backup()
		return lexStateString
	}

	return l.Error(fmt.Sprintf("Invalid char: %s", s))
}

func lexStateNewLine(l *lexer) lexStateFunc {
	//for s := tokenNewLine; s == tokenNewLine; s = l.next() {
	//}
	//l.backup()
	l.emit(tokenNewLineTy)
	l.line += 1
	l.lastNewLine = l.pos
	return lexStateStart
}

// Scan past spaces
func lexStateSpace(l *lexer) lexStateFunc {
	for s := l.next(); isSpace(s); s = l.next() {
	}
	l.backup()
	l.start = l.pos
	return lexStateStart
}

// At an opening quotes, parse until the closing quote
func lexStateQuote(l *lexer) lexStateFunc {
	for s := ""; s != tokenQuote; s = l.next() {
		if s == EOFSTRING {
			return l.Error("Missing ending quote!")
		}
	}
	l.backup()
	l.emit(tokenStringTy)
	l.next()
	l.emit(tokenQuoteTy)
	return lexStateStart

}

// At a command
func lexStateCmd(l *lexer) lexStateFunc {
	l.pos += len(l.temp)
	l.emit(tokenCmdTy)
	return lexStateStart
}

// In a comment. Scan to new line
func lexStateComment(l *lexer) lexStateFunc {
	for r := ""; r != tokenNewLine && r != EOFSTRING; r = l.next() {
	}
	l.backup()
	l.emit(tokenStringTy)
	return lexStateStart
}

// At set of four spaces (alternative to a tab)
func lexStateFourSpaces(l *lexer) lexStateFunc {
	l.pos += len(tokenFourSpaces)
	l.emit(tokenTabTy)
	return lexStateStart
}

// At an arrow (=>)
func lexStateArrow(l *lexer) lexStateFunc {
	l.pos += len(tokenArrow)
	l.emit(tokenArrowTy)
	return lexStateStart
}

// On {{
func lexStateLeftBraces(l *lexer) lexStateFunc {
	l.pos += len(tokenLeftBraces)
	l.emit(tokenLeftBracesTy)
	return lexStateStart
}

// On }}
func lexStateRightBraces(l *lexer) lexStateFunc {
	l.pos += len(tokenRightBraces)
	l.emit(tokenRightBracesTy)
	return lexStateStart
}

// On !{
func lexStateLeftDiff(l *lexer) lexStateFunc {
	l.pos += len(tokenLeftDiff)
	l.emit(tokenLeftDiffTy)
	return lexStateStart
}

// On !}
func lexStateRightDiff(l *lexer) lexStateFunc {
	l.pos += len(tokenRightDiff)
	l.emit(tokenRightDiffTy)
	return lexStateStart
}

// a number (decimal or hex)
func lexStateNumber(l *lexer) lexStateFunc {
	if l.accept("0") && l.accept("xX") {
		l.acceptRun(tokenHex)
	} else {
		l.acceptRun(tokenNumbers)
	}
	l.emit(tokenNumberTy)
	return lexStateStart
}

// a string
func lexStateString(l *lexer) lexStateFunc {
	if !l.acceptRun(tokenChars) {
		return l.Error("Expected a string")
		l.backup()
	}
	l.emit(tokenStringTy)
	return lexStateStart
}

// error!
func lexStateErr(l *lexer) lexStateFunc {
	l.emit(tokenErrTy)
	return nil
}
