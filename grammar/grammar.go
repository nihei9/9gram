package grammar

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/nihei9/9gram/log"
	"github.com/nihei9/9gram/parser"
)

type Grammar struct {
	SymbolTable          *SymbolTable
	Patterns             map[SymbolNum]string
	ProductionSet        *productionSet
	AugmentedStartSymbol Symbol
}

func GenGrammar(root *parser.AST) (*Grammar, error) {
	symTab := newSymbolTable()
	sym2Pat := map[SymbolNum]string{}
	pat2Sym := map[string]Symbol{}
	prods := newProductionSet()
	gram := &Grammar{
		SymbolTable:   symTab,
		Patterns:      sym2Pat,
		ProductionSet: prods,
	}

	defer func() {
		log.Log("--- Symbol Table starts")
		PrintSymbolTable(log.GetWriter(), gram.SymbolTable)
		log.Log("--- Symbol Table ends")
		log.Log("--- Patterns starts")
		{
			syms := []SymbolNum{}
			for sym := range gram.Patterns {
				syms = append(syms, sym)
			}
			sort.Slice(syms, func(i, j int) bool {
				return syms[i].Int() < syms[j].Int()
			})
			for _, sym := range syms {
				patText := gram.Patterns[sym]
				log.Log("%v: %v", sym, patText)
			}
		}
		log.Log("--- Patterns ends")
		log.Log("--- Production Set starts")
		PrintProductionSet(log.GetWriter(), gram.ProductionSet, gram.SymbolTable)
		log.Log("--- Production Set ends")
	}()

	// Register the augmented start symbol with the symbol table and generate its production
	for _, ast := range root.Children {
		if ast.Ty != parser.ASTTypeProduction {
			continue
		}

		lhsAST := ast.Children[0]
		startText, ok := lhsAST.GetText()
		if !ok {
			return nil, fmt.Errorf("a node of the AST does not have a text representation; node: %#v", lhsAST)
		}
		augmentedStartText := fmt.Sprintf("%s'", startText)
		augmentedStartSym, err := symTab.registerStartSymbol(augmentedStartText)
		if err != nil {
			return nil, err
		}
		startSym, err := symTab.registerNonTerminalSymbol(startText)
		if err != nil {
			return nil, err
		}
		prod, err := newProduction(augmentedStartSym, []Symbol{startSym})
		if err != nil {
			return nil, err
		}
		prods.append(prod)

		gram.AugmentedStartSymbol = augmentedStartSym

		break
	}

	// Register all non-terminal symbols with symbol table
	for _, ast := range root.Children {
		if ast.Ty != parser.ASTTypeProduction {
			continue
		}

		lhsAST := ast.Children[0]
		lhsText, _ := lhsAST.GetText()
		_, err := symTab.registerNonTerminalSymbol(lhsText)
		if err != nil {
			return nil, err
		}
	}

	// Generate productions
	patNum := 0
	for _, ast := range root.Children {
		if ast.Ty != parser.ASTTypeProduction {
			continue
		}

		lhsAST := ast.Children[0]
		lhsText, _ := lhsAST.GetText()
		lhsSym, _ := symTab.ToSymbol(lhsText)

		for _, altAST := range ast.Children[1:] {
			rhsSyms := make([]Symbol, len(altAST.Children))
			for i, elemAST := range altAST.Children {
				if elemAST.Ty == parser.ASTTypePattern {
					patText, ok := elemAST.GetText()
					if !ok {
						return nil, fmt.Errorf("text representation of pattern string is not found")
					}
					sym, ok := pat2Sym[patText]
					if !ok {
						symText := fmt.Sprintf("$%v", patNum)
						patNum++
						var err error
						sym, err = symTab.registerTerminalSymbol(symText)
						if err != nil {
							return nil, err
						}
						pat2Sym[patText] = sym
						sym2Pat[sym.Num()] = patText
					}
					rhsSyms[i] = sym
				} else {
					symText, _ := elemAST.GetText()
					sym, err := symTab.registerTerminalSymbol(symText)
					if err != nil {
						return nil, err
					}
					rhsSyms[i] = sym
				}
			}

			prod, err := newProduction(lhsSym, rhsSyms)
			if err != nil {
				return nil, err
			}

			prods.append(prod)
		}
	}

	return gram, nil
}

type Table struct {
	LR           *ParsingTable
	LR0Automaton *LR0Automaton
	Follow       *Follow
	First        *First
}

