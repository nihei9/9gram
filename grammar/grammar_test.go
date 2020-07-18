package grammar

import (
	"strings"
	"testing"

	"github.com/nihei9/9gram/parser"
)

func TestGenGrammar(t *testing.T) {
	src := `
expr: expr add term
    | term
    ;
term: term mul factor
    | factor
    ;
factor: lparen expr rparen
    | sign number
    ;
sign: minus
    |
    ;
`
	parser, err := parser.NewParser(strings.NewReader(src))
	if err != nil {
		t.Fatalf("failed to create a new parser: %v", err)
	}
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("failed to parse the test source: %v", err)
	}

	gram, err := GenGrammar(ast)
	if err != nil {
		t.Fatalf("failed to generate a grammar: %v", err)
	}
	symTab := gram.SymbolTable

	startText := "expr'"
	startSym, ok := symTab.ToSymbol(startText)
	if !ok {
		t.Fatalf("a symbol was not found; augmented start symbol: %v", startText)
	}
	if gram.AugmentedStartSymbol != startSym {
		t.Fatalf("unexpected start symbol; want: %v, got: %v", startSym, gram.AugmentedStartSymbol)
	}

	expectProds := []struct {
		lhs  string
		alts [][]string
	}{
		{
			lhs: "expr'",
			alts: [][]string{
				{"expr"},
			},
		},
		{
			lhs: "expr",
			alts: [][]string{
				{"expr", "add", "term"},
				{"term"},
			},
		},
		{
			lhs: "term",
			alts: [][]string{
				{"term", "mul", "factor"},
				{"factor"},
			},
		},
		{
			lhs: "factor",
			alts: [][]string{
				{"lparen", "expr", "rparen"},
				{"sign", "number"},
			},
		},
		{
			lhs: "sign",
			alts: [][]string{
				{"minus"},
				{},
			},
		},
	}
	expectedNumOfProds := 0
	for _, eProd := range expectProds {
		for i, rhs := range eProd.alts {
			matchProduction(t, eProd.lhs, i, rhs, gram, symTab)
			expectedNumOfProds++
		}
	}
	actualNumOfProds := len(gram.ProductionSet.getAll())
	if actualNumOfProds != expectedNumOfProds {
		t.Fatalf("number of productions is mismatched; want: %v, got: %v", expectedNumOfProds, actualNumOfProds)
	}
}

func matchProduction(t *testing.T, lhs string, num int, rhs []string, gram *Grammar, symTab *SymbolTable) {
	t.Helper()

	lhsSym, ok := symTab.ToSymbol(lhs)
	if !ok {
		t.Fatalf("a symbol was not found; production: %v #%v", lhs, num)
	}
	prods, ok := gram.ProductionSet.findByLHS(lhsSym)
	if !ok {
		t.Fatalf("productions ware not found; production: %v #%v", lhs, num)
	}
	prod := prods[num]
	if prod.lhs != lhsSym {
		t.Fatalf("LHS of is mismatched; production: %v #%v, want: %v (text: %v), got: %v", lhs, num, lhsSym, lhs, prod.lhs)
	}
	if prod.rhsLen != len(rhs) {
		t.Fatalf("length of RHS is mismatched; production: %v #%v want: %v, got: %v", lhs, num, len(rhs), prod.rhsLen)
	}
	if prod.rhsLen > 0 {
		if prod.isEmpty() {
			t.Fatalf("production is empty; production: %v #%v", lhs, num)
		}
		for i, rhsText := range rhs {
			rhsSym, ok := symTab.ToSymbol(rhsText)
			if !ok {
				t.Fatalf("a symbol was not found; production: %v #%v, RHS: %v", lhs, i, rhsText)
			}
			if prod.rhs[i] != rhsSym {
				t.Fatalf("RHS is mismatched; production: %v #%v, want: %v (text: %v), got: %v", lhs, i, rhsSym, rhsText, prod.rhs[i])
			}
		}
	} else {
		if !prod.isEmpty() {
			t.Fatalf("production is not empty; production: %v #%v", lhs, num)
		}
	}
}
