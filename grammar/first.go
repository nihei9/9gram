package grammar

import "fmt"

type FirstEntry struct {
	symbols map[Symbol]struct{}
	empty   bool
}

func newFirstEntry() *FirstEntry {
	return &FirstEntry{
		symbols: map[Symbol]struct{}{},
		empty:   false,
	}
}

func (e *FirstEntry) add(sym Symbol) bool {
	changed := false
	l := len(e.symbols)

	e.symbols[sym] = struct{}{}

	if l != len(e.symbols) {
		changed = true
	}
	return changed
}

func (e *FirstEntry) addEmpty() bool {
	if !e.empty {
		e.empty = true
		return true
	}

	return false
}

func (e *FirstEntry) mergeExceptEmpty(target *FirstEntry) bool {
	if target == nil {
		return false
	}
	isEntryChanged := false
	for sym := range target.symbols {
		added := e.add(sym)
		if added {
			isEntryChanged = true
		}
	}
	return isEntryChanged
}

type First struct {
	set map[Symbol]*FirstEntry
}

func newFirst(prods *productionSet) *First {
	fst := &First{
		set: map[Symbol]*FirstEntry{},
	}
	for _, prod := range prods.getAll() {
		if _, ok := fst.set[prod.lhs]; ok {
			continue
		}
		fst.set[prod.lhs] = newFirstEntry()
	}

	return fst
}

func (fst *First) Get(prod *production, head int) (*FirstEntry, error) {
	entry := newFirstEntry()
	if prod.rhsLen <= head {
		entry.addEmpty()
		return entry, nil
	}
	for _, sym := range prod.rhs[head:] {
		if sym.isTerminal() {
			entry.add(sym)
			return entry, nil
		}

		e := fst.getBySymbol(sym)
		if e == nil {
			return nil, fmt.Errorf("FIRST set was not found; symbol: %s", sym)
		}
		for s := range e.symbols {
			entry.add(s)
		}
		if !e.empty {
			return entry, nil
		}
	}
	entry.addEmpty()
	return entry, nil
}

func (fst *First) getBySymbol(sym Symbol) *FirstEntry {
	return fst.set[sym]
}

type firstComContext struct {
	first *First
}

func newFirstComContext(prods *productionSet) *firstComContext {
	return &firstComContext{
		first: newFirst(prods),
	}
}

func genFirst(prods *productionSet) (*First, error) {
	cc := newFirstComContext(prods)
	for {
		more := false
		for _, prod := range prods.getAll() {
			e := cc.first.getBySymbol(prod.lhs)
			isEntryChanged, err := genProdFirstEntry(cc, e, prod)
			if err != nil {
				return nil, err
			}
			if isEntryChanged {
				more = true
			}
		}
		if !more {
			break
		}
	}
	return cc.first, nil
}

func genProdFirstEntry(cc *firstComContext, acc *FirstEntry, prod *production) (bool, error) {
	if prod.isEmpty() {
		return acc.addEmpty(), nil
	}

	for _, rhsSym := range prod.rhs {
		if rhsSym.isTerminal() {
			return acc.add(rhsSym), nil
		}

		e := cc.first.getBySymbol(rhsSym)
		changed := acc.mergeExceptEmpty(e)
		if !e.empty {
			return changed, nil
		}
	}
	return acc.addEmpty(), nil
}
