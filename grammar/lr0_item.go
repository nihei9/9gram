package grammar

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

type LR0ItemID [32]byte

func (id LR0ItemID) String() string {
	return hex.EncodeToString(id[:])
}

func (id LR0ItemID) num() uint32 {
	return binary.LittleEndian.Uint32(id[:])
}

type LR0Item struct {
	id   LR0ItemID
	prod ProductionID

	// E -> E + T
	//
	// Dot | Dotted Symbol | Item
	// ----+---------------+------------
	// 0   | E             | E ->・E + T
	// 1   | +             | E -> E・+ T
	// 2   | T             | E -> E +・T
	// 3   | Nil           | E -> E + T・
	dot          int
	dottedSymbol Symbol

	// When initial is true, the LHS of the production is the augmented start symbol and dot is 0.
	// It looks like S' ->・S.
	initial bool

	// When reducible is true, the item looks like E -> E + T・.
	reducible bool

	// When kernel is true, the item is kernel item.
	kernel bool
}

func newLR0Item(prod *production, dot int) (*LR0Item, error) {
	if prod == nil {
		return nil, fmt.Errorf("production rule passed is nil")
	}

	if dot < 0 || dot > prod.rhsLen {
		return nil, fmt.Errorf("dot must be between 0 and %v", prod.rhsLen)
	}

	var id LR0ItemID
	{
		b := []byte{}
		b = append(b, prod.id[:]...)
		bDot := make([]byte, 8)
		binary.LittleEndian.PutUint64(bDot, uint64(dot))
		b = append(b, bDot...)
		id = sha256.Sum256(b)
	}

	dottedSymbol := symbolNil
	if dot < prod.rhsLen {
		dottedSymbol = prod.rhs[dot]
	}

	initial := false
	if prod.lhs.isStart() && dot == 0 {
		initial = true
	}

	reducible := false
	if dot == prod.rhsLen {
		reducible = true
	}

	kernel := false
	if initial || dot > 0 {
		kernel = true
	}

	item := &LR0Item{
		id:           id,
		prod:         prod.id,
		dot:          dot,
		dottedSymbol: dottedSymbol,
		initial:      initial,
		reducible:    reducible,
		kernel:       kernel,
	}

	return item, nil
}

type KernelID [32]byte

func (id KernelID) String() string {
	return hex.EncodeToString(id[:])
}

type Kernel struct {
	ID    KernelID
	Items []*LR0Item
}

func newKernel(items []*LR0Item) (*Kernel, error) {
	if len(items) <= 0 {
		return nil, fmt.Errorf("a kernel item is missing")
	}

	// remove duplicates from items and sort it
	var sortedItems []*LR0Item
	{
		m := map[LR0ItemID]*LR0Item{}
		for _, item := range items {
			if !item.kernel {
				return nil, fmt.Errorf("not a kernel item: %v", item)
			}
			m[item.id] = item
		}
		sortedItems = []*LR0Item{}
		for _, item := range m {
			sortedItems = append(sortedItems, item)
		}
		sort.Slice(sortedItems, func(i, j int) bool {
			return sortedItems[i].id.num() < sortedItems[j].id.num()
		})
	}

	// generate a kernel ID
	var id KernelID
	{
		b := []byte{}
		for _, item := range sortedItems {
			b = append(b, item.id[:]...)
		}
		id = sha256.Sum256(b)
	}

	return &Kernel{
		ID:    id,
		Items: sortedItems,
	}, nil
}

type StateNum int

const stateNumInitial = StateNum(0)

func (n StateNum) Int() int {
	return int(n)
}

func (n StateNum) String() string {
	return strconv.Itoa(int(n))
}

func (n StateNum) next() StateNum {
	return StateNum(n + 1)
}

type LR0State struct {
	*Kernel
	Num       StateNum
	Next      map[Symbol]KernelID
	Reducible map[ProductionID]struct{}
}

type LR0Automaton struct {
	initialState KernelID
	states       map[KernelID]*LR0State
}

func genLR0Automaton(prods *productionSet, startSym Symbol) (*LR0Automaton, error) {
	if !startSym.isStart() {
		return nil, fmt.Errorf("symbold passed is not start symbol")
	}

	automaton := &LR0Automaton{
		states: map[KernelID]*LR0State{},
	}

	currentState := stateNumInitial
	knownKernels := map[KernelID]struct{}{}
	uncheckedKernels := []*Kernel{}

	// generate the initial kernel
	{
		prods, _ := prods.findByLHS(startSym)
		initialItem, err := newLR0Item(prods[0], 0)
		if err != nil {
			return nil, err
		}

		k, err := newKernel([]*LR0Item{initialItem})
		if err != nil {
			return nil, err
		}
		automaton.initialState = k.ID
		knownKernels[k.ID] = struct{}{}
		uncheckedKernels = append(uncheckedKernels, k)
	}

	for len(uncheckedKernels) > 0 {
		nextUncheckedKernels := []*Kernel{}
		for _, k := range uncheckedKernels {
			state, neighbours, err := genStateAndNeighbourKernels(k, prods)
			if err != nil {
				return nil, err
			}
			state.Num = currentState
			currentState = currentState.next()

			automaton.states[state.ID] = state

			for _, k := range neighbours {
				if _, known := knownKernels[k.ID]; known {
					continue
				}
				knownKernels[k.ID] = struct{}{}
				nextUncheckedKernels = append(nextUncheckedKernels, k)
			}
		}
		uncheckedKernels = nextUncheckedKernels
	}

	return automaton, nil
}

