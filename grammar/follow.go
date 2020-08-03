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

func (e *FollowEntry) add(sym Symbol) {
	e.symbols[sym] = struct{}{}
}

func (e *FollowEntry) addEOF() {
	e.eof = true
}

func (e *FollowEntry) merge(fst *FirstEntry, flw *FollowEntry) {
	if fst != nil {
		for sym := range fst.symbols {
			e.add(sym)
		}
	}

	if flw != nil {
		for sym := range flw.symbols {
			e.add(sym)
		}
		if flw.eof {
			e.addEOF()
		}
	}
}

type Follow struct {
	set map[Symbol]*FollowEntry
}

func newFollow() *Follow {
	return &Follow{
		set: map[Symbol]*FollowEntry{},
	}
}

func (flw *Follow) Get(sym Symbol) *FollowEntry {
	return flw.set[sym]
}

func (flw *Follow) put(e *FollowEntry, sym Symbol) {
	flw.set[sym] = e
}

type followComContext struct {
	prods  *productionSet
	first  *First
	follow *Follow
	stack  []Symbol
}

func newfollowComContext(prods *productionSet, first *First) *followComContext {
	return &followComContext{
		prods:  prods,
		first:  first,
		follow: newFollow(),
		stack:  []Symbol{},
	}
}

func (cc *followComContext) push(sym Symbol) {
	cc.stack = append(cc.stack, sym)
}

func (cc *followComContext) pop() {
	cc.stack = cc.stack[:len(cc.stack)-1]
}

func (cc *followComContext) isAlreadyStacked(sym Symbol) bool {
	for _, f := range cc.stack {
		if f == sym {
			return true
		}
	}

	return false
}

func genFollow(prods *productionSet, first *First) (*Follow, error) {
	cc := newfollowComContext(prods, first)
	for _, prod := range prods.getAll() {
		_, err := genFollowEntry(cc, prod.lhs)
		if err != nil {
			return nil, err
		}
	}
	return cc.follow, nil
}

func genFollowEntry(cc *followComContext, sym Symbol) (*FollowEntry, error) {
	if sym.isNil() {
		return nil, fmt.Errorf("symbol is nil")
	}

	// guards for avoiding the infinite recursion
	{
		// already computed
		if flw := cc.follow.Get(sym); flw != nil {
			return flw, nil
		}

		if cc.isAlreadyStacked(sym) {
			return newFollowEntry(), nil
		}
	}

	cc.push(sym)
	defer cc.pop()

	entry := newFollowEntry()

	if sym.isStart() {
		entry.addEOF()
	}

	for _, prod := range cc.prods.getAll() {
		for i, rhsSym := range prod.rhs {
			if rhsSym != sym {
				continue
			}

			if i+1 < prod.rhsLen {
				fst := cc.first.Get(prod.id, i+1)
				if fst == nil {
					return nil, fmt.Errorf("failed to get a FIRST set; production: %v, dot: %v", prod.id, i)
				}
				entry.merge(fst, nil)

				if !fst.empty {
					continue
				}
			}

			flw, err := genFollowEntry(cc, prod.lhs)
			if err != nil {
				return nil, err
			}
			entry.merge(nil, flw)
		}
	}
	cc.follow.put(entry, sym)

	return entry, nil
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
		e := follow.Get(nsym)
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
