package grammar

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nihei9/9gram/parser"
)

func TestGenSLRParsingTable(t *testing.T) {
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
	first, err := genFirst(gram.ProductionSet)
	if err != nil {
		t.Fatal(err)
	}
	follow, err := genFollow(gram.ProductionSet, first)
	if err != nil {
		t.Fatal(err)
	}
	automaton, err := genLR0Automaton(gram.ProductionSet, gram.AugmentedStartSymbol)
	if err != nil {
		t.Fatal(err)
	}

	ptab, err := genSLRParsingTable(automaton, gram.ProductionSet, follow)
	if err != nil {
		t.Fatalf("failed to create a SLR parsing table: %v", err)
	}
	if ptab == nil {
		t.Fatal("genSLRParsingTable returns nil without any error")
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

	expectedStates := []struct {
		kernelItems []*LR0Item
		acts        map[Symbol]testActionEntry
		goTos       map[Symbol][]*LR0Item
	}{
		{
			kernelItems: expectedKernels[0],
			acts: map[Symbol]testActionEntry{
				genSym("LPAREN"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[4],
				},
				genSym("NUMBER"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[5],
				},
			},
			goTos: map[Symbol][]*LR0Item{
				genSym("e"): expectedKernels[1],
				genSym("t"): expectedKernels[2],
				genSym("f"): expectedKernels[3],
			},
		},
		{
			kernelItems: expectedKernels[1],
			acts: map[Symbol]testActionEntry{
				genSym("ADD"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[6],
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("e'", "e"),
				},
			},
		},
		{
			kernelItems: expectedKernels[2],
			acts: map[Symbol]testActionEntry{
				genSym("ADD"): {
					ty:         ActionTypeReduce,
					production: genProd("e", "t"),
				},
				genSym("MUL"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[7],
				},
				genSym("RPAREN"): {
					ty:         ActionTypeReduce,
					production: genProd("e", "t"),
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("e", "t"),
				},
			},
		},
		{
			kernelItems: expectedKernels[3],
			acts: map[Symbol]testActionEntry{
				genSym("ADD"): {
					ty:         ActionTypeReduce,
					production: genProd("t", "f"),
				},
				genSym("MUL"): {
					ty:         ActionTypeReduce,
					production: genProd("t", "f"),
				},
				genSym("RPAREN"): {
					ty:         ActionTypeReduce,
					production: genProd("t", "f"),
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("t", "f"),
				},
			},
		},
		{
			kernelItems: expectedKernels[4],
			acts: map[Symbol]testActionEntry{
				genSym("LPAREN"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[4],
				},
				genSym("NUMBER"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[5],
				},
			},
			goTos: map[Symbol][]*LR0Item{
				genSym("e"): expectedKernels[8],
				genSym("t"): expectedKernels[2],
				genSym("f"): expectedKernels[3],
			},
		},
		{
			kernelItems: expectedKernels[5],
			acts: map[Symbol]testActionEntry{
				genSym("ADD"): {
					ty:         ActionTypeReduce,
					production: genProd("f", "NUMBER"),
				},
				genSym("MUL"): {
					ty:         ActionTypeReduce,
					production: genProd("f", "NUMBER"),
				},
				genSym("RPAREN"): {
					ty:         ActionTypeReduce,
					production: genProd("f", "NUMBER"),
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("f", "NUMBER"),
				},
			},
		},
		{
			kernelItems: expectedKernels[6],
			acts: map[Symbol]testActionEntry{
				genSym("LPAREN"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[4],
				},
				genSym("NUMBER"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[5],
				},
			},
			goTos: map[Symbol][]*LR0Item{
				genSym("t"): expectedKernels[9],
				genSym("f"): expectedKernels[3],
			},
		},
		{
			kernelItems: expectedKernels[7],
			acts: map[Symbol]testActionEntry{
				genSym("LPAREN"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[4],
				},
				genSym("NUMBER"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[5],
				},
			},
			goTos: map[Symbol][]*LR0Item{
				genSym("f"): expectedKernels[10],
			},
		},
		{
			kernelItems: expectedKernels[8],
			acts: map[Symbol]testActionEntry{
				genSym("ADD"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[6],
				},
				genSym("RPAREN"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[11],
				},
			},
		},
		{
			kernelItems: expectedKernels[9],
			acts: map[Symbol]testActionEntry{
				genSym("ADD"): {
					ty:         ActionTypeReduce,
					production: genProd("e", "e", "ADD", "t"),
				},
				genSym("MUL"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[7],
				},
				genSym("RPAREN"): {
					ty:         ActionTypeReduce,
					production: genProd("e", "e", "ADD", "t"),
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("e", "e", "ADD", "t"),
				},
			},
		},
		{
			kernelItems: expectedKernels[10],
			acts: map[Symbol]testActionEntry{
				genSym("ADD"): {
					ty:         ActionTypeReduce,
					production: genProd("t", "t", "MUL", "f"),
				},
				genSym("MUL"): {
					ty:         ActionTypeReduce,
					production: genProd("t", "t", "MUL", "f"),
				},
				genSym("RPAREN"): {
					ty:         ActionTypeReduce,
					production: genProd("t", "t", "MUL", "f"),
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("t", "t", "MUL", "f"),
				},
			},
		},
		{
			kernelItems: expectedKernels[11],
			acts: map[Symbol]testActionEntry{
				genSym("ADD"): {
					ty:         ActionTypeReduce,
					production: genProd("f", "LPAREN", "e", "RPAREN"),
				},
				genSym("MUL"): {
					ty:         ActionTypeReduce,
					production: genProd("f", "LPAREN", "e", "RPAREN"),
				},
				genSym("RPAREN"): {
					ty:         ActionTypeReduce,
					production: genProd("f", "LPAREN", "e", "RPAREN"),
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("f", "LPAREN", "e", "RPAREN"),
				},
			},
		},
	}

	t.Run("initial state", func(t *testing.T) {
		iniState := findStateByNum(automaton.states, ptab.InitialState)
		if iniState == nil {
			t.Fatalf("the initial state was not found; state: #%v", ptab.InitialState)
		}
		eIniState, err := newKernel(expectedKernels[0])
		if err != nil {
			t.Fatalf("failed to create a kernel item: %v", err)
		}
		if iniState.ID != eIniState.ID {
			t.Fatalf("initial state is mismatched; want: %v, got: %v", eIniState.ID, iniState.ID)
		}
	})

	for i, eState := range expectedStates {
		t.Run(fmt.Sprintf("#%v", i), func(t *testing.T) {
			k, err := newKernel(eState.kernelItems)
			if err != nil {
				t.Fatalf("failed to create a kernel item: %v", err)
			}
			state, ok := automaton.states[k.ID]
			if !ok {
				t.Fatalf("state was not found; state: #%v", 0)
			}

			actEntries := ptab.Action[state.Num]
			if len(actEntries) != len(eState.acts) {
				t.Errorf("number of action entries is mismatched; want: %v, got: %v", len(eState.acts), len(actEntries))
			}
			for _, act := range actEntries {
				eAct, ok := eState.acts[act.Symbol]
				if !ok {
					t.Fatalf("unknown action entry: %+v", act)
				}
				if act.ActionType != eAct.ty {
					t.Fatalf("action type is mismatched; want: %v, got: %v", eAct.ty, act.ActionType)
				}
				switch act.ActionType {
				case ActionTypeShift:
					eNextState, err := newKernel(eAct.nextState)
					if err != nil {
						t.Fatal(err)
					}
					nextState := findStateByNum(automaton.states, act.State)
					if nextState == nil {
						t.Fatalf("state was not found; state: #%v", act.State)
					}
					if nextState.ID != eNextState.ID {
						t.Fatalf("next state is mismatched; symbol: %v, want: %v, got: %v", act.Symbol, eNextState.ID, nextState.ID)
					}
				case ActionTypeReduce:
					prod := findProductionByNum(gram.ProductionSet, act.Production)
					if prod == nil {
						t.Fatalf("production was not found; production: #%v", act.Production)
					}
					if prod.id != eAct.production.id {
						t.Fatalf("production is mismatched; symbol: %v, want: %v, got: %v", act.Symbol, eAct.production.id, prod.id)
					}
				}
			}

			goToEntries := ptab.GoTo[state.Num]
			if len(goToEntries) != len(eState.goTos) {
				t.Errorf("number of goto entries is mismatched; want: %v, got: %v", len(eState.goTos), len(goToEntries))
			}
			for _, goTo := range goToEntries {
				eGoTo, ok := eState.goTos[goTo.Symbol]
				if !ok {
					t.Fatalf("unknown goto entry: %+v", goTo)
				}
				eNextState, err := newKernel(eGoTo)
				if err != nil {
					t.Fatal(err)
				}
				nextState := findStateByNum(automaton.states, goTo.State)
				if nextState == nil {
					t.Fatalf("state was not found; state: #%v", goTo.State)
				}
				if nextState.ID != eNextState.ID {
					t.Fatalf("next state is mismatched; symbol: %v, want: %v, got: %v", goTo.Symbol, eNextState.ID, nextState.ID)
				}
			}
		})
	}
}

type testActionEntry struct {
	ty         ActionType
	nextState  []*LR0Item
	production *production
}

func findStateByNum(states map[KernelID]*LR0State, num StateNum) *LR0State {
	for _, state := range states {
		if state.Num == num {
			return state
		}
	}
	return nil
}

func findProductionByNum(prods *productionSet, num ProductionNum) *production {
	for _, prod := range prods.getAll() {
		if prod.num == num {
			return prod
		}
	}
	return nil
}
