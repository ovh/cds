package engine

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/proullon/ramsql/engine/log"
	"github.com/proullon/ramsql/engine/parser"
	"github.com/proullon/ramsql/engine/protocol"
)

func attributeExistsInTable(e *Engine, attr string, table string) error {

	r := e.relation(table)
	if r == nil {
		return fmt.Errorf("table \"%s\" does not exist", table)
	}

	found := false
	for _, tAttr := range r.table.attributes {
		if tAttr.name == attr {
			found = true
			break
		}

	}

	if !found {
		return fmt.Errorf("attribute %s does not exist in table %s", attr, table)
	}

	return nil
}

func attributesExistInTables(e *Engine, attributes []Attribute, tables []string) error {

	for _, attr := range attributes {
		if attr.name == "COUNT" {
			continue
		}

		if strings.Contains(attr.name, ".") {
			t := strings.Split(attr.name, ".")
			if err := attributeExistsInTable(e, t[1], t[0]); err != nil {
				return err
			}
			continue
		}

		found := 0
		for _, t := range tables {

			if err := attributeExistsInTable(e, attr.name, t); err == nil {
				found++
			}

			if found == 0 {
				return fmt.Errorf("attribute %s does not exist in tables %v", attr.name, tables)
			}
			if found > 1 {
				return fmt.Errorf("ambiguous attribute %s", attr.name)
			}
		}
	}

	return nil
}

/*
|-> SELECT
	|-> *
	|-> FROM
		|-> account
	|-> WHERE
		|-> email
			|-> =
			|-> foo@bar.com
*/
func selectExecutor(e *Engine, selectDecl *parser.Decl, conn protocol.EngineConn) error {
	var attributes []Attribute
	var tables []*Table
	var predicates []PredicateLinker
	var functors []selectFunctor
	var joiners []joiner
	var err error

	selectDecl.Stringy(0)
	for i := range selectDecl.Decl {
		switch selectDecl.Decl[i].Token {
		case parser.FromToken:
			// get selected tables
			tables = fromExecutor(selectDecl.Decl[i])
		case parser.WhereToken:
			// get WHERE declaration
			pred, err := whereExecutor2(e, selectDecl.Decl[i].Decl, tables[0].name)
			if err != nil {
				return err
			}
			predicates = []PredicateLinker{pred}
		case parser.JoinToken:
			j, err := joinExecutor(selectDecl.Decl[i])
			if err != nil {
				return err
			}
			joiners = append(joiners, j)
		case parser.OrderToken:
			orderFunctor, err := orderbyExecutor(selectDecl.Decl[i], tables)
			if err != nil {
				return err
			}
			functors = append(functors, orderFunctor)
		case parser.LimitToken:
			limit, err := strconv.Atoi(selectDecl.Decl[i].Decl[0].Lexeme)
			if err != nil {
				return fmt.Errorf("wrong limit value: %s", err)
			}
			conn = limitedConn(conn, limit)
		}
	}

	for i := range selectDecl.Decl {
		if selectDecl.Decl[i].Token != parser.StringToken &&
			selectDecl.Decl[i].Token != parser.StarToken &&
			selectDecl.Decl[i].Token != parser.CountToken {
			continue
		}

		// get attribute to selected
		attr, err := getSelectedAttribute(e, selectDecl.Decl[i], tables)
		if err != nil {
			return err
		}
		attributes = append(attributes, attr...)

	}

	if len(functors) == 0 {
		// Instanciate a new select functor
		functors, err = getSelectFunctors(selectDecl)
		if err != nil {
			return err
		}
	}

	err = generateVirtualRows(e, attributes, conn, tables[0].name, joiners, predicates, functors)
	if err != nil {
		return err
	}

	return nil
}

type selectFunctor interface {
	Init(e *Engine, conn protocol.EngineConn, attr []string, alias []string) error
	FeedVirtualRow(row virtualRow) error
	Done() error
}

// getSelectFunctors instanciate new functors for COUNT, MAX, MIN, AVG, ... and default select functor that return rows to client
// If a functor is specified, no attribute can be selected ?
func getSelectFunctors(attr *parser.Decl) ([]selectFunctor, error) {
	var functors []selectFunctor

	for i := range attr.Decl {

		if attr.Decl[i].Token == parser.FromToken {
			break
		}

		if attr.Decl[i].Token == parser.CountToken {
			f := &countSelectFunction{}
			functors = append(functors, f)
		}
	}

	if len(functors) == 0 {
		f := &defaultSelectFunction{}
		functors = append(functors, f)
	}

	return functors, nil

}

type defaultSelectFunction struct {
	e          *Engine
	conn       protocol.EngineConn
	attributes []string
	alias      []string
}

func (f *defaultSelectFunction) Init(e *Engine, conn protocol.EngineConn, attr []string, alias []string) error {
	f.e = e
	f.conn = conn
	f.attributes = attr
	f.alias = alias

	return f.conn.WriteRowHeader(f.alias)
}

