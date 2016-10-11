package engine

import (
	"fmt"
)

// PredicateLinker referes to AND and OR operators.
type PredicateLinker interface {
	Eval(v virtualRow) (bool, error)
}

type andOperator struct {
	pred []PredicateLinker
}

func (o *andOperator) Add(p PredicateLinker) {
	o.pred = append(o.pred, p)
}

func (o *andOperator) Eval(v virtualRow) (bool, error) {

	for i := range o.pred {
		ok, err := o.pred[i].Eval(v)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}

	return true, nil
}

type orOperator struct {
	pred []PredicateLinker
}

func (o *orOperator) Add(p PredicateLinker) {
	o.pred = append(o.pred, p)
}

func (o *orOperator) Eval(v virtualRow) (bool, error) {

	for i := range o.pred {
		ok, err := o.pred[i].Eval(v)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	return false, nil
}

// TruePredicate is a predicate wich return always true
var TruePredicate = Predicate{
	True: true,
}

// Value is a value given to predicates
type Value struct {
	v        interface{}
	valid    bool
	lexeme   string
	constant bool
	table    string
}

// Predicate evaluate if a condition is valid with 2 values and an operator on this 2 values
type Predicate struct {
	LeftValue  Value
	Operator   Operator
	RightValue Value
	True       bool
}

func (p Predicate) String() string {
	var left, right string

	if p.True {
		return "AlwaysTrue"
	}

	left = "?"
	right = "?"

	if p.LeftValue.valid {
		left = p.LeftValue.lexeme
	}

	if p.RightValue.valid {
		right = p.RightValue.lexeme
	}

	return fmt.Sprintf("[%s] vs [%s]", left, right)
}

// Eval fetches operand from virtual row and run operator
func (p *Predicate) Eval(row virtualRow) (bool, error) {

	if p.True {
		return true, nil
	}

	// Find left attribute
	left := p.LeftValue.table + "." + p.LeftValue.lexeme
	val, ok := row[left]
	if !ok {
		return false, fmt.Errorf("Attribute [%s] not found in row", left)
	}
	p.LeftValue.v = val.v

	return p.Operator(p.LeftValue, p.RightValue), nil
}

// Evaluate is deprecated (see Eval). It calls operators and use tuple as operand
// TODO: Delete that
func (p *Predicate) Evaluate(t *Tuple, table *Table) (bool, error) {

	if p.True {
		return true, nil
	}

	// Find left
	var i = 0
	lenTable := len(table.attributes)
	for i = 0; i < lenTable; i++ {
		if table.attributes[i].name == p.LeftValue.lexeme {
			break
		}
	}
	if i == lenTable {
		return false, fmt.Errorf("Attribute [%s] not found in table [%s]", p.LeftValue.lexeme, table.name)
	}

	p.LeftValue.v = t.Values[i]
	return p.Operator(p.LeftValue, p.RightValue), nil
}
