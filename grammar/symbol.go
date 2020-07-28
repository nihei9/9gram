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

type SymbolNum uint16

func (n SymbolNum) Int() int {
	return int(n)
}

type Symbol uint16

func (s Symbol) String() string {
	kind, isStart, isEOF, base := s.describe()
	var prefix string
	switch {
	case isStart:
		prefix = "s"
	case isEOF:
		prefix = "e"
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
	symbolNil = Symbol(0)      // 0000 0000 0000 0000
	SymbolEOF = Symbol(0xc001) // 1100 0000 0000 0001: The EOF symbol is treated as a terminal symbol.

	terminalSymbolNumMin    = SymbolNum(2) // The number 1 is used by the EOF symbol.
	nonTerminalSymbolNumMin = SymbolNum(1)
	symbolBaseMax           = SymbolNum(0xffff) >> 2
)

func newSymbol(kind symbolKind, isStart bool, base SymbolNum) (Symbol, error) {
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
	return Symbol(kindMask | startMask | uint16(base)), nil
}

func (s Symbol) Num() SymbolNum {
	_, _, _, base := s.describe()
	return base
}

func (s Symbol) Byte() []byte {
	if s.isNil() {
		return []byte{0, 0}
	}
	return []byte{byte(uint16(s) >> 8), byte(uint16(s) & 0x00ff)}
}

func (s Symbol) isNil() bool {
	_, _, _, base := s.describe()
	return base == 0
}

func (s Symbol) isStart() bool {
	if s.isNil() {
		return false
	}
	_, isStart, _, _ := s.describe()
	return isStart
}

func (s Symbol) isEOF() bool {
	if s.isNil() {
		return false
	}
	_, _, isEOF, _ := s.describe()
	return isEOF
}

func (s Symbol) isNonTerminal() bool {
	if s.isNil() {
		return false
	}
	kind, _, _, _ := s.describe()
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

func (s Symbol) describe() (symbolKind, bool, bool, SymbolNum) {
	kind := symbolKindNonTerminal
	if s&0x8000 > 0 {
		kind = symbolKindTerminal
	}
	isStart := false
	isEOF := false
	if s&0x4000 > 0 {
		if kind == symbolKindNonTerminal {
			isStart = true
		} else {
			isEOF = true
		}
	}
	base := SymbolNum(s & 0x3fff)
	return kind, isStart, isEOF, base
}

type SymbolTable struct {
	text2Sym map[string]Symbol
	sym2Text map[Symbol]string
	nsymBase SymbolNum
	tsymBase SymbolNum
}

func newSymbolTable() *SymbolTable {
	return &SymbolTable{
		text2Sym: map[string]Symbol{},
		sym2Text: map[Symbol]string{},
		nsymBase: nonTerminalSymbolNumMin,
		tsymBase: terminalSymbolNumMin,
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

func (t *SymbolTable) getNumOfTerminalSymbols() int {
	if t.tsymBase == terminalSymbolNumMin {
		return 0
	}
	return t.tsymBase.Int()
}

func (t *SymbolTable) getNumOfNonTerminalSymbols() int {
	if t.nsymBase == nonTerminalSymbolNumMin {
		return 0
	}
	return t.nsymBase.Int()
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

func (t *SymbolTable) ToTextFromNumT(num SymbolNum) (string, error) {
	sym, err := newSymbol(symbolKindTerminal, false, num)
	if err != nil {
		return "", err
	}
	text, ok := t.ToText(sym)
	if !ok {
		return "", fmt.Errorf("text was not found; symbol: %v", sym)
	}
	return text, nil
}

func (t *SymbolTable) ToTextFromNumN(num SymbolNum) (string, error) {
	sym, err := newSymbol(symbolKindNonTerminal, false, num)
	if err != nil {
		return "", err
	}
	text, ok := t.ToText(sym)
	if !ok {
		return "", fmt.Errorf("text was not found; symbol: %v", sym)
	}
	return text, nil
}
