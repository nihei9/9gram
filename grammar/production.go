package grammar

import (
	"crypto/sha256"
	"fmt"
)

type ProductionID [32]byte

func genProductionID(lhs Symbol, rhs []Symbol) ProductionID {
	seq := lhs.Byte()
	for _, sym := range rhs {
		seq = append(seq, sym.Byte()...)
	}
	return ProductionID(sha256.Sum256(seq))
}

type production struct {
	id     ProductionID
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
}

func newProductionSet() *productionSet {
	return &productionSet{
		lhs2Prods: map[Symbol][]*production{},
		id2Prod:   map[ProductionID]*production{},
	}
}

func (ps *productionSet) append(prod *production) bool {
	if _, ok := ps.id2Prod[prod.id]; ok {
		return false
	}

	if prods, ok := ps.lhs2Prods[prod.lhs]; ok {
		ps.lhs2Prods[prod.lhs] = append(prods, prod)
	} else {
		ps.lhs2Prods[prod.lhs] = []*production{prod}
	}
	ps.id2Prod[prod.id] = prod

	return true
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