func genStateAndNeighbourKernels(kernel *Kernel, prods *productionSet) (*LR0State, []*Kernel, error) {
	items, err := genClosure(kernel, prods)
	if err != nil {
		return nil, nil, err
	}
	neighbours, err := genNeighbourKernels(items, prods)
	if err != nil {
		return nil, nil, err
	}

	next := map[Symbol]KernelID{}
	kernels := []*Kernel{}
	for _, n := range neighbours {
		next[n.symbol] = n.kernel.ID
		kernels = append(kernels, n.kernel)
	}

	reducible := map[ProductionID]struct{}{}
	for _, item := range items {
		if item.reducible {
			reducible[item.prod] = struct{}{}
		}
	}

	return &LR0State{
		Kernel:    kernel,
		Next:      next,
		Reducible: reducible,
	}, kernels, nil
}

func genClosure(kernel *Kernel, prods *productionSet) ([]*LR0Item, error) {
	items := []*LR0Item{}
	knownItems := map[LR0ItemID]struct{}{}
	uncheckedItems := []*LR0Item{}
	for _, item := range kernel.Items {
		items = append(items, item)
		uncheckedItems = append(uncheckedItems, item)
	}
	for len(uncheckedItems) > 0 {
		nextUncheckedItems := []*LR0Item{}
		for _, item := range uncheckedItems {
			if item.dottedSymbol.isTerminal() {
				continue
			}

			ps, _ := prods.findByLHS(item.dottedSymbol)
			for _, prod := range ps {
				item, err := newLR0Item(prod, 0)
				if err != nil {
					return nil, err
				}
				if _, exist := knownItems[item.id]; exist {
					continue
				}
				items = append(items, item)
				knownItems[item.id] = struct{}{}
				nextUncheckedItems = append(nextUncheckedItems, item)
			}
		}
		uncheckedItems = nextUncheckedItems
	}

	return items, nil
}

type neighbourKernel struct {
	symbol Symbol
	kernel *Kernel
}

func genNeighbourKernels(items []*LR0Item, prods *productionSet) ([]*neighbourKernel, error) {
	kernelItemMap := map[Symbol][]*LR0Item{}
	for _, item := range items {
		if item.dottedSymbol.isNil() {
			continue
		}
		prod, ok := prods.findByID(item.prod)
		if !ok {
			return nil, fmt.Errorf("production was not found; production: %v", item.prod)
		}
		kItem, err := newLR0Item(prod, item.dot+1)
		if err != nil {
			return nil, err
		}
		kernelItemMap[item.dottedSymbol] = append(kernelItemMap[item.dottedSymbol], kItem)
	}

	nextSymbols := []Symbol{}
	for sym := range kernelItemMap {
		nextSymbols = append(nextSymbols, sym)
	}
	sort.Slice(nextSymbols, func(i, j int) bool {
		return nextSymbols[i] < nextSymbols[j]
	})

	kernels := []*neighbourKernel{}
	for _, sym := range nextSymbols {
		k, err := newKernel(kernelItemMap[sym])
		if err != nil {
			return nil, err
		}
		kernels = append(kernels, &neighbourKernel{
			symbol: sym,
			kernel: k,
		})
	}

	return kernels, nil
}

func printLR0Automaton(w io.Writer, automaton *LR0Automaton, prods *productionSet, symTab *SymbolTable) {
	sortedStates := make([]*LR0State, len(automaton.states))
	for _, state := range automaton.states {
		sortedStates[state.Num] = state
	}

	w.Write([]byte("LR0 Automaton:\n"))
	for _, state := range sortedStates {
		var b strings.Builder
		if state.ID == automaton.initialState {
			fmt.Fprintf(&b, "#%v (initial):\n", state.Num)
		} else {
			fmt.Fprintf(&b, "#%v:\n", state.Num)
		}
		fmt.Fprintf(&b, "  ID: %v\n", state.ID)
		fmt.Fprintf(&b, "  Kernel:\n")
		for _, kItem := range state.Items {
			prod, _ := prods.findByID(kItem.prod)
			lhs, _ := symTab.ToText(prod.lhs)
			fmt.Fprintf(&b, "    %v →", lhs)
			for i := 0; i < prod.rhsLen; i++ {
				rhs, _ := symTab.ToText(prod.rhs[i])
				if i == kItem.dot {
					fmt.Fprintf(&b, "・%v", rhs)
				} else {
					fmt.Fprintf(&b, " %v", rhs)
				}
			}
			if kItem.reducible {
				fmt.Fprintf(&b, "・")
			}
			fmt.Fprintf(&b, " (%v)\n", kItem.id)
		}
		fmt.Fprintf(&b, "  Next:\n")
		for sym, kID := range state.Next {
			symText, _ := symTab.ToText(sym)
			nextState := automaton.states[kID]
			fmt.Fprintf(&b, "    %v → %v\n", symText, nextState.Num)
		}
		fmt.Fprintf(&b, "  Reducible:\n")
		for prodID := range state.Reducible {
			prod, _ := prods.findByID(prodID)
			fmt.Fprintf(&b, "    %v\n", prod.num)
		}
		w.Write([]byte(b.String()))
	}
}
