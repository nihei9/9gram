package grammar

import (
	"fmt"
	"io"
	"sort"
)

type FollowEntry struct {
	symbols map[Symbol]struct{}
	eof     bool
}

func newFollowEntry() *FollowEntry {
	return &FollowEntry{
		symbols: map[Symbol]struct{}{},
		eof:     false,
	}
}

func (e *FollowEntry) add(sym Symbol) bool {
	changed := false
	l := len(e.symbols)

	e.symbols[sym] = struct{}{}

	if l != len(e.symbols) {
		changed = true
	}
	return changed
}

func (e *FollowEntry) addEOF() bool {
	if !e.eof {
		e.eof = true
		return true
	}
	return false
}

func (e *FollowEntry) merge(fst *FirstEntry, flw *FollowEntry) bool {
	changed := false

	if fst != nil {
		for sym := range fst.symbols {
			added := e.add(sym)
			if added {
				changed = true
			}
		}
	}

	if flw != nil {
		for sym := range flw.symbols {
			added := e.add(sym)
			if added {
				changed = true
			}
		}
		if flw.eof {
			added := e.addEOF()
			if added {
				changed = true
			}
		}
	}

	return changed
}

type Follow struct {
	set map[Symbol]*FollowEntry
}

func newFollow(prods *productionSet) *Follow {
	flw := &Follow{
		set: map[Symbol]*FollowEntry{},
	}
	for _, prod := range prods.getAll() {
		if _, ok := flw.set[prod.lhs]; ok {
			continue
		}
		flw.set[prod.lhs] = newFollowEntry()
	}
	return flw
}

func (flw *Follow) Get(sym Symbol) (*FollowEntry, error) {
	e, ok := flw.set[sym]
	if !ok {
		return nil, fmt.Errorf("FOLLOW set was not found; symbol: %s", sym)
	}
	return e, nil
}

type followComContext struct {
	prods  *productionSet
	first  *First
	follow *Follow
}

func newFollowComContext(prods *productionSet, first *First) *followComContext {
	return &followComContext{
		prods:  prods,
		first:  first,
		follow: newFollow(prods),
	}
}

func genFollow(prods *productionSet, first *First) (*Follow, error) {
	ntsyms := map[Symbol]struct{}{}
	for _, prod := range prods.getAll() {
		if _, ok := ntsyms[prod.lhs]; ok {
			continue
		}
		ntsyms[prod.lhs] = struct{}{}
	}

	cc := newFollowComContext(prods, first)
	for {
		more := false
		for ntsym := range ntsyms {
			e, err := cc.follow.Get(ntsym)
			if err != nil {
				return nil, err
			}
			if ntsym.isStart() {
				changed := e.addEOF()
				if changed {
					more = true
				}
			}
			for _, prod := range prods.getAll() {
				for i, sym := range prod.rhs {
					if sym != ntsym {
						continue
					}
					fst, err := first.Get(prod, i+1)
					if err != nil {
						return nil, err
					}
					changed := e.merge(fst, nil)
					if changed {
						more = true
					}
					if fst.empty {
						flw, err := cc.follow.Get(prod.lhs)
						if err != nil {
							return nil, err
						}
						changed := e.merge(nil, flw)
						if changed {
							more = true
						}
					}
				}
			}
		}
		if !more {
			break
		}
	}

	return cc.follow, nil
}

func genFollowEntry(cc *followComContext, acc *FollowEntry, ntsym Symbol) (bool, error) {
	isEntryChanged := false

	if ntsym.isStart() {
		changed := acc.addEOF()
		if changed {
			isEntryChanged = true
		}
	}
	for _, prod := range cc.prods.getAll() {
		for i, sym := range prod.rhs {
			if sym != ntsym {
				continue
			}
			fst, err := cc.first.Get(prod, i+1)
			if err != nil {
				return false, err
			}
			changed := acc.merge(fst, nil)
			if changed {
				isEntryChanged = true
			}
			if fst.empty {
				flw, err := cc.follow.Get(prod.lhs)
				if err != nil {
					return false, err
				}
				changed := acc.merge(nil, flw)
				if changed {
					isEntryChanged = true
				}
			}
		}
	}

	return isEntryChanged, nil
}

func PrintFollow(w io.Writer, follow *Follow, symTab *SymbolTable) {
	if w == nil {
		return
	}

	var nsyms []Symbol
	for nsym := range follow.set {
		nsyms = append(nsyms, nsym)
	}
	sort.Slice(nsyms, func(i, j int) bool {
		return nsyms[i].Num() < nsyms[j].Num()
	})

	for _, nsym := range nsyms {
		nsymText, ok := symTab.ToText(nsym)
		if !ok {
			nsymText = "<Symbol Not Found>"
		}
		e, err := follow.Get(nsym)
		if err != nil {
			fmt.Fprintf(w, "! %v", err)
			continue
		}
		fmt.Fprintf(w, "%v:", nsymText)
		if e.eof {
			fmt.Fprintf(w, " <eof>")
		}
		var tsyms []Symbol
		for tsym := range e.symbols {
			tsyms = append(tsyms, tsym)
		}
		sort.Slice(tsyms, func(i, j int) bool {
			return tsyms[i].Num() < tsyms[j].Num()
		})
		for _, tsym := range tsyms {
			tsymText, ok := symTab.ToText(tsym)
			if !ok {
				tsymText = "<Symbol Not Found>"
			}
			fmt.Fprintf(w, " %v", tsymText)
		}
		fmt.Fprintf(w, "\n")
	}
}
