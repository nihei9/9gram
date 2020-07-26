package grammar

import (
	"fmt"

	"github.com/nihei9/9gram/parser"
)

type Grammar struct {
	SymbolTable          *SymbolTable
	ProductionSet        *productionSet
	AugmentedStartSymbol Symbol
}

func GenGrammar(root *parser.AST) (*Grammar, error) {
	symTab := newSymbolTable()
	prods := newProductionSet()
	gram := &Grammar{
		SymbolTable:   symTab,
		ProductionSet: prods,
	}

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
		startSym, err := symTab.registerNonTerminalSymbol(startText)
		if err != nil {
			return nil, err
		}
		augmentedStartText := fmt.Sprintf("%s'", startText)
		augmentedStartSym, err := symTab.registerStartSymbol(augmentedStartText)
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
	for _, ast := range root.Children {
		if ast.Ty != parser.ASTTypeProduction {
			continue
		}

		lhsAST := ast.Children[0]
		lhsText, _ := lhsAST.GetText()
		lhsSym, _ := symTab.ToSymbol(lhsText)

		for i := 1; i < len(ast.Children); i++ {
			altAST := ast.Children[i]
			rhsSyms := make([]Symbol, len(altAST.Children))
			for i, symAST := range altAST.Children {
				symText, _ := symAST.GetText()
				sym, err := symTab.registerTerminalSymbol(symText)
				if err != nil {
					return nil, err
				}
				rhsSyms[i] = sym
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

	automaton, err := genLR0Automaton(gram.ProductionSet, gram.AugmentedStartSymbol)
	if err != nil {
		return nil, fmt.Errorf("failed to create a LR0 automaton: %v", err)
	}

	numOfTSyms := gram.SymbolTable.getNumOfTerminalSymbols()
	numOfNSyms := gram.SymbolTable.getNumOfNonTerminalSymbols()
	ptab, err := genSLRParsingTable(automaton, gram.ProductionSet, flw, numOfTSyms, numOfNSyms)
	if err != nil {
		return nil, fmt.Errorf("failed to create a SLR parsing table: %v", err)
	}

	return &Table{
		LR:           ptab,
		LR0Automaton: automaton,
		Follow:       flw,
		First:        fst,
	}, nil
}