func (f *defaultSelectFunction) FeedVirtualRow(vrow virtualRow) error {
	var row []string

	for _, attr := range f.attributes {
		val, ok := vrow[attr]
		if !ok {
			return fmt.Errorf("could not select attribute %s", attr)
		}
		row = append(row, fmt.Sprintf("%v", val.v))
	}

	return f.conn.WriteRow(row)
}

func (f *defaultSelectFunction) Done() error {
	return f.conn.WriteRowEnd()
}

type countSelectFunction struct {
	e          *Engine
	conn       protocol.EngineConn
	attributes []string
	alias      []string
	Count      int64
}

func (f *countSelectFunction) Init(e *Engine, conn protocol.EngineConn, attr []string, alias []string) error {
	f.e = e
	f.conn = conn
	f.attributes = attr
	f.alias = alias
	return nil
}

func (f *countSelectFunction) FeedVirtualRow(row virtualRow) error {
	f.Count++
	return nil
}

func (f *countSelectFunction) Done() error {
	err := f.conn.WriteRowHeader(f.alias)
	if err != nil {
		return err
	}

	err = f.conn.WriteRow([]string{fmt.Sprintf("%d", f.Count)})
	if err != nil {
		return err
	}

	return f.conn.WriteRowEnd()
}

func inExecutor(inDecl *parser.Decl, p *Predicate) error {
	inDecl.Stringy(0)

	p.Operator = inOperator

	// Put everything in a []string
	var values []string
	for i := range inDecl.Decl {
		log.Debug("inExecutor: Appending [%s]", inDecl.Decl[i].Lexeme)
		values = append(values, inDecl.Decl[i].Lexeme)
	}
	p.RightValue.v = values

	return nil
}

func isExecutor(isDecl *parser.Decl, p *Predicate) error {
	isDecl.Stringy(0)

	if isDecl.Decl[0].Token == parser.NullToken {
		p.Operator = isNullOperator
	} else {
		p.Operator = isNotNullOperator
	}

	return nil
}

func or(e *Engine, left []*parser.Decl, right []*parser.Decl, tableName string) (PredicateLinker, error) {
	p := &orOperator{}

	if len(left) > 0 {
		lPred, err := whereExecutor2(e, left, tableName)
		if err != nil {
			return nil, err
		}
		p.Add(lPred)
	}

	if len(right) > 0 {
		rPred, err := whereExecutor2(e, right, tableName)
		if err != nil {
			return nil, err
		}
		p.Add(rPred)
	}

	return p, nil
}

func and(e *Engine, left []*parser.Decl, right []*parser.Decl, tableName string) (PredicateLinker, error) {
	p := &andOperator{}

	if len(left) > 0 {
		lPred, err := whereExecutor2(e, left, tableName)
		if err != nil {
			return nil, err
		}
		p.Add(lPred)
	}

	if len(right) > 0 {
		rPred, err := whereExecutor2(e, right, tableName)
		if err != nil {
			return nil, err
		}
		p.Add(rPred)
	}

	return p, nil
}

func whereExecutor2(e *Engine, decl []*parser.Decl, fromTableName string) (PredicateLinker, error) {

	for i, cond := range decl {

		if cond.Token == parser.AndToken {
			if i+1 == len(decl) {
				return nil, fmt.Errorf("query error: AND not followed by any predicate")
			}

			p, err := and(e, decl[:i], decl[i+1:], fromTableName)
			return p, err
		}

		if cond.Token == parser.OrToken {
			if i+1 == len(decl) {
				return nil, fmt.Errorf("query error: OR not followd by any predicate")
			}
			p, err := or(e, decl[:i], decl[i+1:], fromTableName)
			return p, err
		}
	}

	p := &Predicate{}
	var err error
	cond := decl[0]

	// 1 PREDICATE
	if cond.Lexeme == "1" {
		return &TruePredicate, nil
	}

	switch cond.Decl[0].Token {
	case parser.IsToken, parser.InToken, parser.EqualityToken, parser.LeftDipleToken, parser.RightDipleToken:
		break
	default:
		fromTableName = cond.Decl[0].Lexeme
		cond.Decl = cond.Decl[1:]
		break
	}

	p.LeftValue.lexeme = cond.Lexeme

	if err := attributeExistsInTable(e, p.LeftValue.lexeme, fromTableName); err != nil {
		return nil, err
	}

	// Handle IN keyword
	if cond.Decl[0].Token == parser.InToken {
		err := inExecutor(cond.Decl[0], p)
		if err != nil {
			return nil, err
		}
		p.LeftValue.table = fromTableName
		return p, nil
	}

	// Handle IS NULL and IS NOT NULL
	if cond.Decl[0].Token == parser.IsToken {
		err := isExecutor(cond.Decl[0], p)
		if err != nil {
			return nil, err
		}
		p.LeftValue.table = fromTableName
		return p, nil
	}

	if len(cond.Decl) < 2 {
		return nil, fmt.Errorf("Malformed predicate \"%s\"", cond.Lexeme)
	}

	// The first element of the list is then the relation of the attribute
	op := cond.Decl[0]
	val := cond.Decl[1]

	p.Operator, err = NewOperator(op.Token, op.Lexeme)
	if err != nil {
		return nil, err
	}
	p.RightValue.lexeme = val.Lexeme
	p.RightValue.valid = true

	p.LeftValue.table = fromTableName
	return p, nil
}

