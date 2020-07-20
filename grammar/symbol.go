package grammar

import (
	"fmt"
)

type symbolKind string

const (
	symbolKindNonTerminal = symbolKind("non-terminal")
	symbolKindTerminal    = symbolKind("terminal")
)

func (t symbolKind) String() string {
	return string(t)
}

type Symbol uint16

func (s Symbol) String() string {
	kind, isStart, base := s.describe()
	var prefix string
	switch {
	case isStart:
		prefix = "s"
	case kind == symbolKindNonTerminal:
		prefix = "n"
	case kind == symbolKindTerminal:
		prefix = "t"
	default:
		prefix = "?"
	}
	return fmt.Sprintf("%v%v", prefix, base)
}

const (
	symbolNil = Symbol(0)

	symbolBaseMin = uint16(1)
	symbolBaseMax = uint16(0xffff) >> 2
)

func newSymbol(kind symbolKind, isStart bool, base uint16) (Symbol, error) {
	if base > symbolBaseMax {
		return symbolNil, fmt.Errorf("a base of a symbol exceeds the limit; limit: %v, passed: %v", symbolBaseMax, base)
	}

	var kindMask uint16 = 0x0000
	if kind == symbolKindTerminal {
		kindMask = 0x8000
	}
	var startMask uint16 = 0x0000
	if isStart {
		startMask = 0x4000
	}
	return Symbol(kindMask | startMask | base), nil
}

func (s Symbol) Byte() []byte {
	if s.isNil() {
		return []byte{0, 0}
	}
	return []byte{byte(uint16(s) >> 8), byte(uint16(s) & 0x00ff)}
}

func (s Symbol) isNil() bool {
	_, _, base := s.describe()
	return base == 0
}

func (s Symbol) isStart() bool {
	if s.isNil() {
		return false
	}
	_, isStart, _ := s.describe()
	return isStart
}

func (s Symbol) isNonTerminal() bool {
	if s.isNil() {
		return false
	}
	kind, _, _ := s.describe()
	if kind == symbolKindNonTerminal {
		return true
	}
	return false
}

func (s Symbol) isTerminal() bool {
	if s.isNil() {
		return false
	}
	return !s.isNonTerminal()
}

func (s Symbol) describe() (symbolKind, bool, uint16) {
	kind := symbolKindNonTerminal
	if uint16(s)&0x8000 > 0 {
		kind = symbolKindTerminal
	}
	isStart := false
	if uint16(s)&0x4000 > 0 {
		isStart = true
	}
	base := uint16(s) & 0x3fff
	return kind, isStart, base
}

type SymbolTable struct {
	text2Sym map[string]Symbol
	sym2Text map[Symbol]string
	nsymBase uint16
	tsymBase uint16
}

func newSymbolTable() *SymbolTable {
	return &SymbolTable{
		text2Sym: map[string]Symbol{},
		sym2Text: map[Symbol]string{},
		nsymBase: symbolBaseMin,
		tsymBase: symbolBaseMin,
	}
}

func (t *SymbolTable) registerStartSymbol(text string) (Symbol, error) {
	if sym, ok := t.text2Sym[text]; ok {
		return sym, nil
	}
	sym, err := newSymbol(symbolKindNonTerminal, true, t.nsymBase)
	if err != nil {
		return symbolNil, err
	}
	t.nsymBase++
	t.text2Sym[text] = sym
	t.sym2Text[sym] = text
	return sym, nil
}

func (t *SymbolTable) registerNonTerminalSymbol(text string) (Symbol, error) {
	if sym, ok := t.text2Sym[text]; ok {
		return sym, nil
	}
	sym, err := newSymbol(symbolKindNonTerminal, false, t.nsymBase)
	if err != nil {
		return symbolNil, err
	}
	t.nsymBase++
	t.text2Sym[text] = sym
	t.sym2Text[sym] = text
	return sym, nil
}

func (t *SymbolTable) registerTerminalSymbol(text string) (Symbol, error) {
	if sym, ok := t.text2Sym[text]; ok {
		return sym, nil
	}
	sym, err := newSymbol(symbolKindTerminal, false, t.tsymBase)
	if err != nil {
		return symbolNil, err
	}
	t.tsymBase++
	t.text2Sym[text] = sym
	t.sym2Text[sym] = text
	return sym, nil
}

func (t *SymbolTable) ToSymbol(text string) (Symbol, bool) {
	if sym, ok := t.text2Sym[text]; ok {
		return sym, true
	}
	return symbolNil, false
}

func (t *SymbolTable) ToText(sym Symbol) (string, bool) {
	if text, ok := t.sym2Text[sym]; ok {
		return text, true
	}
	return "", false
}
