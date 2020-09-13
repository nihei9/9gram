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
		if isLexemeProduction(ast) {
			_, err := symTab.registerTerminalSymbol(lhsText)
			if err != nil {
				return nil, err
			}
		} else {
			_, err := symTab.registerNonTerminalSymbol(lhsText)
			if err != nil {
				return nil, err
			}
		}
	}

	// Generate productions
	patNum := 0
	prodNum := 0
	for _, ast := range root.Children {
		if ast.Ty != parser.ASTTypeProduction {
			continue
		}
		if isLexemeProduction(ast) {
			registerLexemes(ast, symTab, sym2Pat, pat2Sym)
		} else {
			err := registerProds(ast, prods, symTab, sym2Pat, pat2Sym, &patNum, &prodNum)
			if err != nil {
				return nil, err
			}
		}
	}

	return gram, nil
}

func isLexemeProduction(prodAST *parser.AST) bool {
	if prodAST.Ty != parser.ASTTypeProduction {
		return false
	}
	if len(prodAST.Children) == 2 && len(prodAST.Children[1].Children) == 1 && prodAST.Children[1].Children[0].Ty == parser.ASTTypePattern {
		return true
	}
	return false
}

func registerLexemes(ast *parser.AST, symTab *SymbolTable, sym2Pat map[SymbolNum]string, pat2Sym map[string]Symbol) {
	lhsAST := ast.Children[0]
	lhsText, _ := lhsAST.GetText()
	lhsSym, _ := symTab.ToSymbol(lhsText)
	patText, _ := ast.Children[1].Children[0].GetText()
	pat2Sym[patText] = lhsSym
	sym2Pat[lhsSym.Num()] = patText
}

func registerProds(ast *parser.AST, prods *productionSet, symTab *SymbolTable, sym2Pat map[SymbolNum]string, pat2Sym map[string]Symbol, patNum *int, prodNum *int) error {
	lhsAST := ast.Children[0]
	lhsText, _ := lhsAST.GetText()
	lhsSym, _ := symTab.ToSymbol(lhsText)
	for _, altAST := range ast.Children[1:] {
		err := registerAlternative(altAST, prods, lhsSym, symTab, sym2Pat, pat2Sym, patNum, prodNum)
		if err != nil {
			return err
		}
	}
	return nil
}

func registerAlternative(altAST *parser.AST, prods *productionSet, lhsSym Symbol, symTab *SymbolTable, sym2Pat map[SymbolNum]string, pat2Sym map[string]Symbol, patNum *int, prodNum *int) error {
	var rhsSyms []Symbol
	i := 0
	for i < len(altAST.Children) {
		var rhsSym Symbol
		elemAST := altAST.Children[i]
		if elemAST.Ty == parser.ASTTypePattern {
			patText, ok := elemAST.GetText()
			if !ok {
				return fmt.Errorf("text representation of pattern string is not found")
			}
			sym, ok := pat2Sym[patText]
			if !ok {
				symText := fmt.Sprintf("$%v", *patNum)
				*patNum = *patNum + 1
				var err error
				sym, err = symTab.registerTerminalSymbol(symText)
				if err != nil {
					return err
				}
				pat2Sym[patText] = sym
				sym2Pat[sym.Num()] = patText
			}
			rhsSym = sym
		} else if elemAST.Ty == parser.ASTTypeSymbol {
			symText, _ := elemAST.GetText()
			sym, err := symTab.registerTerminalSymbol(symText)
			if err != nil {
				return err
			}
			rhsSym = sym
		} else {
			return fmt.Errorf("invalid symbol sequence")
		}
		i++

		if i >= len(altAST.Children) {
			rhsSyms = append(rhsSyms, rhsSym)
			break
		}

		switch altAST.Children[i].Ty {
		case parser.ASTTypeOptional:
			optSym := rhsSym

			lhsText := fmt.Sprintf("$$%v", *prodNum)
			*prodNum = *prodNum + 1
			lhsSym, err := symTab.registerNonTerminalSymbol(lhsText)
			if err != nil {
				return err
			}
			optProd1, err := newProduction(lhsSym, []Symbol{optSym})
			if err != nil {
				return err
			}
			optProd2, err := newProduction(lhsSym, []Symbol{})
			if err != nil {
				return err
			}
			prods.append(optProd1)
			prods.append(optProd2)

			rhsSym = lhsSym
			i++
		case parser.ASTTypeZeroOrMore:
			repeatSym := rhsSym

			lhsText := fmt.Sprintf("$$%v", *prodNum)
			*prodNum = *prodNum + 1
			lhsSym, err := symTab.registerNonTerminalSymbol(lhsText)
			if err != nil {
				return err
			}
			repeatProd1, err := newProduction(lhsSym, []Symbol{repeatSym, lhsSym})
			if err != nil {
				return err
			}
			repeatProd2, err := newProduction(lhsSym, []Symbol{})
			if err != nil {
				return err
			}
			prods.append(repeatProd1)
			prods.append(repeatProd2)

			rhsSym = lhsSym
			i++
		case parser.ASTTypeOneOrMore:
			repeatSym := rhsSym

			lhsText := fmt.Sprintf("$$%v", *prodNum)
			*prodNum = *prodNum + 1
			lhsSym, err := symTab.registerNonTerminalSymbol(lhsText)
			if err != nil {
				return err
			}
			repeatProd1, err := newProduction(lhsSym, []Symbol{repeatSym, lhsSym})
			if err != nil {
				return err
			}
			repeatProd2, err := newProduction(lhsSym, []Symbol{repeatSym})
			if err != nil {
				return err
			}
			prods.append(repeatProd1)
			prods.append(repeatProd2)

			rhsSym = lhsSym
			i++
		}

		rhsSyms = append(rhsSyms, rhsSym)
	}
	prod, err := newProduction(lhsSym, rhsSyms)
	if err != nil {
		return err
	}
	prods.append(prod)

	return nil
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
	var unusedTSyms []int
	{
		tsymUseChek := make([]bool, tsymCount)
		tsymUseChek[symbolNil.Num().Int()] = true
		tsymUseChek[SymbolEOF.Num().Int()] = true
		for _, prod := range gram.ProductionSet.getAll() {
			for _, rhsSym := range prod.rhs {
				if !rhsSym.isTerminal() {
					continue
				}
				tsymUseChek[rhsSym.Num().Int()] = true
			}
		}
		for num, used := range tsymUseChek {
			if used {
				continue
			}
			unusedTSyms = append(unusedTSyms, num)
		}
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
		UnusedTerminalSymbols   []int         `json:"unused_terminal_symbols"`
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
		UnusedTerminalSymbols:   unusedTSyms,
		NonTerminalSymbols:      nsyms,
		NonTerminalSymbolCount:  nsymCount,
	})
}
