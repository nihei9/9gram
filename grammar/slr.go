package grammar

import (
	"fmt"
	"io"
)

type ActionType string

const (
	ActionTypeShift  = ActionType("shift")
	ActionTypeReduce = ActionType("reduce")
	ActionTypeError  = ActionType("error")
)

type actionEntry int16

const actionEntryEmpty = actionEntry(0)

func newShiftActionEntry(state StateNum) actionEntry {
	return actionEntry(state * -1)
}

func newReduceActionEntry(prod ProductionNum) actionEntry {
	return actionEntry(prod)
}

func (e actionEntry) isEmpty() bool {
	return e == actionEntryEmpty
}

func (e actionEntry) describe() (ActionType, StateNum, ProductionNum) {
	if e == actionEntryEmpty {
		return ActionTypeError, stateNumInitial, productionNumMin
	}
	if e < 0 {
		return ActionTypeShift, StateNum(e * -1), productionNumMin
	}
	return ActionTypeReduce, stateNumInitial, ProductionNum(e)
}

type GoToType string

const (
	GoToTypeRegistered = GoToType("registered")
	GoToTypeError      = GoToType("error")
)

type goToEntry uint16

const goToEntryEmpty = goToEntry(0)

func newGoToEntry(state StateNum) goToEntry {
	return goToEntry(state)
}

func (e goToEntry) isEmpty() bool {
	return e == goToEntryEmpty
}

func (e goToEntry) describe() (GoToType, StateNum) {
	if e == goToEntryEmpty {
		return GoToTypeError, stateNumInitial
	}
	return GoToTypeRegistered, StateNum(e)
}

type ParsingTable struct {
	actionTable   []actionEntry
	goToTable     []goToEntry
	numOfStates   int
	numOfTSymbols int
	numOfNSymbols int

	InitialState StateNum
}

func (t *ParsingTable) getAction(state StateNum, sym SymbolNum) (ActionType, StateNum, ProductionNum) {
	pos := state.Int()*t.numOfTSymbols + sym.Int()
	return t.actionTable[pos].describe()
}

func (t *ParsingTable) getGoTo(state StateNum, sym SymbolNum) (GoToType, StateNum) {
	pos := state.Int()*t.numOfNSymbols + sym.Int()
	return t.goToTable[pos].describe()
}

func (t *ParsingTable) writeShiftAction(state StateNum, sym Symbol, nextState StateNum) error {
	pos := state.Int()*t.numOfTSymbols + sym.num().Int()
	act := t.actionTable[pos]
	if !act.isEmpty() {
		ty, _, _ := act.describe()
		if ty == ActionTypeReduce {
			return fmt.Errorf("shift/reduce conflict")
		}
	}
	t.actionTable[pos] = newShiftActionEntry(nextState)

	return nil
}

func (t *ParsingTable) writeReduceAction(state StateNum, sym Symbol, prod ProductionNum) error {
	pos := state.Int()*t.numOfTSymbols + sym.num().Int()
	act := t.actionTable[pos]
	if !act.isEmpty() {
		ty, _, p := act.describe()
		if ty == ActionTypeReduce && p != prod {
			return fmt.Errorf("reduce/reduce conflict")
		}
		return fmt.Errorf("shift/reduce conflict")
	}
	t.actionTable[pos] = newReduceActionEntry(prod)

	return nil
}

func (t *ParsingTable) writeGoTo(state StateNum, sym Symbol, nextState StateNum) {
	pos := state.Int()*t.numOfNSymbols + sym.num().Int()
	t.goToTable[pos] = newGoToEntry(nextState)
}

func genSLRParsingTable(automaton *LR0Automaton, prods *productionSet, follow *Follow, numOfTSyms, numOfNSyms int) (*ParsingTable, error) {
	var ptab *ParsingTable
	{
		initialState := automaton.states[automaton.initialState]
		ptab = &ParsingTable{
			actionTable:   make([]actionEntry, len(automaton.states)*numOfTSyms),
			goToTable:     make([]goToEntry, len(automaton.states)*numOfNSyms),
			numOfStates:   len(automaton.states),
			numOfTSymbols: numOfTSyms,
			numOfNSymbols: numOfNSyms,
			InitialState:  initialState.Num,
		}
	}

	for _, state := range automaton.states {
		for sym, kID := range state.Next {
			nextState := automaton.states[kID]
			if sym.isTerminal() {
				err := ptab.writeShiftAction(state.Num, sym, nextState.Num)
				if err != nil {
					return nil, err
				}
			} else {
				ptab.writeGoTo(state.Num, sym, nextState.Num)
			}
		}

		for prodID := range state.Reducible {
			prod, _ := prods.findByID(prodID)
			flw := follow.Get(prod.lhs)
			for sym := range flw.symbols {
				err := ptab.writeReduceAction(state.Num, sym, prod.num)
				if err != nil {
					return nil, err
				}
			}
			if flw.eof {
				err := ptab.writeReduceAction(state.Num, symbolEOF, prod.num)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return ptab, nil
}

func printParsingTable(w io.Writer, ptab *ParsingTable) {
	fmt.Fprintf(w, "Action:\n")
	for stateNum := 0; stateNum < ptab.numOfStates; stateNum++ {
		for symNum := 0; symNum < ptab.numOfTSymbols; symNum++ {
			fmt.Fprintf(w, "  %v-%v: ", stateNum, symNum)
			ty, nextState, prod := ptab.getAction(StateNum(stateNum), SymbolNum(symNum))
			switch ty {
			case ActionTypeShift:
				fmt.Fprintf(w, "shift %v", nextState)
			case ActionTypeReduce:
				fmt.Fprintf(w, "reduce %v", prod)
			default:
				fmt.Fprintf(w, "error")
			}
			fmt.Fprintf(w, "\n")
		}
	}

	fmt.Fprintf(w, "GoTo:\n")
	for stateNum := 0; stateNum < ptab.numOfStates; stateNum++ {
		for symNum := 0; symNum < ptab.numOfNSymbols; symNum++ {
			fmt.Fprintf(w, "  %v-%v: ", stateNum, symNum)
			ty, nextState := ptab.getGoTo(StateNum(stateNum), SymbolNum(symNum))
			switch ty {
			case GoToTypeRegistered:
				fmt.Fprintf(w, "%v", nextState)
			default:
				fmt.Fprintf(w, "error")
			}
			fmt.Fprintf(w, "\n")
		}
	}
}
