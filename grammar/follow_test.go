package grammar

import (
	"strings"
	"testing"

	"github.com/nihei9/9gram/parser"
)

type follow struct {
	nSym    string
	symbols []string
	eof     bool
}

func TestFollowSet(t *testing.T) {
	tests := []struct {
		caption string
		src     string
		follow  []follow
	}{
		{
			caption: "productions contain only non-empty productions",
			src:     "e: e PLUS t | t; t: t STAR f | f; f: LPAREN e RPAREN | NUMBER;",
			follow: []follow{
				{nSym: "e'", symbols: []string{}, eof: true},
				{nSym: "e", symbols: []string{"PLUS", "RPAREN"}, eof: true},
				{nSym: "t", symbols: []string{"PLUS", "STAR", "RPAREN"}, eof: true},
				{nSym: "f", symbols: []string{"PLUS", "STAR", "RPAREN"}, eof: true},
			},
		},
		{
			caption: "productions contain the empty start production",
			src:     "s: ;",
			follow: []follow{
				{nSym: "s'", symbols: []string{}, eof: true},
				{nSym: "s", symbols: []string{}, eof: true},
			},
		},
		{
			caption: "productions contain a empty production",
			src:     "s: foo; foo: ;",
			follow: []follow{
				{nSym: "s'", symbols: []string{}, eof: true},
				{nSym: "s", symbols: []string{}, eof: true},
				{nSym: "foo", symbols: []string{}, eof: true},
			},
		},
		{
			caption: "productions contain a non-empty start production and empty production",
			src:     "s: FOO | ;",
			follow: []follow{
				{nSym: "s'", symbols: []string{}, eof: true},
				{nSym: "s", symbols: []string{}, eof: true},
			},
		},
		{
			caption: "productions contain non-empty production and empty one",
			src:     "s: foo; foo: BAR | ;",
			follow: []follow{
				{nSym: "s'", symbols: []string{}, eof: true},
				{nSym: "s", symbols: []string{}, eof: true},
				{nSym: "foo", symbols: []string{}, eof: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.caption, func(t *testing.T) {
			flw, gram := genActualFollow(t, tt.src)

			for _, ttFollow := range tt.follow {
				nSym, ok := gram.SymbolTable.ToSymbol(ttFollow.nSym)
				if !ok {
					t.Fatalf("a symbol was not found; symbol: %v", ttFollow.nSym)
				}

				actualFollow := flw.Get(nSym)
				if actualFollow == nil {
					t.Fatalf("failed to get a FOLLOW set; non-terminal symbol: %v (%v)", ttFollow.nSym, nSym)
				}

				expectedFollow := genExpectedFollowEntry(t, ttFollow.symbols, ttFollow.eof, gram.SymbolTable)

				testFollow(t, actualFollow, expectedFollow)
			}
		})
	}
}

func genActualFollow(t *testing.T, src string) (*Follow, *Grammar) {
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
	flw, err := genFollow(gram.ProductionSet, fst)
	if flw == nil {
		t.Fatal("genFollow returned nil without any error")
	}

	return flw, gram
}

func genExpectedFollowEntry(t *testing.T, symbols []string, eof bool, symTab *SymbolTable) *FollowEntry {
	t.Helper()

	entry := newFollowEntry()
	if eof {
		entry.addEOF()
	}
	for _, sym := range symbols {
		symID, _ := symTab.ToSymbol(sym)
		if symID.isNil() {
			t.Fatalf("a symbol was not found; symbol: %v", sym)
		}

		entry.add(symID)
	}

	return entry
}

func testFollow(t *testing.T, actual, expected *FollowEntry) {
	if actual.eof != expected.eof {
		t.Errorf("eof is mismatched\nwant: %v\ngot: %v", expected.eof, actual.eof)
	}

	if len(actual.symbols) != len(expected.symbols) {
		t.Fatalf("invalid FOLLOW set\nwant: %v\ngot: %v", expected.symbols, actual.symbols)
	}

	for eSym := range expected.symbols {
		if _, ok := actual.symbols[eSym]; !ok {
			t.Fatalf("invalid FOLLOW set\nwant: %v\ngot: %v", expected.symbols, actual.symbols)
		}
	}
}
