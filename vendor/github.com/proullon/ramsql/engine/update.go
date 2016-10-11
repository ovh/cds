package engine

import (
	"fmt"
	"strings"
	"time"

	"github.com/proullon/ramsql/engine/log"
	"github.com/proullon/ramsql/engine/parser"
	"github.com/proullon/ramsql/engine/protocol"
)

/*
|-> update
	|-> account
	|-> set
	      |-> email
					|-> =
					|-> roger@gmail.com
  |-> where
        |-> id
					|-> =
					|-> 2
*/
func updateExecutor(e *Engine, updateDecl *parser.Decl, conn protocol.EngineConn) error {
	var num int64

	updateDecl.Stringy(0)

	// Fetch table from name and write lock it
	r := e.relation(updateDecl.Decl[0].Lexeme)
	if r == nil {
		return fmt.Errorf("Table %s does not exists", updateDecl.Decl[0].Lexeme)
	}
	r.Lock()
	r.Unlock()

	// Set decl
	values, err := setExecutor(updateDecl.Decl[1])
	if err != nil {
		return err
	}

	// Where decl
	predicates, err := whereExecutor(updateDecl.Decl[2], r.table.name)
	if err != nil {
		return err
	}

	var ok, res bool
	for i := range r.rows {
		ok = true
		// If the row validate all predicates, write it
		for _, predicate := range predicates {
			if res, err = predicate.Evaluate(r.rows[i], r.table); err != nil {
				return err
			}
			if res == false {
				ok = false
				continue
			}
		}

		if ok {
			num++
			err = updateValues(r, i, values)
			if err != nil {
				return err
			}
		}
	}

	return conn.WriteResult(0, num)
}

/*
	|-> set
	      |-> email
					|-> =
					|-> roger@gmail.com
*/
func setExecutor(setDecl *parser.Decl) (map[string]interface{}, error) {

	values := make(map[string]interface{})

	for _, attr := range setDecl.Decl {
		values[attr.Lexeme] = attr.Decl[1].Lexeme
	}

	return values, nil
}

func updateValues(r *Relation, row int, values map[string]interface{}) error {

	for i := range r.table.attributes {
		val, ok := values[r.table.attributes[i].name]
		if !ok {
			continue
		}
		log.Debug("Type of '%s' is '%s'\n", r.table.attributes[i].name, r.table.attributes[i].typeName)
		switch strings.ToLower(r.table.attributes[i].typeName) {
		case "timestamp", "localtimestamp":
			s, ok := val.(string)
			if ok && (s == "current_timestamp" || s == "now()") {
				r.rows[row].Values[i] = fmt.Sprintf("%s", time.Now())
			} else {
				r.rows[row].Values[i] = fmt.Sprintf("%v", val)
			}
		default:
			r.rows[row].Values[i] = fmt.Sprintf("%v", val)
		}
	}

	return nil
}