func GenTable(gram *Grammar) (*Table, error) {
	fst, err := genFirst(gram.ProductionSet)
	if err != nil {
		return nil, fmt.Errorf("failed to create a FIRST set: %v", err)
	}

	flw, err := genFollow(gram.ProductionSet, fst)
	if err != nil {
		return nil, fmt.Errorf("failed to create a FOLLOW set: %v", err)
	}
	log.Log("--- Follow starts")
	PrintFollow(log.GetWriter(), flw, gram.SymbolTable)
	log.Log("--- Follow ends")

	automaton, err := genLR0Automaton(gram.ProductionSet, gram.AugmentedStartSymbol)
	if err != nil {
		return nil, fmt.Errorf("failed to create a LR0 automaton: %v", err)
	}
	log.Log("--- LR0 Automaton starts")
	PrintLR0Automaton(log.GetWriter(), automaton, gram.ProductionSet, gram.SymbolTable)
	log.Log("--- LR0 Automaton ends")

	numOfTSyms := gram.SymbolTable.getNumOfTerminalSymbols()
	numOfNSyms := gram.SymbolTable.getNumOfNonTerminalSymbols()
	ptab, err := genSLRParsingTable(automaton, gram.ProductionSet, flw, numOfTSyms, numOfNSyms)
	if err != nil {
		return nil, fmt.Errorf("failed to create a SLR parsing table: %v", err)
	}
	log.Log("--- Parsing Table starts")
	PrintParsingTable(log.GetWriter(), ptab)
	log.Log("--- ParsingTable ends")

	return &Table{
		LR:           ptab,
		LR0Automaton: automaton,
		Follow:       flw,
		First:        fst,
	}, nil
}

func GenJSON(gram *Grammar, tab *Table) ([]byte, error) {
	headSyms := make([]int, len(gram.ProductionSet.getAll())+1)
	altSymCounts := make([]int, len(gram.ProductionSet.getAll())+1)
	for _, p := range gram.ProductionSet.getAll() {
		headSyms[p.num] = p.lhs.Num().Int()
		altSymCounts[p.num] = p.rhsLen
	}

	tsymCount := gram.SymbolTable.getNumOfTerminalSymbols()
	tsyms := make([]string, tsymCount)
	patterns := make([]string, tsymCount)
	for num := terminalSymbolNumMin.Int(); num < tsymCount; num++ {
		text, err := gram.SymbolTable.ToTextFromNumT(SymbolNum(num))
		if err != nil {
			return nil, err
		}
		tsyms[num] = text
		patterns[num] = gram.Patterns[SymbolNum(num)]
	}

	nsymCount := gram.SymbolTable.getNumOfNonTerminalSymbols()
	nsyms := make([]string, nsymCount)
	// nonTerminalSymbolNumMin represents the augmented start symbol.
	for num := nonTerminalSymbolNumMin.Int() + 1; num < nsymCount; num++ {
		text, err := gram.SymbolTable.ToTextFromNumN(SymbolNum(num))
		if err != nil {
			return nil, err
		}
		nsyms[num] = text
	}

	return json.Marshal(struct {
		Action                  []actionEntry `json:"action"`
		GoTo                    []goToEntry   `json:"goto"`
		StateCount              int           `json:"state_count"`
		InitialState            StateNum      `json:"initial_state"`
		StartProduction         int           `json:"start_production"`
		HeadSymbols             []int         `json:"head_symbols"`
		AlternativeSymbolCounts []int         `json:"alternative_symbol_counts"`
		EOFSymbol               int           `json:"eof_symbol"`
		TerminalSymbols         []string      `json:"terminal_symbols"`
		TerminalSymbolPatterns  []string      `json:"terminal_symbol_patterns"`
		TerminalSymbolCount     int           `json:"terminal_symbol_count"`
		NonTerminalSymbols      []string      `json:"non_terminal_symbols"`
		NonTerminalSymbolCount  int           `json:"non_terminal_symbol_count"`
	}{
		Action:                  tab.LR.actionTable,
		GoTo:                    tab.LR.goToTable,
		StateCount:              len(tab.LR0Automaton.states),
		InitialState:            tab.LR.InitialState,
		StartProduction:         ProductionNumStart.Int(),
		HeadSymbols:             headSyms,
		AlternativeSymbolCounts: altSymCounts,
		EOFSymbol:               SymbolEOF.Num().Int(),
		TerminalSymbols:         tsyms,
		TerminalSymbolPatterns:  patterns,
		TerminalSymbolCount:     tsymCount,
		NonTerminalSymbols:      nsyms,
		NonTerminalSymbolCount:  nsymCount,
	})
}
