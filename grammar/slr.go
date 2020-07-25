package grammar

import (
	"fmt"
	"io"
	"strings"
)

type ActionType string

const (
	ActionTypeShift  = ActionType("shift")
	ActionTypeReduce = ActionType("reduce")
)

type ActionEntry struct {
	Symbol     Symbol
	ActionType ActionType
	State      StateNum
	Production ProductionNum
}

type GoToEntry struct {
	Symbol Symbol
	State  StateNum
}

type ParsingTable struct {
	Action       [][]ActionEntry
	GoTo         [][]GoToEntry
	InitialState StateNum
}

func (t *ParsingTable) shift(state StateNum, sym Symbol, nextState StateNum) error {
	for _, e := range t.Action[state] {
		if e.Symbol != sym {
			continue
		}
		if e.ActionType == ActionTypeReduce {
			return fmt.Errorf("shift/reduce conflict")
		}
		return fmt.Errorf("an entry of a parsing table is already registerd")
	}

	t.Action[state] = append(t.Action[state], ActionEntry{
		Symbol:     sym,
		ActionType: ActionTypeShift,
		State:      nextState,
	})

	return nil
}

func (t *ParsingTable) reduce(state StateNum, sym Symbol, prod ProductionNum) error {
	for _, e := range t.Action[state] {
		if e.Symbol != sym {
			continue
		}
		if e.ActionType == ActionTypeReduce {
			return fmt.Errorf("reduce/reduce conflict")
		}
		return fmt.Errorf("shift/reduce conflict")
	}

	t.Action[state] = append(t.Action[state], ActionEntry{
		Symbol:     sym,
		ActionType: ActionTypeReduce,
		Production: prod,
	})

	return nil
}

func (t *ParsingTable) goTo(state StateNum, sym Symbol, nextState StateNum) {
	t.GoTo[state] = append(t.GoTo[state], GoToEntry{
		Symbol: sym,
		State:  nextState,
	})
}

func genSLRParsingTable(automaton *LR0Automaton, prods *productionSet, follow *Follow) (*ParsingTable, error) {
	var ptab *ParsingTable
	{
		initialState := automaton.states[automaton.initialState]
		ptab = &ParsingTable{
			Action:       make([][]ActionEntry, len(automaton.states)),
			GoTo:         make([][]GoToEntry, len(automaton.states)),
			InitialState: initialState.Num,
		}
	}

	for _, state := range automaton.states {
		for sym, kID := range state.Next {
			nextState := automaton.states[kID]
			if sym.isTerminal() {
				err := ptab.shift(state.Num, sym, nextState.Num)
				if err != nil {
					return nil, err
				}
			} else {
				ptab.goTo(state.Num, sym, nextState.Num)
			}
		}

		for prodID := range state.Reducible {
			prod, _ := prods.findByID(prodID)
			flw := follow.Get(prod.lhs)
			for sym := range flw.symbols {
				err := ptab.reduce(state.Num, sym, prod.num)
				if err != nil {
					return nil, err
				}
			}
			if flw.eof {
				err := ptab.reduce(state.Num, symbolEOF, prod.num)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return ptab, nil
}

func printParsingTable(w io.Writer, ptab *ParsingTable, symTab *SymbolTable) {
	w.Write([]byte("Action:\n"))
	for i, entries := range ptab.Action {
		var b strings.Builder
		fmt.Fprintf(&b, "#%v:", i)
		for _, entry := range entries {
			var symText string
			if entry.Symbol.isEOF() {
				symText = "EOF"
			} else {
				symText, _ = symTab.ToText(entry.Symbol)
			}
			switch entry.ActionType {
			case ActionTypeShift:
				fmt.Fprintf(&b, "  (%v, s%v)", symText, entry.State)
			case ActionTypeReduce:
				fmt.Fprintf(&b, "  (%v, r%v)", symText, entry.Production)
			}
		}
		fmt.Fprintf(&b, "\n")
		w.Write([]byte(b.String()))
	}
}
