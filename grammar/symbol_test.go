package grammar

import "testing"

func TestSymbol(t *testing.T) {
	tab := newSymbolTable()
	tab.registerStartSymbol("s")
	tab.registerNonTerminalSymbol("n")
	tab.registerTerminalSymbol("t")

	tests := []struct {
		text          string
		isStart       bool
		isNonTerminal bool
		isTerminal    bool
	}{
		{
			text:          "s",
			isStart:       true,
			isNonTerminal: true,
			isTerminal:    false,
		},
		{
			text:          "n",
			isStart:       false,
			isNonTerminal: true,
			isTerminal:    false,
		},
		{
			text:          "t",
			isStart:       false,
			isNonTerminal: false,
			isTerminal:    true,
		},
	}
	for _, tt := range tests {
		sym, ok := tab.ToSymbol(tt.text)
		if !ok {
			t.Fatalf("\"%s\" was not found", tt.text)
		}
		if v := sym.isStart(); v != tt.isStart {
			t.Fatalf("isStart property of \"%s\" is mismatched; want: %v, got: %v", tt.text, tt.isStart, v)
		}
		if v := sym.isNonTerminal(); v != tt.isNonTerminal {
			t.Fatalf("isNonTerminal property of \"%s\" is mismatched; want: %v, got: %v", tt.text, tt.isNonTerminal, v)
		}
		if v := sym.isTerminal(); v != tt.isTerminal {
			t.Fatalf("isTerminal property of \"%s\" is mismatched; want: %v, got: %v", tt.text, tt.isTerminal, v)
		}
	}
}
