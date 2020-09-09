package grammar

import (
	"strings"
	"testing"

	"github.com/nihei9/9gram/parser"
)

type first struct {
	lhs     string
	num     int
	dot     int
	symbols []string
	empty   bool
}

func TestGenFirst(t *testing.T) {
	tests := []struct {
		caption string
		src     string
		first   []first
	}{
		{
			caption: "productions contain only non-empty productions",
			src:     "e: e PLUS t | t; t: t STAR f | f; f: LPAREN e RPAREN | NUMBER;",
			first: []first{
				{lhs: "e'", num: 0, dot: 0, symbols: []string{"LPAREN", "NUMBER"}},
				{lhs: "e", num: 0, dot: 0, symbols: []string{"LPAREN", "NUMBER"}},
				{lhs: "e", num: 0, dot: 1, symbols: []string{"PLUS"}},
				{lhs: "e", num: 0, dot: 2, symbols: []string{"LPAREN", "NUMBER"}},
				{lhs: "e", num: 1, dot: 0, symbols: []string{"LPAREN", "NUMBER"}},
				{lhs: "t", num: 0, dot: 0, symbols: []string{"LPAREN", "NUMBER"}},
				{lhs: "t", num: 0, dot: 1, symbols: []string{"STAR"}},
				{lhs: "t", num: 0, dot: 2, symbols: []string{"LPAREN", "NUMBER"}},
				{lhs: "t", num: 1, dot: 0, symbols: []string{"LPAREN", "NUMBER"}},
				{lhs: "f", num: 0, dot: 0, symbols: []string{"LPAREN"}},
				{lhs: "f", num: 0, dot: 1, symbols: []string{"LPAREN", "NUMBER"}},
				{lhs: "f", num: 0, dot: 2, symbols: []string{"RPAREN"}},
				{lhs: "f", num: 1, dot: 0, symbols: []string{"NUMBER"}},
			},
		},
		{
			caption: "productions contain the empty start production",
			src:     "s: ;",
			first: []first{
				{lhs: "s'", num: 0, dot: 0, symbols: []string{}, empty: true},
				{lhs: "s", num: 0, dot: 0, symbols: []string{}, empty: true},
			},
		},
		{
			caption: "productions contain a empty production",
			src:     "s: foo; foo: ;",
			first: []first{
				{lhs: "s'", num: 0, dot: 0, symbols: []string{}, empty: true},
				{lhs: "s", num: 0, dot: 0, symbols: []string{}, empty: true},
				{lhs: "foo", num: 0, dot: 0, symbols: []string{}, empty: true},
			},
		},
		{
			caption: "productions contain a non-empty start production and empty production",
			src:     "s: FOO | ;",
			first: []first{
				{lhs: "s'", num: 0, dot: 0, symbols: []string{"FOO"}, empty: true},
				{lhs: "s", num: 0, dot: 0, symbols: []string{"FOO"}},
				{lhs: "s", num: 1, dot: 0, symbols: []string{}, empty: true},
			},
		},
		{
			caption: "productions contain non-empty production and empty one",
			src:     "s: foo; foo: BAR | ;",
			first: []first{
				{lhs: "s'", num: 0, dot: 0, symbols: []string{"BAR"}, empty: true},
				{lhs: "s", num: 0, dot: 0, symbols: []string{"BAR"}, empty: true},
				{lhs: "foo", num: 0, dot: 0, symbols: []string{"BAR"}},
				{lhs: "foo", num: 1, dot: 0, symbols: []string{}, empty: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.caption, func(t *testing.T) {
			fst, gram := genActualFirst(t, tt.src)

			for _, ttFirst := range tt.first {
				lhsSym, ok := gram.SymbolTable.ToSymbol(ttFirst.lhs)
				if !ok {
					t.Fatalf("a symbol was not found; symbol: %v", ttFirst.lhs)
				}

				prod, ok := gram.ProductionSet.findByLHS(lhsSym)
				if !ok {
					t.Fatalf("a production was not found; LHS: %v (%v)", ttFirst.lhs, lhsSym)
				}

				actualFirst, err := fst.Get(prod[ttFirst.num], ttFirst.dot)
				if err != nil {
					t.Fatalf("failed to get a FIRST set; LHS: %v (%v), num: %v, dot: %v, error: %v", ttFirst.lhs, lhsSym, ttFirst.num, ttFirst.dot, err)
				}

				expectedFirst := genExpectedFirstEntry(t, ttFirst.symbols, ttFirst.empty, gram.SymbolTable)

				testFirst(t, actualFirst, expectedFirst)
			}
		})
	}
}

func genActualFirst(t *testing.T, src string) (*First, *Grammar) {
	parser, err := parser.NewParser(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	ast, err := parser.Parse()
	if err != nil {
		t.Fatal(err)
	}
	gram, err := GenGrammar(ast)
	if err != nil {
		t.Fatal(err)
	}
	fst, err := genFirst(gram.ProductionSet)
	if err != nil {
		t.Fatal(err)
	}
	if fst == nil {
		t.Fatal("genFiest returned nil without any error")
	}

	return fst, gram
}

func genExpectedFirstEntry(t *testing.T, symbols []string, empty bool, symTab *SymbolTable) *FirstEntry {
	t.Helper()

	entry := newFirstEntry()
	if empty {
		entry.addEmpty()
	}
	for _, sym := range symbols {
		symSym, ok := symTab.ToSymbol(sym)
		if !ok {
			t.Fatalf("a symbol was not found; symbol: %v", sym)
		}
		entry.add(symSym)
	}

	return entry
}

func testFirst(t *testing.T, actual, expected *FirstEntry) {
	if actual.empty != expected.empty {
		t.Errorf("empty is mismatched\nwant: %v\ngot: %v", expected.empty, actual.empty)
	}

	if len(actual.symbols) != len(expected.symbols) {
		t.Fatalf("invalid FIRST set\nwant: %+v\ngot: %+v", expected.symbols, actual.symbols)
	}

	for eSym := range expected.symbols {
		if _, ok := actual.symbols[eSym]; !ok {
			t.Fatalf("invalid FIRST set\nwant: %+v\ngot: %+v", expected.symbols, actual.symbols)
		}
	}
}
