package grammar

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/nihei9/9gram/parser"
)

func TestGenLR0Automaton(t *testing.T) {
	src := "e: e ADD t | t; t: t MUL f | f; f: LPAREN e RPAREN | NUMBER;"

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

	automaton, err := genLR0Automaton(gram.ProductionSet, gram.AugmentedStartSymbol)
	if err != nil {
		t.Fatalf("failed to create a LR0 automaton: %v", err)
	}
	if automaton == nil {
		t.Fatalf("GenLR0Automaton returns nil without any error")
	}

	initialState := automaton.states[automaton.initialState]
	if initialState == nil {
		t.Errorf("failed to get the initial status; id: %v", automaton.initialState)
	}

	genSym := newTestSymbolGenerator(t, gram.SymbolTable)
	genProd := newTestProductionGenerator(t, genSym)
	genLR0Item := newTestLR0ItemGenerator(t, genProd)

	expectedKernels := map[int][]*LR0Item{
		0: {
			genLR0Item("e'", 0, "e"),
		},
		1: {
			genLR0Item("e'", 1, "e"),
			genLR0Item("e", 1, "e", "ADD", "t"),
		},
		2: {
			genLR0Item("e", 1, "t"),
			genLR0Item("t", 1, "t", "MUL", "f"),
		},
		3: {
			genLR0Item("t", 1, "f"),
		},
		4: {
			genLR0Item("f", 1, "LPAREN", "e", "RPAREN"),
		},
		5: {
			genLR0Item("f", 1, "NUMBER"),
		},
		6: {
			genLR0Item("e", 2, "e", "ADD", "t"),
		},
		7: {
			genLR0Item("t", 2, "t", "MUL", "f"),
		},
		8: {
			genLR0Item("e", 1, "e", "ADD", "t"),
			genLR0Item("f", 2, "LPAREN", "e", "RPAREN"),
		},
		9: {
			genLR0Item("e", 3, "e", "ADD", "t"),
			genLR0Item("t", 1, "t", "MUL", "f"),
		},
		10: {
			genLR0Item("t", 3, "t", "MUL", "f"),
		},
		11: {
			genLR0Item("f", 3, "LPAREN", "e", "RPAREN"),
		},
	}

	expectedStates := []expectedLR0State{
		{
			kernelItems: expectedKernels[0],
			nextStates: map[Symbol][]*LR0Item{
				genSym("e"):      expectedKernels[1],
				genSym("t"):      expectedKernels[2],
				genSym("f"):      expectedKernels[3],
				genSym("LPAREN"): expectedKernels[4],
				genSym("NUMBER"): expectedKernels[5],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[1],
			nextStates: map[Symbol][]*LR0Item{
				genSym("ADD"): expectedKernels[6],
			},
			reducibleProds: []*production{
				genProd("e'", "e"),
			},
		},
		{
			kernelItems: expectedKernels[2],
			nextStates: map[Symbol][]*LR0Item{
				genSym("MUL"): expectedKernels[7],
			},
			reducibleProds: []*production{
				genProd("e", "t"),
			},
		},
		{
			kernelItems: expectedKernels[3],
			nextStates:  map[Symbol][]*LR0Item{},
			reducibleProds: []*production{
				genProd("t", "f"),
			},
		},
		{
			kernelItems: expectedKernels[4],
			nextStates: map[Symbol][]*LR0Item{
				genSym("e"):      expectedKernels[8],
				genSym("t"):      expectedKernels[2],
				genSym("f"):      expectedKernels[3],
				genSym("LPAREN"): expectedKernels[4],
				genSym("NUMBER"): expectedKernels[5],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[5],
			nextStates:  map[Symbol][]*LR0Item{},
			reducibleProds: []*production{
				genProd("f", "NUMBER"),
			},
		},
		{
			kernelItems: expectedKernels[6],
			nextStates: map[Symbol][]*LR0Item{
				genSym("t"):      expectedKernels[9],
				genSym("f"):      expectedKernels[3],
				genSym("LPAREN"): expectedKernels[4],
				genSym("NUMBER"): expectedKernels[5],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[7],
			nextStates: map[Symbol][]*LR0Item{
				genSym("f"):      expectedKernels[10],
				genSym("LPAREN"): expectedKernels[4],
				genSym("NUMBER"): expectedKernels[5],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[8],
			nextStates: map[Symbol][]*LR0Item{
				genSym("ADD"):    expectedKernels[6],
				genSym("RPAREN"): expectedKernels[11],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[9],
			nextStates: map[Symbol][]*LR0Item{
				genSym("MUL"): expectedKernels[7],
			},
			reducibleProds: []*production{
				genProd("e", "e", "ADD", "t"),
			},
		},
		{
			kernelItems: expectedKernels[10],
			nextStates:  map[Symbol][]*LR0Item{},
			reducibleProds: []*production{
				genProd("t", "t", "MUL", "f"),
			},
		},
		{
			kernelItems: expectedKernels[11],
			nextStates:  map[Symbol][]*LR0Item{},
			reducibleProds: []*production{
				genProd("f", "LPAREN", "e", "RPAREN"),
			},
		},
	}

	if len(automaton.states) != len(expectedStates) {
		t.Errorf("number of states is mismatched; want: %v, got: %v", len(expectedStates), len(automaton.states))
	}

	for i, eState := range expectedStates {
		t.Run(fmt.Sprintf("state #%v", i), func(t *testing.T) {
			k, err := newKernel(eState.kernelItems)
			if err != nil {
				t.Fatalf("failed to create a kernel item: %v", err)
			}

			state, ok := automaton.states[k.ID]
			if !ok {
				t.Fatalf("kernel was not found; kernel ID: %v", k.ID)
			}

			// test next states
			{
				if len(state.Next) != len(eState.nextStates) {
					t.Errorf("number of next states is mismcthed; want: %v, got: %v", len(eState.nextStates), len(state.Next))
				}
				for eSym, eKItems := range eState.nextStates {
					nextStateKernel, err := newKernel(eKItems)
					if err != nil {
						t.Fatalf("failed to create a kernel item: %v", err)
					}
					nextState, ok := state.Next[eSym]
					if !ok {
						t.Fatalf("next state was not found; state: %v, symbol: %v (%v)", state.ID, "e", eSym)
					}
					if nextState != nextStateKernel.ID {
						t.Fatalf("kernel ID of the next state is mismatched;\nwant: %v\ngot: %v", nextStateKernel.ID, nextState)
					}
				}
			}

			// test reducible productions
			{
				if len(state.Reducible) != len(eState.reducibleProds) {
					t.Errorf("number of reducible production is mismatched; want: %v, got: %v", len(eState.reducibleProds), len(state.Reducible))
				}
				for _, eProd := range eState.reducibleProds {
					if _, ok := state.Reducible[eProd.id]; !ok {
						t.Errorf("reducible production was not found; production ID: %v", eProd.id)
					}
				}
			}
		})
	}

	printLR0Automaton(os.Stdout, automaton, gram.ProductionSet, gram.SymbolTable)
}

type expectedLR0State struct {
	kernelItems    []*LR0Item
	nextStates     map[Symbol][]*LR0Item
	reducibleProds []*production
}

type testSymbolGenerator func(text string) Symbol

func newTestSymbolGenerator(t *testing.T, symTab *SymbolTable) testSymbolGenerator {
	return func(text string) Symbol {
		t.Helper()

		sym, ok := symTab.ToSymbol(text)
		if !ok {
			t.Fatalf("symbol was not found; text: %v", text)
		}
		return sym
	}
}

type testProductionGenerator func(lhs string, rhs ...string) *production

func newTestProductionGenerator(t *testing.T, genSym testSymbolGenerator) testProductionGenerator {
	return func(lhs string, rhs ...string) *production {
		t.Helper()

		rhsSym := []Symbol{}
		for _, text := range rhs {
			rhsSym = append(rhsSym, genSym(text))
		}
		prod, err := newProduction(genSym(lhs), rhsSym)
		if err != nil {
			t.Fatalf("failed to create a production: %v", err)
		}

		return prod
	}
}

type testLR0ItemGenerator func(lhs string, dot int, rhs ...string) *LR0Item

func newTestLR0ItemGenerator(t *testing.T, genProd testProductionGenerator) testLR0ItemGenerator {
	return func(lhs string, dot int, rhs ...string) *LR0Item {
		t.Helper()

		prod := genProd(lhs, rhs...)
		item, err := newLR0Item(prod, dot)
		if err != nil {
			t.Fatalf("failed to create a LR0 item: %v", err)
		}

		return item
	}
}
