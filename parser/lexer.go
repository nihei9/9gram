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
	tokenKindColon      = tokenKind(":")
	tokenKindVBar       = tokenKind("|")
	tokenKindSemicolon  = tokenKind(";")
	tokenKindOptional   = tokenKind("?")
	tokenKindZeorOrMore = tokenKind("*")
	tokenKindOneOrMore  = tokenKind("+")
	tokenKindID         = tokenKind("id")
	tokenKindPattern    = tokenKind("pattern")
	tokenKindComment    = tokenKind("comment")
	tokenKindEOF        = tokenKind("eof")
	tokenKindUnknown    = tokenKind("unknown")
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

func newPatternToken(pos Position, text string) *token {
	return &token{
		kind: tokenKindPattern,
		pos:  pos,
		text: text,
	}
}

func newCommentToken(pos Position, text string) *token {
	return &token{
		kind: tokenKindComment,
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
	prevChar    rune
	prevCharPos Position
	reachedEOF  bool
}

func newLexer(src io.Reader) *lexer {
	return &lexer{
		src:         bufio.NewReader(src),
		pos:         newPosition(),
		lastChar:    nullChar,
		lastCharPos: newPosition(),
		prevChar:    nullChar,
		prevCharPos: newPosition(),
		reachedEOF:  false,
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
	case c == '?':
		return newSymbolToken(pos, tokenKindOptional), nil
	case c == '*':
		return newSymbolToken(pos, tokenKindZeorOrMore), nil
	case c == '+':
		return newSymbolToken(pos, tokenKindOneOrMore), nil
	case isIDChar(c):
		text, err := l.readID()
		if err != nil {
			return nil, err
		}
		return newIDToken(pos, text), nil
	case c == '"':
		text, err := l.readPattern()
		if err != nil {
			return nil, err
		}
		return newPatternToken(pos, text), nil
	case c == '/':
		c, _, err := l.read()
		if err != nil {
			return nil, err
		}
		if c != '/' {
			l.restore()
			break
		}
		text, err := l.readComment()
		if err != nil {
			return nil, err
		}
		return newCommentToken(pos, text), nil
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

func (l *lexer) readPattern() (string, error) {
	var b strings.Builder
	for {
		c, terminated, eof, err := l.readEscapedChar()
		if err != nil {
			return "", err
		}
		if eof {
			return "", fmt.Errorf("unclosed pattern string")
		}
		if terminated {
			break
		}

		fmt.Fprint(&b, string(c))
	}
	if b.Len() <= 0 {
		return "", fmt.Errorf("empty pattern string")
	}

	return b.String(), nil
}

func (l *lexer) readEscapedChar() (rune, bool, bool, error) {
	c, eof, err := l.read()
	if err != nil {
		return nullChar, false, false, err
	}
	if c == '\\' {
		ec, _, err := l.read()
		if err != nil {
			return nullChar, false, false, err
		}
		if ec == '"' || ec == '\\' {
			return ec, false, false, nil
		}
		return nullChar, false, false, fmt.Errorf("unsupported escape sequence: \\%s", string(ec))
	}
	if c == '"' {
		return nullChar, true, false, nil
	}
	return c, false, eof, nil
}

func (l *lexer) readComment() (string, error) {
	var b strings.Builder
	for {
		c, eof, err := l.read()
		if err != nil {
			return "", err
		}
		if c == '\n' || c == '\r' || eof {
			break
		}
		fmt.Fprint(&b, string(c))
	}

	return b.String(), nil
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
	return c == ':' || c == '|' || c == ';' || c == '?' || c == '*' || c == '+' || isIDHeadChar(c) || c == '"' || c == '/' || isWhitespace(c)
}

func (l *lexer) read() (rune, bool, error) {
	c, _, err := l.src.ReadRune()
	if err != nil {
		if err == io.EOF {
			l.prevChar = l.lastChar
			l.prevCharPos = l.lastCharPos
			l.lastChar = nullChar
			l.lastCharPos = l.pos
			l.reachedEOF = true
			return nullChar, true, nil
		}
		return nullChar, false, err
	}
	l.prevChar = l.lastChar
	l.prevCharPos = l.lastCharPos
	l.lastChar = c
	l.lastCharPos = l.pos
	l.pos.increment(c)
	return c, false, nil
}

func (l *lexer) restore() error {
	if l.reachedEOF {
		l.pos = l.lastCharPos
		l.lastChar = l.prevChar
		l.lastCharPos = l.prevCharPos
		l.prevChar = nullChar
		l.reachedEOF = false
		return l.src.UnreadRune()
	}
	if l.lastChar == nullChar {
		return fmt.Errorf("since the previous character is null, the lexer failed to call the restore")
	}
	l.pos = l.lastCharPos
	l.lastChar = l.prevChar
	l.lastCharPos = l.prevCharPos
	l.prevChar = nullChar
	return l.src.UnreadRune()
}
