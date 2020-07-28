package grammar

import "testing"

func TestSymbol(t *testing.T) {
	tab := newSymbolTable()
	tab.registerStartSymbol("s")
	tab.registerNonTerminalSymbol("n")
	tab.registerTerminalSymbol("t")

	tests := []struct {
		caption       string
		text          string
		isNil         bool
		isStart       bool
		isEOF         bool
		isNonTerminal bool
		isTerminal    bool
	}{
		{
			caption:       "s is the start symbol",
			text:          "s",
			isNil:         false,
			isStart:       true,
			isEOF:         false,
			isNonTerminal: true,
			isTerminal:    false,
		},
		{
			caption:       "n is a non-terminal symbol",
			text:          "n",
			isNil:         false,
			isStart:       false,
			isEOF:         false,
			isNonTerminal: true,
			isTerminal:    false,
		},
		{
			caption:       "t is a terminal symbol",
			text:          "t",
			isNil:         false,
			isStart:       false,
			isEOF:         false,
			isNonTerminal: false,
			isTerminal:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.caption, func(t *testing.T) {
			sym, ok := tab.ToSymbol(tt.text)
			if !ok {
				t.Fatalf("symbol was not found")
			}
			testSymbolProperty(t, sym, tt.isNil, tt.isStart, tt.isEOF, tt.isNonTerminal, tt.isTerminal)
			text, ok := tab.ToText(sym)
			if !ok {
				t.Fatalf("text was not found")
			}
			if text != tt.text {
				t.Fatalf("text representation of a symbol is mismatched; want: %v, got: %v", tt.text, text)
			}
		})
	}

	t.Run("SymbolEOF is the EOF symbol", func(t *testing.T) {
		testSymbolProperty(t, SymbolEOF, false, false, true, false, true)
	})

	t.Run("symbolNil is the nil symbol", func(t *testing.T) {
		testSymbolProperty(t, symbolNil, true, false, false, false, false)
	})
}

func testSymbolProperty(t *testing.T, sym Symbol, null, start, eof, nonTerminal, terminal bool) {
	t.Helper()

	if v := sym.isNil(); v != null {
		t.Fatalf("isNil property is mismatched; want: %v, got: %v", null, v)
	}
	if v := sym.isStart(); v != start {
		t.Fatalf("isStart property is mismatched; want: %v, got: %v", start, v)
	}
	if v := sym.isEOF(); v != eof {
		t.Fatalf("isEOF property is mismatched; want: %v, got: %v", eof, v)
	}
	if v := sym.isNonTerminal(); v != nonTerminal {
		t.Fatalf("isNonTerminal property is mismatched; want: %v, got: %v", nonTerminal, v)
	}
	if v := sym.isTerminal(); v != terminal {
		t.Fatalf("isTerminal property is mismatched; want: %v, got: %v", terminal, v)
	}
}
