package parser

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
)

type tokenKind string

const (
	tokenKindColon     = tokenKind(":")
	tokenKindVBar      = tokenKind("|")
	tokenKindSemicolon = tokenKind(";")
	tokenKindID        = tokenKind("id")
	tokenKindEOF       = tokenKind("eof")
	tokenKindUnknown   = tokenKind("unknown")
)

type Position struct {
	Line   int
	Column int
}

func newPosition() Position {
	return Position{
		Line:   1,
		Column: 1,
	}
}

func (p *Position) increment(c rune) {
	if c == '\n' || c == '\r' {
		p.Line += 1
		p.Column = 1
	} else {
		p.Column += 1
	}
}

type token struct {
	kind tokenKind
	pos  Position
	text string
}

func newSymbolToken(pos Position, kind tokenKind) *token {
	return &token{
		kind: kind,
		pos:  pos,
	}
}

func newIDToken(pos Position, text string) *token {
	return &token{
		kind: tokenKindID,
		pos:  pos,
		text: text,
	}
}

func newEOFToken(pos Position) *token {
	return &token{
		kind: tokenKindEOF,
		pos:  pos,
	}
}

func newUnknownToken(pos Position, text string) *token {
	return &token{
		kind: tokenKindUnknown,
		pos:  pos,
		text: text,
	}
}

const nullChar = '\u0000'

type lexer struct {
	src         *bufio.Reader
	pos         Position
	lastChar    rune
	lastCharPos Position
}

func newLexer(src io.Reader) *lexer {
	return &lexer{
		src:         bufio.NewReader(src),
		pos:         newPosition(),
		lastChar:    nullChar,
		lastCharPos: newPosition(),
	}
}

func (l *lexer) next() (*token, error) {
	err := l.skipWhitespace()
	if err != nil {
		return nil, err
	}

	pos := l.pos
	c, eof, err := l.read()
	if err != nil {
		return nil, err
	}
	if eof {
		return newEOFToken(pos), nil
	}

	switch {
	case c == ':':
		return newSymbolToken(pos, tokenKindColon), nil
	case c == '|':
		return newSymbolToken(pos, tokenKindVBar), nil
	case c == ';':
		return newSymbolToken(pos, tokenKindSemicolon), nil
	case isIDChar(c):
		text, err := l.readID()
		if err != nil {
			return nil, err
		}
		return newIDToken(pos, text), nil
	}

	text, err := l.readUnknown()
	if err != nil {
		return nil, err
	}
	return newUnknownToken(pos, text), nil
}

func (l *lexer) skipWhitespace() error {
	for {
		c, eof, err := l.read()
		if err != nil {
			return err
		}
		if eof {
			return nil
		}
		if !isWhitespace(c) {
			err := l.restore()
			if err != nil {
				return err
			}
			break
		}
	}

	return nil
}

func isWhitespace(c rune) bool {
	return unicode.IsSpace(c)
}

func (l *lexer) readID() (string, error) {
	var b strings.Builder
	fmt.Fprint(&b, string(l.lastChar))
	for {
		c, eof, err := l.read()
		if err != nil {
			return "", err
		}
		if eof {
			break
		}
		if !isIDChar(c) {
			err := l.restore()
			if err != nil {
				return "", err
			}
			break
		}
		fmt.Fprint(&b, string(c))
	}

	return b.String(), nil
}

func isIDChar(c rune) bool {
	return isIDHeadChar(c) || c >= '0' && c <= '9'
}

func isIDHeadChar(c rune) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_'
}

func (l *lexer) readUnknown() (string, error) {
	var b strings.Builder
	fmt.Fprint(&b, string(l.lastChar))
	for {
		c, eof, err := l.read()
		if err != nil {
			return "", err
		}
		if eof {
			break
		}
		if !isUnknownChar(c) {
			err := l.restore()
			if err != nil {
				return "", err
			}
			break
		}
		fmt.Fprint(&b, string(c))
	}

	return b.String(), nil
}

func isUnknownChar(c rune) bool {
	return !isHeadChar(c)
}

func isHeadChar(c rune) bool {
	return c == ':' || c == '|' || c == ';' || isIDHeadChar(c) || isWhitespace(c)
}

func (l *lexer) read() (rune, bool, error) {
	c, _, err := l.src.ReadRune()
	if err != nil {
		if err == io.EOF {
			return nullChar, true, nil
		}
		return nullChar, false, err
	}
	l.lastChar = c
	l.lastCharPos = l.pos
	l.pos.increment(c)
	return c, false, nil
}

func (l *lexer) restore() error {
	if l.lastChar == nullChar {
		return fmt.Errorf("since the previous character is null, the lexer failed to call the restore")
	}
	l.pos = l.lastCharPos
	l.lastChar = nullChar
	return l.src.UnreadRune()
}
