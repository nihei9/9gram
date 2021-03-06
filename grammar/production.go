package grammar

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
)

type ProductionID [32]byte

func (id ProductionID) String() string {
	return hex.EncodeToString(id[:])
}

func genProductionID(lhs Symbol, rhs []Symbol) ProductionID {
	seq := lhs.Byte()
	for _, sym := range rhs {
		seq = append(seq, sym.Byte()...)
	}
	return ProductionID(sha256.Sum256(seq))
}

type ProductionNum uint16

const (
	ProductionNumStart = ProductionNum(1)

	// Avoid using 0 as a production number.
	// In ACTION table, 0 means an empty entry.
	productionNumMin = ProductionNum(2)
)

func (n ProductionNum) Int() int {
	return int(n)
}

type production struct {
	id     ProductionID
	num    ProductionNum
	lhs    Symbol
	rhs    []Symbol
	rhsLen int
}

func newProduction(lhs Symbol, rhs []Symbol) (*production, error) {
	if lhs.isNil() {
		return nil, fmt.Errorf("LHS must be a non-nil symbol; LHS: %v, RHS: %v", lhs, rhs)
	}
	for _, sym := range rhs {
		if sym.isNil() {
			return nil, fmt.Errorf("a symbol of RHS must be a non-nil symbol; LHS: %v, RHS: %v", lhs, rhs)
		}
	}

	p := &production{
		id:     genProductionID(lhs, rhs),
		lhs:    lhs,
		rhs:    rhs,
		rhsLen: len(rhs),
	}

	return p, nil
}

func (p *production) equals(q *production) bool {
	return q.id == p.id
}

func (p *production) isEmpty() bool {
	return p.rhsLen <= 0
}

type productionSet struct {
	lhs2Prods map[Symbol][]*production
	id2Prod   map[ProductionID]*production
	num       ProductionNum
}

func newProductionSet() *productionSet {
	return &productionSet{
		lhs2Prods: map[Symbol][]*production{},
		id2Prod:   map[ProductionID]*production{},
		num:       productionNumMin,
	}
}

func (ps *productionSet) append(prod *production) bool {
	if _, ok := ps.id2Prod[prod.id]; ok {
		return false
	}

	if prod.lhs.isStart() {
		prod.num = ProductionNumStart
	} else {
		prod.num = ps.num
		ps.num += 1
	}

	if prods, ok := ps.lhs2Prods[prod.lhs]; ok {
		ps.lhs2Prods[prod.lhs] = append(prods, prod)
	} else {
		ps.lhs2Prods[prod.lhs] = []*production{prod}
	}
	ps.id2Prod[prod.id] = prod

	return true
}

func (ps *productionSet) findByID(id ProductionID) (*production, bool) {
	prod, ok := ps.id2Prod[id]
	return prod, ok
}

func (ps *productionSet) findByLHS(lhs Symbol) ([]*production, bool) {
	if lhs.isNil() {
		return nil, false
	}

	prods, ok := ps.lhs2Prods[lhs]
	return prods, ok
}

func (ps *productionSet) getAll() map[ProductionID]*production {
	return ps.id2Prod
}

func PrintProductionSet(w io.Writer, prods *productionSet, symTab *SymbolTable) {
	if w == nil {
		return
	}

	var ps []*production
	for _, p := range prods.getAll() {
		ps = append(ps, p)
	}
	sort.Slice(ps, func(i, j int) bool {
		return ps[i].num < ps[j].num
	})

	for _, p := range ps {
		lhsText, ok := symTab.ToText(p.lhs)
		if !ok {
			lhsText = "<Symbol Not Found>"
		}
		fmt.Fprintf(w, "#%v: %v →", p.num, lhsText)
		for _, rhsSym := range p.rhs {
			rhsText, ok := symTab.ToText(rhsSym)
			if !ok {
				rhsText = "<Symbol Not Found>"
			}
			fmt.Fprintf(w, "　%v", rhsText)
		}
		fmt.Fprintf(w, "\n")
	}
}
