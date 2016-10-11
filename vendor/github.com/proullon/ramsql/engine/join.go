package engine

import (
	"fmt"
	"strings"

	"github.com/proullon/ramsql/engine/log"
	"github.com/proullon/ramsql/engine/parser"
	"github.com/proullon/ramsql/engine/protocol"
)

// virtualRow is the resultset after FROM and JOIN transformations
// The key of the map is the lexeme (table.attribute) of the value (i.e: user.name)
type virtualRow map[string]Value

func (v virtualRow) String() string {
	var l1, l2 string
	l1 = "\n"
	l2 = "\n"
	for key, val := range v {
		l1 = fmt.Sprintf("%s %25s", l1, key)
		l2 = fmt.Sprintf("%s %25v", l2, val.v)
	}
	return l1 + l2
}

// 3 types of predicates
// INNER, LEFT, RIGHT, FULL
// with NATURAL option
type joiner interface {
	Evaluate(virtualRow, *Relation, int) (bool, error)
	On() string
}

// default joiner implementation
type inner struct {
	table   string
	t1Value Value
	t2Value Value
}

func (i *inner) On() string {
	return i.table
}

func (i *inner) Evaluate(row virtualRow, r *Relation, index int) (bool, error) {
	var t1, t2 Value

	// I want t1 to be the attribute already in the virtual row
	// So if t1 table is the current one...swap !
	if i.t1Value.table == r.table.name {
		t1 = i.t2Value
		i.t2Value = i.t1Value
		i.t1Value = t1
	}

	// let's find t1Value
	val, ok := row[i.t1Value.table+"."+i.t1Value.lexeme]
	if !ok {
		return false, fmt.Errorf("JOIN: joining on %s, not found", i.t1Value.table+"."+i.t1Value.lexeme)
	}
	t1 = val

	if r.table.name != i.t2Value.table {
		return false, fmt.Errorf("JOIN: joining on table %s, got %s", i.t2Value.table, r.table.name)
	}
	for attrIndex, attr := range r.table.attributes {
		if attr.name == i.t2Value.lexeme {
			t2 = Value{
				v:      r.rows[index].Values[attrIndex],
				lexeme: attr.name,
				table:  r.table.name,
				valid:  true,
			}
			break
		}
	}
	if t2.valid == false {
		return false, fmt.Errorf("JOIN: joining on table %s, attribute %s not found", i.t2Value.table, i.t2Value.lexeme)
	}

	// let's say for now the only operator is '='
	if fmt.Sprintf("%v", t1.v) == fmt.Sprintf("%v", t2.v) {
		return true, nil
	}

	return false, nil
}

// The optional WHERE, GROUP BY, and HAVING clauses in the table expression specify a pipeline of successive transformations performed on the table derived in the FROM clause.
// All these transformations produce a virtual table that provides the rows that are passed to the select list to compute the output rows of the query.
func generateVirtualRows(e *Engine, attr []Attribute, conn protocol.EngineConn, t1Name string, joinPredicates []joiner, selectPredicates []PredicateLinker, functors []selectFunctor) error {

	// get t1 and lock it
	t1 := e.relation(t1Name)
	if t1 == nil {
		return fmt.Errorf("table %s not found", t1Name)
	}
	t1.RLock()
	defer t1.RUnlock()

	// all joined tables in a map of relation
	relations := make(map[string]*Relation)
	for _, j := range joinPredicates {
		r := e.relation(j.On())
		if r == nil {
			return fmt.Errorf("table %s not found", j.On())
		}
		r.RLock()
		defer r.RUnlock()
		relations[j.On()] = r
	}

	// Write header
	var header []string
	var alias []string
	for _, a := range attr {
		alias = append(alias, a.name)
		if strings.Contains(a.name, ".") == false {
			a.name = t1Name + "." + a.name
		}
		header = append(header, a.name)
	}

	// Initialize functors here
	for i := range functors {
		if err := functors[i].Init(e, conn, header, alias); err != nil {
			return err
		}
	}

	// for each row in t1
	for i := range t1.rows {
		// create virtualrow
		row := make(virtualRow)
		for index := range t1.rows[i].Values {
			v := Value{
				v:      t1.rows[i].Values[index],
				valid:  true,
				lexeme: t1.table.attributes[index].name,
				table:  t1Name,
			}
			row[v.table+"."+v.lexeme] = v
		}

		// for first join predicates
		err := join(row, relations, joinPredicates, 0, selectPredicates, functors)
		if err != nil {
			return err
		}

	}

	for i := range functors {
		err := functors[i].Done()
		if err != nil {
			return err
		}
	}
	return nil
}

// Recursive virtual row creation
func join(row virtualRow, relations map[string]*Relation, predicates []joiner, predicateIndex int, selectPredicates []PredicateLinker, functors []selectFunctor) error {

	// Skip directly to selectRows if there is no joiner to run
	if len(predicates) == 0 {
		return selectRows(row, selectPredicates, functors)
	}

	// get current predicates
	predicate := predicates[predicateIndex]

	// last := is it last join ?
	last := false
	if predicateIndex >= len(predicates)-1 {
		last = true
	}

	// for each row in relations[pred.Table()]
	r := relations[predicate.On()]
	for i := range r.rows {
		ok, err := predicate.Evaluate(row, r, i)
		if err != nil {
			return err
		}
		// if predicate not ok
		if !ok {
			continue
		}

		// combine columns to existing virtual row
		for index := range r.rows[i].Values {
			v := Value{
				v:      r.rows[i].Values[index],
				valid:  true,
				lexeme: r.table.attributes[index].name,
				table:  r.table.name,
			}
			row[v.table+"."+v.lexeme] = v
		}

		// if last predicate
		if last {
			err = selectRows(row, selectPredicates, functors)
		} else {
			err = join(row, relations, predicates, predicateIndex+1, selectPredicates, functors)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

/*
-> join
       |-> user_project
       |-> on
           |-> project_id
               |-> user_project
           |-> =
           |-> id
               |-> project

*/
func joinExecutor(decl *parser.Decl) (joiner, error) {
	decl.Stringy(0)

	j := &inner{}

	// Table name
	if decl.Decl[0].Token != parser.StringToken {
		return nil, fmt.Errorf("join: expected table name, got %v", decl.Decl[0])
	}
	j.table = decl.Decl[0].Lexeme

	// Predicate should be ON
	on := decl.Decl[1]
	if on.Token != parser.OnToken {
		return nil, fmt.Errorf("join: expected ON, got %v", on)
	}

	// Set first value
	j.t1Value.valid = true
	j.t1Value.lexeme = on.Decl[0].Lexeme
	j.t1Value.table = on.Decl[0].Decl[0].Lexeme

	// TODO: Skip operator here, expect '='

	// Set second value
	j.t2Value.valid = true
	j.t2Value.lexeme = on.Decl[2].Lexeme
	j.t2Value.table = on.Decl[2].Decl[0].Lexeme

	log.Debug("JOIN %s ON %s = %s !", j.table, j.t1Value.table+"."+j.t1Value.lexeme, j.t2Value.table+"."+j.t2Value.lexeme)
	return j, nil
}
