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

func (e *FirstEntry) add(sym Symbol) {
	e.symbols[sym] = struct{}{}
}

func (e *FirstEntry) addEmpty() {
	e.empty = true
}

func (e *FirstEntry) mergeExceptEmpty(target *FirstEntry) {
	for sym := range target.symbols {
		e.add(sym)
	}
}

type First struct {
	set map[ProductionID][]*FirstEntry
}

func newFirst(prods *productionSet) *First {
	fst := &First{
		set: map[ProductionID][]*FirstEntry{},
	}
	for _, prod := range prods.getAll() {
		len := prod.rhsLen
		if prod.isEmpty() {
			len = 1
		}
		fst.set[prod.id] = make([]*FirstEntry, len)
	}

	return fst
}

func (fst *First) Get(prod ProductionID, head int) *FirstEntry {
	return fst.set[prod][head]
}

func (fst First) put(e *FirstEntry, prod ProductionID, head int) {
	fst.set[prod][head] = e
}

type firstComFrame struct {
	prodID ProductionID
	head   int
	prev   *firstComFrame
}

type firstComContext struct {
	prods    *productionSet
	first    *First
	frameTop *firstComFrame
}

func newfirstComContext(prods *productionSet) *firstComContext {
	return &firstComContext{
		prods:    prods,
		first:    newFirst(prods),
		frameTop: nil,
	}
}

func (cc *firstComContext) push(prod *production, head int) {
	cc.frameTop = &firstComFrame{
		prodID: prod.id,
		head:   head,
		prev:   cc.frameTop,
	}
}

func (cc *firstComContext) pop() {
	cc.frameTop = cc.frameTop.prev
}

func (cc *firstComContext) isAlreadyStacked(prod *production, head int) bool {
	for f := cc.frameTop; f != nil; f = f.prev {
		if f.prodID == prod.id && f.head == head {
			return true
		}
	}
	return false
}

func genFirst(prods *productionSet) (*First, error) {
	cc := newfirstComContext(prods)
	for _, prod := range prods.getAll() {
		if prod.isEmpty() {
			_, err := genFirstEntry(cc, prod, 0)
			if err != nil {
				return nil, err
			}
		} else {
			for i := 0; i < prod.rhsLen; i++ {
				_, err := genFirstEntry(cc, prod, i)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return cc.first, nil
}

func genFirstEntry(cc *firstComContext, prod *production, head int) (*FirstEntry, error) {
	if prod.isEmpty() {
		if head != 0 {
			return nil, fmt.Errorf("production passed is an empty rule but head is not 0; production: %v, head: %v", prod.id, head)
		}
	} else {
		if head < 0 || head >= prod.rhsLen {
			return nil, fmt.Errorf("head must be between 0 and %v; production: %v, head: %v", prod.rhsLen-1, prod.id, head)
		}
	}

	// guards for avoiding the infinite recursion
	{
		// A FIRST set is already computed
		if fst := cc.first.Get(prod.id, head); fst != nil {
			return fst, nil
		}

		if cc.isAlreadyStacked(prod, head) {
			return newFirstEntry(), nil
		}
	}

	cc.push(prod, head)
	defer cc.pop()

	// When the production is empty, a FIRST set contains the only EMPTY symbol.
	if prod.isEmpty() || head >= prod.rhsLen {
		e := newFirstEntry()
		e.addEmpty()
		cc.first.put(e, prod.id, head)

		return e, nil
	}

	headSym := prod.rhs[head]
	if headSym.isNil() {
		return nil, fmt.Errorf("head symbol must be a non-nil symbol; production: %v, head: %v", prod.id, head)
	}

	// When the head symbol is a terminal symbol, a FIRST set contains only it.
	if headSym.isTerminal() {
		e := newFirstEntry()
		e.add(headSym)
		cc.first.put(e, prod.id, head)

		return e, nil
	}

	entry := newFirstEntry()
	headSymProds, ok := cc.prods.findByLHS(headSym)
	if !ok {
		return nil, fmt.Errorf("production was not found; LHS: %v", headSym)
	}
	for _, headSymProd := range headSymProds {
		e, err := genFirstEntry(cc, headSymProd, 0)
		if err != nil {
			return nil, err
		}
		entry.mergeExceptEmpty(e)

		if e.empty {
			nextHead := head + 1
			if !headSymProd.isEmpty() && nextHead < headSymProd.rhsLen {
				f, err := genFirstEntry(cc, headSymProd, nextHead)
				if err != nil {
					return nil, err
				}
				entry.mergeExceptEmpty(f)
			} else {
				entry.addEmpty()
			}
		}
	}
	cc.first.put(entry, prod.id, head)

	return entry, nil
}
