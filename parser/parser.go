package parser

import (
	"fmt"
	"io"
)

type ASTType string

const (
	ASTTypeStart       = ASTType("start")
	ASTTypeProduction  = ASTType("production")
	ASTTypeAlternative = ASTType("alternative")
	ASTTypeSymbol      = ASTType("symbol")
	ASTTypePattern     = ASTType("pattern")
	ASTTypeOptional    = ASTType("optional")
)

type AST struct {
	Ty       ASTType
	Children []*AST

	token *token
	prev  *AST
}

func (ast *AST) GetText() (string, bool) {
	if ast.token == nil {
		return "", false
	}
	if ast.token.kind == tokenKindID || ast.token.kind == tokenKindPattern {
		return ast.token.text, true
	}
	return "", false
}

func (ast *AST) appendChild(child *AST) {
	if ast.Children == nil {
		ast.Children = []*AST{}
	}
	ast.Children = append(ast.Children, child)
}

type SyntaxError struct {
	pos     Position
	message string
}

func newSyntaxError(pos Position, message string) *SyntaxError {
	return &SyntaxError{
		pos:     pos,
		message: message,
	}
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("syntax error: %s (%v, %v)", e.message, e.pos.Line, e.pos.Column)
}

type Parser interface {
	Parse() (*AST, error)
}

type parser struct {
	lex         *lexer
	peekedTok   *token
	lastTok     *token
	root        *AST
	currentNode *AST
}

func NewParser(src io.Reader) (Parser, error) {
	return &parser{
		lex:         newLexer(src),
		peekedTok:   nil,
		lastTok:     nil,
		root:        nil,
		currentNode: nil,
	}, nil
}

func (p *parser) Parse() (ast *AST, retErr error) {
	defer func() {
		err := recover()
		if err != nil {
			retErr = err.(error)
			return
		}
	}()

	p.parseStart()

	return p.root, nil
}

func (p *parser) parseStart() {
	p.enter(ASTTypeStart)
	defer p.leave()

	p.parseProduction()
	for {
		if p.consume(tokenKindEOF) {
			break
		}
		p.parseProduction()
	}
}

func (p *parser) parseProduction() {
	p.enter(ASTTypeProduction)
	defer p.leave()

	p.expect(tokenKindID)
	p.as(ASTTypeSymbol)
	p.expect(tokenKindColon)
	p.parseAlternative()
	for {
		if !p.consume(tokenKindVBar) {
			break
		}
		p.parseAlternative()
	}
	p.expect(tokenKindSemicolon)
}

func (p *parser) parseAlternative() {
	p.enter(ASTTypeAlternative)
	defer p.leave()

	for {
		if p.consume(tokenKindID) {
			p.as(ASTTypeSymbol)
			p.parseQualifier()
			continue
		}
		if p.consume(tokenKindPattern) {
			p.as(ASTTypePattern)
			p.parseQualifier()
			continue
		}
		break
	}
}

func (p *parser) parseQualifier() {
	if p.consume(tokenKindOptional) {
		p.as(ASTTypeOptional)
	}
}

func (p *parser) enter(ty ASTType) {
	ast := &AST{
		Ty: ty,
	}
	if p.root == nil {
		p.root = ast
	}
	if p.currentNode != nil {
		p.currentNode.appendChild(ast)
	}
	ast.prev = p.currentNode
	p.currentNode = ast
}

func (p *parser) leave() {
	if p.currentNode.prev == nil {
		return
	}
	ast := p.currentNode
	p.currentNode = ast.prev
	ast.prev = nil
}

func (p *parser) expect(expected tokenKind) {
	if !p.consume(expected) {
		tok := p.peekedTok
		errMsg := fmt.Sprintf("unexpected token; expected: %v, actual: %v", expected, tok.kind)
		raiseSyntaxError(tok.pos, errMsg)
	}
}

func (p *parser) consume(expected tokenKind) bool {
	var tok *token
	var err error
	if p.peekedTok != nil {
		tok = p.peekedTok
		p.peekedTok = nil
	} else {
		tok, err = p.lex.next()
		if err != nil {
			panic(err)
		}
	}
	p.lastTok = tok
	if tok.kind == tokenKindUnknown {
		errMsg := fmt.Sprintf("unknown token: \"%s\"", tok.text)
		raiseSyntaxError(tok.pos, errMsg)
	}
	if tok.kind == expected {
		return true
	}
	p.peekedTok = tok
	p.lastTok = nil

	return false
}

func (p *parser) as(ty ASTType) {
	if p.lastTok == nil {
		return
	}
	p.currentNode.appendChild(&AST{
		Ty:    ty,
		token: p.lastTok,
	})
	p.lastTok = nil
}

func raiseSyntaxError(pos Position, message string) {
	panic(newSyntaxError(pos, message))
}