/*
   |-> WHERE
	   |-> email
		   |-> =
		   |-> foo@bar.com
*/
func whereExecutor(whereDecl *parser.Decl, fromTableName string) ([]Predicate, error) {
	var predicates []Predicate
	var err error
	whereDecl.Stringy(0)

	for i := range whereDecl.Decl {
		var p Predicate
		tableName := fromTableName
		cond := whereDecl.Decl[i]

		// 1 PREDICATE
		if cond.Lexeme == "1" {
			predicates = append(predicates, TruePredicate)
			continue
		}

		if len(cond.Decl) == 0 {
			log.Debug("whereExecutor: HUm hum you must be AND or OR: %v", cond)
			continue
		}

		switch cond.Decl[0].Token {
		case parser.EqualityToken, parser.LeftDipleToken, parser.RightDipleToken:
			log.Debug("whereExecutor: it's = < >\n")
			break
		case parser.InToken:
			log.Debug("whereExecutor: it's IN\n")
			break
		case parser.IsToken:
			log.Debug("whereExecutor: it's IS token\n")
			log.Debug("whereExecutor: %+v\n", cond.Decl[0])
			break
		default:
			log.Debug("it's the table name ! -> %s", cond.Decl[0].Lexeme)
			tableName = cond.Decl[0].Lexeme
			cond.Decl = cond.Decl[1:]
			break
		}

		p.LeftValue.lexeme = whereDecl.Decl[i].Lexeme

		// Handle IN keyword
		if cond.Decl[0].Token == parser.InToken {
			err := inExecutor(cond.Decl[0], &p)
			if err != nil {
				return nil, err
			}
			p.LeftValue.table = tableName
			predicates = append(predicates, p)
			continue
		}

		// Handle IS NULL and IS NOT NULL
		if cond.Decl[0].Token == parser.IsToken {
			err := isExecutor(cond.Decl[0], &p)
			if err != nil {
				return nil, err
			}
			p.LeftValue.table = tableName
			predicates = append(predicates, p)
			continue
		}

		if len(cond.Decl) < 2 {
			return nil, fmt.Errorf("Malformed predicate \"%s\"", cond.Lexeme)
		}

		// The first element of the list is then the relation of the attribute
		op := cond.Decl[0]
		val := cond.Decl[1]

		p.Operator, err = NewOperator(op.Token, op.Lexeme)
		if err != nil {
			return nil, err
		}
		p.RightValue.lexeme = val.Lexeme
		p.RightValue.valid = true

		p.LeftValue.table = tableName

		predicates = append(predicates, p)
	}

	if len(predicates) == 0 {
		return nil, fmt.Errorf("No predicates provided")
	}

	return predicates, nil
}

/*
|-> FROM
	|-> account
*/
func fromExecutor(fromDecl *parser.Decl) []*Table {
	var tables []*Table
	for _, t := range fromDecl.Decl {
		tables = append(tables, NewTable(t.Lexeme))
	}

	return tables
}

func getSelectedAttribute(e *Engine, attr *parser.Decl, tables []*Table) ([]Attribute, error) {
	var attributes []Attribute
	var t []string

	for i := range tables {
		t = append(t, tables[i].name)
	}

	switch attr.Token {
	case parser.StarToken:
		for _, table := range tables {
			r := e.relation(table.name)
			if r == nil {
				return nil, errors.New("Relation " + table.name + " not found")
			}
			attributes = append(attributes, r.table.attributes...)
		}
	case parser.CountToken:
		err := attributesExistInTables(e, []Attribute{NewAttribute(attr.Decl[0].Lexeme, "", false)}, t)
		if err != nil && attr.Decl[0].Lexeme != "*" {
			return nil, err
		}
		attributes = append(attributes, NewAttribute("COUNT", "int", false))
	case parser.StringToken:
		attribute := attr.Lexeme
		if len(attr.Decl) > 0 {
			if err := attributeExistsInTable(e, attr.Lexeme, attr.Decl[0].Lexeme); err != nil {
				return nil, err
			}
			attribute = attr.Decl[0].Lexeme + "." + attribute
		}
		newAttr := NewAttribute(attribute, "text", false)
		if err := attributesExistInTables(e, []Attribute{newAttr}, t); err != nil {
			return nil, err
		}
		attributes = append(attributes, newAttr)
	}

	return attributes, nil
}

// Perform actual check of predicates present in virtualrow.
func selectRows(row virtualRow, predicates []PredicateLinker, functors []selectFunctor) error {
	var res bool
	var err error

	// If the row validate all predicates, write it
	for _, predicate := range predicates {
		if res, err = predicate.Eval(row); err != nil {
			return err
		}
		if res == false {
			return nil
		}
	}

	for i := range functors {
		err := functors[i].FeedVirtualRow(row)
		if err != nil {
			return err
		}
	}
	return nil
}
